package main

import (
	"container/list"
	"flag"
	"fmt"
	"net"
	"os"

	"github.com/montalex/Peerster/gossiper"
)

func main() {
	var UIPort = flag.Int("UIPort", 4242, "Port for UI client")
	var gossipAddr = flag.String("gossipAddr", "127.0.0.1:5000", "ip:port for the gossiper")
	var name = flag.String("name", "myNode", "name of the gossiper")
	var peers = flag.String("peers", "", "comma separated list of peers of the form ip:port")
	var simple = flag.Bool("simple", true, "run gossiper in simple broadcast mode")
	//Todo: plus de flag non défini = problems

	flag.Parse()

	gos := NewGossiper(*gossipAddr, "myNode")

}

// NewGossiper, create a new gossiper or exit if there is an error
func NewGossiper(address, name string) *gossiper.Gossiper {
	udpAddr, err := net.ResolveUDPAddr("udp4", address)
	udpConn, err := net.ListenUDP("udp4", udpAddr)
	if err != nil {
		fmt.Println(err.Error)
		os.Exit(1)
	}

	return &gossiper.Gossiper{
		Address:    udpAddr,
		Conn:       udpConn,
		Name:       name,
		KnownPeers: *list.New(),
	}
}
