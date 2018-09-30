package main

import (
	"flag"
	"net"

	"github.com/montalex/Peerster/errors"
)

func main() {
	var UIPort = flag.String("UIPort", "4242", "Port for UI client")
	var msg = flag.String("msg", "", "message to be sent")
	flag.Parse()

	destAddr, err := net.ResolveUDPAddr("udp4", "127.0.0.1:"+*UIPort)
	errors.CheckErr(err, "Error when resolving UDP dest address: ", true)

	myAddr, err := net.ResolveUDPAddr("udp", "127.0.0.1:4284")
	errors.CheckErr(err, "Error when resolving UDP local address: ", true)

	udpConn, err := net.DialUDP("udp", myAddr, destAddr)
	errors.CheckErr(err, "Error with UDP connection: ", true)
	defer udpConn.Close()

	_, err = udpConn.Write([]byte(*msg))
	errors.CheckErr(err, "Error when sending UDP msg: ", true)
}
