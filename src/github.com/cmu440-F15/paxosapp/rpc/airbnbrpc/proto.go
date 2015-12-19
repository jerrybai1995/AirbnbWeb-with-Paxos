// This file contains constants and arguments used to perform RPCs for airbnbNode

package airbnbrpc

// Status represents the status of a RPC's reply.
type Status int
type Lookup int

type HousingInfo struct {
	Id       int /* The id of this housing */
	Owner    string
	Address  string
	City     string
	Zip      string
	Rating   string /* Rating of this accommodation, out of 5 */
	Downtown bool   /* Is this house/apartment in downtown? Just make it up. */
	Price    string
	Reserved string
}

const (
	OK Status = iota + 1
	Reject
)

const (
	KeyFound    Lookup = iota + 1 // GetValue key found
	KeyNotFound                   // GetValue key not found
)

type RegisterUserArgs struct {
	Username string
	Password string
}

type RegisterUserReply struct {
	Status Status
}

type CheckValidityReply struct {
	Valid     bool
	HUsername string
}

type AddNewHouseArgs struct {
	Owner    string
	Address  string
	City     string
	Zip      string
	Rating   string /* Rating of this accommodation, out of 5 */
	Downtown bool   /* Is this house/apartment in downtown? Just make it up. */
	Price    string
}

type AddNewHouseReply struct {
	Status Status
}

type SelectHousingArgs struct {
	UserId  string
	HouseId int
}

type FetchArgs struct {
	UserId string
}

type FetchReply struct {
	Rst []HousingInfo
}

type SearchArgs struct {
	Metric string
}
