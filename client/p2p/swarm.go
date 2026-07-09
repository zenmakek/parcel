package p2p

import (
	"fmt"
	"sync"
	"time"

	"github.com/zenmakek/parcel/client/identity"
	"github.com/zenmakek/parcel/client/store"
	"github.com/zenmakek/parcel/shared/chunk"
	"github.com/zenmakek/parcel/shared/hash"
	"github.com/zenmakek/parcel/shared/protocol"
)

const (
	maxConcurrentChunks = 4
	chunkRequestTimeout = 30 * time.Second
	maxRetries          = 3
)

// Swarm manages connections to multiple peers and
// downloads chunks from them in parallel.
type Swarm struct {
	manifest *chunk.Manifest
	store    *store.Store
	id       *identity.Identity
	peers    []*PeerConn
	mu       sync.Mutex
}

// NewSwarm creates a Swarm for downloading a file.
func NewSwarm(m *chunk.Manifest, s *store.Store, id *identity.Identity) *Swarm {
	return &Swarm{
		manifest: m,
		store:    s,
		id:       id,
	}
}

// AddPeer adds an authenticated peer connection to the swarm.
func (sw *Swarm) AddPeer(pc *PeerConn) {
	sw.mu.Lock()
	defer sw.mu.Unlock()
	sw.peers = append(sw.peers, pc)
	fmt.Printf("[swarm] added peer %s (%d total)\n", pc.RemotePeerID[:12], len(sw.peers))
}

// Download fetches all missing chunks from available peers in parallel.
// Returns when all chunks are downloaded and verified.
func (sw *Swarm) Download() error {
	have := sw.store.HaveChunks(sw.manifest)
	missing := sw.manifest.MissingChunks(have)

	if len(missing) == 0 {
		fmt.Println("[swarm] all chunks already present")
		return nil
	}

	fmt.Printf("[swarm] downloading %d chunks from %d peers\n", len(missing), len(sw.peers))

	// semaphore limits concurrent chunk requests
	sem := make(chan struct{}, maxConcurrentChunks)
	errCh := make(chan error, len(missing))
	var wg sync.WaitGroup

	for _, idx := range missing {
		wg.Add(1)
		go func(chunkIndex int) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			if err := sw.fetchChunkWithRetry(chunkIndex); err != nil {
				errCh <- fmt.Errorf("chunk %d failed: %w", chunkIndex, err)
			}
		}(idx)
	}

	wg.Wait()
	close(errCh)

	for err := range errCh {
		if err != nil {
			return err
		}
	}

	fmt.Printf("[swarm] all %d chunks downloaded\n", len(missing))
	return nil
}

// fetchChunkWithRetry attempts to fetch a chunk, retrying on
// different peers if one fails.
func (sw *Swarm) fetchChunkWithRetry(chunkIndex int) error {
	for attempt := 0; attempt < maxRetries; attempt++ {
		peer := sw.selectPeer(chunkIndex, attempt)
		if peer == nil {
			return fmt.Errorf("no peers available for chunk %d", chunkIndex)
		}

		c, err := sw.fetchChunk(peer, chunkIndex)
		if err != nil {
			fmt.Printf("[swarm] chunk %d failed from peer %s (attempt %d): %v\n",
				chunkIndex, peer.RemotePeerID[:12], attempt+1, err)
			continue
		}

		if err := sw.store.WriteChunk(c); err != nil {
			return fmt.Errorf("failed to store chunk %d: %w", chunkIndex, err)
		}

		return nil
	}
	return fmt.Errorf("chunk %d failed after %d attempts", chunkIndex, maxRetries)
}

// fetchChunk requests and verifies a single chunk from a peer.
func (sw *Swarm) fetchChunk(peer *PeerConn, chunkIndex int) (*chunk.Chunk, error) {
	peer.mu.Lock()
	defer peer.mu.Unlock()

	err := peer.Send(protocol.PacketChunkRequest, protocol.ChunkRequestPayload{
		FileHash:   sw.manifest.FileHash,
		ChunkIndex: chunkIndex,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to send chunk request: %w", err)
	}

	peer.conn.(interface{ SetDeadline(time.Time) error }).SetDeadline(
		time.Now().Add(chunkRequestTimeout),
	)

	packet, err := peer.Receive()
	if err != nil {
		return nil, fmt.Errorf("failed to receive chunk response: %w", err)
	}

	if packet.Type != protocol.PacketChunkResponse {
		return nil, fmt.Errorf("expected CHUNK_RESPONSE got %s", packet.Type)
	}

	var resp protocol.ChunkResponsePayload
	if err := protocol.DecodePayload(packet.Payload, &resp); err != nil {
		return nil, fmt.Errorf("failed to decode chunk response: %w", err)
	}

	if resp.ChunkIndex != chunkIndex {
		return nil, fmt.Errorf("wrong chunk index: expected %d got %d", chunkIndex, resp.ChunkIndex)
	}

	actual := hash.HashBytes(resp.Data)
	expected := sw.manifest.ChunkHashes[chunkIndex]
	if actual != expected {
		peer.Send(protocol.PacketChunkVerifyFail, protocol.ChunkVerifyFailPayload{
			FileHash:   sw.manifest.FileHash,
			ChunkIndex: chunkIndex,
			Expected:   expected,
			Got:        actual,
		})
		return nil, fmt.Errorf("chunk %d hash mismatch", chunkIndex)
	}

	return &chunk.Chunk{
		Index: chunkIndex,
		Hash:  actual,
		Data:  resp.Data,
		Size:  len(resp.Data),
	}, nil
}

// selectPeer picks a peer for a given chunk index and attempt number.
// Uses round-robin with attempt offset so retries use different peers.
func (sw *Swarm) selectPeer(chunkIndex, attempt int) *PeerConn {
	sw.mu.Lock()
	defer sw.mu.Unlock()

	if len(sw.peers) == 0 {
		return nil
	}

	idx := (chunkIndex + attempt) % len(sw.peers)
	return sw.peers[idx]
}

// ConnectToPeers dials all peers from a peer list and adds
// successful connections to the swarm.
func (sw *Swarm) ConnectToPeers(peers []protocol.PeerInfo) {
	var wg sync.WaitGroup
	for _, p := range peers {
		wg.Add(1)
		go func(info protocol.PeerInfo) {
			defer wg.Done()
			pc, err := Connect(info.Address, sw.id)
			if err != nil {
				fmt.Printf("[swarm] failed to connect to peer %s: %v\n", info.PeerID[:12], err)
				return
			}
			sw.AddPeer(pc)
		}(p)
	}
	wg.Wait()
}

// Close closes all peer connections in the swarm.
func (sw *Swarm) Close() {
	sw.mu.Lock()
	defer sw.mu.Unlock()
	for _, pc := range sw.peers {
		pc.Close()
	}
	sw.peers = nil
}

// Progress returns downloaded vs total chunk counts.
func (sw *Swarm) Progress() (int, int) {
	have := sw.store.HaveChunks(sw.manifest)
	return len(have), sw.manifest.ChunkCount
}