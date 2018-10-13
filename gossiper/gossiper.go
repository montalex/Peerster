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
simple: run gossiper in simple mode
want: the gossiper's peer status
timers: a synchronized map of timers in order to timeout after a msg is send to a peer
pastMsg: the list of messages recieved and sent
lastSent: the last msg sent to each peer
*/
type Gossiper struct {
	peersAddr, clientAddr *net.UDPAddr
	peersConn, clientConn *net.UDPConn
	name                  string
	knownPeers            []string
	simple                bool
	want                  map[string]*messages.PeerStatus
	timers                *sync.Map
	pastMsg               messages.PastMessages
	lastSent              map[string]*messages.RumorMessage
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

	peersList := make([]string, 0)
	if peers != "" {
		peersList = strings.Split(peers, ",")
	}
	var initWant = make(map[string]*messages.PeerStatus)
	initWant[name] = &messages.PeerStatus{
		Identifier: name,
		NextID:     1,
	}

	return &Gossiper{
		peersAddr:  udpPeersAddr,
		clientAddr: udpClientAddr,
		peersConn:  udpPeersConn,
		clientConn: udpClientConn,
		name:       name,
		knownPeers: peersList,
		simple:     simple,
		want:       initWant,
		timers:     &sync.Map{},
		pastMsg:    messages.PastMessages{MessagesList: make(map[string][]*messages.RumorMessage)},
		lastSent:   make(map[string]*messages.RumorMessage),
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
				gos.SendRumor(msg)
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

				//Checks if this peer is in the list already
				if _, ok := gos.want[nameOrigin]; !ok {
					gos.want[nameOrigin] = &messages.PeerStatus{Identifier: nameOrigin, NextID: uint32(1)}
				}

				//Checks if id is the one we need, otherwise drop msg.
				//If already seen, no need to read it.
				//If more recent then needed, we will recieve it later in order.
				if id == gos.want[nameOrigin].NextID {
					gos.want[nameOrigin].NextID++
					gos.pastMsg.MessagesList[nameOrigin] = append(gos.pastMsg.MessagesList[nameOrigin], packet.Rumor)

					//Send status answer
					gos.sendStatus(relayAddr)

					//Start rumormongering
					gos.rumormongering(packet.Rumor, []string{relayAddr}, true)
				}

			} else if packet.Status != nil {
				fmt.Println("STATUS from", relayAddr, packet.ReadStatusMessage())
				gos.addPrintPeers(relayAddr)

				//Checks if status recieved from waiting list
				//TODO: Check if correct/needed.
				timer, ok := gos.timers.Load(relayAddr)
				if ok {
					//Checks if timer was stopped or timed out if true status was recieved as an answer
					if timer.(*time.Timer).Stop() {
						gos.timers.Delete(relayAddr)
					}
				}

				//Compare with own status
				gos.compareStatus(packet.Status.Want, relayAddr)
			} else { //Should never happen!
				fmt.Println("Error: MESSAGE FORM UNKNOWN. Sent by", relayAddr)
			}
		}
	}
}

