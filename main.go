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
	var rtimer = flag.Int("rtimer", 0, "route rumors sending period in seconds, 0 to disable sending of route rumors (default 0)")
	var simple = flag.Bool("simple", false, "run gossiper in simple broadcast mode")
	var webserver = flag.Bool("web", false, "run Peerster with the web server mode")
	flag.Parse()

	gos := gossiper.NewGossiper(*gossipAddr, *UIPort, *name, *peers, *simple)

	go gos.ListenClient(make([]byte, 65535))
	//go gos.Mine()

	if *webserver {
		go web.Run(gos, *UIPort)
	}
	if !*simple {
		go gos.AntiEntropy()
	}
	if *rtimer > 0 {
		//Make yourself known, first broadcast empty rumor
		gos.Hello()
		go gos.RoutingRumors(*rtimer)
	}

	gos.ListenPeers(make([]byte, 65535))
}
