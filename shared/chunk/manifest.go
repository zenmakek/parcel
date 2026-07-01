package chunk

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/zenmakek/parcel/shared/hash"
)

// Manifest describes a complete file in terms of its chunks.
// It is what gets shared between peers — not the file itself.
// The receiver uses it to know exactly which chunks to fetch
// and to verify each one after receiving it.
type Manifest struct {
	FileHash    string   `json:"file_hash"`
	Filename    string   `json:"filename"`
	TotalSize   int64    `json:"total_size"`
	ChunkSize   int      `json:"chunk_size"`
	ChunkCount  int      `json:"chunk_count"`
	ChunkHashes []string `json:"chunk_hashes"`
	IsArchive   bool     `json:"is_archive"`
}

// BuildManifest creates a Manifest from a slice of chunks and file metadata.
func BuildManifest(chunks []*Chunk, filename string, totalSize int64, isArchive bool, fileHash string) *Manifest {
	hashes := make([]string, len(chunks))
	for i, c := range chunks {
		hashes[i] = c.Hash
	}

	return &Manifest{
		FileHash:    fileHash,
		Filename:    filename,
		TotalSize:   totalSize,
		ChunkSize:   DefaultChunkSize,
		ChunkCount:  len(chunks),
		ChunkHashes: hashes,
		IsArchive:   isArchive,
	}
}


// Save writes the manifest to a JSON file at the given path.
func (m *Manifest) Save(path string) error {
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal manifest: %w", err)
	}
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write manifest: %w", err)
	}
	return nil
}

// LoadManifest reads a manifest from a JSON file.
func LoadManifest(path string) (*Manifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read manifest: %w", err)
	}
	var m Manifest
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("failed to unmarshal manifest: %w", err)
	}
	return &m, nil
}

// VerifyChunk checks that a chunk's data matches the expected hash
// from the manifest at the given index.
func (m *Manifest) VerifyChunk(index int, data []byte) error {
	if index < 0 || index >= len(m.ChunkHashes) {
		return fmt.Errorf("chunk index %d out of range (0-%d)", index, len(m.ChunkHashes)-1)
	}
	actual := hash.HashBytes(data)
	expected := m.ChunkHashes[index]
	if actual != expected {
		return fmt.Errorf("chunk %d hash mismatch: expected %s got %s", index, expected[:12], actual[:12])
	}
	return nil
}

// MissingChunks returns the indices of chunks not present in the provided set.
// Used during resume — tells the receiver which chunks still need fetching.
func (m *Manifest) MissingChunks(have map[int]bool) []int {
	var missing []int
	for i := 0; i < m.ChunkCount; i++ {
		if !have[i] {
			missing = append(missing, i)
		}
	}
	return missing
}