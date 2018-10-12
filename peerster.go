package main

import (
	"flag"

	"github.com/montalex/Peerster/gossiper"
	"github.com/montalex/Peerster/web"
)

func main() {
	var UIPort = flag.String("UIPort", "4242", "Port for UI client")
	var gossipAddr = flag.String("gossipAddr", "127.0.0.1:5000", "ip:port for the gossiper")
	var name = flag.String("name", "Alexis-245433", "name of the gossiper")
	var peers = flag.String("peers", "", "comma separated list of peers of the form ip:port")
	var simple = flag.Bool("simple", false, "run gossiper in simple broadcast mode")
	var webserver = flag.Bool("web", false, "run Peerster with the web server mode")
	flag.Parse()

	gos := gossiper.NewGossiper(*gossipAddr, *UIPort, *name, *peers, *simple)
	clientReadBuffer := make([]byte, 4096)
	peersReadBuffer := make([]byte, 4096)

	if *webserver {
		go web.Run(gos)
	}
	go gos.AntiEntropy()
	go gos.ListenClient(clientReadBuffer)
	gos.ListenPeers(peersReadBuffer)
}
