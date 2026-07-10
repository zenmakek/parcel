package dht

import (
	"encoding/hex"
	"fmt"
	"math/bits"
	"sync"
	"time"
)

// bucket holds up to k nodes at a specific distance range.
type bucket struct {
	nodes []*Node
	mu    sync.RWMutex
}

func (b *bucket) add(node *Node) {
	b.mu.Lock()
	defer b.mu.Unlock()

	for _, n := range b.nodes {
		if n.ID == node.ID {
			n.Address = node.Address
			n.LastSeen = time.Now()
			return
		}
	}

	if len(b.nodes) < k {
		b.nodes = append(b.nodes, node)
		return
	}

	// bucket full — drop oldest node (simplified; real Kademlia pings oldest first)
	b.nodes[0] = node
}

func (b *bucket) nodes_() []*Node {
	b.mu.RLock()
	defer b.mu.RUnlock()
	result := make([]*Node, len(b.nodes))
	copy(result, b.nodes)
	return result
}

// RoutingTable is a Kademlia routing table.
// Organizes peers into buckets by XOR distance from self.
type RoutingTable struct {
	self    *Node
	buckets [idBits]*bucket
	mu      sync.RWMutex
}

// NewRoutingTable creates a routing table for the given self node.
func NewRoutingTable(self *Node) *RoutingTable {
	rt := &RoutingTable{self: self}
	for i := range rt.buckets {
		rt.buckets[i] = &bucket{}
	}
	return rt
}

// Add inserts a node into the appropriate bucket.
func (rt *RoutingTable) Add(node *Node) {
	if node.ID == rt.self.ID {
		return
	}
	idx := rt.bucketIndex(node.ID)
	rt.buckets[idx].add(node)
}

// ClosestNodes returns the n closest nodes to the target ID.
func (rt *RoutingTable) ClosestNodes(target NodeID, n int) []*Node {
	rt.mu.RLock()
	defer rt.mu.RUnlock()

	var all []*Node
	for _, b := range rt.buckets {
		all = append(all, b.nodes_()...)
	}

	SortByDistance(target, all)

	if len(all) > n {
		return all[:n]
	}
	return all
}

// Size returns the total number of nodes in the routing table.
func (rt *RoutingTable) Size() int {
	rt.mu.RLock()
	defer rt.mu.RUnlock()

	total := 0
	for _, b := range rt.buckets {
		b.mu.RLock()
		total += len(b.nodes)
		b.mu.RUnlock()
	}
	return total
}

// bucketIndex returns the index of the bucket for a given node ID.
// Based on the position of the highest differing bit from self.
func (rt *RoutingTable) bucketIndex(id NodeID) int {
	dist := XORDistance(rt.self.ID, id)
	for i, b := range dist {
		if b != 0 {
			return i*8 + (7 - bits.LeadingZeros8(b))
		}
	}
	return 0
}

// String returns a summary of the routing table.
func (rt *RoutingTable) String() string {
	return fmt.Sprintf("RoutingTable{self=%s, size=%d}", hex.EncodeToString(rt.self.ID[:])[:12], rt.Size())
}
