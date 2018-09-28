package gossiper

import (
	"container/list"
	"net"

	"github.com/montalex/Peerster/errors"
)

//Gossiper, the gossiper entity
type Gossiper struct {
	OutAddr, ClientAddr *net.UDPAddr
	OutConn, ClientConn *net.UDPConn
	Name                string
	KnownPeers          list.List
}

//NewGossiper, create a new gossiper or exit if there is an error
func NewGossiper(address, UIPort, name string) *Gossiper {
	//Gossiper outside UDP listener
	udpOutAddr, err := net.ResolveUDPAddr("udp4", address)
	errors.CheckErr(err, "Error when resolving UDP address: ", true)
	udpOutConn, err := net.ListenUDP("udp4", udpOutAddr)
	errors.CheckErr(err, "Error with UDP connection: ", true)

	//Gossiper client UDP listener
	udpClientAddr, err := net.ResolveUDPAddr("udp4", "127.0.0.1:"+UIPort)
	errors.CheckErr(err, "Error when resolving UDP address: ", true)
	udpClientConn, err := net.ListenUDP("udp4", udpClientAddr)
	errors.CheckErr(err, "Error with UDP connection: ", true)
	return &Gossiper{
		OutAddr:    udpOutAddr,
		ClientAddr: udpClientAddr,
		OutConn:    udpOutConn,
		ClientConn: udpClientConn,
		Name:       name,
		KnownPeers: *list.New(),
	}
}
