package paxos

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/cmu440-F15/paxosapp/rpc/paxosrpc"
	"net"
	"net/http"
	"net/rpc"
	"sync"
	"time"
)

type CommitInfo struct {
	Info *paxosrpc.CommitArgs `json:"info"`
}

type paxosNode struct {
	numNodes int
	cliMap   map[int]*rpc.Client // This should be filled before NewPaxosNode() returns.
	srvId    int
	/* BOOKKEEPING */
	highestPropNumKeyMap map[string]int                  // We must have separate bookkeeping for the highest proposal number seen.
	highestAcceValKeyMap map[string]*paxosrpc.AcceptArgs // Maps from key to the highest accepted key-value so far. WILL BE USED IN RecvPrepare and RecvAccept.

	dataStorage        map[string]*CommitInfo // Maps from key to the struct containing key-value pair.
	myHostPort         string
	connCounterLock    *sync.Mutex
	connCounter        int
	failCounter        int
	highestPropLock    *sync.Mutex
	cliMapLock         *sync.Mutex
	propNumCounterLock *sync.Mutex
	highestAcceLock    *sync.Mutex
	commitLock         *sync.Mutex
	catchupLock        *sync.RWMutex
	proposalNumCounter int
	commitSigChanLock  *sync.Mutex
	commitSigChanMap   map[string]chan interface{}
}

// NewPaxosNode creates a new PaxosNode. This function should return only when
// all nodes have joined the ring, and should return a non-nil error if the node
// could not be started in spite of dialing the other nodes numRetries times.
//
// hostMap is a map from node IDs to their hostports, numNodes is the number
// of nodes in the ring, replace is a flag which indicates whether this node
// is a replacement for a node which failed.
func NewPaxosNode(myHostPort string, hostMap map[int]string, numNodes, srvId, numRetries int, replace bool) (PaxosNode, error) {
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

	newPaxos := &paxosNode{
		numNodes:             numNodes,
		cliMap:               make(map[int]*rpc.Client),
		srvId:                srvId,
		highestPropNumKeyMap: make(map[string]int),
		highestAcceValKeyMap: make(map[string]*paxosrpc.AcceptArgs),
		dataStorage:          make(map[string]*CommitInfo),
		myHostPort:           myHostPort,
		connCounterLock:      &sync.Mutex{},
		connCounter:          0,
		failCounter:          0,
		highestPropLock:      &sync.Mutex{},
		cliMapLock:           &sync.Mutex{},
		propNumCounterLock:   &sync.Mutex{},
		highestAcceLock:      &sync.Mutex{},
		commitLock:           &sync.Mutex{},
		catchupLock:          &sync.RWMutex{},
		proposalNumCounter:   1,
		commitSigChanLock:    &sync.Mutex{},
		commitSigChanMap:     make(map[string]chan interface{}),
	}
	for {
		err2 := rpc.RegisterName("PaxosNode", paxosrpc.Wrap(newPaxos))
		if err2 != nil {
			fmt.Println(err2)
			time.Sleep(500 * time.Millisecond)
		} else {
			break
		}
	}

	rpc.HandleHTTP()
	go http.Serve(listener, nil)

	for id, hp := range hostMap {
		go newPaxos.DialNode(id, hp, numRetries)
	}

	for {
		if newPaxos.connCounter < numNodes {
			// If there is any failure when we establish the connection, return error.
			if newPaxos.failCounter > 0 {
				return nil, errors.New(fmt.Sprintf("Failed to connect to node with server ID %d", newPaxos.failCounter-1))
			} else {
				time.Sleep(1 * time.Second)
			}
		} else {
			break
		}
	}

	if replace {
		// Tell other nodes to replace
		replaceArgs := &paxosrpc.ReplaceServerArgs{srvId, myHostPort}
		var replaceReply paxosrpc.ReplaceServerReply
		for _, cli := range newPaxos.cliMap {
			// Shouldn't fail
			cli.Call("PaxosNode.RecvReplaceServer", replaceArgs, &replaceReply)
		}

		go newPaxos.CatchupData()
	}

	return newPaxos, nil
}

