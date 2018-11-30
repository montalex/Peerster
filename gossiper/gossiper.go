package gossiper

import (
	"bufio"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"math/rand"
	"net"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/dedis/protobuf"
	"github.com/montalex/Peerster/errorhandler"
	"github.com/montalex/Peerster/messages"
)

const buffSize = 8192

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
	knownPeers            SafePeers
	simple                bool
	want                  SafeStatus
	timers                *sync.Map
	currentSearch         *sync.Map
	pastMsg               SafePast
	pastPrivate           SafePast
	lastSent              SafeMsgMap
	routingTable          SafeTable
	metaFiles             SafeMetaMap
	matches               SafeMatchMap
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
	//Gossiper's outside UDP listener
	udpPeersAddr, err := net.ResolveUDPAddr("udp4", address)
	errorhandler.CheckErr(err, "Error when resolving UDP address: ", true)
	udpPeersConn, err := net.ListenUDP("udp4", udpPeersAddr)
	errorhandler.CheckErr(err, "Error with UDP connection: ", true)

	//Gossiper's client UDP listener
	udpClientAddr, err := net.ResolveUDPAddr("udp4", "127.0.0.1:"+UIPort)
	errorhandler.CheckErr(err, "Error when resolving UDP address: ", true)
	udpClientConn, err := net.ListenUDP("udp4", udpClientAddr)
	errorhandler.CheckErr(err, "Error with UDP connection: ", true)

	//Gossiper's known peers list
	peersList := make([]string, 0)
	if peers != "" {
		peersList = strings.Split(peers, ",")
	}

	//Gossiper's peer status
	var initWant = make(map[string]*messages.PeerStatus)
	initWant[name] = &messages.PeerStatus{
		Identifier: name,
		NextID:     1,
	}

	return &Gossiper{
		peersAddr:     udpPeersAddr,
		clientAddr:    udpClientAddr,
		peersConn:     udpPeersConn,
		clientConn:    udpClientConn,
		name:          name,
		knownPeers:    SafePeers{peers: peersList},
		simple:        simple,
		want:          SafeStatus{m: initWant},
		timers:        &sync.Map{},
		currentSearch: &sync.Map{},
		pastMsg:       SafePast{messagesList: make(map[string][]*messages.RumorMessage)},
		pastPrivate:   SafePast{messagesList: make(map[string][]*messages.RumorMessage)},
		lastSent:      SafeMsgMap{messages: make(map[string]*messages.RumorMessage)},
		routingTable:  SafeTable{table: make(map[string]string)},
		metaFiles:     SafeMetaMap{meta: make(map[[32]byte]*File), data: make(map[[32]byte]*File)},
		matches:       SafeMatchMap{matches: make(map[string]*Match)},
	}
}

