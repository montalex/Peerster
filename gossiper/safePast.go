package gossiper

import (
	"sync"

	"github.com/montalex/Peerster/messages"
)

/*SafePast represent the Gossiper's memory for the messages as a list with a lock*/
type SafePast struct {
	messagesList map[string][]*messages.RumorMessage
	mux          sync.RWMutex
}

/*SafeRead reads the whole list of past mesages*/
func (p *SafePast) SafeRead() map[string][]*messages.RumorMessage {
	p.mux.RLock()
	defer p.mux.RUnlock()

	return p.messagesList
}

/*SafeReadSpec reads the list of past mesages for the given peer
name: the name of the peer
*/
func (p *SafePast) SafeReadSpec(name string) []*messages.RumorMessage {
	p.mux.RLock()
	defer p.mux.RUnlock()

	return p.messagesList[name]
}

/*SafeAdd safely add the given message to the list
name: the name of the peer
newMsg: the new message
*/
func (p *SafePast) SafeAdd(name string, newMsg *messages.RumorMessage) {
	p.mux.Lock()
	p.messagesList[name] = append(p.messagesList[name], newMsg)
	p.mux.Unlock()
}
