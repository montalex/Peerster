package gossiper

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"sync"

	"github.com/montalex/Peerster/messages"
)

/*BlockChain represents a gossiper's block chain with a lock*/
type BlockChain struct {
	chain    map[[32]byte]*messages.Block
	leafHash [32]byte
	rootHash [32]byte
	size     int
	mux      sync.RWMutex
}

/*CurrentBlock represents a gossiper's current block to mine*/
type CurrentBlock struct {
	b   *messages.Block
	mux sync.RWMutex
}

/*ClaimedNames represents the gossiper's memory for the claimed file names*/
type ClaimedNames struct {
	names map[string]bool
	mux   sync.RWMutex
}

/*SafeUpdateBlock safely updates the current transaction list
newT: the new transaction to include
*/
func (c *CurrentBlock) SafeUpdateBlock(newT *messages.TxPublish) {
	c.mux.Lock()
	c.b.Transactions = append(c.b.Transactions, *newT)
	c.mux.Unlock()
}

/*SafeClean safetly cleans the current transaction list*/
func (c *CurrentBlock) SafeClean() {
	c.mux.Lock()
	c.b.Transactions = nil
	c.mux.Unlock()
}

/*WasSeen checks is the given transaction was already seen
t: the transaction to check*/
func (c *CurrentBlock) WasSeen(t *messages.TxPublish) bool {
	c.mux.RLock()
	defer c.mux.RUnlock()
	hash := t.Hash()
	for _, t := range c.b.Transactions {
		if hash == t.Hash() {
			return true
		}
	}
	return false
}

/*SafeUpdateNames safely updates the claimed names list
newN: the new name to append*/
func (n *ClaimedNames) SafeUpdateNames(newN string) {
	n.mux.Lock()
	n.names[newN] = true
	n.mux.Unlock()
}

/*IsNameClaimed checks if the given name is already claimed
name: the name to check*/
func (n *ClaimedNames) IsNameClaimed(name string) bool {
	n.mux.RLock()
	defer n.mux.RUnlock()

	return n.names[name]
}

/*GetPrevHash returns the preHash for the new block*/
func (b *BlockChain) GetPrevHash() (ret [32]byte) {
	b.mux.RLock()
	defer b.mux.RUnlock()
	copy(ret[:], b.leafHash[:])
	return
}

/*GetChainPrint returns the print for the chain*/
func (b *BlockChain) GetChainPrint() string {
	b.mux.RLock()
	defer b.mux.RUnlock()

	res := ""
	hash := b.leafHash
	for !bytes.Equal(hash[:], b.rootHash[:]) {
		current := b.chain[hash]
		newB := hex.EncodeToString(hash[:]) + ":" + hex.EncodeToString(current.PrevHash[:]) + ":"
		size := len(current.Transactions)
		for i, t := range current.Transactions {
			if i == (size - 1) {
				newB += t.File.Name
			} else {
				newB += t.File.Name + ","
			}
		}
		res += newB + " "
		hash = current.PrevHash
	}
	return res
}

/*SafeUpdateChain safely updates the blockchains with the given block
newB: the block to append to the chain*/
func (b *BlockChain) SafeUpdateChain(newB *messages.Block) {
	b.mux.Lock()
	hash := newB.Hash()
	b.chain[hash] = newB
	if bytes.Equal(newB.PrevHash[:], b.leafHash[:]) {
		b.leafHash = hash
	}
	b.mux.Unlock()
}

/*ParentSeen is the parent of the given block has been seen in the chain*/
func (b *BlockChain) ParentSeen(block *messages.Block) bool {
	b.mux.RLock()
	defer b.mux.RUnlock()
	res := [32]byte{}
	if bytes.Equal(block.PrevHash[:], res[:]) {
		return true
	}
	_, ok := b.chain[block.PrevHash]
	return ok
}

/*RandomNonce creates a random nonce for the current block*/
func RandomNonce() (ret [32]byte) {
	rand.Read(ret[:])
	return
}

/*SafeUpdateNonce safely update the nonce of the current block*/
func (c *CurrentBlock) SafeUpdateNonce() {
	c.mux.Lock()
	c.b.Nonce = RandomNonce()
	c.mux.Unlock()
}

/*SafeHash safely compute the hash of the current block*/
func (c *CurrentBlock) SafeHash() [32]byte {
	c.mux.RLock()
	defer c.mux.RUnlock()

	return c.b.Hash()
}