/*ListenClient listens for UDP packets sent from client
readBuffer: the byte buffer needed to read messages
*/
func (gos *Gossiper) ListenClient(readBuffer []byte) {
	for {
		size, _, err := gos.clientConn.ReadFromUDP(readBuffer)
		errorhandler.CheckErr(err, "Error when reading message: ", false)
		if size != 0 {
			var packet messages.GossipPacket
			protobuf.Decode(readBuffer[:size], &packet)

			//SimpleMessage can carry a simple message (if simple mode activate), a rumor or an index command
			if packet.Simple != nil {
				//Prepare packet and send it
				name, _, msg := packet.ReadSimpleMessage()

				//Client asks for a file to be indexed
				if name == "file" {
					err = gos.indexFile(msg)
					errorhandler.CheckErr(err, "Error indexing file: ", false)
				} else {
					gos.PrintClientMsg(msg)
					if gos.simple {
						packet.Simple.OriginalName = gos.name
						packet.Simple.RelayPeerAddr = gos.peersAddr.String()
						serializedPacket, err := protobuf.Encode(&packet)
						errorhandler.CheckErr(err, "Error when encoding packet: ", false)

						//Transmit to all peers
						gos.sendToAll(serializedPacket, []string{})
					} else {
						gos.SendRumor(msg)
					}
				}
			} else if packet.Private != nil {
				_, _, msg, dest, _ := packet.ReadPrivateMessage()
				gos.PrintClientMsg(msg)
				packet.Private.Origin = gos.name
				serializedPacket, err := protobuf.Encode(&packet)
				errorhandler.CheckErr(err, "Error when encoding packet: ", false)
				if destAddr, ok := gos.routingTable.SafeReadSpec(dest); ok {
					gos.pastPrivate.SafeAdd(dest, &messages.RumorMessage{Origin: gos.name, ID: 0, Text: msg})
					fmt.Println("SENDING PRIVATE MESSAGE", msg, "TO", dest) //Test Gv2
					gos.sendToPeer(serializedPacket, destAddr)
				} else {
					fmt.Println("Error sending private message: I do not know", dest)
				}
			} else if packet.DataRequest != nil {
				fmt.Println("REQUESTING filename", packet.DataRequest.Origin, "from", packet.DataRequest.Destination, "hash", hex.EncodeToString(packet.DataRequest.HashValue)) //Test Gv2
				gos.clientRequest(packet)
			} else if packet.SearchRequest != nil {
				packet.SearchRequest.Origin = gos.name
				if packet.SearchRequest.Budget == 0 {
					packet.SearchRequest.Budget = 2
					gos.fileSearch(packet, true)
				} else {
					gos.fileSearch(packet, false)
				}
			} else { //Should never happen!
				fmt.Println("Error: CLIENT MESSAGE FORM UNKNOWN")
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
		errorhandler.CheckErr(err, "Error when reading message: ", false)
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
				errorhandler.CheckErr(err, "Error when encoding packet: ", false)

				//Send to all peers except relay address
				gos.sendToAll(serializedPacket, []string{relayAddr})

			} else if packet.Rumor != nil {
				nameOrigin, id, content := packet.ReadRumorMessage()

				//If message is empty -> Routing Rumor
				if content == "" {
					gos.AddPeer(relayAddr)
				} else {
					fmt.Println("RUMOR origin", nameOrigin, "from", relayAddr, "ID", id, "contents", content)
					gos.addPrintPeers(relayAddr)
				}

				//Checks if this peer is in the list already else add new entry with ID = 1
				if _, ok := gos.want.SafeRead(nameOrigin); !ok {
					gos.want.SafeUpdate(nameOrigin, &messages.PeerStatus{Identifier: nameOrigin, NextID: uint32(1)})
				}

				//Checks if id is the one we need, otherwise drop msg.
				//If already seen, no need to read it.
				//If more recent then needed, we will recieve it later in order.
				//Makes sure if people resend my own message I drop them
				if id == gos.want.SafeID(nameOrigin) && nameOrigin != gos.name {
					gos.want.SafeInc(nameOrigin)
					gos.pastMsg.SafeAdd(nameOrigin, packet.Rumor)
					gos.routingTable.SafeUpdate(nameOrigin, relayAddr)

					//Send status answer
					gos.sendStatus(relayAddr)

					//Start rumormongering
					gos.rumormongering(packet.Rumor, []string{relayAddr}, true)

				}

			} else if packet.Status != nil {
				fmt.Println("STATUS from", relayAddr, packet.ReadStatusMessage())
				gos.addPrintPeers(relayAddr)

				//Checks if status recieved from waiting list

				if timer, ok := gos.timers.Load(relayAddr); ok {
					//Checks if timer was stopped or timed out if true status was recieved as an answer
					if timer.(*time.Timer).Stop() {
						gos.timers.Delete(relayAddr)
					}
				}

				//Compare with own status
				gos.compareStatus(packet.Status.Want, relayAddr)
			} else if packet.Private != nil {
				name, _, content, dest, nHop := packet.ReadPrivateMessage()

				if dest == gos.name {
					fmt.Println("PRIVATE origin", name, "hop-limit", nHop, "contents", content)
					gos.pastPrivate.SafeAdd(name, &messages.RumorMessage{Origin: name, ID: 0, Text: content})
				} else {
					if nHop > 1 {
						packet.Private.HopLimit--
						gos.forwardPacket(&packet, dest)
					}
				}
			} else if packet.DataRequest != nil {
				name, dest, nHop, hash := packet.ReadDataRequest()
				if dest == gos.name {
					fmt.Println("DATA REQUEST origin", name, "hop-limit", nHop, "hash", hash)
					var data = make([]byte, 0)
					var hash32 [32]byte
					copy(hash32[:], hash)

					f, ok := gos.metaFiles.SafeReadData(hash32)
					if ok {
						data = gos.metaFiles.SafeReadChunk(hash32)
					} else {
						f, ok = gos.metaFiles.SafeReadMeta(hash32)
						if ok {
							data = f.SafeReadMetaFile()
						}
					}

					reply := messages.GossipPacket{DataReply: &messages.DataReply{
						Origin:      gos.name,
						Destination: name,
						HopLimit:    10,
						HashValue:   hash,
						Data:        data}}
					serializedPacket, err := protobuf.Encode(&reply)
					fmt.Println("SEND REPLY to", name, "hash", hash, "data", data)
					errorhandler.CheckErr(err, "Error when encoding packet: ", false)
					if destAddr, ok := gos.routingTable.SafeReadSpec(name); ok {
						gos.sendToPeer(serializedPacket, destAddr)
					}
				} else {
					if nHop > 1 {
						packet.DataRequest.HopLimit--
						gos.forwardPacket(&packet, dest)
					}
				}
			} else if packet.DataReply != nil {
				name, dest, nHop, hash, data := packet.ReadDataReply()
				fmt.Println("DATA REPLY origin", name, "hop-limit", nHop, "hash", hash)
				hashTest := make([]byte, 32)
				x := sha256.Sum256(data)
				copy(hashTest, x[:])
				if bytes.Equal(hashTest, hash) {
					var hash32 [32]byte
					copy(hash32[:], hash)
					hashSize := 32
					if dest == gos.name {
						f, ok := gos.metaFiles.SafeReadData(hash32)
						if ok {
							//Recieved a chunk >> Update File & send more request if needed
							f.SafeUpdateChunk(hash32, data)
							nChunk := f.SafeReadNextChunk()
							s := f.SafeReadSize()
							fmt.Println("DOWNLOADING", f.SafeReadName(), "chunk", nChunk, "from", name)
							if nChunk == s {
								fmt.Println("RECONSTRUCTED file", f.SafeReadName())
								buffer := make([]byte, 0)
								for i := 0; i < nChunk; i++ {
									copy(hash32[:], f.SafeReadMetaFile()[i*hashSize:(i+1)*hashSize])
									buffer = append(buffer, gos.metaFiles.SafeReadChunk(hash32)...)
								}
								err := ioutil.WriteFile("_Downloads/"+f.SafeReadName(), buffer, 0644)
								if err != nil {
									fmt.Println("ERROR: Could not open file for rebuild")
								}
							} else {
								gos.sendChunkRequest(f.SafeNextChuck(), name)
							}
						} else {
							f, ok = gos.metaFiles.SafeReadMeta(hash32)
							if ok {
								//Recieved a MetaFile >> Update File and start requsting chunks
								fmt.Println("DOWNLOADING metafile of", f.SafeReadName(), "from", name)
								f.SafeUpdateMetaFile(data)
								nChunks := len(data) / hashSize
								f.SafeUpdateSize(nChunks)
								for i := 0; i < nChunks; i++ {
									var cHash [32]byte
									copy(cHash[:], data[i*hashSize:(i+1)*hashSize])
									gos.metaFiles.SafeUpdateData(cHash, f)
								}
								gos.sendChunkRequest(f.SafeNextChuck(), name)
							}
						}
						hashStr := hex.EncodeToString(hash)
						if timer, ok := gos.timers.Load(hashStr); ok {
							//Checks if timer was stopped or timed out if true answer was recieved
							if timer.(*time.Timer).Stop() {
								gos.timers.Delete(hashStr)
							}
						}
					} else {
						if nHop > 1 {
							packet.DataReply.HopLimit--
							gos.forwardPacket(&packet, dest)
						}
					}
				}
			} else if packet.SearchRequest != nil {
				name, budget, keywords := packet.ReadSearchRequest()
				if name != gos.name { //TODO: check this
					id := packet.SearchRequest.Origin + strings.Join(packet.SearchRequest.Keywords, "")
					fmt.Println(id)
					if _, ok := gos.currentSearch.Load(id); !ok {
						timer := time.AfterFunc(500*time.Millisecond, func() {
							gos.currentSearch.Delete(id)
							return
						})
						gos.currentSearch.Store(id, timer)
						fmt.Println("SEARCH REQUEST origin", name, "budget", budget, "keywords", keywords)
						packet.SearchRequest.Budget--
						if packet.SearchRequest.Budget > 0 {
							gos.fileSearch(packet, false)
						}

						//Search locally & send reply
						fileMap := gos.metaFiles.SafeCheckKeywords(keywords)
						if len(fileMap) > 0 {
							res := make([]*messages.SearchResult, len(fileMap))
							for h, f := range fileMap {
								res = append(res, gos.makeSearchResult(h, f))
							}
							serializedPacket, err := protobuf.Encode(&messages.GossipPacket{SearchReply: &messages.SearchReply{
								Origin:      gos.name,
								Destination: name,
								HopLimit:    10,
								Results:     res}})
							errorhandler.CheckErr(err, "Error when encoding packet: ", false)
							if destAddr, ok := gos.routingTable.SafeReadSpec(name); ok {
								gos.sendToPeer(serializedPacket, destAddr)
							}
						}
					}
				}
			} else if packet.SearchReply != nil {
				name, dest, nHop, searchRes := packet.ReadSearchReply()
				if dest == gos.name {
					for _, res := range searchRes {
						hashStr := hex.EncodeToString(res.MetafileHash)
						chunkList := strings.Trim(strings.Join(strings.Fields(fmt.Sprint(res.ChunkMap)), ","), "[]")
						fmt.Println("FOUND match", res.FileName, "at", name, "metafile="+hashStr, "chunks="+chunkList)

						id := name + res.FileName
						if match, ok := gos.matches.SafeReadMatch(id); ok {
							match.SafeUpdateChunks(res.ChunkMap)
						} else {
							newM := Match{
								fileName:  res.FileName,
								totChunks: res.ChunkCount,
								metaHash:  res.MetafileHash,
								chunks:    make([]uint64, 0),
								fullMatch: false}
							newM.SafeUpdateChunks(res.ChunkMap)
							gos.matches.SafeUpdateMatch(id, &newM)
						}
					}
				} else {
					if nHop > 1 {
						packet.SearchReply.HopLimit--
						gos.forwardPacket(&packet, dest)
					}
				}
			} else { //Should never happen!
				fmt.Println("Error: MESSAGE FORM UNKNOWN. Sent by", relayAddr)
			}
		}
	}
}

