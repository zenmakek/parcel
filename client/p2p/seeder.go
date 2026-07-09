package p2p

import (
	"fmt"
	"sync"

	"github.com/zenmakek/parcel/client/identity"
	"github.com/zenmakek/parcel/client/store"
	"github.com/zenmakek/parcel/shared/protocol"
)

// Seeder listens for incoming peer connections and serves
// chunks from the local store on request.
type Seeder struct {
	listener *Listener
	store    *store.Store
	id       *identity.Identity
	active   map[string]*PeerConn // peerID → connection
	mu       sync.RWMutex
	done     chan struct{}
}

// NewSeeder creates a Seeder on the given port.
func NewSeeder(port string, s *store.Store, id *identity.Identity) (*Seeder, error) {
	ln, err := NewListener(port, id)
	if err != nil {
		return nil, fmt.Errorf("failed to create listener: %w", err)
	}

	return &Seeder{
		listener: ln,
		store:    s,
		id:       id,
		active:   make(map[string]*PeerConn),
		done:     make(chan struct{}),
	}, nil
}

// Start begins accepting peer connections and serving chunks.
// Runs until Stop is called.
func (sd *Seeder) Start() {
	fmt.Printf("[seeder] serving on :%s\n", sd.listener.Port)
	go sd.acceptLoop()
}

func (sd *Seeder) acceptLoop() {
	for {
		select {
		case <-sd.done:
			return
		default:
		}

		pc, err := sd.listener.Accept()
		if err != nil {
			select {
			case <-sd.done:
				return
			default:
				fmt.Printf("[seeder] accept error: %v\n", err)
				continue
			}
		}

		sd.mu.Lock()
		sd.active[pc.RemotePeerID] = pc
		sd.mu.Unlock()

		fmt.Printf("[seeder] peer connected: %s\n", pc.RemotePeerID[:12])
		go sd.servePeer(pc)
	}
}

func (sd *Seeder) servePeer(pc *PeerConn) {
	defer func() {
		sd.mu.Lock()
		delete(sd.active, pc.RemotePeerID)
		sd.mu.Unlock()
		pc.Close()
		fmt.Printf("[seeder] peer disconnected: %s\n", pc.RemotePeerID[:12])
	}()

	for {
		packet, err := pc.Receive()
		if err != nil {
			return
		}

		switch packet.Type {
		case protocol.PacketChunkRequest:
			sd.handleChunkRequest(pc, packet)
		case protocol.PacketManifestRequest:
			sd.handleManifestRequest(pc, packet)
		case protocol.PacketBitfield:
			sd.handleBitfield(pc, packet)
		default:
			fmt.Printf("[seeder] unknown packet from %s: %s\n", pc.RemotePeerID[:12], packet.Type)
		}
	}
}

func (sd *Seeder) handleChunkRequest(pc *PeerConn, packet *protocol.Packet) {
	var req protocol.ChunkRequestPayload
	if err := protocol.DecodePayload(packet.Payload, &req); err != nil {
		fmt.Printf("[seeder] bad chunk request: %v\n", err)
		return
	}

	manifest, ok := sd.store.GetManifest(req.FileHash)
	if !ok {
		fmt.Printf("[seeder] manifest not found for hash %s\n", req.FileHash[:12])
		return
	}

	if req.ChunkIndex >= len(manifest.ChunkHashes) {
		fmt.Printf("[seeder] chunk index %d out of range\n", req.ChunkIndex)
		return
	}

	chunkHash := manifest.ChunkHashes[req.ChunkIndex]
	c, err := sd.store.ReadChunk(chunkHash, req.ChunkIndex)
	if err != nil {
		fmt.Printf("[seeder] failed to read chunk %d: %v\n", req.ChunkIndex, err)
		return
	}

	err = pc.Send(protocol.PacketChunkResponse, protocol.ChunkResponsePayload{
		FileHash:   req.FileHash,
		ChunkIndex: req.ChunkIndex,
		Hash:       c.Hash,
		Data:       c.Data,
	})
	if err != nil {
		fmt.Printf("[seeder] failed to send chunk %d: %v\n", req.ChunkIndex, err)
		return
	}

	fmt.Printf("[seeder] served chunk %d to %s\n", req.ChunkIndex, pc.RemotePeerID[:12])
}

func (sd *Seeder) handleManifestRequest(pc *PeerConn, packet *protocol.Packet) {
	var req protocol.ManifestRequestPayload
	if err := protocol.DecodePayload(packet.Payload, &req); err != nil {
		fmt.Printf("[seeder] bad manifest request: %v\n", err)
		return
	}

	manifest, ok := sd.store.GetManifest(req.FileHash)
	if !ok {
		fmt.Printf("[seeder] manifest not found: %s\n", req.FileHash[:12])
		return
	}

	err := pc.Send(protocol.PacketManifestResponse, protocol.ManifestResponsePayload{
		FileHash:    manifest.FileHash,
		Filename:    manifest.Filename,
		TotalSize:   manifest.TotalSize,
		ChunkSize:   manifest.ChunkSize,
		ChunkCount:  manifest.ChunkCount,
		ChunkHashes: manifest.ChunkHashes,
		IsArchive:   manifest.IsArchive,
	})
	if err != nil {
		fmt.Printf("[seeder] failed to send manifest: %v\n", err)
		return
	}

	fmt.Printf("[seeder] served manifest %s to %s\n", req.FileHash[:12], pc.RemotePeerID[:12])
}

func (sd *Seeder) handleBitfield(pc *PeerConn, packet *protocol.Packet) {
	var bf protocol.BitfieldPayload
	if err := protocol.DecodePayload(packet.Payload, &bf); err != nil {
		fmt.Printf("[seeder] bad bitfield: %v\n", err)
		return
	}

	manifest, ok := sd.store.GetManifest(bf.FileHash)
	if !ok {
		return
	}

	// send our bitfield back
	have := sd.store.HaveChunks(manifest)
	bitfield := make([]bool, manifest.ChunkCount)
	for i := range bitfield {
		bitfield[i] = have[i]
	}

	pc.Send(protocol.PacketBitfield, protocol.BitfieldPayload{
		FileHash: bf.FileHash,
		Bitfield: bitfield,
	})
}

// Stop shuts down the seeder and closes all peer connections.
func (sd *Seeder) Stop() {
	close(sd.done)
	sd.listener.Close()

	sd.mu.Lock()
	defer sd.mu.Unlock()
	for _, pc := range sd.active {
		pc.Close()
	}
	sd.active = make(map[string]*PeerConn)
	fmt.Println("[seeder] stopped")
}

// ActivePeers returns the number of currently connected peers.
func (sd *Seeder) ActivePeers() int {
	sd.mu.RLock()
	defer sd.mu.RUnlock()
	return len(sd.active)
}