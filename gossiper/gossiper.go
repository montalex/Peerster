package gossiper

import (
	"container/list"
	"net"
)

//Gossiper, the gossiper entity
type Gossiper struct {
	Address    *net.UDPAddr
	Conn       *net.UDPConn
	Name       string
	KnownPeers list.List
}
