package gossiper

import (
	"sync"
)

/*MetaFile represent the Gossiper's meta file*/
type MetaFile struct {
	name     string
	size     int
	metaFile []byte
}

/*SafeMetaMap represent a map of the Gossiper's meta files with a lock*/
type SafeMetaMap struct {
	files map[[32]byte]*MetaFile
	mux   sync.RWMutex
}

/*SafeUpdate safely updates the meta file map for the given hash
hash: the meta file's hash
mFile: the new meta file
*/
func (m *SafeMetaMap) SafeUpdate(hash [32]byte, mFile *MetaFile) {
	m.mux.Lock()
	m.files[hash] = mFile
	m.mux.Unlock()
}
