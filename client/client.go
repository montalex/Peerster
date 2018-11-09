package main

import (
	"encoding/hex"
	"flag"
	"net"

	"github.com/dedis/protobuf"
	"github.com/montalex/Peerster/errorhandler"
	"github.com/montalex/Peerster/messages"
)

func main() {
	var UIPort = flag.String("UIPort", "4242", "Port for UI client")
	var dest = flag.String("dest", "", "destination for the private message")
	var filename = flag.String("file", "", "file to be indexed by the gossiper")
	var msg = flag.String("msg", "", "message to be sent")
	var request = flag.String("request", "", "request a chunk or metafile of this hash")
	flag.Parse()

	destAddr, err := net.ResolveUDPAddr("udp4", "127.0.0.1:"+*UIPort)
	errorhandler.CheckErr(err, "Error when resolving UDP dest address: ", true)

	myAddr, err := net.ResolveUDPAddr("udp", "127.0.0.1:4284")
	errorhandler.CheckErr(err, "Error when resolving UDP local address: ", true)

	udpConn, err := net.ListenUDP("udp", myAddr)
	errorhandler.CheckErr(err, "Error with UDP connection: ", true)
	defer udpConn.Close()

	var packet messages.GossipPacket
	if *msg != "" {
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
	}

	if *filename != "" {
		if *request != "" {
			hash, err := hex.DecodeString(*request)
			errorhandler.CheckErr(err, "Error when decoding hash: ", true)
			packet = messages.GossipPacket{DataRequest: &messages.DataRequest{
				Origin:      *filename,
				Destination: *dest,
				HopLimit:    10,
				HashValue:   hash}}
		} else {
			packet = messages.GossipPacket{Simple: &messages.SimpleMessage{
				OriginalName:  "file",
				RelayPeerAddr: "",
				Contents:      *filename}}
		}
	}

	serializedPacket, err := protobuf.Encode(&packet)
	errorhandler.CheckErr(err, "Error when encoding packet: ", false)

	_, err = udpConn.WriteToUDP(serializedPacket, destAddr)
	errorhandler.CheckErr(err, "Error when sending UDP msg: ", true)
}
