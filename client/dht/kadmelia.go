package dht

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net"
	"sort"
	"sync"
	"time"

	"github.com/zenmakek/parcel/shared/protocol"
)

const (
	k              = 20  // k-bucket size — max peers per bucket
	alpha          = 3   // concurrency factor for lookups
	idBits         = 256 // SHA256 node ID length in bits
	lookupTimeout  = 10 * time.Second
)

// NodeID is a 256-bit identifier derived from a peer's SHA256 hash.
type NodeID [32]byte

// Node represents a peer in the DHT network.
type Node struct {
	ID      NodeID
	Address string
	LastSeen time.Time
}

// DHT is a Kademlia distributed hash table.
type DHT struct {
	self     *Node
	table    *RoutingTable
	mu       sync.RWMutex
	announces map[string][]string // file_hash → [peer addresses]
}

// New creates a DHT node with the given peer ID and address.
func New(peerID string, address string) (*DHT, error) {
	id, err := nodeIDFromHex(peerID)
	if err != nil {
		return nil, fmt.Errorf("invalid peer ID: %w", err)
	}

	self := &Node{ID: id, Address: address, LastSeen: time.Now()}

	return &DHT{
		self:      self,
		table:     NewRoutingTable(self),
		announces: make(map[string][]string),
	}, nil
}

// Announce stores that this node has the file with the given hash.
func (d *DHT) Announce(fileHash string) {
	d.mu.Lock()
	defer d.mu.Unlock()

	for _, addr := range d.announces[fileHash] {
		if addr == d.self.Address {
			return
		}
	}
	d.announces[fileHash] = append(d.announces[fileHash], d.self.Address)
	fmt.Printf("[dht] announced: hash=%s\n", fileHash[:12])
}

// FindPeers returns known peers that have announced the given file hash.
// Searches local store first, then queries k-closest nodes.
func (d *DHT) FindPeers(fileHash string) ([]string, error) {
	d.mu.RLock()
	local := d.announces[fileHash]
	d.mu.RUnlock()

	if len(local) > 0 {
		fmt.Printf("[dht] found %d local peers for %s\n", len(local), fileHash[:12])
		return local, nil
	}

	// query closest nodes in routing table
	target, err := nodeIDFromHex(fileHash)
	if err != nil {
		return nil, fmt.Errorf("invalid file hash: %w", err)
	}

	closest := d.table.ClosestNodes(target, alpha)
	if len(closest) == 0 {
		return nil, fmt.Errorf("no peers in routing table")
	}

	return d.iterativeLookup(fileHash, closest)
}

// AddNode adds a node to the routing table.
func (d *DHT) AddNode(id string, address string) error {
	nodeID, err := nodeIDFromHex(id)
	if err != nil {
		return fmt.Errorf("invalid node ID: %w", err)
	}
	node := &Node{ID: nodeID, Address: address, LastSeen: time.Now()}
	d.table.Add(node)
	return nil
}

// iterativeLookup queries nodes to find peers announcing a file hash.
func (d *DHT) iterativeLookup(fileHash string, initial []*Node) ([]string, error) {
	visited := make(map[NodeID]bool)
	var results []string
	var mu sync.Mutex

	queue := make([]*Node, len(initial))
	copy(queue, initial)

	deadline := time.Now().Add(lookupTimeout)

	for len(queue) > 0 && time.Now().Before(deadline) {
		batch := queue
		if len(batch) > alpha {
			batch = batch[:alpha]
		}
		queue = queue[len(batch):]

		var wg sync.WaitGroup
		for _, node := range batch {
			if visited[node.ID] {
				continue
			}
			visited[node.ID] = true

			wg.Add(1)
			go func(n *Node) {
				defer wg.Done()
				peers, err := queryNode(n.Address, fileHash)
				if err != nil {
					return
				}
				mu.Lock()
				results = append(results, peers...)
				mu.Unlock()
			}(node)
		}
		wg.Wait()
	}

	return deduplicate(results), nil
}

// queryNode contacts a single DHT node and asks for peers with fileHash.
func queryNode(address string, fileHash string) ([]string, error) {
	conn, err := net.DialTimeout("tcp", address, 5*time.Second)
	if err != nil {
		return nil, err
	}
	defer conn.Close()
	conn.SetDeadline(time.Now().Add(5 * time.Second))

	req, err := protocol.Encode(protocol.PacketLookup, protocol.LookupPayload{
		FileHash: fileHash,
	})
	if err != nil {
		return nil, err
	}

	fmt.Fprint(conn, req)

	buf := make([]byte, 4096)
	n, err := conn.Read(buf)
	if err != nil {
		return nil, err
	}

	packet, err := protocol.Decode(string(buf[:n]))
	if err != nil {
		return nil, err
	}

	if packet.Type != protocol.PacketPeerList {
		return nil, fmt.Errorf("expected PEER_LIST got %s", packet.Type)
	}

	var payload protocol.PeerListPayload
	if err := protocol.DecodePayload(packet.Payload, &payload); err != nil {
		return nil, err
	}

	addrs := make([]string, len(payload.Peers))
	for i, p := range payload.Peers {
		addrs[i] = p.Address
	}
	return addrs, nil
}

// XORDistance computes the XOR distance between two node IDs.
// Kademlia uses XOR as the distance metric.
func XORDistance(a, b NodeID) NodeID {
	var dist NodeID
	for i := range dist {
		dist[i] = a[i] ^ b[i]
	}
	return dist
}

// Less returns true if a is closer to target than b.
func Less(target, a, b NodeID) bool {
	distA := XORDistance(target, a)
	distB := XORDistance(target, b)
	for i := range distA {
		if distA[i] < distB[i] {
			return true
		}
		if distA[i] > distB[i] {
			return false
		}
	}
	return false
}

func nodeIDFromHex(s string) (NodeID, error) {
	if len(s) > 64 {
		s = s[:64]
	}
	b, err := hex.DecodeString(s)
	if err != nil {
		return NodeID{}, fmt.Errorf("invalid hex: %w", err)
	}
	var id NodeID
	copy(id[:], b)
	return id, nil
}

func deduplicate(addrs []string) []string {
	seen := make(map[string]bool)
	result := make([]string, 0, len(addrs))
	for _, a := range addrs {
		if !seen[a] {
			seen[a] = true
			result = append(result, a)
		}
	}
	return result
}

// RandomNodeID generates a random 256-bit node ID.
// Used for testing.
func RandomNodeID() (NodeID, error) {
	var id NodeID
	if _, err := rand.Read(id[:]); err != nil {
		return id, err
	}
	return id, nil
}

// SortByDistance sorts nodes by XOR distance to target.
func SortByDistance(target NodeID, nodes []*Node) {
	sort.Slice(nodes, func(i, j int) bool {
		return Less(target, nodes[i].ID, nodes[j].ID)
	})
}