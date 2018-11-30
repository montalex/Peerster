package gossiper

import (
	"strings"
	"sync"
)

/*Match represent a SearchRequest's match*/
type Match struct {
	fileName  string
	totChunks uint64
	metaHash  []byte
	chunks    []uint64
	fullMatch bool
	mux       sync.RWMutex
}

/*SafeMatchMap represent the Gossiper's matches map with a lock*/
type SafeMatchMap struct {
	matches map[string]*Match
	mux     sync.RWMutex
}

/*SafeReadMatch safely reads the matche's map
id: the match's id*/
func (m *SafeMatchMap) SafeReadMatch(id string) (*Match, bool) {
	m.mux.RLock()
	defer m.mux.RUnlock()

	res, ok := m.matches[id]
	return res, ok
}

/*FileRequest safely reads the matche's map and looks for a full match for the given file name
fileName: the name of the file to look for*/
func (m *SafeMatchMap) FileRequest(fileName string) (string, bool) {
	m.mux.RLock()
	defer m.mux.RUnlock()

	for name, match := range m.matches {
		if fileName == match.SafeReadName() && match.IsFullMatch() {
			dest := strings.Replace(name, fileName, "", -1)
			return dest, true
		}
	}
	return "", false
}

/*SafeUpdateMatch safely updates the matche's map*/
func (m *SafeMatchMap) SafeUpdateMatch(id string, newM *Match) {
	m.mux.Lock()
	m.matches[id] = newM
	m.mux.Unlock()
}

/*IsSearchOver checks if we already have two full matches for this search*/
func (m *SafeMatchMap) IsSearchOver(keywords []string) bool {
	m.mux.RLock()
	defer m.mux.RUnlock()

	fullM := 0
	for _, match := range m.matches {
		for _, word := range keywords {
			if strings.Contains(match.SafeReadName(), word) {
				if match.IsFullMatch() {
					fullM++
					if fullM >= 2 {
						return true
					}
				}
			}
		}
	}
	return false
}

/*SafeReadName safely reads the file's name*/
func (m *Match) SafeReadName() string {
	m.mux.RLock()
	defer m.mux.RUnlock()

	return m.fileName
}

/*SafeUpdateMetaHash safely update the match's metaFile*/
func (m *Match) SafeUpdateMetaHash(newMF []byte) {
	var mf = make([]byte, len(newMF))
	copy(mf, newMF)
	m.mux.Lock()
	m.metaHash = mf
	m.mux.Unlock()
}

/*SafeUpdateChunks safely update the match's chunks*/
func (m *Match) SafeUpdateChunks(newC []uint64) {
	m.mux.Lock()
	for _, i := range newC {
		if !UINTcontains(m.chunks, i) {
			m.chunks = append(m.chunks, i)
		}
	}
	if uint64(len(m.chunks)) == m.totChunks {
		m.fullMatch = true
	}
	m.mux.Unlock()
}

/*IsFullMatch checks if the match is a Full match*/
func (m *Match) IsFullMatch() bool {
	m.mux.RLock()
	defer m.mux.RUnlock()

	return m.fullMatch
}

/*SafeReadMetaHash safely reads the match's meta file*/
func (m *Match) SafeReadMetaHash() []byte {
	m.mux.RLock()
	defer m.mux.RUnlock()

	return m.metaHash
}