func (pn *paxosNode) CatchupData() {
	// Catchup
	var replaceCatchupArgs paxosrpc.ReplaceCatchupArgs
	replaceCatchupReply := &paxosrpc.ReplaceCatchupReply{}
	for _, cli := range pn.cliMap {
		err := cli.Call("PaxosNode.RecvReplaceCatchup", &replaceCatchupArgs, replaceCatchupReply)
		if err == nil {
			dict := make(map[string]*CommitInfo)
			json.Unmarshal(replaceCatchupReply.Data, &dict)
			pn.commitLock.Lock()
			for s, ci := range dict {
				pn.dataStorage[s] = ci
			}
			pn.commitLock.Unlock()
			break
		}
	}
}

// Dial another paxos node with node ID id, hostport hp and upper bound for dial
// retries given by numRetries. If numRetries seconds elapse and there is no connection
// established then we return an error.
func (pn *paxosNode) DialNode(id int, hp string, numRetries int) error {
	dialCount := 0
	for {
		cli, err := rpc.DialHTTP("tcp", hp)
		if err != nil {
			dialCount += 1
			if dialCount == numRetries {
				pn.failCounter = id + 1
				return errors.New(fmt.Sprintf("Failed to connect to node with server ID %d", id))
			}
			time.Sleep(1 * time.Second)
		} else {
			pn.cliMapLock.Lock()
			pn.cliMap[id] = cli
			pn.cliMapLock.Unlock()
			break
		}
	}
	pn.connCounterLock.Lock()
	pn.connCounter += 1
	pn.connCounterLock.Unlock()
	return nil
}

func (pn *paxosNode) GetNextProposalNumber(args *paxosrpc.ProposalNumberArgs, reply *paxosrpc.ProposalNumberReply) error {
	pn.propNumCounterLock.Lock()
	defer pn.propNumCounterLock.Unlock()

	// Every time this function is called, we return a number that is exactly pn.numNodes higher
	// than the number we returned last time. Note that the integer set is partitioned based on
	// the modulus space (mod pn.numNodes) and each node occupies the space that pn.srvId belongs to.
	reply.N = pn.proposalNumCounter*pn.numNodes + pn.srvId
	pn.proposalNumCounter += 1
	return nil
}

// Adds a (key, value) pair. This function responds with the committed value in
// the reply of the RPC, or returns an error. It should not return until a value
// is successfully committed or an error occurs. Note that the committed value
// may not be the same as the value passed to Propose.
func (pn *paxosNode) Propose(args *paxosrpc.ProposeArgs, reply *paxosrpc.ProposeReply) error {
	proposeFinish := make(chan interface{}, 1)
	// Start a new goroutine to do the propose, and meanwhile start the 15 seconds timer.
	go pn.ConPropose(args, proposeFinish)
	timeout := time.After(15 * time.Second)

	select {
	case <-timeout:
		return errors.New("Nothing committed within 15 seconds.")
	case v := <-proposeFinish:
		reply.V = v
	}

	return nil
}

