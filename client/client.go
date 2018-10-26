package main

import (
	"flag"
	"net"

	"github.com/dedis/protobuf"
	"github.com/montalex/Peerster/errors"
	"github.com/montalex/Peerster/messages"
)

func main() {
	var UIPort = flag.String("UIPort", "4242", "Port for UI client")
	var dest = flag.String("dest", "", "destination for the private message")
	var msg = flag.String("msg", "", "message to be sent")
	flag.Parse()

	destAddr, err := net.ResolveUDPAddr("udp4", "127.0.0.1:"+*UIPort)
	errors.CheckErr(err, "Error when resolving UDP dest address: ", true)

	myAddr, err := net.ResolveUDPAddr("udp", "127.0.0.1:4284")
	errors.CheckErr(err, "Error when resolving UDP local address: ", true)

	//Test 1 too fast to send...needed that
	//time.Sleep(5 * time.Millisecond)

	udpConn, err := net.ListenUDP("udp", myAddr)
	errors.CheckErr(err, "Error with UDP connection: ", true)
	defer udpConn.Close()

	var packet messages.GossipPacket
	if *dest != "" {
		packet = messages.GossipPacket{Private: &messages.PrivateMessage{
			Origin:      "",
			ID:          0,
			Text:        *msg,
			Destination: *dest,
			HopLimit:    10}}
	} else {
		packet = messages.GossipPacket{Simple: &messages.SimpleMessage{
			OriginalName:  "",
			RelayPeerAddr: "",
			Contents:      *msg}}
	}

	serializedPacket, err := protobuf.Encode(&packet)
	errors.CheckErr(err, "Error when encoding packet: ", false)

	_, err = udpConn.WriteToUDP(serializedPacket, destAddr)
	errors.CheckErr(err, "Error when sending UDP msg: ", true)
}
