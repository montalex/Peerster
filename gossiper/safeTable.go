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
	fmt.Println("SafeTable start SafeReadSpec")
	t.mux.RLock()
	defer t.mux.RUnlock()

	v, ok := t.table[name]
	fmt.Println("SafeTable end SafeReadSpec")
	return v, ok
}

/*GetSafeKeys reads the whole table and returns all the keys*/
func (t *SafeTable) GetSafeKeys() []string {
	fmt.Println("SafeTable start GetSafeKeys")
	t.mux.RLock()
	defer t.mux.RUnlock()

	allNames := make([]string, 0)
	for name := range t.table {
		allNames = append(allNames, name)
	}
	fmt.Println("SafeTable end GetSafeKeys")
	return allNames
}

/*SafeUpdate updates safely the routing table
name: the name of the peer
newRoute: the new route to take to reach the peer (of the form IP:Port)
*/
func (t *SafeTable) SafeUpdate(name string, newRoute string) {
	fmt.Println("SafeTable start SafeUpdate")
	fmt.Println("DSDV", name, newRoute)
	t.mux.Lock()
	t.table[name] = newRoute
	t.mux.Unlock()
	fmt.Println("SafeTable end SafeUpdate")
}
