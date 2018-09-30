package gossiper

import (
	"fmt"
	"math/rand"
	"net"
	"strings"
	"time"

	"github.com/dedis/protobuf"
	"github.com/montalex/Peerster/errors"
	"github.com/montalex/Peerster/messages"
)

/*Gossiper is the entity in charge of gossiping
peersAddr: the gossiping side's UDP address
clientAddr: the client side's UDP address
peersConn: the gossiping side's UDP connection
clientConn: the client side's UDP connection
name: the gossiper's name
knownPeers: slice of the known peers address of the form (IP:Port)
*/
type Gossiper struct {
	peersAddr, clientAddr *net.UDPAddr
	peersConn, clientConn *net.UDPConn
	name                  string
	knownPeers            []string
	simple                bool
}

/*NewGossiper creates a new gossiper
address: the gossiping side's address of the form (IP:Port)
UIPort: the client side's port
name: the gossiper's name
peers: slice of the known peers address of the form (IP:Port)
Returns the gossiper
*/
func NewGossiper(address, UIPort, name, peers string, simple bool) *Gossiper {
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
		simple:     simple,
	}
}

/*ListenClient listens for UDP packets sent from client
readBuffer: the byte buffer needed to read messages
*/
func (gos *Gossiper) ListenClient(readBuffer []byte) {
	for {
		size, _, err := gos.clientConn.ReadFromUDP(readBuffer)
		errors.CheckErr(err, "Error when reading message: ", false)
		if size != 0 {
			msg := string(readBuffer[:size])
			fmt.Println("CLIENT MESSAGE", msg)
			fmt.Println("PEERS", strings.Join(gos.knownPeers, ","))

			//Prepare packet and send it
			if gos.simple {
				packet := messages.GossipPacket{Simple: &messages.SimpleMessage{
					OriginalName:  gos.name,
					RelayPeerAddr: gos.peersAddr.String(),
					Contents:      msg}}
				serializedPacket, err := protobuf.Encode(&packet)
				errors.CheckErr(err, "Error when encoding packet: ", false)

				//Transmit to all peers
				gos.sendToAll(serializedPacket, []string{})
			} else {
				packet := messages.GossipPacket{Rumor: &messages.RumorMessage{
					Origin: gos.name,
					ID:     1, //TODO: handle this ID
					Text:   msg}}
				serializedPacket, err := protobuf.Encode(&packet)
				errors.CheckErr(err, "Error when encoding packet: ", false)

				//Transmit to a random peer if at least one is known
				//TODO: HANDLE WAITING TIME
				if len(gos.knownPeers) == 0 {
					fmt.Println("Error: could not retransmit message, I do not know any other peers!")
				} else {
					randomPeer := gos.knownPeers[randomSelect(len(gos.knownPeers))]
					fmt.Println("MONGERING with", randomPeer)
					gos.sendToPeer(serializedPacket, randomPeer)
				}
			}
		}
	}
}

/*ListenPeers listens for UDP packets sent from other peers
readBuffer: the byte buffer needed to read messages
*/
func (gos *Gossiper) ListenPeers(readBuffer []byte) {
	for {
		size, addr, err := gos.peersConn.ReadFromUDP(readBuffer)
		errors.CheckErr(err, "Error when reading message: ", false)
		if size != 0 {
			var packet messages.GossipPacket
			protobuf.Decode(readBuffer[:size], &packet)

			//Check for Message type
			if packet.Simple != nil {
				nameOrigin, relayAddr, content := packet.ReadSimpleMessage()
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
				gos.sendToAll(serializedPacket, []string{relayAddr})

			} else if packet.Rumor != nil {

			} else if packet.Status != nil {

			} else { //Should never happen!
				fmt.Println("Error: MESSAGE FORM UNKNOWN. Sent by", addr.String())
			}
		}
	}
}

/*sendToPeer sends a packet to a given peer
packet: the packet to send
peer: the peer's address of the form (IP:Port)
*/
func (gos *Gossiper) sendToPeer(packet []byte, peer string) {
	peerAddr, err := net.ResolveUDPAddr("udp4", peer)
	errors.CheckErr(err, "Error when resolving peer UDP addr: ", false)
	gos.peersConn.WriteToUDP(packet, peerAddr)
}

/*sendToAll sends a packet to all peers known by the gossiper, can include an exception
packet: the packet to send
exception: a slice of peers to omit in the transmission (if none set to "")
*/
func (gos *Gossiper) sendToAll(packet []byte, exception []string) {
	if len(gos.knownPeers) != 0 {
		for _, peer := range gos.knownPeers {
			if !contains(exception, peer) {
				gos.sendToPeer(packet, peer)
			}
		}
	}
}

/*contains checks if the given string is contained in the given slice
peers: slice of strings to check
p: the string to look for
*/
func contains(peers []string, p string) bool {
	for _, test := range peers {
		if test == p {
			return true
		}
	}
	return false
}

func randomSelect(sliceLength int) int {
	rand.Seed(time.Now().UnixNano())
	return rand.Intn(sliceLength)
}
