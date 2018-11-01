package gossiper

import (
	"fmt"
	"sync"
)

/*SafePeers represent the Gossiper's known peers list as a slice with a lock*/
type SafePeers struct {
	peers []string
	mux   sync.RWMutex
}

/*SafeRead reads the slice of known peers*/
func (p *SafePeers) SafeRead() []string {
	fmt.Println("SafePeers start SafeRead")
	p.mux.RLock()
	defer p.mux.RUnlock()

	return p.peers
}

/*SafeReadSpec reads a specific place in the slice of known peers*/
func (p *SafePeers) SafeReadSpec(num int) string {
	fmt.Println("SafePeers start SafeReadSpec")
	p.mux.RLock()
	defer p.mux.RUnlock()

	return p.peers[num]
}

/*SafeSize returns the size of the known peers list*/
func (p *SafePeers) SafeSize() int {
	fmt.Println("SafePeers start SafeSize")
	p.mux.RLock()
	defer p.mux.RUnlock()

	return len(p.peers)
}

/*SafeAdd adds safely a peer to the gossiper's list
newPeer: the new peer to add
*/
func (p *SafePeers) SafeAdd(newPeer string) {
	fmt.Println("SafePeers start SafeAdd")
	p.mux.Lock()
	if !contains(p.peers, newPeer) {
		p.peers = append(p.peers, newPeer)
	}
	p.mux.Unlock()
	fmt.Println("SafePeers end SafeAdd")
}
