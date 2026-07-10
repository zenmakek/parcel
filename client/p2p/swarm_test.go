package p2p

import (
	"fmt"
	"net"
	"os"
	"testing"
	"time"

	"github.com/zenmakek/parcel/client/identity"
	"github.com/zenmakek/parcel/client/store"
	"github.com/zenmakek/parcel/shared/chunk"
	"github.com/zenmakek/parcel/shared/hash"
	"github.com/zenmakek/parcel/shared/protocol"
)

// mockSeeder listens on a port and serves chunks from a manifest.
func mockSeeder(t *testing.T, port string, chunks []*chunk.Chunk, manifest *chunk.Manifest, id *identity.Identity) {
	t.Helper()
	ln, err := net.Listen("tcp", ":"+port)
	if err != nil {
		t.Fatalf("mock seeder failed to listen: %v", err)
	}
	t.Cleanup(func() { ln.Close() })

	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			go func(c net.Conn) {
				pc, err := handshake(c, id)
				if err != nil {
					c.Close()
					return
				}
				for {
					pkt, err := pc.Receive()
					if err != nil {
						return
					}
					if pkt.Type != protocol.PacketChunkRequest {
						continue
					}
					var req protocol.ChunkRequestPayload
					protocol.DecodePayload(pkt.Payload, &req)

					if req.ChunkIndex >= len(chunks) {
						continue
					}
					c := chunks[req.ChunkIndex]
					pc.Send(protocol.PacketChunkResponse, protocol.ChunkResponsePayload{
						FileHash:   manifest.FileHash,
						ChunkIndex: c.Index,
						Hash:       c.Hash,
						Data:       c.Data,
					})
				}
			}(conn)
		}
	}()
	time.Sleep(30 * time.Millisecond)
}

func TestSwarmDownload(t *testing.T) {
	seederID := makeIdentity(t)
	downloaderID := makeIdentity(t)

	// build test chunks
	chunkData := [][]byte{
		[]byte("chunk zero data for swarm test"),
		[]byte("chunk one data for swarm test"),
		[]byte("chunk two data for swarm test"),
	}

	chunks := make([]*chunk.Chunk, len(chunkData))
	hashes := make([]string, len(chunkData))
	for i, d := range chunkData {
		h := hash.HashBytes(d)
		chunks[i] = &chunk.Chunk{Index: i, Hash: h, Data: d, Size: len(d)}
		hashes[i] = h
	}

	fileHash := hash.HashBytes([]byte("combined"))
	manifest := &chunk.Manifest{
		FileHash:    fileHash,
		Filename:    "test.bin",
		TotalSize:   int64(len(chunkData[0]) * 3),
		ChunkSize:   chunk.DefaultChunkSize,
		ChunkCount:  3,
		ChunkHashes: hashes,
	}

	mockSeeder(t, "19010", chunks, manifest, seederID)

	dir, _ := os.MkdirTemp("", "parcel-swarm-*")
	t.Cleanup(func() { os.RemoveAll(dir) })

	s, err := store.NewAt(dir)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}

	swarm := NewSwarm(manifest, s, downloaderID)

	pc, err := Dial("localhost:19010", downloaderID)
	if err != nil {
		t.Fatalf("failed to dial seeder: %v", err)
	}
	swarm.AddPeer(pc)

	if err := swarm.Download(); err != nil {
		t.Fatalf("Download failed: %v", err)
	}

	have := s.HaveChunks(manifest)
	if len(have) != 3 {
		t.Errorf("expected 3 chunks have %d", len(have))
	}

	for i, c := range chunks {
		stored, err := s.ReadChunk(c.Hash, i)
		if err != nil {
			t.Errorf("chunk %d not found in store: %v", i, err)
			continue
		}
		if string(stored.Data) != string(c.Data) {
			t.Errorf("chunk %d data mismatch", i)
		}
	}

	t.Logf("swarm downloaded %d chunks successfully", len(have))
}

func TestSwarmProgress(t *testing.T) {
	id := makeIdentity(t)
	dir, _ := os.MkdirTemp("", "parcel-progress-*")
	t.Cleanup(func() { os.RemoveAll(dir) })

	s, _ := store.NewAt(dir)

	manifest := &chunk.Manifest{
		FileHash:    hash.HashBytes([]byte("progress test")),
		ChunkCount:  5,
		ChunkHashes: make([]string, 5),
	}
	for i := range manifest.ChunkHashes {
		manifest.ChunkHashes[i] = hash.HashBytes([]byte(fmt.Sprintf("chunk%d", i)))
	}

	swarm := NewSwarm(manifest, s, id)
	got, total := swarm.Progress()

	if total != 5 {
		t.Errorf("expected total 5 got %d", total)
	}
	if got != 0 {
		t.Errorf("expected 0 chunks initially got %d", got)
	}

	t.Logf("progress: %d/%d", got, total)
}
