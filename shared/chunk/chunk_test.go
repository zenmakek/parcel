package chunk

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/zenmakek/parcel/shared/hash"
)

func TestChunkAndAssemble(t *testing.T) {
	tmp, err := os.CreateTemp("", "parcel-chunk-*.bin")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer os.Remove(tmp.Name())

	data := make([]byte, 600*1024)
	for i := range data {
		data[i] = byte(i % 256)
	}
	tmp.Write(data)
	tmp.Close()

	// compute file hash BEFORE chunking
	fileHash, err := hash.HashFile(tmp.Name())
	if err != nil {
		t.Fatalf("HashFile failed: %v", err)
	}

	chunks, err := ChunkFile(tmp.Name(), DefaultChunkSize)
	if err != nil {
		t.Fatalf("ChunkFile failed: %v", err)
	}

	if len(chunks) != 3 {
		t.Errorf("expected 3 chunks got %d", len(chunks))
	}

	t.Logf("chunk 0: %d bytes, hash: %s", chunks[0].Size, chunks[0].Hash[:12])
	t.Logf("chunk 1: %d bytes, hash: %s", chunks[1].Size, chunks[1].Hash[:12])
	t.Logf("chunk 2: %d bytes, hash: %s", chunks[2].Size, chunks[2].Hash[:12])

	info, _ := os.Stat(tmp.Name())
	manifest := BuildManifest(chunks, filepath.Base(tmp.Name()), info.Size(), false, fileHash)

	t.Logf("file hash: %s", manifest.FileHash[:12])
	t.Logf("chunk count: %d", manifest.ChunkCount)

	destDir, err := os.MkdirTemp("", "parcel-assemble-*")
	if err != nil {
		t.Fatalf("failed to create dest dir: %v", err)
	}
	defer os.RemoveAll(destDir)

	destPath := filepath.Join(destDir, "assembled.bin")
	assembler := NewAssembler(manifest)

	for _, c := range chunks {
		if err := assembler.AddChunk(c); err != nil {
			t.Fatalf("AddChunk failed for index %d: %v", c.Index, err)
		}
	}

	if !assembler.IsComplete() {
		t.Fatal("expected assembler to be complete")
	}

	if err := assembler.Assemble(destPath); err != nil {
		t.Fatalf("Assemble failed: %v", err)
	}

	assembled, _ := os.ReadFile(destPath)
	if len(assembled) != len(data) {
		t.Errorf("size mismatch: expected %d got %d", len(data), len(assembled))
	}
	for i := range data {
		if assembled[i] != data[i] {
			t.Errorf("data mismatch at byte %d", i)
			break
		}
	}

	t.Log("chunk and assemble round-trip successful")
}

func TestMissingChunks(t *testing.T) {
	manifest := &Manifest{
		ChunkCount:  5,
		ChunkHashes: make([]string, 5),
	}

	have := map[int]bool{0: true, 2: true, 4: true}
	missing := manifest.MissingChunks(have)

	if len(missing) != 2 {
		t.Errorf("expected 2 missing chunks got %d", len(missing))
	}
	if missing[0] != 1 || missing[1] != 3 {
		t.Errorf("expected missing [1 3] got %v", missing)
	}

	t.Logf("missing chunks: %v", missing)
}

func TestVerifyChunkTampering(t *testing.T) {
	tmp, err := os.CreateTemp("", "parcel-tamper-*.bin")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer os.Remove(tmp.Name())

	data := make([]byte, 1024)
	tmp.Write(data)
	tmp.Close()

	fileHash, err := hash.HashFile(tmp.Name())
	if err != nil {
		t.Fatalf("HashFile failed: %v", err)
	}

	chunks, err := ChunkFile(tmp.Name(), DefaultChunkSize)
	if err != nil {
		t.Fatalf("ChunkFile failed: %v", err)
	}

	info, _ := os.Stat(tmp.Name())
	manifest := BuildManifest(chunks, "test.bin", info.Size(), false, fileHash)

	tampered := make([]byte, len(chunks[0].Data))
	copy(tampered, chunks[0].Data)
	tampered[0] = tampered[0] ^ 0xFF

	err = manifest.VerifyChunk(0, tampered)
	if err == nil {
		t.Fatal("expected verification to fail for tampered chunk")
	}

	t.Logf("correctly detected tampering: %v", err)
}

func TestManifestSaveLoad(t *testing.T) {
	manifest := &Manifest{
		FileHash:    "abc123abc123abc123abc123abc123abc123abc123abc123abc123abc123abcd",
		Filename:    "test.txt",
		TotalSize:   1024,
		ChunkSize:   DefaultChunkSize,
		ChunkCount:  1,
		ChunkHashes: []string{"def456def456def456def456def456def456def456def456def456def456deff"},
		IsArchive:   false,
	}

	tmp, err := os.CreateTemp("", "parcel-manifest-*.json")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	tmp.Close()
	defer os.Remove(tmp.Name())

	if err := manifest.Save(tmp.Name()); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	loaded, err := LoadManifest(tmp.Name())
	if err != nil {
		t.Fatalf("LoadManifest failed: %v", err)
	}

	if loaded.FileHash != manifest.FileHash {
		t.Errorf("file hash mismatch")
	}
	if loaded.ChunkCount != manifest.ChunkCount {
		t.Errorf("chunk count mismatch")
	}

	t.Log("manifest save/load round-trip successful")
}
