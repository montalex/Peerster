package gossiper

import (
	"sync"

	"github.com/montalex/Peerster/messages"
)

/*SafeStatus represent the Gossiper's peer status as a map with a lock*/
type SafeStatus struct {
	m   map[string]*messages.PeerStatus
	mux sync.RWMutex
}

/*SafeRead read the value of the given peer's status
name: the name of the peer
*/
func (s *SafeStatus) SafeRead(name string) (*messages.PeerStatus, bool) {
	s.mux.RLock()
	defer s.mux.RUnlock()

	res, ok := s.m[name]
	return res, ok
}

/*SafeID returns the ID of the given peer
name: the name of the peer
*/
func (s *SafeStatus) SafeID(name string) uint32 {
	s.mux.RLock()
	defer s.mux.RUnlock()

	res := s.m[name]
	return res.NextID
}

/*SafeUpdate safely update the peer status
name: the name of the peer
newStatus: the new status
*/
func (s *SafeStatus) SafeUpdate(name string, newStatus *messages.PeerStatus) {
	s.mux.Lock()
	s.m[name] = newStatus
	s.mux.Unlock()
}

/*SafeInc safely increments the ID in the peer status
name: the name of the peer
*/
func (s *SafeStatus) SafeInc(name string) {
	s.mux.Lock()
	s.m[name].NextID++
	s.mux.Unlock()
}

/*SafeStatusList returns a slice of the peerStatus*/
func (s *SafeStatus) SafeStatusList() []messages.PeerStatus {
	s.mux.RLock()
	defer s.mux.RUnlock()
	wantSlice := []messages.PeerStatus{}
	for _, p := range s.m {
		wantSlice = append(wantSlice, messages.PeerStatus{Identifier: p.Identifier, NextID: p.NextID})
	}
	return wantSlice
}

/*MakeSafeCopy returns a copy of the peer status map*/
func (s *SafeStatus) MakeSafeCopy() map[string]bool {
	s.mux.RLock()
	defer s.mux.RUnlock()

	peersCopy := make(map[string]bool)
	for key := range s.m {
		peersCopy[key] = false
	}
	return peersCopy
}
