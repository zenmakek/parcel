package store

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/zenmakek/parcel/shared/chunk"
	"github.com/zenmakek/parcel/shared/hash"
)

// Store manages chunks on disk under ~/.parcel/store/
// Chunks are stored by their own hash — two files sharing
// a chunk store it exactly once (content-addressed deduplication).
type Store struct {
	root      string // ~/.parcel/store/chunks/
	indexPath string // ~/.parcel/store/index.json
	index     *Index
}

// New creates or opens a Store at the default location.
func New() (*Store, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}
	return NewAt(filepath.Join(home, ".parcel", "store"))
}

// NewAt creates or opens a Store at a specific path.
// Used in tests to avoid polluting the real store.
func NewAt(root string) (*Store, error) {
	chunksDir := filepath.Join(root, "chunks")
	if err := os.MkdirAll(chunksDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create chunk store directory: %w", err)
	}

	indexPath := filepath.Join(root, "index.json")
	index, err := loadOrCreateIndex(indexPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load index: %w", err)
	}

	return &Store{
		root:      chunksDir,
		indexPath: indexPath,
		index:     index,
	}, nil
}

// WriteChunk saves a chunk to disk by its hash.
// If the chunk already exists (same hash), it is not rewritten.
// This is content-addressed deduplication — identical chunks
// across different files are stored exactly once.
func (s *Store) WriteChunk(c *chunk.Chunk) error {
	path := s.chunkPath(c.Hash)

	if _, err := os.Stat(path); err == nil {
		return nil // already exists
	}

	if err := os.WriteFile(path, c.Data, 0644); err != nil {
		return fmt.Errorf("failed to write chunk %s: %w", c.Hash[:12], err)
	}

	return nil
}

// ReadChunk loads a chunk from disk by its hash and index.
// Verifies the data matches the hash before returning.
func (s *Store) ReadChunk(chunkHash string, index int) (*chunk.Chunk, error) {
	path := s.chunkPath(chunkHash)

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("chunk %s not found in store", chunkHash[:12])
		}
		return nil, fmt.Errorf("failed to read chunk %s: %w", chunkHash[:12], err)
	}

	actual := hash.HashBytes(data)
	if actual != chunkHash {
		return nil, fmt.Errorf("chunk %s is corrupt: hash mismatch", chunkHash[:12])
	}

	return &chunk.Chunk{
		Index: index,
		Hash:  chunkHash,
		Data:  data,
		Size:  len(data),
	}, nil
}

// HasChunk returns true if a chunk with the given hash exists on disk.
func (s *Store) HasChunk(chunkHash string) bool {
	_, err := os.Stat(s.chunkPath(chunkHash))
	return err == nil
}

// WriteManifest saves a manifest to the index and persists it.
func (s *Store) WriteManifest(m *chunk.Manifest) error {
	s.index.AddManifest(m)
	return s.index.Save(s.indexPath)
}

// GetManifest retrieves a manifest by file hash.
func (s *Store) GetManifest(fileHash string) (*chunk.Manifest, bool) {
	return s.index.GetManifest(fileHash)
}

// HaveChunks returns a map of which chunk indices are present
// on disk for a given manifest.
// Used to compute missing chunks for resume.
func (s *Store) HaveChunks(m *chunk.Manifest) map[int]bool {
	have := make(map[int]bool)
	for i, h := range m.ChunkHashes {
		if s.HasChunk(h) {
			have[i] = true
		}
	}
	return have
}

// IsComplete returns true if all chunks for a manifest are on disk.
func (s *Store) IsComplete(m *chunk.Manifest) bool {
	have := s.HaveChunks(m)
	return len(have) == m.ChunkCount
}

// DeleteChunks removes all chunks for a manifest from disk.
// Only deletes chunks not referenced by any other manifest.
func (s *Store) DeleteChunks(m *chunk.Manifest) error {
	referencedByOthers := s.index.ChunksReferencedByOthers(m.FileHash)

	for _, chunkHash := range m.ChunkHashes {
		if referencedByOthers[chunkHash] {
			continue // shared chunk — don't delete
		}
		path := s.chunkPath(chunkHash)
		if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("failed to delete chunk %s: %w", chunkHash[:12], err)
		}
	}

	s.index.RemoveManifest(m.FileHash)
	return s.index.Save(s.indexPath)
}

// StoreSize returns the total bytes used by all chunks on disk.
func (s *Store) StoreSize() (int64, error) {
	var total int64
	entries, err := os.ReadDir(s.root)
	if err != nil {
		return 0, fmt.Errorf("failed to read store directory: %w", err)
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		total += info.Size()
	}
	return total, nil
}

func (s *Store) chunkPath(chunkHash string) string {
	return filepath.Join(s.root, chunkHash+".chunk")
}