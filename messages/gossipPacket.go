package messages

/*GossipPacket is only type of packets sent between peers
!! Can only send one of the following at a time !!

Simple: if this is a simple message
Rumor: if this is a rumor message
Status: if this is a status message
*/
type GossipPacket struct {
	Simple *SimpleMessage
	Rumor  *RumorMessage
	Status *StatusPacket
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

/*ReadSimpleMessage reads the simple message from a GossipPacket
Returns the original name, the relay peer's address and the content*/
func (packet GossipPacket) ReadSimpleMessage() (string, string, string) {
	simple := *packet.Simple
	return simple.OriginalName, simple.RelayPeerAddr, simple.Contents
}
