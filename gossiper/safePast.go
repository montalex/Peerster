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

/*GetSafePast reads the whole list of past mesages in the form Origin: Message*/
func (p *SafePast) GetSafePast() []string {
	p.mux.RLock()
	defer p.mux.RUnlock()

	allMsg := make([]string, 0)
	for key, rumList := range p.messagesList {
		for _, msg := range rumList {
			//Do not display routing rumor in GUI
			if msg.Text != "" {
				allMsg = append(allMsg, key+": "+msg.Text)
			}
		}
	}
	return allMsg
}

/*SafeReadSpec reads the list of past mesages for the given peer
name: the name of the peer
*/
func (p *SafePast) SafeReadSpec(name string) ([]*messages.RumorMessage, bool) {
	p.mux.RLock()
	defer p.mux.RUnlock()

	res, ok := p.messagesList[name]
	return res, ok
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
