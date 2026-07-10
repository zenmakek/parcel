package dht

import (
	"fmt"
	"testing"
)

func TestXORDistance(t *testing.T) {
	var a, b NodeID
	a[0] = 0xFF
	b[0] = 0x0F

	dist := XORDistance(a, b)
	if dist[0] != 0xF0 {
		t.Errorf("expected 0xF0 got 0x%X", dist[0])
	}
	t.Logf("XOR distance: 0x%X", dist[0])
}

func TestRoutingTableAddAndQuery(t *testing.T) {
	selfID, _ := RandomNodeID()
	self := &Node{ID: selfID, Address: "localhost:9000"}
	rt := NewRoutingTable(self)

	for i := 0; i < 10; i++ {
		id, _ := RandomNodeID()
		rt.Add(&Node{
			ID:      id,
			Address: "localhost:900" + string(rune('0'+i)),
		})
	}

	if rt.Size() != 10 {
		t.Errorf("expected 10 nodes got %d", rt.Size())
	}

	target, _ := RandomNodeID()
	closest := rt.ClosestNodes(target, 3)

	if len(closest) != 3 {
		t.Errorf("expected 3 closest nodes got %d", len(closest))
	}

	t.Logf("routing table size: %d", rt.Size())
	t.Logf("closest 3 nodes found")
}

func TestDHTAnnounceAndFind(t *testing.T) {
	selfID, _ := RandomNodeID()

	dht, err := New(fmt.Sprintf("%x", selfID), "localhost:9100")
	if err != nil {
		t.Fatalf("failed to create DHT: %v", err)
	}

	fileHash := "3a7bd3e2a7b9f1c2d3e4f5a6b7c8d9e0f1a2b3c4d5e6f7a8b9c0d1e2f3a4b5c6"
	dht.Announce(fileHash)

	peers, err := dht.FindPeers(fileHash)
	if err != nil {
		t.Fatalf("FindPeers failed: %v", err)
	}

	if len(peers) == 0 {
		t.Fatal("expected at least one peer")
	}

	if peers[0] != "localhost:9100" {
		t.Errorf("expected localhost:9100 got %s", peers[0])
	}

	t.Logf("found %d peers for hash %s", len(peers), fileHash[:12])
}

func TestSortByDistance(t *testing.T) {
	target, _ := RandomNodeID()

	nodes := make([]*Node, 5)
	for i := range nodes {
		id, _ := RandomNodeID()
		nodes[i] = &Node{ID: id}
	}

	SortByDistance(target, nodes)

	for i := 1; i < len(nodes); i++ {
		if Less(target, nodes[i].ID, nodes[i-1].ID) {
			t.Errorf("nodes not sorted correctly at index %d", i)
		}
	}

	t.Log("nodes sorted by XOR distance correctly")
}
