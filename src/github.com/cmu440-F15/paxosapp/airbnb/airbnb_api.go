package airbnb

import (
	"github.com/cmu440-F15/paxosapp/rpc/airbnbrpc"
)

// RegisterUser registered a new user into the system, return Status OK if it is done successfully
// CheckValidity is used for login, it will return the Status OK if the username
// and password exists and matched
// AddNewHouse add a new house into the system using paxos rpc call
// SelectHousing reserve a house for the user and return a Status OK if it is done successfully
// FetchCurrentReservations get current reservation of a user
// SearchHousingBy fetch and select those house with the key word provided
type AirbnbNode interface {
	RegisterUser(args *airbnbrpc.RegisterUserArgs, reply *airbnbrpc.RegisterUserReply) error
	CheckValidity(args *airbnbrpc.RegisterUserArgs, reply *airbnbrpc.CheckValidityReply) error
	AddNewHouse(args *airbnbrpc.AddNewHouseArgs, reply *airbnbrpc.RegisterUserReply) error
	SelectHousing(args *airbnbrpc.SelectHousingArgs, reply *airbnbrpc.RegisterUserReply) error
	FetchCurrentReservations(args *airbnbrpc.FetchArgs, reply *airbnbrpc.FetchReply) error
	SearchHousingBy(args *airbnbrpc.SearchArgs, reply *airbnbrpc.FetchReply) error
}
