package messages

import (
	"crypto/sha256"
	"encoding/binary"
	"strconv"
)

/*GossipPacket is only type of packets sent between peers
!! Can only send one of the following at a time !!

Simple: if this is a simple message
Rumor: if this is a rumor message
Status: if this is a status message
*/
type GossipPacket struct {
	Simple        *SimpleMessage
	Rumor         *RumorMessage
	Status        *StatusPacket
	Private       *PrivateMessage
	DataRequest   *DataRequest
	DataReply     *DataReply
	SearchRequest *SearchRequest
	SearchReply   *SearchReply
	TxPublish     *TxPublish
	BlockPublish  *BlockPublish
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

/*DataRequest is used to request a file from a specific peer
Origin: the original sender's name
Desitnation: the destination name (ex: Alice or Bob)
HopLimit: the maximum number of hop this message can do
HashValue: the meta hash
*/
type DataRequest struct {
	Origin      string
	Destination string
	HopLimit    uint32
	HashValue   []byte
}

/*DataReply is used to send a file to a specific peer
Origin: the original sender's name
Desitnation: the destination name (ex: Alice or Bob)
HopLimit: the maximum number of hop this message can do
HashValue: the meta hash
Data: the data (metafile or chunk)
*/
type DataReply struct {
	Origin      string
	Destination string
	HopLimit    uint32
	HashValue   []byte
	Data        []byte
}

/*SearchRequest is used to search for a file using keywords
Origin: the original sender's name
Budget: amount of maximum forwarding for this request
Keywords: the keywords to search for
*/
type SearchRequest struct {
	Origin   string
	Budget   uint64
	Keywords []string
}

/*SearchReply is used to reply to a search request
Origin: the original sender's name
Desitnation: the destination name (ex: Alice or Bob)
HopLimit: the maximum number of hop this message can do
Results: the list of results
*/
type SearchReply struct {
	Origin      string
	Destination string
	HopLimit    uint32
	Results     []*SearchResult
}

/*SearchResult is used to store results of a file search
FileName: the file's name
MetafileHash: the file's meta hash
ChunkMap: a map of chunks for this file
ChunkCount: the number of chunks for the whole file
*/
type SearchResult struct {
	FileName     string
	MetafileHash []byte
	ChunkMap     []uint64
	ChunkCount   uint64
}

/*TxPublish is used to publish transactions in the gossiping process
File: the published file
HopLimit: the maximum number of hop this message can do
*/
type TxPublish struct {
	File     File
	HopLimit uint32
}

/*File represents a gossiper's file
Name: the name of the file
Size: the size of the file
MetafileHash: the file's metahash
*/
type File struct {
	Name         string
	Size         int64
	MetafileHash []byte
}

/*BlockPublish is used to publish blocks in the gossiping process
Block: the published block
HopLimit: the maximum number of hop this message can do
*/
type BlockPublish struct {
	Block    Block
	HopLimit uint32
}

/*Block represents a block in the chain
PrevHash: the previous hash in the chain
Nonce: the new Nonce
Transaction: the list of accepted transactions
*/
type Block struct {
	PrevHash     [32]byte
	Nonce        [32]byte
	Transactions []TxPublish
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

/*ReadDataRequest reads the data request message from a GossipPacket
Returns the origin's name, the destination name, the hop limit and the hash*/
func (packet *GossipPacket) ReadDataRequest() (string, string, uint32, []byte) {
	request := *packet.DataRequest
	return request.Origin, request.Destination, request.HopLimit, request.HashValue
}

/*ReadDataReply reads the data reply message from a GossipPacket
Returns the origin's name, the destination name, the hop limit, the hash and the data*/
func (packet *GossipPacket) ReadDataReply() (string, string, uint32, []byte, []byte) {
	reply := *packet.DataReply
	return reply.Origin, reply.Destination, reply.HopLimit, reply.HashValue, reply.Data
}

/*ReadSearchRequest reads the search request message from a GossipPacket
Returns the origin's name, the budget, and the keywords list*/
func (packet *GossipPacket) ReadSearchRequest() (string, uint64, []string) {
	request := *packet.SearchRequest
	return request.Origin, request.Budget, request.Keywords
}

/*ReadSearchReply reads the search reply message from a GossipPacket
Returns the origin's name, the destination, the hop limit and the results list*/
func (packet *GossipPacket) ReadSearchReply() (string, string, uint32, []*SearchResult) {
	reply := *packet.SearchReply
	return reply.Origin, reply.Destination, reply.HopLimit, reply.Results
}

/*Hash function for blocks*/
func (b *Block) Hash() (out [32]byte) {
	h := sha256.New()
	h.Write(b.PrevHash[:])
	h.Write(b.Nonce[:])
	binary.Write(h, binary.LittleEndian, uint32(len(b.Transactions)))
	for _, t := range b.Transactions {
		th := t.Hash()
		h.Write(th[:])
	}
	copy(out[:], h.Sum(nil))
	return
}

/*Hash function for transaction*/
func (t *TxPublish) Hash() (out [32]byte) {
	h := sha256.New()
	binary.Write(h, binary.LittleEndian, uint32(len(t.File.Name)))
	h.Write([]byte(t.File.Name))
	h.Write(t.File.MetafileHash)
	copy(out[:], h.Sum(nil))
	return
}
