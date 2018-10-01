package gossiper

import (
	"fmt"
	"math/rand"
	"net"
	"strings"
	"sync"
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
	id                    uint32
	want                  map[string]messages.PeerStatus
	timers                sync.Map
	pastMsg               messages.PastMessages
}

/*NewGossiper creates a new gossiper
address: the gossiping side's address of the form (IP:Port)
UIPort: the client side's port
name: the gossiper's name
peers: slice of the known peers address of the form (IP:Port)
simple: only using simple message if set to true
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

	peersList := strings.Split(peers, ",")
	/*var initWant = make([]messages.PeerStatus, len(peersList))
	for _, p := range peersList {
		initWant = append(initWant, messages.PeerStatus{
			Identifier: p,
			NextID:     1})
	}*/

	return &Gossiper{
		peersAddr:  udpPeersAddr,
		clientAddr: udpClientAddr,
		peersConn:  udpPeersConn,
		clientConn: udpClientConn,
		name:       name,
		knownPeers: peersList,
		simple:     simple,
		id:         uint32(1),
		want:       make(map[string]messages.PeerStatus),
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
				if len(gos.knownPeers) == 0 {
					fmt.Println("Error: could not retransmit message, I do not know any other peers!")
				} else {
					packet := messages.GossipPacket{Rumor: &messages.RumorMessage{
						Origin: gos.name,
						ID:     gos.id,
						Text:   msg}}
					serializedPacket, err := protobuf.Encode(&packet)
					errors.CheckErr(err, "Error when encoding packet: ", false)

					//Transmit to a random peer if at least one is known
					gos.rumormongering(serializedPacket, true)
					gos.id++
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
		relayAddr := addr.String()
		if size != 0 {
			var packet messages.GossipPacket
			protobuf.Decode(readBuffer[:size], &packet)

			//Check for Message type
			if packet.Simple != nil {
				nameOrigin, relayAddr, content := packet.ReadSimpleMessage()
				fmt.Println("SIMPLE MESSAGE origin", nameOrigin, "from", relayAddr, "contents", content)
				gos.addPrintPeers(relayAddr)

				//Modify relay address & prepare packet to send
				packet.Simple.RelayPeerAddr = gos.peersAddr.String()
				serializedPacket, err := protobuf.Encode(&packet)
				errors.CheckErr(err, "Error when encoding packet: ", false)

				//Send to all peers except relay address
				gos.sendToAll(serializedPacket, []string{relayAddr})

			} else if packet.Rumor != nil {
				nameOrigin, id, content := packet.ReadRumorMessage()
				fmt.Println("RUMOR origin", nameOrigin, "from", relayAddr, "ID", id, "contents", content)
				gos.addPrintPeers(relayAddr)
				gos.want[nameOrigin] = messages.PeerStatus{Identifier: nameOrigin, NextID: packet.Rumor.ID}
				gos.pastMsg.MessagesList[nameOrigin] = append(gos.pastMsg.MessagesList[nameOrigin], packet.Rumor)

				//Send status answer
				wantSlice := []messages.PeerStatus{}
				for _, p := range gos.want {
					wantSlice = append(wantSlice, p)
				}
				packet := messages.GossipPacket{Status: &messages.StatusPacket{
					Want: wantSlice}}
				serializedPacket, err := protobuf.Encode(&packet)
				errors.CheckErr(err, "Error when encoding packet: ", false)
				gos.sendToPeer(serializedPacket, relayAddr)
			} else if packet.Status != nil {
				fmt.Println("STATUS from", relayAddr, packet.ReadStatusMessage())
				gos.addPrintPeers(relayAddr)

				//Checks if status recieved from waiting list
				timer, ok := gos.timers.Load(relayAddr)
				if ok {
					//Checks if timer was stopped or timed out if true status was recieved as an answer
					if timer.(*time.Timer).Stop() {
						//TODO: maybe need to move this
						gos.timers.Delete(relayAddr)
					}
				}

				//Compare with own status
				var status = messages.PeerStatus{Identifier: "OK"}
				for _, s := range gos.want {
					for _, t := range packet.Status.Want {
						if t.Identifier == s.Identifier {
							if t.NextID > s.NextID {
								status = t
							} else if t.NextID < s.NextID {

							}
						}
					}
				}
				if status.Identifier != "OK" {
					//DO STUFF
				} else {
					fmt.Println("IN SYNC WITH", relayAddr)

				}
			} else { //Should never happen!
				fmt.Println("Error: MESSAGE FORM UNKNOWN. Sent by", relayAddr)
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

/*addPrintPeers adds relay address if not contained already and prints all known peers*/
func (gos *Gossiper) addPrintPeers(addr string) {
	//Adds relay address if not contained already
	if !contains(gos.knownPeers, addr) {
		gos.knownPeers = append(gos.knownPeers, addr)
	}
	fmt.Println("PEERS ", strings.Join(gos.knownPeers, ","))
}

/*rumormongering is a simple rumoring protocol
msg: the message to send
first: set to true when starting a new rumor
*/
func (gos *Gossiper) rumormongering(packet []byte, first bool) {
	randomPeer := gos.knownPeers[rand.Int()%len(gos.knownPeers)]
	if first {
		fmt.Println("MONGERING with", randomPeer)
	} else {
		fmt.Println("FLIPPED COIN sending rumor to", randomPeer)
	}

	gos.sendToPeer(packet, randomPeer)
	timer := time.AfterFunc(time.Second, func() {
		gos.timers.Delete(randomPeer)
		if (rand.Int() % 2) == 0 {
			gos.rumormongering(packet, false)
		} else {
			return
		}
	})
	gos.timers.Store(randomPeer, timer)
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
