package messages

//GossipPacket, The only type of packets sent between peers
type GossipPacket struct {
	Simple *SimpleMessage
}

//SimpleMessage, the simplest message a peer can send
type SimpleMessage struct {
	OriginalName  string
	RelayPeerAddr string
	Contents      string
}

//ReadMessage, reads the simple message from a GossipPacket
func (packet GossipPacket) ReadMessage() (string, string, string) {
	simple := *packet.Simple
	return simple.OriginalName, simple.RelayPeerAddr, simple.Contents
}