/*AntiEntropy makes sure everybody enventually gets all messages by sending a status message
to a random peer every second*/
func (gos *Gossiper) AntiEntropy() {
	for {
		time.Sleep(time.Second)
		if len(gos.knownPeers) > 0 {
			randomPeer := gos.knownPeers[rand.Int()%len(gos.knownPeers)]
			gos.sendStatus(randomPeer)
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
except: a slice of peers to omit in the transmission (if none is empty)
*/
func (gos *Gossiper) sendToAll(packet []byte, except []string) {
	if len(gos.knownPeers) != 0 {
		for _, peer := range gos.knownPeers {
			if !contains(except, peer) {
				gos.sendToPeer(packet, peer)
			}
		}
	}
}

/*sendToPeer sends a packet to a given peer
peer: the peer's address of the form (IP:Port)
*/
func (gos *Gossiper) sendStatus(peer string) {
	wantSlice := []messages.PeerStatus{}
	for _, p := range gos.want {
		wantSlice = append(wantSlice, *p)
	}

	packet := messages.GossipPacket{Status: &messages.StatusPacket{
		Want: wantSlice}}
	serializedPacket, err := protobuf.Encode(&packet)
	errors.CheckErr(err, "Error when encoding packet: ", false)
	gos.sendToPeer(serializedPacket, peer)
}

/*addPrintPeers adds relay address if not contained already and prints all known peers*/
func (gos *Gossiper) addPrintPeers(addr string) {
	//Adds relay address if not contained already
	gos.AddPeer(addr)
	fmt.Println("PEERS ", strings.Join(gos.knownPeers, ","))
}

/*rumormongering is a simple rumoring protocol
msg: the message to send
except: a slice of peers to omit in the transmission (if none is empty) of the form (IP:Port)
first: set to true when starting a new rumor
*/
func (gos *Gossiper) rumormongering(rumor *messages.RumorMessage, except []string, first bool) {
	if len(except) < len(gos.knownPeers) {
		randomPeer := gos.knownPeers[rand.Int()%len(gos.knownPeers)]
		for contains(except, randomPeer) {
			randomPeer = gos.knownPeers[rand.Int()%len(gos.knownPeers)]
		}

		packet := messages.GossipPacket{Rumor: rumor}
		serializedPacket, err := protobuf.Encode(&packet)
		errors.CheckErr(err, "Error when encoding packet: ", false)

		if first {
			fmt.Println("MONGERING with", randomPeer)
		} else {
			fmt.Println("FLIPPED COIN sending rumor to", randomPeer)
		}

		gos.sendToPeer(serializedPacket, randomPeer)
		gos.lastSent[randomPeer] = rumor
		timer := time.AfterFunc(time.Second, func() {
			gos.timers.Delete(randomPeer)
			if (rand.Int() % 2) == 0 {
				gos.rumormongering(rumor, append(except, randomPeer), false)
			}
			return
		})
		gos.timers.Store(randomPeer, timer)
	}
}

/*compareStatus compares the given status with the gossiper's status and send the appropriate message
msgStatus: the status to compare with
peer: the peer's address of the form (IP:Port)
*/
func (gos *Gossiper) compareStatus(msgStatus []messages.PeerStatus, peer string) {
	iNeed := false

	peersCopy := make(map[string]*messages.PeerStatus)

	// Copy from the Want map
	for key, value := range gos.want {
		peersCopy[key] = value
	}

	for _, hisStatus := range msgStatus {
		myStatus, ok := gos.want[hisStatus.Identifier]
		if ok {
			delete(peersCopy, hisStatus.Identifier)

			if hisStatus.NextID > myStatus.NextID {
				iNeed = true
			}
			if hisStatus.NextID < myStatus.NextID {
				//Send him packet
				packet := messages.GossipPacket{Rumor: gos.pastMsg.MessagesList[hisStatus.Identifier][hisStatus.NextID-1]}
				serializedPacket, err := protobuf.Encode(&packet)
				errors.CheckErr(err, "Error when encoding packet: ", false)
				gos.sendToPeer(serializedPacket, peer)
				return
			}
		} else {
			iNeed = true
			gos.want[hisStatus.Identifier] = &messages.PeerStatus{
				Identifier: hisStatus.Identifier,
				NextID:     1,
			}
		}
	}

	if len(peersCopy) > 0 {
		id := ""
		for key, stat := range peersCopy {
			if stat.NextID > 1 {
				id = key
				break
			}
		}
		if id != "" {
			//Send him packet
			packet := messages.GossipPacket{Rumor: gos.pastMsg.MessagesList[id][0]}
			serializedPacket, err := protobuf.Encode(&packet)
			errors.CheckErr(err, "Error when encoding packet: ", false)
			gos.sendToPeer(serializedPacket, peer)
			return
		}
	}

	if iNeed {
		//Ask him for rumors
		gos.sendStatus(peer)
	} else {
		fmt.Println("IN SYNC WITH", peer)
		if (rand.Int() % 2) == 0 {
			gos.rumormongering(gos.lastSent[peer], []string{peer}, false)
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

/*GetName returns the list of known peers*/
func (gos *Gossiper) GetName() string {
	return gos.name
}

/*GetPeers returns the list of known peers*/
func (gos *Gossiper) GetPeers() []string {
	return gos.knownPeers
}

/*AddPeer adds a peer to the gossiper's list
newPeer: the new peer to add
*/
func (gos *Gossiper) AddPeer(newPeer string) {
	if !contains(gos.knownPeers, newPeer) {
		gos.knownPeers = append(gos.knownPeers, newPeer)
	}
}

/*GetMessages returns the list of messages recieves in the form Origin: Message*/
func (gos *Gossiper) GetMessages() []string {
	allMsg := make([]string, 0)
	for key, rumList := range gos.pastMsg.MessagesList {
		for _, msg := range rumList {
			allMsg = append(allMsg, key+": "+msg.Text)
		}
	}
	return allMsg
}

/*SendRumor start the rumor mongering process
msg: the message to send
*/
func (gos *Gossiper) SendRumor(msg string) {
	if len(gos.knownPeers) == 0 {
		fmt.Println("Error: could not retransmit message, I do not know any other peers!")
	} else {
		rumor := messages.RumorMessage{
			Origin: gos.name,
			ID:     gos.want[gos.name].NextID,
			Text:   msg}

		//Add to past
		gos.pastMsg.MessagesList[gos.name] = append(gos.pastMsg.MessagesList[gos.name], &rumor)
		gos.want[gos.name].NextID++

		//Transmit to a random peer if at least one is known
		gos.rumormongering(&rumor, []string{}, true)
	}
}
