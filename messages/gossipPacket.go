package messages

// GossipPacket, The only type of packets sent between peers
type GossipPacket struct {
	Simple *SimpleMessage
}

// SimpleMessage, the simplest message a peer can send
type SimpleMessage struct {
	OriginalName  string
	RelayPeerAddr string
	Contents      string
}
