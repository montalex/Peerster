package gossiper

import (
	"sync"

	"github.com/montalex/Peerster/messages"
)

/*SafeMsgMap represent the Gossiper's last message sent for each peers with a lock*/
type SafeMsgMap struct {
	messages map[string]*messages.RumorMessage
	mux      sync.RWMutex
}

/*SafeUpdate safely updates the messages map for the given peers
name: the name of the peer
newMsg: the new message
*/
func (m *SafeMsgMap) SafeUpdate(name string, newMsg *messages.RumorMessage) {
	m.mux.Lock()
	m.messages[name] = newMsg
	m.mux.Unlock()
}

/*SafeRead safely reads the last sent message for the given peers
name: the name of the peer
*/
func (m *SafeMsgMap) SafeRead(name string) *messages.RumorMessage {
	m.mux.RLock()
	defer m.mux.RUnlock()

	return m.messages[name]
}