/*AntiEntropy makes sure everybody enventually gets all messages by sending a status message
to a random peer every second*/
func (gos *Gossiper) AntiEntropy() {
	ticker := time.NewTicker(time.Second)
	for {
		select {
		case _ = <-ticker.C:
			size := gos.knownPeers.SafeSize()
			if size > 0 {
				randomPeer := gos.selectRandomPeers(1)[0]
				gos.sendStatus(randomPeer)
			}
		default:
		}
	}
}

func (gos *Gossiper) forwardPacket(packet *messages.GossipPacket, dest string) {
	serializedPacket, err := protobuf.Encode(packet)
	errorhandler.CheckErr(err, "Error when encoding packet: ", false)
	if destAddr, ok := gos.routingTable.SafeReadSpec(dest); ok {
		gos.sendToPeer(serializedPacket, destAddr)
	}
}

/*RoutingRumors sends periodically empty rumor message to keep routing table up to date
rtimer: the period in seconds
*/
func (gos *Gossiper) RoutingRumors(rtimer int) {
	ticker := time.NewTicker(time.Duration(rtimer) * time.Second)
	for {
		select {
		case _ = <-ticker.C:
			gos.SendRumor("")
		default:
		}
	}
}

/*Hello sends the first broadcast to all peers to make yourself known*/
func (gos *Gossiper) Hello() {
	rumor := gos.prepRumor("")
	gos.pastMsg.SafeAdd(gos.name, rumor)
	gos.want.SafeInc(gos.name)
	serializedPacket, err := protobuf.Encode(&messages.GossipPacket{Rumor: rumor})
	errorhandler.CheckErr(err, "Error when encoding packet: ", false)
	gos.sendToAll(serializedPacket, []string{})
}

