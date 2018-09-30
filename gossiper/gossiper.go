package gossiper

import (
	"fmt"
	"net"
	"strings"

	"github.com/dedis/protobuf"
	"github.com/montalex/Peerster/errors"
	"github.com/montalex/Peerster/messages"
)

//Gossiper, the gossiper entity
type Gossiper struct {
	peersAddr, clientAddr *net.UDPAddr
	peersConn, clientConn *net.UDPConn
	name                  string
	knownPeers            []string
}

//NewGossiper, create a new gossiper or exit if there is an error
func NewGossiper(address, UIPort, name, peers string) *Gossiper {
	//Gossiper outside UDP listener
	udpPeersAddr, err := net.ResolveUDPAddr("udp4", address)
	errors.CheckErr(err, "Error when resolving UDP address: ", true)
	udpPeersConn, err := net.ListenUDP("udp4", udpPeersAddr)
	errors.CheckErr(err, "Error with UDP connection: ", true)

	//Gossiper client UDP listener
	udpClientAddr, err := net.ResolveUDPAddr("udp4", "127.0.0.1:"+UIPort)
	errors.CheckErr(err, "Error when resolving UDP address: ", true)
	udpClientConn, err := net.ListenUDP("udp4", udpClientAddr)
	errors.CheckErr(err, "Error with UDP connection: ", true)

	return &Gossiper{
		peersAddr:  udpPeersAddr,
		clientAddr: udpClientAddr,
		peersConn:  udpPeersConn,
		clientConn: udpClientConn,
		name:       name,
		knownPeers: strings.Split(peers, ","),
	}
}

//ListenClient, listen for UDP packets sent from client
func (gos *Gossiper) ListenClient(readBuffer []byte) {
	for {
		size, _, err := gos.clientConn.ReadFromUDP(readBuffer)
		errors.CheckErr(err, "Error when reading message: ", false)
		if size != 0 {
			msg := string(readBuffer[:size])
			fmt.Println("CLIENT MESSAGE", msg)
			fmt.Println("PEERS", strings.Join(gos.knownPeers, ","))
			packet := messages.GossipPacket{Simple: &messages.SimpleMessage{
				OriginalName:  gos.name,
				RelayPeerAddr: gos.peersAddr.String(),
				Contents:      msg}}

			//Prepare packet to send
			serializedPacket, err := protobuf.Encode(&packet)
			errors.CheckErr(err, "Error when encoding packet: ", false)

			//Transmit to all peers
			if len(gos.knownPeers) != 0 {
				for _, peer := range gos.knownPeers {
					gos.sendToPeer(serializedPacket, peer)
				}
			}
		}
	}
}

//ListenPeers listen for UDP packets sent from other peers
func (gos *Gossiper) ListenPeers(readBuffer []byte) {
	for {
		size, _, err := gos.peersConn.ReadFromUDP(readBuffer)
		errors.CheckErr(err, "Error when reading message: ", false)
		if size != 0 {
			var packet messages.GossipPacket
			protobuf.Decode(readBuffer[:size], &packet)
			nameOrigin, relayAddr, content := packet.ReadMessage()
			fmt.Println("SIMPLE MESSAGE origin", nameOrigin, "from", relayAddr, "contents", content)
			//Adds relay address if not contained already
			if !contains(gos.knownPeers, relayAddr) {
				gos.knownPeers = append(gos.knownPeers, relayAddr)
			}
			fmt.Println("PEERS ", strings.Join(gos.knownPeers, ","))

			//Modify relay address & prepare packet to send
			packet.Simple.RelayPeerAddr = gos.peersAddr.String()
			serializedPacket, err := protobuf.Encode(&packet)
			errors.CheckErr(err, "Error when encoding packet: ", false)

			//Send to all peers except relay address
			if len(gos.knownPeers) != 0 {
				for _, peer := range gos.knownPeers {
					if peer != relayAddr {
						gos.sendToPeer(serializedPacket, peer)
					}
				}
			}
		}
	}
}

//sendToPeer, sends a packet to a given peer
func (gos *Gossiper) sendToPeer(packet []byte, peer string) {
	peerAddr, err := net.ResolveUDPAddr("udp4", peer)
	errors.CheckErr(err, "Error when resolving peer UDP addr: ", false)
	gos.peersConn.WriteToUDP(packet, peerAddr)
}

//contains, check if the given string is contained in the given slice
func contains(peers []string, p string) bool {
	for _, test := range peers {
		if test == p {
			return true
		}
	}
	return false
}
