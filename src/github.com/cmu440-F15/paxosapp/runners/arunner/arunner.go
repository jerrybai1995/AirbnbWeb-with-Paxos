package main

import (
	"fmt"
	"github.com/cmu440-F15/paxosapp/airbnb"
)

func main() {
	myHostPort := "localhost:9999"

	// Default we have 1 paxos nodes
	hostMap := make(map[int]string)
	hostMap[0] = "localhost:9990"

	_, err := airbnb.NewAirbnbNode(myHostPort, hostMap, 1)
	if err != nil {
		fmt.Println(err)
		return
	}

	// Run forever
	select {}
}