/*clientRequest sends a data request asked by the client and adds a new entry in the SafeMetaMap
packet: the gossip packet sent by the client
*/
func (gos *Gossiper) clientRequest(packet messages.GossipPacket) {
	fmt.Println("CLIENT REQUEST")
	//Create new entry in the SafeMetaMap
	if destAddr, ok := gos.routingTable.SafeReadSpec(packet.DataRequest.Destination); ok {
		newF := File{name: packet.DataRequest.Origin, nextChunk: 0, chunks: make(map[[32]byte][]byte)}
		var hash32 [32]byte
		copy(hash32[:], packet.DataRequest.HashValue)
		gos.metaFiles.SafeUpdateMeta(hash32, &newF)

		packet.DataRequest.Origin = gos.name
		serializedPacket, err := protobuf.Encode(&packet)
		errorhandler.CheckErr(err, "Error when encoding packet: ", false)
		hash := hex.EncodeToString(packet.DataRequest.HashValue)
		timer := time.AfterFunc(5*time.Second, func() {
			gos.timers.Delete(hash)
			gos.clientRequest(packet)
		})
		gos.timers.Store(hash, timer)
		gos.sendToPeer(serializedPacket, destAddr)
	} else {
		fmt.Println("ERROR SENDING DATA REQUEST: I do not know", packet.DataRequest.Destination)
	}
}

