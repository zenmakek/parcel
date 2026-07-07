package store

import (
	"os"
	"testing"

	"github.com/zenmakek/parcel/shared/chunk"
	"github.com/zenmakek/parcel/shared/hash"
)

func newTestStore(t *testing.T) (*Store, func()) {
	t.Helper()
	dir, err := os.MkdirTemp("", "parcel-store-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	s, err := NewAt(dir)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	return s, func() { os.RemoveAll(dir) }
}

func makeChunk(t *testing.T, data []byte, index int) *chunk.Chunk {
	t.Helper()
	return &chunk.Chunk{
		Index: index,
		Hash:  hash.HashBytes(data),
		Data:  data,
		Size:  len(data),
	}
}

func TestWriteAndReadChunk(t *testing.T) {
	s, cleanup := newTestStore(t)
	defer cleanup()

	data := []byte("hello parcel chunk")
	c := makeChunk(t, data, 0)

	if err := s.WriteChunk(c); err != nil {
		t.Fatalf("WriteChunk failed: %v", err)
	}

	if !s.HasChunk(c.Hash) {
		t.Fatal("expected chunk to exist after write")
	}

	read, err := s.ReadChunk(c.Hash, 0)
	if err != nil {
		t.Fatalf("ReadChunk failed: %v", err)
	}

	if string(read.Data) != string(data) {
		t.Errorf("data mismatch: expected %q got %q", data, read.Data)
	}

	t.Logf("chunk written and read: %s", c.Hash[:12])
}

func TestDeduplication(t *testing.T) {
	s, cleanup := newTestStore(t)
	defer cleanup()

	data := []byte("identical chunk data")
	c := makeChunk(t, data, 0)

	if err := s.WriteChunk(c); err != nil {
		t.Fatalf("first WriteChunk failed: %v", err)
	}

	// write same chunk again — should not error or duplicate
	if err := s.WriteChunk(c); err != nil {
		t.Fatalf("second WriteChunk failed: %v", err)
	}

	size, err := s.StoreSize()
	if err != nil {
		t.Fatalf("StoreSize failed: %v", err)
	}

	expectedSize := int64(len(data))
	if size != expectedSize {
		t.Errorf("expected store size %d got %d — chunk was duplicated", expectedSize, size)
	}

	t.Logf("deduplication confirmed: store size = %d bytes", size)
}

func TestHaveChunksAndResume(t *testing.T) {
	s, cleanup := newTestStore(t)
	defer cleanup()

	chunks := []*chunk.Chunk{
		makeChunk(t, []byte("chunk zero"), 0),
		makeChunk(t, []byte("chunk one"), 1),
		makeChunk(t, []byte("chunk two"), 2),
	}

	hashes := make([]string, len(chunks))
	for i, c := range chunks {
		hashes[i] = c.Hash
	}

	manifest := &chunk.Manifest{
		FileHash:    hash.HashBytes([]byte("fake file")),
		Filename:    "test.bin",
		TotalSize:   30,
		ChunkSize:   chunk.DefaultChunkSize,
		ChunkCount:  3,
		ChunkHashes: hashes,
	}

	// write only chunks 0 and 2 — simulating interrupted transfer
	s.WriteChunk(chunks[0])
	s.WriteChunk(chunks[2])

	have := s.HaveChunks(manifest)
	if !have[0] || have[1] || !have[2] {
		t.Errorf("have map wrong: %v", have)
	}

	missing := manifest.MissingChunks(have)
	if len(missing) != 1 || missing[0] != 1 {
		t.Errorf("expected missing [1] got %v", missing)
	}

	t.Logf("resume: have chunks %v, missing %v", have, missing)
}

func TestManifestIndexPersistence(t *testing.T) {
	dir, err := os.MkdirTemp("", "parcel-index-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(dir)

	s1, err := NewAt(dir)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}

	manifest := &chunk.Manifest{
		FileHash:    "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		Filename:    "persist.bin",
		TotalSize:   1024,
		ChunkSize:   chunk.DefaultChunkSize,
		ChunkCount:  1,
		ChunkHashes: []string{"bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"},
	}

	if err := s1.WriteManifest(manifest); err != nil {
		t.Fatalf("WriteManifest failed: %v", err)
	}

	// open a new store instance pointing to the same dir
	s2, err := NewAt(dir)
	if err != nil {
		t.Fatalf("failed to reopen store: %v", err)
	}

	loaded, ok := s2.GetManifest(manifest.FileHash)
	if !ok {
		t.Fatal("expected manifest to persist across store instances")
	}

	if loaded.Filename != manifest.Filename {
		t.Errorf("filename mismatch: expected %s got %s", manifest.Filename, loaded.Filename)
	}

	t.Log("manifest persisted and loaded correctly across store instances")
}

func TestDeleteChunksWithSharing(t *testing.T) {
	s, cleanup := newTestStore(t)
	defer cleanup()

	sharedData := []byte("shared chunk between two files")
	sharedChunk := makeChunk(t, sharedData, 0)
	s.WriteChunk(sharedChunk)

	manifest1 := &chunk.Manifest{
		FileHash:    "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		ChunkHashes: []string{sharedChunk.Hash},
		ChunkCount:  1,
	}
	manifest2 := &chunk.Manifest{
		FileHash:    "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
		ChunkHashes: []string{sharedChunk.Hash},
		ChunkCount:  1,
	}

	s.WriteManifest(manifest1)
	s.WriteManifest(manifest2)

	// delete manifest1 — shared chunk should NOT be deleted
	if err := s.DeleteChunks(manifest1); err != nil {
		t.Fatalf("DeleteChunks failed: %v", err)
	}

	if !s.HasChunk(sharedChunk.Hash) {
		t.Error("shared chunk was incorrectly deleted")
	}

	t.Log("shared chunk correctly preserved after deleting one referencing manifest")
}