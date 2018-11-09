package gossiper

import (
	"fmt"
	"sync"
)

/*File represent a Gossiper's file*/
type File struct {
	name      string
	totChunks int
	metaFile  []byte
	nextChunk int
	chunks    map[[32]byte][]byte
	mux       sync.RWMutex
}

/*SafeReadName safely reads the file's name*/
func (f *File) SafeReadName() string {
	f.mux.RLock()
	defer f.mux.RUnlock()

	return f.name
}

/*SafeNextChuck safely returns the file's next required chunk*/
func (f *File) SafeNextChuck() []byte {
	f.mux.RLock()
	defer f.mux.RUnlock()
	fmt.Println("f.nextChunk", f.nextChunk)
	fmt.Print("f.metafile", f.metaFile)

	var next = make([]byte, 32)
	copy(next, f.metaFile[f.nextChunk*32:(f.nextChunk+1)*32])
	return next
}

/*SafeUpdateMetaFile safely update the file's metaFile*/
func (f *File) SafeUpdateMetaFile(newMF []byte) {
	var mf = make([]byte, len(newMF))
	fmt.Println("SIZE MF", len(newMF))
	copy(mf, newMF)
	fmt.Println("COPIED MF", mf)
	f.mux.Lock()
	f.metaFile = mf
	f.mux.Unlock()
}

/*SafeReadMetaFile safely reads the file's meta file*/
func (f *File) SafeReadMetaFile() []byte {
	f.mux.RLock()
	defer f.mux.RUnlock()

	return f.metaFile
}

/*SafeUpdateSize safely update the file's size*/
func (f *File) SafeUpdateSize(n int) {
	f.mux.Lock()
	f.totChunks = n
	f.mux.Unlock()
}

/*SafeReadSize safely reads the file's size*/
func (f *File) SafeReadSize() int {
	f.mux.RLock()
	defer f.mux.RUnlock()

	return f.totChunks
}

/*SafeReadNextChunk safely reads the file's next chunk*/
func (f *File) SafeReadNextChunk() int {
	f.mux.RLock()
	defer f.mux.RUnlock()

	return f.nextChunk
}

/*SafeUpdateChunk safely updates the file's chunk map for the given hash and increment next needed
hash: the meta file's hash
data: the new chunk
*/
func (f *File) SafeUpdateChunk(hash [32]byte, data []byte) {
	f.mux.Lock()
	f.chunks[hash] = make([]byte, len(data))
	copy(f.chunks[hash], data)
	f.nextChunk++
	f.mux.Unlock()
}

/*SafeMetaMap represent a map of the Gossiper's meta files with a lock*/
type SafeMetaMap struct {
	meta map[[32]byte]*File
	data map[[32]byte]*File
	mux  sync.RWMutex
}

/*SafeUpdateMeta safely updates the meta file map for the given hash
hash: the meta file's hash
mFile: the new meta file
*/
func (m *SafeMetaMap) SafeUpdateMeta(hash [32]byte, mFile *File) {
	m.mux.Lock()
	m.meta[hash] = mFile
	m.mux.Unlock()
}

/*SafeReadMeta safely reads the meta map for the given hash, returns the file's pointer
hash: the chunk's hash
*/
func (m *SafeMetaMap) SafeReadMeta(hash [32]byte) (*File, bool) {
	m.mux.RLock()
	defer m.mux.RUnlock()

	res, ok := m.meta[hash]
	return res, ok
}

/*SafeUpdateData safely updates the data map for the given hash
hash: the meta file's hash
mFile: the new meta file
*/
func (m *SafeMetaMap) SafeUpdateData(hash [32]byte, mFile *File) {
	m.mux.Lock()
	m.data[hash] = mFile
	m.mux.Unlock()
}

/*SafeReadData safely reads the data map for the given hash, returns the file's pointer
hash: the chunk's hash
*/
func (m *SafeMetaMap) SafeReadData(hash [32]byte) (*File, bool) {
	m.mux.RLock()
	defer m.mux.RUnlock()

	fmt.Println(m.data)

	res, ok := m.data[hash]
	return res, ok
}

/*SafeReadChunk safely reads the chunk map for the given hash, returns emtpy slice of byte if hash doesn't exists
hash: the chunk's hash
*/
func (m *SafeMetaMap) SafeReadChunk(hash [32]byte) []byte {
	m.mux.RLock()
	defer m.mux.RUnlock()

	fmt.Println("READING CHUNK", hash)
	res, ok := m.data[hash]
	if ok {
		res.mux.RLock()
		defer res.mux.RUnlock()
		return res.chunks[hash]
	}
	return []byte{}
}
