package gossiper

import (
	"fmt"
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
	fmt.Println("SafeStatus start SafeRead")
	s.mux.RLock()
	defer s.mux.RUnlock()

	res, ok := s.m[name]
	fmt.Println("SafeStatus end SafeRead")
	return res, ok
}

/*SafeID returns the ID of the given peer
name: the name of the peer
*/
func (s *SafeStatus) SafeID(name string) uint32 {
	fmt.Println("SafeStatus start SafeID")
	s.mux.RLock()
	defer s.mux.RUnlock()

	res := s.m[name]
	fmt.Println("SafeStatus end SafeID")
	return res.NextID
}

/*SafeUpdate safely update the peer status
name: the name of the peer
newStatus: the new status
*/
func (s *SafeStatus) SafeUpdate(name string, newStatus *messages.PeerStatus) {
	fmt.Println("SafeStatus start SafeUpdate")
	s.mux.Lock()
	s.m[name] = newStatus
	s.mux.Unlock()
	fmt.Println("SafeStatus end SafeUpdate")
}

/*SafeInc safely increments the ID in the peer status
name: the name of the peer
*/
func (s *SafeStatus) SafeInc(name string) {
	fmt.Println("SafeStatus start SafeInc")
	s.mux.Lock()
	s.m[name].NextID++
	s.mux.Unlock()
	fmt.Println("SafeStatus end SafeInc")
}

/*SafeStatusList returns a slice of the peerStatus*/
func (s *SafeStatus) SafeStatusList() []messages.PeerStatus {
	fmt.Println("OUPS")
	s.mux.RLock()
	fmt.Println("NOPE")
	defer s.mux.RUnlock()
	fmt.Println("MAYBE")
	wantSlice := []messages.PeerStatus{}
	fmt.Println("X")
	for _, p := range s.m {
		fmt.Println("Y")
		wantSlice = append(wantSlice, messages.PeerStatus{Identifier: p.Identifier, NextID: p.NextID})
	}
	fmt.Println("Z")
	return wantSlice
}

/*MakeSafeCopy returns a copy of the peer status map*/
func (s *SafeStatus) MakeSafeCopy() map[string]bool {
	fmt.Println("SafeStatus start MakeSafeCopy")
	s.mux.RLock()
	defer s.mux.RUnlock()

	peersCopy := make(map[string]bool)
	for key := range s.m {
		peersCopy[key] = false
	}
	fmt.Println("SafeStatus end MakeSafeCopy")
	return peersCopy
}