/*sendChunkRequest send a request for a chunk
hash: the chunk's hash value
dest: the final destination to send to
*/
func (gos *Gossiper) sendChunkRequest(hash []byte, dest string) {
	if destAddr, ok := gos.routingTable.SafeReadSpec(dest); ok {
		packet := messages.GossipPacket{DataRequest: &messages.DataRequest{
			Origin:      gos.name,
			Destination: dest,
			HopLimit:    10,
			HashValue:   hash}}
		serializedPacket, err := protobuf.Encode(&packet)
		errorhandler.CheckErr(err, "Error when encoding packet: ", false)
		hashStr := hex.EncodeToString(hash)
		timer := time.AfterFunc(5*time.Second, func() {
			gos.timers.Delete(hashStr)
			gos.sendChunkRequest(hash, dest)
		})
		gos.timers.Store(hashStr, timer)
		gos.sendToPeer(serializedPacket, destAddr)
	} else {
		fmt.Println("ERROR SENDING DATA REQUEST: I do not know", dest)
	}
}

/*fileSearch sends a search request with the available budget
packet: the gossip packet containing the search request
*/
func (gos *Gossiper) fileSearch(packet messages.GossipPacket, withIncrement bool) {

	//First check if we have two full match for this search
	if gos.matches.IsSearchOver(packet.SearchRequest.Keywords) {
		fmt.Println("SEARCH FINISHED")
		return
	}

	budget := int(packet.SearchRequest.Budget)
	nPeers := gos.knownPeers.SafeSize()
	peers := gos.selectRandomPeers(budget)

	if nPeers > budget {
		packet.SearchRequest.Budget = 1
		serializedPacket, err := protobuf.Encode(&packet)
		errorhandler.CheckErr(err, "Error when encoding packet: ", false)
		for _, peer := range peers {
			gos.sendToPeer(serializedPacket, peer)
		}
	} else {
		available := budget / nPeers
		rest := budget - nPeers*available
		for i, peer := range peers {
			if i < rest {
				packet.SearchRequest.Budget = uint64(available + 1)
			} else {
				packet.SearchRequest.Budget = uint64(available)
			}
			serializedPacket, err := protobuf.Encode(&packet)
			errorhandler.CheckErr(err, "Error when encoding packet: ", false)
			gos.sendToPeer(serializedPacket, peer)
		}
	}

	if withIncrement {
		id := packet.SearchRequest.Origin + strings.Join(packet.SearchRequest.Keywords, "")
		timer := time.AfterFunc(time.Second, func() {
			gos.timers.Delete(id)
			packet.SearchRequest.Budget *= 2
			if packet.SearchRequest.Budget <= 32 {
				gos.fileSearch(packet, true)
			}
			return
		})

		//Check if not already a timer for that search, in that case stops it and stores the new one
		if oldTimer, ok := gos.timers.Load(id); ok {
			oldTimer.(*time.Timer).Stop()
			gos.timers.Delete(id)
		}
		gos.timers.Store(id, timer)
	}
}

