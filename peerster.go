package main

import (
	"flag"
	"fmt"

	"github.com/montalex/Peerster/errors"
	"github.com/montalex/Peerster/gossiper"
)

func main() {
	var UIPort = flag.String("UIPort", "4242", "Port for UI client")
	var gossipAddr = flag.String("gossipAddr", "127.0.0.1:5000", "ip:port for the gossiper")
	var name = flag.String("name", "myNode", "name of the gossiper")
	var peers = flag.String("peers", "", "comma separated list of peers of the form ip:port")
	var simple = flag.Bool("simple", true, "run gossiper in simple broadcast mode")

	flag.Parse()

	fmt.Println(*gossipAddr, *name, *peers, *simple)

	gos := gossiper.NewGossiper(*gossipAddr, *UIPort, "Alexis")
	readBuffer := make([]byte, 4096)

	for {
		size, addr, err := gos.ClientConn.ReadFromUDP(readBuffer)
		if size != 0 {
			fmt.Println(size, " Got something")
			fmt.Println("Addr: ", addr.String())
			errors.CheckErr(err, "Error when reading message: ", false)
			fmt.Println("CLIENT MESSAGE ", string(readBuffer[:size]))
			//Checks if client send message
			/*if addr.String() == *gossipAddr {
				fmt.Println("CLIENT MESSAGE ", readBuffer[:size])
				//TODO: send msg to all known peers
				for s := range strings.Split(*peers, ",") {

				}
			}*/
			/*var packet messages.GossipPacket
			protobuf.Decode(readBuffer[:size], &packet)

			//fmt.Println(*packet.Simple)
			nameOrigin, relayAddr, content := packet.ReadMessage()
			fmt.Println("SIMPLE MESSAGE origin ", nameOrigin, " from ", relayAddr, " contents ", content)*/
		}
	}

}
