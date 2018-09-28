package main

import (
	"flag"

	"github.com/montalex/Peerster/gossiper"
)

func main() {
	var UIPort = flag.String("UIPort", "4242", "Port for UI client")
	var gossipAddr = flag.String("gossipAddr", "127.0.0.1:5000", "ip:port for the gossiper")
	var name = flag.String("name", "Alexis", "name of the gossiper")
	var peers = flag.String("peers", "", "comma separated list of peers of the form ip:port")
	var simple = flag.Bool("simple", true, "run gossiper in simple broadcast mode")

	flag.Parse()
	if *simple {

	}
	gos := gossiper.NewGossiper(*gossipAddr, *UIPort, *name, *peers)
	clientReadBuffer := make([]byte, 4096)
	peersReadBuffer := make([]byte, 4096)

	go gos.ListenClient(clientReadBuffer)
	gos.ListenPeers(peersReadBuffer)
}
