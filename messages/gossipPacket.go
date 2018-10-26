package messages

import "strconv"

/*GossipPacket is only type of packets sent between peers
!! Can only send one of the following at a time !!

Simple: if this is a simple message
Rumor: if this is a rumor message
Status: if this is a status message
*/
type GossipPacket struct {
	Simple  *SimpleMessage
	Rumor   *RumorMessage
	Status  *StatusPacket
	Private *PrivateMessage
}

/*StatusPacket is the packet send for PeerStatus
Want: a slice of PeerStatus messages
*/
type StatusPacket struct {
	Want []PeerStatus
}

/*SimpleMessage is the simplest message a peer can send
OriginalName: the original sender's name
RelayPeerAddr: the relay peer's address
Content: the content of the message
*/
type SimpleMessage struct {
	OriginalName  string
	RelayPeerAddr string
	Contents      string
}

/*RumorMessage is the gossip protocol message a peer can send
Origin: the original sender's name
ID: sequence number of the message
Text: the content of the message
*/
type RumorMessage struct {
	Origin string
	ID     uint32
	Text   string
}

/*PeerStatus is used to summarize the set of message a peer has seen so far
Identifier: the original sender's name
NextID: the next unseen message sequence number
*/
type PeerStatus struct {
	Identifier string
	NextID     uint32
}

/*PrivateMessage is used to transmit a private message to a specific peer
Origin: the original sender's name
ID: the message ID (set to 0 by default as no ordering is enforced in private messaging for now)
Text: the text of the message
Desitnation: the destination name (ex: Alice or Bob)
HopLimit: the maximum number of hop this message can do
*/
type PrivateMessage struct {
	Origin      string
	ID          uint32
	Text        string
	Destination string
	HopLimit    uint32
}

/*ReadSimpleMessage reads the simple message from a GossipPacket
Returns the original name, the relay peer's address and the content*/
func (packet *GossipPacket) ReadSimpleMessage() (string, string, string) {
	simple := *packet.Simple
	return simple.OriginalName, simple.RelayPeerAddr, simple.Contents
}

/*ReadRumorMessage reads the rumor message from a GossipPacket
Returns the original name, the ID and the content*/
func (packet *GossipPacket) ReadRumorMessage() (string, uint32, string) {
	rumor := *packet.Rumor
	return rumor.Origin, rumor.ID, rumor.Text
}

/*ReadStatusMessage reads the status message from a GossipPacket
Returns the pairs of peer-nextID*/
func (packet *GossipPacket) ReadStatusMessage() string {
	var statusString = ""
	status := *packet.Status
	for _, s := range status.Want {
		statusString = statusString + "peer " + s.Identifier + " nextID " + strconv.FormatUint(uint64(s.NextID), 10) + " "
	}
	return statusString
}

/*ReadPrivateMessage reads the private message from a GossipPacket
Returns the origin's name, the ID, the content, the destination name and the hop limit*/
func (packet *GossipPacket) ReadPrivateMessage() (string, uint32, string, string, uint32) {
	private := *packet.Private
	return private.Origin, private.ID, private.Text, private.Destination, private.HopLimit
}
