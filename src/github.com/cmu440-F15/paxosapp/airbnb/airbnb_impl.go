package airbnb

import (
	"encoding/gob"
	"errors"
	"github.com/cmu440-F15/paxosapp/paxos"
	"github.com/cmu440-F15/paxosapp/rpc/airbnbrpc"
	"github.com/cmu440-F15/paxosapp/rpc/paxosrpc"
	"net"
	"net/http"
	"net/rpc"
	"strconv"
	"sync"
	"time"
)

type airbnbNode struct {
	cliMap       map[int]*rpc.Client
	userc        int
	usercLock    *sync.Mutex
	houseid      int
	houseidLock  *sync.Mutex
	reserved     map[string][]int
	reservedLock *sync.Mutex
}

type userInfo struct {
	Username string
	Password string
	Id       int
}

func NewAirbnbNode(myHostPort string, hostMap map[int]string, numNodes int) (AirbnbNode, error) {
	// Step 1: initialize all paxosNode
	for srvId, hostPort := range hostMap {
		go paxos.NewPaxosNode(hostPort, hostMap, numNodes, srvId, 10, false)
	}

	newAirbnb := &airbnbNode{
		cliMap:       make(map[int]*rpc.Client),
		userc:        0,
		usercLock:    &sync.Mutex{},
		houseid:      0,
		houseidLock:  &sync.Mutex{},
		reserved:     make(map[string][]int),
		reservedLock: &sync.Mutex{},
	}

	for id, hp := range hostMap {
		cli, err := rpc.DialHTTP("tcp", hp)
		if err != nil {
			return nil, errors.New("Failed to connect to new Paxos Node")
		}
		newAirbnb.cliMap[id] = cli
	}

	// Step 2: start listening
	var listener net.Listener
	var err error
	for {
		listener, err = net.Listen("tcp", myHostPort)
		if err != nil {
			time.Sleep(500 * time.Millisecond)
		} else {
			break
		}
	}

	for {
		err2 := rpc.RegisterName("AirbnbNode", airbnbrpc.Wrap(newAirbnb))
		if err2 != nil {
			time.Sleep(500 * time.Millisecond)
		} else {
			break
		}
	}

	go http.Serve(listener, nil)

	return newAirbnb, nil
}

func userInfoEq(a, b userInfo) bool {
	if a.Password == b.Password && a.Username == b.Username && a.Id == b.Id {
		return true
	}
	return false
}

func (an *airbnbNode) RegisterUser(args *airbnbrpc.RegisterUserArgs, reply *airbnbrpc.RegisterUserReply) error {
	// Dont allow it to be house since we are using this key, also cant be empty
	if args.Username == "house" || args.Username == "" {
		reply.Status = airbnbrpc.Reject
		return nil
	}

	// First try to get the username and see if the username already exists
	flag := false
	getValArgs := &paxosrpc.GetValueArgs{args.Username}
	for _, cli := range an.cliMap {

		var getValReply paxosrpc.GetValueReply
		err := cli.Call("PaxosNode.GetValue", getValArgs, &getValReply)
		if err == nil {
			if getValReply.Status == paxosrpc.KeyFound {
				reply.Status = airbnbrpc.Reject
				return nil
			} else {
				flag = true
				break
			}
		}
	}

	// If the username is valid, then try to register it
	if flag {
		pN := -1
		for _, cli := range an.cliMap {
			proposeNumArges := &paxosrpc.ProposalNumberArgs{args.Username}
			var proposeNumReply paxosrpc.ProposalNumberReply
			err := cli.Call("PaxosNode.GetNextProposalNumber", proposeNumArges, &proposeNumReply)
			if err == nil {
				pN = proposeNumReply.N
				an.usercLock.Lock()
				newValue := userInfo{
					Username: args.Username,
					Password: args.Password,
					Id:       an.userc,
				}

				an.userc += 1
				an.usercLock.Unlock()
				gob.Register(newValue)
				proposeArges := &paxosrpc.ProposeArgs{pN, args.Username, newValue}
				var proposeReply paxosrpc.ProposeReply
				err2 := cli.Call("PaxosNode.Propose", proposeArges, &proposeReply)
				if err2 == nil && userInfoEq(proposeReply.V.(userInfo), newValue) {
					// Here we use paxos to make sure that even if someone else is trying
					// to register the same username with a different password at the same
					// time, only one will succeed.
					reply.Status = airbnbrpc.OK
					return nil
				}
				break
			}
		}
	}

	reply.Status = airbnbrpc.Reject
	return nil
}

func (an *airbnbNode) CheckValidity(args *airbnbrpc.RegisterUserArgs, reply *airbnbrpc.CheckValidityReply) error {
	// Just get the value from paxos and check that passwords match with each other
	getValArgs := &paxosrpc.GetValueArgs{args.Username}
	for _, cli := range an.cliMap {
		var getValReply paxosrpc.GetValueReply
		err := cli.Call("PaxosNode.GetValue", getValArgs, &getValReply)
		if err == nil {
			if getValReply.Status == paxosrpc.KeyNotFound {
				reply.Valid = false
				return nil
			} else {
				if getValReply.V.(userInfo).Password == args.Password {
					reply.Valid = true
					reply.HUsername = getValReply.V.(userInfo).Username + strconv.Itoa(getValReply.V.(userInfo).Id)
					return nil
				} else {
					reply.Valid = false
					return nil
				}
			}
		}
	}
	reply.Valid = false
	return nil
}

// Helper function used to update info within the node
func updateReserved(an *airbnbNode, userid string, houseid int) {
	an.reservedLock.Lock()
	defer an.reservedLock.Unlock()
	original, ok := an.reserved[userid]
	if ok {
		original = append(original, houseid)
		an.reserved[userid] = original

	} else {
		var a = make([]int, 1, 1)
		a[0] = houseid
		an.reserved[userid] = a
	}
}

