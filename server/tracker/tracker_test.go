package tracker

import (
	"testing"
	"time"
)

func TestAnnounceAndLookup(t *testing.T) {
	ps := NewPeerStore()

	ps.Announce(&PeerEntry{
		PeerID:   "peer-aaa",
		Address:  "1.2.3.4:9000",
		FileHash: "abc123abc123abc123abc123abc123abc123abc123abc123abc123abc123abcd",
		Filename: "video.mp4",
		Size:     1024,
	})

	entries := ps.Lookup("abc123abc123abc123abc123abc123abc123abc123abc123abc123abc123abcd")
	if len(entries) != 1 {
		t.Fatalf("expected 1 peer got %d", len(entries))
	}
	if entries[0].PeerID != "peer-aaa" {
		t.Errorf("wrong peer ID: %s", entries[0].PeerID)
	}

	t.Logf("lookup returned peer: %s at %s", entries[0].PeerID, entries[0].Address)
}

func TestMultiplePeersForSameHash(t *testing.T) {
	ps := NewPeerStore()
	hash := "abc123abc123abc123abc123abc123abc123abc123abc123abc123abc123abcd"

	ps.Announce(&PeerEntry{PeerID: "peer-aaa", Address: "1.2.3.4:9000", FileHash: hash})
	ps.Announce(&PeerEntry{PeerID: "peer-bbb", Address: "5.6.7.8:9000", FileHash: hash})
	ps.Announce(&PeerEntry{PeerID: "peer-ccc", Address: "9.10.11.12:9000", FileHash: hash})

	entries := ps.Lookup(hash)
	if len(entries) != 3 {
		t.Fatalf("expected 3 peers got %d", len(entries))
	}

	t.Logf("found %d peers for hash", len(entries))
}

func TestDuplicateAnnounceUpdates(t *testing.T) {
	ps := NewPeerStore()
	hash := "abc123abc123abc123abc123abc123abc123abc123abc123abc123abc123abcd"

	ps.Announce(&PeerEntry{PeerID: "peer-aaa", Address: "1.2.3.4:9000", FileHash: hash})
	ps.Announce(&PeerEntry{PeerID: "peer-aaa", Address: "9.9.9.9:9000", FileHash: hash})

	entries := ps.Lookup(hash)
	if len(entries) != 1 {
		t.Fatalf("expected 1 peer after duplicate announce, got %d", len(entries))
	}
	if entries[0].Address != "9.9.9.9:9000" {
		t.Errorf("expected updated address, got %s", entries[0].Address)
	}

	t.Log("duplicate announce correctly updates address")
}

func TestRemovePeer(t *testing.T) {
	ps := NewPeerStore()
	hash := "abc123abc123abc123abc123abc123abc123abc123abc123abc123abc123abcd"

	ps.Announce(&PeerEntry{PeerID: "peer-aaa", Address: "1.2.3.4:9000", FileHash: hash})
	ps.Remove(hash, "peer-aaa")

	entries := ps.Lookup(hash)
	if len(entries) != 0 {
		t.Fatalf("expected 0 peers after remove, got %d", len(entries))
	}

	t.Log("peer removed successfully")
}

func TestPeerTTLExpiry(t *testing.T) {
	ps := NewPeerStore()
	hash := "abc123abc123abc123abc123abc123abc123abc123abc123abc123abc123abcd"

	ps.Announce(&PeerEntry{PeerID: "peer-aaa", Address: "1.2.3.4:9000", FileHash: hash})

	ps.mu.Lock()
	ps.peers[hash][0].SeenAt = time.Now().Add(-31 * time.Minute)
	ps.mu.Unlock()

	entries := ps.Lookup(hash)
	if len(entries) != 0 {
		t.Fatalf("expected 0 peers after TTL expiry, got %d", len(entries))
	}

	t.Log("expired peer correctly excluded from lookup")
}