func (pn *paxosNode) ConPropose(args *paxosrpc.ProposeArgs, sigChan chan interface{}) error {
	pn.catchupLock.RLock()
	defer pn.catchupLock.RUnlock()

	value := args.V
	// Phase 1: Prepare
	ok := 0
	reject := 0

	highestAcceptedNum := -1
	var highestAccepted interface{}

	prepArgs := &paxosrpc.PrepareArgs{args.Key, args.N}
	prepareSummary := make(chan paxosrpc.PrepareReply, pn.numNodes)

	timeout := time.After(15 * time.Second)
	for _, cli := range pn.cliMap {
		go func(c *rpc.Client) {
			var prepReply paxosrpc.PrepareReply
			c.Call("PaxosNode.RecvPrepare", prepArgs, &prepReply)
			prepareSummary <- prepReply
		}(cli)
	}

	for i := 0; i < pn.numNodes; i++ {
		select {
		case prepReply := <-prepareSummary:
			if prepReply.Status == paxosrpc.OK {
				ok += 1
				// Keep track of the highest n_a and v_a received.
				if prepReply.N_a > highestAcceptedNum {
					highestAcceptedNum = prepReply.N_a
					highestAccepted = prepReply.V_a
				}

			} else {
				reject += 1
			}
		case <-timeout:
			return nil
		}
	}

	if reject > (pn.numNodes / 2) {
		commitSig := make(chan interface{}, 1)
		pn.commitSigChanLock.Lock()
		pn.commitSigChanMap[args.Key] = commitSig
		defer delete(pn.commitSigChanMap, args.Key)
		pn.commitSigChanLock.Unlock()
		for {
			select {
			case <-timeout:
				return nil
			case value = <-commitSig:
				sigChan <- value
				return nil
			}
		}
		// Should never reach here
		return errors.New("Nothing committed but should not be here.")
	}

	// Phase 2: Accept (with highestAccepted and args.N)
	if highestAcceptedNum >= 0 {
		value = highestAccepted
	}

	ok = 0
	reject = 0

	acceptArgs := &paxosrpc.AcceptArgs{args.Key, args.N, value}
	acceptSummary := make(chan paxosrpc.AcceptReply, pn.numNodes)

	for _, cli := range pn.cliMap {
		go func(c *rpc.Client) {
			var acceptReply paxosrpc.AcceptReply
			c.Call("PaxosNode.RecvAccept", acceptArgs, &acceptReply)
			acceptSummary <- acceptReply
		}(cli)
	}

	for i := 0; i < pn.numNodes; i++ {
		select {
		case acceptReply := <-acceptSummary:
			if acceptReply.Status == paxosrpc.OK {
				ok += 1
			} else {
				reject += 1
			}
		case <-timeout:
			return nil
		}
	}

	if reject > (pn.numNodes / 2) {
		commitSig := make(chan interface{}, 1)
		pn.commitSigChanLock.Lock()
		pn.commitSigChanMap[args.Key] = commitSig
		defer delete(pn.commitSigChanMap, args.Key)
		pn.commitSigChanLock.Unlock()
		for {
			select {
			case <-timeout:
				return nil
			case value = <-commitSig:
				sigChan <- value
				return nil
			}
		}
		// Should never reach here
		return errors.New("Nothing committed but should not be here.")
	}

	// Phase 3: Commit
	commitArgs := &paxosrpc.CommitArgs{args.Key, value}
	commitSummary := make(chan paxosrpc.CommitReply, pn.numNodes)
	for _, cli := range pn.cliMap {
		go func(c *rpc.Client) {
			var commitReply paxosrpc.CommitReply
			c.Call("PaxosNode.RecvCommit", commitArgs, &commitReply)
			commitSummary <- commitReply
		}(cli)
	}

	for i := 0; i < pn.numNodes; i++ {
		select {
		case <-commitSummary:
			continue
		case <-timeout:
			return nil
		}
	}

	sigChan <- value
	return nil
}

// Gets the value for a given key. This function should NOT block. If a key is
// not found, it should return KeyNotFound. It checks committed values only.
func (pn *paxosNode) GetValue(args *paxosrpc.GetValueArgs, reply *paxosrpc.GetValueReply) error {
	pn.commitLock.Lock()
	cInfo, ok := pn.dataStorage[args.Key]
	pn.commitLock.Unlock()
	if ok {
		reply.V = cInfo.Info.V
		reply.Status = paxosrpc.KeyFound
	} else {
		reply.Status = paxosrpc.KeyNotFound
	}
	return nil
}

