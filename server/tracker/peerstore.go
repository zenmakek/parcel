package tracker

import (
	"sync"
	"time"
)

const peerTTL = 30 * time.Minute

type PeerEntry struct {
	PeerID    string
	Address   string
	FileHash  string
	Filename  string
	Size      int64
	SeenAt    time.Time
}

type PeerStore struct {
	mu    sync.RWMutex
	peers map[string][]*PeerEntry // file_hash → peers
}

func NewPeerStore() *PeerStore {
	ps := &PeerStore{
		peers: make(map[string][]*PeerEntry),
	}
	go ps.cleanupLoop()
	return ps
}

func (ps *PeerStore) Announce(entry *PeerEntry) {
	ps.mu.Lock()
	defer ps.mu.Unlock()

	entry.SeenAt = time.Now()
	existing := ps.peers[entry.FileHash]

	for _, e := range existing {
		if e.PeerID == entry.PeerID {
			e.Address = entry.Address
			e.SeenAt = time.Now()
			return
		}
	}

	ps.peers[entry.FileHash] = append(existing, entry)
}

func (ps *PeerStore) Lookup(fileHash string) []*PeerEntry {
	ps.mu.RLock()
	defer ps.mu.RUnlock()

	entries := ps.peers[fileHash]
	result := make([]*PeerEntry, 0, len(entries))
	now := time.Now()

	for _, e := range entries {
		if now.Sub(e.SeenAt) < peerTTL {
			result = append(result, e)
		}
	}
	return result
}

func (ps *PeerStore) Remove(fileHash, peerID string) {
	ps.mu.Lock()
	defer ps.mu.Unlock()

	entries := ps.peers[fileHash]
	filtered := entries[:0]
	for _, e := range entries {
		if e.PeerID != peerID {
			filtered = append(filtered, e)
		}
	}
	ps.peers[fileHash] = filtered
}

func (ps *PeerStore) cleanupLoop() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		ps.mu.Lock()
		now := time.Now()
		for hash, entries := range ps.peers {
			active := entries[:0]
			for _, e := range entries {
				if now.Sub(e.SeenAt) < peerTTL {
					active = append(active, e)
				}
			}
			if len(active) == 0 {
				delete(ps.peers, hash)
			} else {
				ps.peers[hash] = active
			}
		}
		ps.mu.Unlock()
	}
}