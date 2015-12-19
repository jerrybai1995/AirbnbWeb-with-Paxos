package airbnbrpc

type RemoteAirbnbNode interface {
	RegisterUser(args *RegisterUserArgs, reply *RegisterUserReply) error
	CheckValidity(args *RegisterUserArgs, reply *CheckValidityReply) error
	AddNewHouse(args *AddNewHouseArgs, reply *RegisterUserReply) error
	SelectHousing(args *SelectHousingArgs, reply *RegisterUserReply) error
	FetchCurrentReservations(args *FetchArgs, reply *FetchReply) error
	SearchHousingBy(args *SearchArgs, reply *FetchReply) error
}

type AirbnbNode struct {
	RemoteAirbnbNode
}

func Wrap(t RemoteAirbnbNode) RemoteAirbnbNode {
	return &AirbnbNode{t}
}
