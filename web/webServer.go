package web

import (
	"encoding/hex"
	"encoding/json"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"strings"

	"github.com/dedis/protobuf"
	"github.com/gorilla/mux"
	"github.com/montalex/Peerster/errorhandler"
	"github.com/montalex/Peerster/gossiper"
	"github.com/montalex/Peerster/messages"
)

/*Run runs the server for the Peerster application*/
func Run(gos *gossiper.Gossiper, UIPort string) {
	router := mux.NewRouter()

	destAddr, err := net.ResolveUDPAddr("udp4", "127.0.0.1:"+UIPort)
	errorhandler.CheckErr(err, "Error when resolving UDP dest address: ", true)

	myAddr, err := net.ResolveUDPAddr("udp", "127.0.0.1:4285")
	errorhandler.CheckErr(err, "Error when resolving UDP local address: ", true)

	udpConn, err := net.ListenUDP("udp", myAddr)
	errorhandler.CheckErr(err, "Error with UDP connection: ", true)
	defer udpConn.Close()

	router.HandleFunc("/id", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(gos.GetName())
	}).Methods("GET")

	router.HandleFunc("/node", func(w http.ResponseWriter, r *http.Request) {
		for _, elem := range gos.GetPeers() {
			json.NewEncoder(w).Encode(elem)
		}
	}).Methods("GET")

	router.HandleFunc("/name", func(w http.ResponseWriter, r *http.Request) {
		for _, elem := range gos.GetNodesName() {
			json.NewEncoder(w).Encode(elem)
		}
	}).Methods("GET")

	router.HandleFunc("/node", func(w http.ResponseWriter, r *http.Request) {
		body, _ := ioutil.ReadAll(r.Body)
		newPeer := string(body)
		gos.AddPeer(newPeer)
	}).Methods("POST")

	router.HandleFunc("/message", func(w http.ResponseWriter, r *http.Request) {
		for _, elem := range gos.GetRumorMessages() {
			json.NewEncoder(w).Encode(elem)
		}
	}).Methods("GET")

	router.HandleFunc("/message", func(w http.ResponseWriter, r *http.Request) {
		body, _ := ioutil.ReadAll(r.Body)
		msg := string(body)
		packet := messages.GossipPacket{Simple: &messages.SimpleMessage{
			OriginalName:  "",
			RelayPeerAddr: "",
			Contents:      msg}}
		serializedPacket, err := protobuf.Encode(&packet)
		errorhandler.CheckErr(err, "Error when encoding packet: ", false)

		_, err = udpConn.WriteToUDP(serializedPacket, destAddr)
		errorhandler.CheckErr(err, "Error when sending UDP msg: ", true)
	}).Methods("POST")

	router.HandleFunc("/file", func(w http.ResponseWriter, r *http.Request) {
		body, _ := ioutil.ReadAll(r.Body)
		msg := string(body)
		packet := messages.GossipPacket{Simple: &messages.SimpleMessage{
			OriginalName:  "file",
			RelayPeerAddr: "",
			Contents:      msg}}
		serializedPacket, err := protobuf.Encode(&packet)
		errorhandler.CheckErr(err, "Error when encoding packet: ", false)

		_, err = udpConn.WriteToUDP(serializedPacket, destAddr)
		errorhandler.CheckErr(err, "Error when sending UDP msg: ", true)
	}).Methods("POST")

	router.HandleFunc("/private/{name}", func(w http.ResponseWriter, r *http.Request) {
		name := mux.Vars(r)["name"]
		for _, elem := range gos.GetPrivateMessages(name) {
			json.NewEncoder(w).Encode(elem)
		}
	}).Methods("GET")

	router.HandleFunc("/private/{name}", func(w http.ResponseWriter, r *http.Request) {
		name := mux.Vars(r)["name"]
		body, _ := ioutil.ReadAll(r.Body)
		msg := string(body)

		packet := messages.GossipPacket{Private: &messages.PrivateMessage{
			Origin:      "",
			ID:          0,
			Text:        msg,
			Destination: name,
			HopLimit:    10}}
		serializedPacket, err := protobuf.Encode(&packet)
		errorhandler.CheckErr(err, "Error when encoding packet: ", false)

		_, err = udpConn.WriteToUDP(serializedPacket, destAddr)
		errorhandler.CheckErr(err, "Error when sending UDP msg: ", true)
	}).Methods("POST")

	router.HandleFunc("/match/{keywords}", func(w http.ResponseWriter, r *http.Request) {
		keywords := mux.Vars(r)["keywords"]
		words := strings.Split(keywords, ",")
		for _, name := range gos.GetMatches(words) {
			json.NewEncoder(w).Encode(name)
		}
	}).Methods("GET")

	router.HandleFunc("/match/{keywords}", func(w http.ResponseWriter, r *http.Request) {
		keywords := mux.Vars(r)["keywords"]
		words := strings.Split(keywords, ",")
		packet := messages.GossipPacket{SearchRequest: &messages.SearchRequest{
			Origin:   "",
			Budget:   0,
			Keywords: words}}
		serializedPacket, err := protobuf.Encode(&packet)
		errorhandler.CheckErr(err, "Error when encoding packet: ", false)

		_, err = udpConn.WriteToUDP(serializedPacket, destAddr)
		errorhandler.CheckErr(err, "Error when sending UDP msg: ", true)
	}).Methods("POST")

	router.HandleFunc("/download/{name}", func(w http.ResponseWriter, r *http.Request) {
		name := mux.Vars(r)["name"]
		gos.Download(name)
	}).Methods("POST")

	router.HandleFunc("/request/{name}", func(w http.ResponseWriter, r *http.Request) {
		name := mux.Vars(r)["name"]
		body, _ := ioutil.ReadAll(r.Body)
		msg := string(body)
		req := strings.Split(msg, ":")

		hash, err := hex.DecodeString(req[1])
		errorhandler.CheckErr(err, "Error when decoding hash: ", true)
		packet := messages.GossipPacket{DataRequest: &messages.DataRequest{
			Origin:      req[0],
			Destination: name,
			HopLimit:    10,
			HashValue:   hash}}
		serializedPacket, err := protobuf.Encode(&packet)
		errorhandler.CheckErr(err, "Error when encoding packet: ", false)

		_, err = udpConn.WriteToUDP(serializedPacket, destAddr)
		errorhandler.CheckErr(err, "Error when sending UDP msg: ", true)
	}).Methods("POST")

	router.PathPrefix("/").Handler(http.FileServer(http.Dir("./web/static/")))

	log.Fatal(http.ListenAndServe(":8080", router))
}
