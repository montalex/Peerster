package gossiper

import (
	"fmt"
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
	fmt.Println("SafeMsgMap start SafeUpdate")
	m.mux.Lock()
	m.messages[name] = newMsg
	m.mux.Unlock()
	fmt.Println("SafeMsgMap end SafeUpdate")
}

/*SafeRead safely reads the last sent message for the given peers
name: the name of the peer
*/
func (m *SafeMsgMap) SafeRead(name string) *messages.RumorMessage {
	fmt.Println("SafeMsgMap start SafeRead")
	m.mux.RLock()
	defer m.mux.RUnlock()

	return m.messages[name]
}
