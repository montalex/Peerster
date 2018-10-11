package web

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/gorilla/mux"
)

func handlePeers(w http.ResponseWriter, r *http.Request) {
	json.NewEncoder(w).Encode("128.0.0.2:5000")
}

/*Run runs the server for the Peerster application*/
func Run() {
	r := mux.NewRouter()
	r.HandleFunc("/node", handlePeers)
	r.PathPrefix("/").Handler(http.FileServer(http.Dir("./web/static/")))

	log.Fatal(http.ListenAndServe(":8080", r))
}