func (gos *Gossiper) makeSearchResult(hash [32]byte, file *File) *messages.SearchResult {
	count := file.SafeReadSize()
	list := make([]uint64, 0)
	var temp = make([]byte, 32)
	copy(temp, hash[:])
	for i := 1; i <= count; i++ {
		list = append(list, uint64(i))
	}
	return &messages.SearchResult{
		FileName:     file.SafeReadName(),
		MetafileHash: temp,
		ChunkMap:     list,
		ChunkCount:   uint64(count)}
}

/*sendToPeer sends a packet to a given peer
packet: the packet to send
peer: the peer's address of the form (IP:Port)
*/
func (gos *Gossiper) sendToPeer(packet []byte, peer string) {
	peerAddr, err := net.ResolveUDPAddr("udp4", peer)
	errorhandler.CheckErr(err, "Error when resolving peer UDP addr: ", false)
	gos.peersConn.WriteToUDP(packet, peerAddr)
}

/*sendToAll sends a packet to all peers known by the gossiper, can include an exception
packet: the packet to send
except: a slice of peers to omit in the transmission (if none is empty)
*/
func (gos *Gossiper) sendToAll(packet []byte, except []string) {
	if gos.knownPeers.SafeSize() != 0 {
		for _, peer := range gos.knownPeers.SafeRead() {
			if !contains(except, peer) {
				gos.sendToPeer(packet, peer)
			}
		}
	}
}

/*sendStatus sends a packet to a given peer
peer: the peer's address of the form (IP:Port)
*/
func (gos *Gossiper) sendStatus(peer string) {
	serializedPacket, err := protobuf.Encode(&messages.GossipPacket{Status: &messages.StatusPacket{Want: gos.want.SafeStatusList()}})
	errorhandler.CheckErr(err, "Error when encoding packet: ", false)
	gos.sendToPeer(serializedPacket, peer)
}

/*addPrintPeers adds relay address if not contained already and prints all known peers*/
func (gos *Gossiper) addPrintPeers(addr string) {
	//Adds relay address if not contained already
	gos.AddPeer(addr)
	fmt.Println("PEERS ", strings.Join(gos.knownPeers.SafeRead(), ","))
}

/*PrintClientMsg outputs the content of the client message in the console*/
func (gos *Gossiper) PrintClientMsg(msg string) {
	fmt.Println("CLIENT MESSAGE", msg)
	fmt.Println("PEERS", strings.Join(gos.knownPeers.SafeRead(), ","))
}