func (an *airbnbNode) AddNewHouse(args *airbnbrpc.AddNewHouseArgs, reply *airbnbrpc.RegisterUserReply) error {
	// Just try to propose without checking since we allow people to post same info several times
	an.houseidLock.Lock()
	newHouse := airbnbrpc.HousingInfo{
		Id:       an.houseid,
		Owner:    args.Owner,
		Address:  args.Address,
		City:     args.City,
		Zip:      args.Zip,
		Rating:   args.Rating,
		Downtown: args.Downtown,
		Price:    args.Price,
		Reserved: "",
	}
	an.houseid += 1
	an.houseidLock.Unlock()
	key := "house" + strconv.Itoa(newHouse.Id)

	for _, cli := range an.cliMap {
		proposeNumArges := &paxosrpc.ProposalNumberArgs{newHouse.Owner}
		var proposeNumReply paxosrpc.ProposalNumberReply
		err := cli.Call("PaxosNode.GetNextProposalNumber", proposeNumArges, &proposeNumReply)
		if err == nil {
			gob.Register(newHouse)
			proposeArges := &paxosrpc.ProposeArgs{proposeNumReply.N, key, newHouse}
			var proposeReply paxosrpc.ProposeReply
			err2 := cli.Call("PaxosNode.Propose", proposeArges, &proposeReply)
			if err2 == nil {
				reply.Status = airbnbrpc.OK
				return nil
			}
		}
	}
	reply.Status = airbnbrpc.Reject
	return nil
}

func (an *airbnbNode) SelectHousing(args *airbnbrpc.SelectHousingArgs, reply *airbnbrpc.RegisterUserReply) error {
	// We use house+id as key to store housing info in the paxos so that if several people are trying to reserve
	// the same house at the same time, only one will succeed.
	flag := false
	key := "house" + strconv.Itoa(args.HouseId)
	var house airbnbrpc.HousingInfo
	getValArgs := &paxosrpc.GetValueArgs{key}
	for _, cli := range an.cliMap {
		var getValReply paxosrpc.GetValueReply
		err := cli.Call("PaxosNode.GetValue", getValArgs, &getValReply)
		if err == nil {
			if getValReply.Status == paxosrpc.KeyNotFound || getValReply.V.(airbnbrpc.HousingInfo).Reserved != "" {
				reply.Status = airbnbrpc.Reject
				return nil
			} else {
				flag = true
				house = getValReply.V.(airbnbrpc.HousingInfo)
				break
			}
		}
	}

	if flag {
		house.Reserved = args.UserId
		for _, cli := range an.cliMap {
			proposeNumArges := &paxosrpc.ProposalNumberArgs{house.Owner}
			var proposeNumReply paxosrpc.ProposalNumberReply
			err := cli.Call("PaxosNode.GetNextProposalNumber", proposeNumArges, &proposeNumReply)
			if err == nil {
				gob.Register(house)
				proposeArges := &paxosrpc.ProposeArgs{proposeNumReply.N, key, house}
				var proposeReply paxosrpc.ProposeReply
				err2 := cli.Call("PaxosNode.Propose", proposeArges, &proposeReply)
				// Check that the userid is indeed the guy who succeed in the competition
				if err2 == nil && proposeReply.V.(airbnbrpc.HousingInfo).Reserved == args.UserId {
					reply.Status = airbnbrpc.OK
					updateReserved(an, args.UserId, args.HouseId)
					return nil
				} else {
					break
				}
			}
		}
	}

	reply.Status = airbnbrpc.Reject
	return nil
}

func (an *airbnbNode) FetchCurrentReservations(args *airbnbrpc.FetchArgs, reply *airbnbrpc.FetchReply) error {
	// We store the info of user locally at the airbnbNode, and fetch the data from paxos when called
	l, ok := an.reserved[args.UserId]
	if ok {
		rst := make([]airbnbrpc.HousingInfo, 0, 0)
		for _, i := range l {
			key := "house" + strconv.Itoa(i)
			getValArgs := &paxosrpc.GetValueArgs{key}
			for _, cli := range an.cliMap {
				var getValReply paxosrpc.GetValueReply
				err := cli.Call("PaxosNode.GetValue", getValArgs, &getValReply)
				if err == nil {
					rst = append(rst, getValReply.V.(airbnbrpc.HousingInfo))
					break
				}
			}
		}
		reply.Rst = rst
		return nil
	}
	reply.Rst = make([]airbnbrpc.HousingInfo, 0, 0)
	return nil
}

func (an *airbnbNode) SearchHousingBy(args *airbnbrpc.SearchArgs, reply *airbnbrpc.FetchReply) error {
	// Just get all houses and fetch the matched data
	rst := make([]airbnbrpc.HousingInfo, 0, 0)
	i := 0
	an.houseidLock.Lock()
	total := an.houseid
	an.houseidLock.Unlock()
	for i < total {
		key := "house" + strconv.Itoa(i)
		getValArgs := &paxosrpc.GetValueArgs{key}
		for _, cli := range an.cliMap {
			var getValReply paxosrpc.GetValueReply
			err := cli.Call("PaxosNode.GetValue", getValArgs, &getValReply)
			if err == nil {
				if getValReply.V.(airbnbrpc.HousingInfo).Reserved == "" && getValReply.V.(airbnbrpc.HousingInfo).City == args.Metric {
					rst = append(rst, getValReply.V.(airbnbrpc.HousingInfo))
				}
				break
			}
		}
		i += 1
	}
	reply.Rst = rst
	return nil
}
