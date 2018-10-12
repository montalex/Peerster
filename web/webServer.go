package web

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/montalex/Peerster/gossiper"
)

/*Run runs the server for the Peerster application*/
func Run(gos *gossiper.Gossiper) {
	r := mux.NewRouter()
	r.HandleFunc("/node", func(w http.ResponseWriter, r *http.Request) {
		for _, elem := range gos.GetPeers() {
			json.NewEncoder(w).Encode(elem)
		}
	})
	r.PathPrefix("/").Handler(http.FileServer(http.Dir("./web/static/")))

	log.Fatal(http.ListenAndServe(":8080", r))
}