// An RPC service that is provided to other nodes. When other nodes call RecvPrepare,
// it is equivalent to them sending a prepare message. The reply will be based on the
// highest proposal number we have seen; either we reject the prepare or we accept it
// and send back the highest accepted value we contains.
func (pn *paxosNode) RecvPrepare(args *paxosrpc.PrepareArgs, reply *paxosrpc.PrepareReply) error {
	pn.highestPropLock.Lock()
	highestSeen, ok := pn.highestPropNumKeyMap[args.Key]
	if !ok {
		highestSeen = -1
	}
	pn.highestPropLock.Unlock()

	if args.N < highestSeen {
		reply.Status = paxosrpc.Reject
		reply.N_a = -1
	} else {
		pn.highestPropLock.Lock()
		pn.highestPropNumKeyMap[args.Key] = args.N
		pn.highestPropLock.Unlock()

		pn.highestAcceLock.Lock()
		accepted, exists := pn.highestAcceValKeyMap[args.Key]
		pn.highestAcceLock.Unlock()

		reply.Status = paxosrpc.OK
		if exists {
			// We reply the info of th previous accepted proposal
			reply.N_a = accepted.N
			reply.V_a = accepted.V
		} else {
			reply.N_a = -1
		}
	}
	return nil
}

// An RPC service that is provided to other nodes. When other nodes call RecvAccept,
// it is equivalent to them sending an accept message. The reply will be based on the
// highest proposal number we have seen so far. If the received number is too small, then
// we reject to accept; if it is larger than the largest proposal number we have seen so
// far then we update n_a and v_a.
func (pn *paxosNode) RecvAccept(args *paxosrpc.AcceptArgs, reply *paxosrpc.AcceptReply) error {
	pn.highestPropLock.Lock()
	highestSeen, ok := pn.highestPropNumKeyMap[args.Key]
	if !ok {
		pn.highestPropNumKeyMap[args.Key] = args.N
		highestSeen = args.N
	}
	pn.highestPropLock.Unlock()

	if args.N < highestSeen {
		reply.Status = paxosrpc.Reject
	} else {
		pn.highestPropLock.Lock()
		pn.highestPropNumKeyMap[args.Key] = args.N
		pn.highestPropLock.Unlock()

		pn.highestAcceLock.Lock()
		pn.highestAcceValKeyMap[args.Key] = args
		pn.highestAcceLock.Unlock()
		reply.Status = paxosrpc.OK
	}
	return nil
}

// An RPC service that is provided to other nodes. When other nodes call RecvCommit,
// it is equivalent to them sending a commit message. Commit is mandatory
func (pn *paxosNode) RecvCommit(args *paxosrpc.CommitArgs, reply *paxosrpc.CommitReply) error {
	// A mandatory commit command
	pn.commitLock.Lock()
	defer pn.commitLock.Unlock()
	pn.dataStorage[args.Key] = &CommitInfo{args}
	pn.commitSigChanLock.Lock()
	sigChan, ok := pn.commitSigChanMap[args.Key]
	if ok {
		sigChan <- args.V
	}
	pn.commitSigChanLock.Unlock()

	pn.highestAcceLock.Lock()
	delete(pn.highestAcceValKeyMap, args.Key) // Clear the bookkeeping data
	pn.highestAcceLock.Unlock()

	pn.highestPropLock.Lock()
	delete(pn.highestPropNumKeyMap, args.Key) // Clear the bookkeeping data
	pn.highestPropLock.Unlock()

	return nil
}

func (pn *paxosNode) RecvReplaceServer(args *paxosrpc.ReplaceServerArgs, reply *paxosrpc.ReplaceServerReply) error {
	go pn.DialNode(args.SrvID, args.Hostport, 10)
	return nil
}

func (pn *paxosNode) RecvReplaceCatchup(args *paxosrpc.ReplaceCatchupArgs, reply *paxosrpc.ReplaceCatchupReply) error {
	pn.catchupLock.Lock()
	defer pn.catchupLock.Unlock()

	pn.commitLock.Lock()
	defer pn.commitLock.Unlock()

	data, _ := json.Marshal(&pn.dataStorage)
	reply.Data = data
	return nil
}
