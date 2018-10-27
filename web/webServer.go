package web

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/montalex/Peerster/gossiper"
)

/*Run runs the server for the Peerster application*/
func Run(gos *gossiper.Gossiper) {
	router := mux.NewRouter()

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
		for _, elem := range gos.GetMessages() {
			json.NewEncoder(w).Encode(elem)
		}
	}).Methods("GET")

	router.HandleFunc("/message", func(w http.ResponseWriter, r *http.Request) {
		body, _ := ioutil.ReadAll(r.Body)
		msg := string(body)
		gos.PrintClientMsg(msg)
		gos.SendRumor(msg)
	}).Methods("POST")

	router.PathPrefix("/").Handler(http.FileServer(http.Dir("./web/static/")))

	log.Fatal(http.ListenAndServe(":8080", router))
}
