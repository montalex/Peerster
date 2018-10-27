package gossiper

import (
	"fmt"
	"sync"
)

/*SafeTable represent the Gossiper's routing table with a lock*/
type SafeTable struct {
	table map[string]string
	mux   sync.RWMutex
}

/*SafeReadSpec reads the table for the given name
name: the name of the peer
*/
func (t *SafeTable) SafeReadSpec(name string) (string, bool) {
	t.mux.RLock()
	defer t.mux.RUnlock()

	v, ok := t.table[name]
	return v, ok
}

/*SafeRead reads the whole table*/
func (t *SafeTable) SafeRead() map[string]string {
	t.mux.RLock()
	defer t.mux.RUnlock()

	return t.table
}

/*SafeUpdate updates safely the routing table
name: the name of the peer
newRoute: the new route to take to reach the peer (of the form IP:Port)
*/
func (t *SafeTable) SafeUpdate(name string, newRoute string) {
	fmt.Println("DSDV", name, newRoute)
	t.mux.Lock()
	t.table[name] = newRoute
	t.mux.Unlock()
}