/*rumormongering is a simple rumoring protocol
msg: the message to send
except: a slice of peers to omit in the transmission (if none is empty) of the form (IP:Port)
first: set to true when starting a new rumor
*/
func (gos *Gossiper) rumormongering(rumor *messages.RumorMessage, except []string, first bool) {
	size := gos.knownPeers.SafeSize()
	if len(except) < size {
		randomPeer := gos.selectRandomPeers(1)[0]
		for contains(except, randomPeer) {
			randomPeer = gos.selectRandomPeers(1)[0]
		}

		serializedPacket, err := protobuf.Encode(&messages.GossipPacket{Rumor: rumor})
		errorhandler.CheckErr(err, "Error when encoding packet: ", false)

		if first {
			fmt.Println("MONGERING with", randomPeer)
		} else {
			fmt.Println("FLIPPED COIN sending rumor to", randomPeer)
		}

		gos.sendToPeer(serializedPacket, randomPeer)
		gos.lastSent.SafeUpdate(randomPeer, rumor)
		timer := time.AfterFunc(time.Second, func() {
			gos.timers.Delete(randomPeer)
			if (rand.Int() % 2) == 0 {
				gos.rumormongering(rumor, append(except, randomPeer), false)
			}
			return
		})

		//Check if not already a timer for that peer, in that case stops it and stores the new one
		if oldTimer, ok := gos.timers.Load(randomPeer); ok {
			oldTimer.(*time.Timer).Stop()
			gos.timers.Delete(randomPeer)
		}
		gos.timers.Store(randomPeer, timer)
	}
}

/*compareStatus compares the given status with the gossiper's status and send the appropriate message
msgStatus: the status to compare with
peer: the peer's address of the form (IP:Port)
*/
//TODO: sendToPeer or rumonmongering ??
func (gos *Gossiper) compareStatus(msgStatus []messages.PeerStatus, peer string) {
	iNeed := false

	peersCopy := gos.want.MakeSafeCopy()

	for _, hisStatus := range msgStatus {
		if _, ok := gos.want.SafeRead(hisStatus.Identifier); ok {
			peersCopy[hisStatus.Identifier] = true

			if hisStatus.NextID > gos.want.SafeID(hisStatus.Identifier) {
				iNeed = true
			}
			if hisStatus.NextID < gos.want.SafeID(hisStatus.Identifier) {
				//Send him packet if I have it
				if list, ok := gos.pastMsg.SafeReadSpec(hisStatus.Identifier); ok {
					serializedPacket, err := protobuf.Encode(&messages.GossipPacket{Rumor: list[hisStatus.NextID-1]})
					errorhandler.CheckErr(err, "Error when encoding packet: ", false)
					gos.sendToPeer(serializedPacket, peer)
					return
				}
			}
		} else {
			iNeed = true
			gos.want.SafeUpdate(hisStatus.Identifier, &messages.PeerStatus{
				Identifier: hisStatus.Identifier,
				NextID:     1,
			})
		}
	}

	for key, val := range peersCopy {
		if !val {
			//Send him packet if I have it
			if list, ok := gos.pastMsg.SafeReadSpec(key); ok {
				serializedPacket, err := protobuf.Encode(&messages.GossipPacket{Rumor: list[0]})
				errorhandler.CheckErr(err, "Error when encoding packet: ", false)
				gos.sendToPeer(serializedPacket, peer)
				return
			}
		}
	}

	if iNeed {
		//Ask him for rumors
		gos.sendStatus(peer)
	} else {
		fmt.Println("IN SYNC WITH", peer)
		if (rand.Int() % 2) == 0 {
			gos.rumormongering(gos.lastSent.SafeRead(peer), []string{peer}, false)
		}
	}
}

