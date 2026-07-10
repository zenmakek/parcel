package p2p

import (
	"os"
	"testing"
	"time"

	"github.com/zenmakek/parcel/client/store"
	"github.com/zenmakek/parcel/shared/chunk"
	"github.com/zenmakek/parcel/shared/hash"
	"github.com/zenmakek/parcel/shared/protocol"
)

func TestSeederServesChunk(t *testing.T) {
	seederID := makeIdentity(t)
	downloaderID := makeIdentity(t)

	dir, _ := os.MkdirTemp("", "parcel-seeder-*")
	t.Cleanup(func() { os.RemoveAll(dir) })

	s, err := store.NewAt(dir)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}

	data := []byte("seeder test chunk data")
	c := &chunk.Chunk{
		Index: 0,
		Hash:  hash.HashBytes(data),
		Data:  data,
		Size:  len(data),
	}

	if err := s.WriteChunk(c); err != nil {
		t.Fatalf("failed to write chunk: %v", err)
	}

	manifest := &chunk.Manifest{
		FileHash:    hash.HashBytes([]byte("seeder test file")),
		Filename:    "test.bin",
		TotalSize:   int64(len(data)),
		ChunkSize:   chunk.DefaultChunkSize,
		ChunkCount:  1,
		ChunkHashes: []string{c.Hash},
	}

	if err := s.WriteManifest(manifest); err != nil {
		t.Fatalf("failed to write manifest: %v", err)
	}

	seeder, err := NewSeeder("19020", s, seederID)
	if err != nil {
		t.Fatalf("failed to create seeder: %v", err)
	}
	seeder.Start()
	defer seeder.Stop()

	time.Sleep(30 * time.Millisecond)

	pc, err := Dial("localhost:19020", downloaderID)
	if err != nil {
		t.Fatalf("failed to dial seeder: %v", err)
	}
	defer pc.Close()

	if err := pc.Send(protocol.PacketChunkRequest, protocol.ChunkRequestPayload{
		FileHash:   manifest.FileHash,
		ChunkIndex: 0,
	}); err != nil {
		t.Fatalf("failed to send chunk request: %v", err)
	}

	pkt, err := pc.Receive()
	if err != nil {
		t.Fatalf("failed to receive response: %v", err)
	}

	if pkt.Type != protocol.PacketChunkResponse {
		t.Fatalf("expected CHUNK_RESPONSE got %s", pkt.Type)
	}

	var resp protocol.ChunkResponsePayload
	if err := protocol.DecodePayload(pkt.Payload, &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if string(resp.Data) != string(data) {
		t.Errorf("data mismatch")
	}

	t.Logf("seeder served chunk %d (%d bytes) to downloader", resp.ChunkIndex, len(resp.Data))
}

func TestSeederServesManifest(t *testing.T) {
	seederID := makeIdentity(t)
	downloaderID := makeIdentity(t)

	dir, _ := os.MkdirTemp("", "parcel-seeder-manifest-*")
	t.Cleanup(func() { os.RemoveAll(dir) })

	s, _ := store.NewAt(dir)

	manifest := &chunk.Manifest{
		FileHash:    hash.HashBytes([]byte("manifest test")),
		Filename:    "movie.mp4",
		TotalSize:   1024 * 1024,
		ChunkSize:   chunk.DefaultChunkSize,
		ChunkCount:  4,
		ChunkHashes: []string{"aaa", "bbb", "ccc", "ddd"},
	}
	s.WriteManifest(manifest)

	seeder, err := NewSeeder("19021", s, seederID)
	if err != nil {
		t.Fatalf("failed to create seeder: %v", err)
	}
	seeder.Start()
	defer seeder.Stop()

	time.Sleep(30 * time.Millisecond)

	pc, err := Dial("localhost:19021", downloaderID)
	if err != nil {
		t.Fatalf("failed to dial seeder: %v", err)
	}
	defer pc.Close()

	pc.Send(protocol.PacketManifestRequest, protocol.ManifestRequestPayload{
		FileHash: manifest.FileHash,
	})

	pkt, err := pc.Receive()
	if err != nil {
		t.Fatalf("failed to receive manifest: %v", err)
	}

	if pkt.Type != protocol.PacketManifestResponse {
		t.Fatalf("expected MANIFEST_RESPONSE got %s", pkt.Type)
	}

	var resp protocol.ManifestResponsePayload
	protocol.DecodePayload(pkt.Payload, &resp)

	if resp.Filename != "movie.mp4" {
		t.Errorf("filename mismatch: %s", resp.Filename)
	}
	if resp.ChunkCount != 4 {
		t.Errorf("chunk count mismatch: %d", resp.ChunkCount)
	}

	t.Logf("seeder served manifest: %s (%d chunks)", resp.Filename, resp.ChunkCount)
}

func TestSeederActivepeers(t *testing.T) {
	seederID := makeIdentity(t)
	downloaderID := makeIdentity(t)

	dir, _ := os.MkdirTemp("", "parcel-seeder-peers-*")
	t.Cleanup(func() { os.RemoveAll(dir) })

	s, _ := store.NewAt(dir)

	seeder, err := NewSeeder("19022", s, seederID)
	if err != nil {
		t.Fatalf("failed to create seeder: %v", err)
	}
	seeder.Start()
	defer seeder.Stop()

	time.Sleep(30 * time.Millisecond)

	if seeder.ActivePeers() != 0 {
		t.Errorf("expected 0 active peers got %d", seeder.ActivePeers())
	}

	pc, err := Dial("localhost:19022", downloaderID)
	if err != nil {
		t.Fatalf("failed to dial: %v", err)
	}

	time.Sleep(30 * time.Millisecond)

	if seeder.ActivePeers() != 1 {
		t.Errorf("expected 1 active peer got %d", seeder.ActivePeers())
	}

	pc.Close()
	time.Sleep(50 * time.Millisecond)

	if seeder.ActivePeers() != 0 {
		t.Errorf("expected 0 peers after disconnect got %d", seeder.ActivePeers())
	}

	t.Log("active peer tracking works correctly")
}