/*selectRandomPeers selects n random peers from the known peers
If n is bigger than the number of peers, returns all peers.
n: the number of peers to select
*/
func (gos *Gossiper) selectRandomPeers(n int) []string {
	res := make([]string, 0)
	size := gos.knownPeers.SafeSize()

	if n > size {
		return gos.knownPeers.SafeRead()
	}

	for len(res) < n {
		randomPeer := gos.knownPeers.SafeReadSpec(rand.Int() % size)
		if !contains(res, randomPeer) {
			res = append(res, randomPeer)
		}
	}
	return res
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

/*UINTcontains checks if the given uint64 is contained in the given slice
list: slice of uint64 to check
i: the uint64 to look for
*/
func UINTcontains(list []uint64, i uint64) bool {
	for _, test := range list {
		if test == i {
			return true
		}
	}
	return false
}

/*GetName returns the name of the Gossiper*/
func (gos *Gossiper) GetName() string {
	return gos.name
}

/*GetPeers returns the list of known peers*/
func (gos *Gossiper) GetPeers() []string {
	return gos.knownPeers.SafeRead()
}

/*AddPeer returns the list of known peers*/
func (gos *Gossiper) AddPeer(newPeer string) {
	gos.knownPeers.SafeAdd(newPeer)
}

/*GetRumorMessages returns the list of rumor messages in the form Origin: Message*/
func (gos *Gossiper) GetRumorMessages() []string {
	return gos.pastMsg.GetSafePast()
}

/*GetPrivateMessages returns the list of private messages exchanged with the given peer in the form Origin: Message
name: the name of the peer
*/
func (gos *Gossiper) GetPrivateMessages(name string) []string {
	allMsg := make([]string, 0)
	if rumList, ok := gos.pastPrivate.SafeReadSpec(name); ok {
		for _, msg := range rumList {
			allMsg = append(allMsg, msg.Origin+": "+msg.Text)
		}
	}
	return allMsg
}

/*GetNodesName returns the list of peers name for private messaging*/
func (gos *Gossiper) GetNodesName() []string {
	return gos.routingTable.GetSafeKeys()
}

/*SendRumor start the rumor mongering process and save message in memory
msg: the message to send
*/
func (gos *Gossiper) SendRumor(msg string) {
	if gos.knownPeers.SafeSize() == 0 {
		fmt.Println("Error: could not retransmit message, I do not know any other peers!")
	} else {
		//Add to past
		newRumor := gos.prepRumor(msg)
		gos.pastMsg.SafeAdd(gos.name, newRumor)
		gos.want.SafeInc(gos.name)

		//Transmit to a random peer if at least one is known
		gos.rumormongering(newRumor, []string{}, true)
	}
}

/*prepRumor returns the rumor message with the given text
msg: the text of the rumor message
*/
func (gos *Gossiper) prepRumor(msg string) *messages.RumorMessage {
	rumor := messages.RumorMessage{
		Origin: gos.name,
		ID:     gos.want.SafeID(gos.name),
		Text:   msg}
	return &rumor
}

/*indexFile indexes the given file and returns the err code appropriate
filename: the file to index
*/
func (gos *Gossiper) indexFile(filename string) error {
	fmt.Println("REQUESTING INDEXING filename", filename) //Test Gv2
	const sizeMax = 2097152                               //256 * 8192
	newF := File{name: filename, nextChunk: -1, chunks: make(map[[32]byte][]byte)}

	file, err := os.Open("_SharedFiles/" + filename)
	if err != nil {
		return err
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		return err
	}
	if info.Size() > sizeMax {
		return errors.New("ERROR: File too big to be indexed " + filename)
	}

	reader := bufio.NewReader(file)
	buffer := make([]byte, 0)
	chunk := make([]byte, buffSize)
	s := 0
	var count int

	for {
		if count, err = reader.Read(chunk); err != nil {
			break
		}
		s++
		hash := sha256.Sum256(chunk[:count])
		buffer = append(buffer, hash[:]...)
		gos.metaFiles.SafeUpdateData(hash, &newF)
		newF.SafeUpdateChunk(hash, chunk[:count])
	}

	if err == io.EOF {
		err = nil
	} else {
		return err
	}

	hashFile := sha256.Sum256(buffer)
	var temp = make([]byte, 32)
	copy(temp, hashFile[:])
	fmt.Println("HASH", hex.EncodeToString(temp))
	newF.totChunks = s
	newF.metaFile = make([]byte, len(buffer))
	copy(newF.metaFile, buffer)
	gos.metaFiles.SafeUpdateMeta(hashFile, &newF)
	return err
}
