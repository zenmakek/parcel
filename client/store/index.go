package store

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/zenmakek/parcel/shared/chunk"
)

// indexEntry stores a manifest and its completion status.
type indexEntry struct {
	Manifest  *chunk.Manifest `json:"manifest"`
	Complete  bool            `json:"complete"`
}

// Index maps file_hash → manifest + completion state.
// Persisted to disk as JSON at ~/.parcel/store/index.json.
type Index struct {
	Entries map[string]*indexEntry `json:"entries"`
}

func newIndex() *Index {
	return &Index{
		Entries: make(map[string]*indexEntry),
	}
}

func loadOrCreateIndex(path string) (*Index, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return newIndex(), nil
		}
		return nil, fmt.Errorf("failed to read index: %w", err)
	}

	var idx Index
	if err := json.Unmarshal(data, &idx); err != nil {
		return nil, fmt.Errorf("failed to parse index: %w", err)
	}

	if idx.Entries == nil {
		idx.Entries = make(map[string]*indexEntry)
	}

	return &idx, nil
}

// Save persists the index to disk atomically.
// Writes to a temp file first then renames — prevents corruption
// if the process is killed mid-write.
func (idx *Index) Save(path string) error {
	data, err := json.MarshalIndent(idx, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal index: %w", err)
	}

	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0644); err != nil {
		return fmt.Errorf("failed to write index temp file: %w", err)
	}

	if err := os.Rename(tmp, path); err != nil {
		return fmt.Errorf("failed to rename index file: %w", err)
	}

	return nil
}

// AddManifest adds or updates a manifest entry in the index.
func (idx *Index) AddManifest(m *chunk.Manifest) {
	idx.Entries[m.FileHash] = &indexEntry{
		Manifest: m,
		Complete: false,
	}
}

// MarkComplete marks a file as fully downloaded and verified.
func (idx *Index) MarkComplete(fileHash string) {
	if entry, ok := idx.Entries[fileHash]; ok {
		entry.Complete = true
	}
}

// GetManifest retrieves a manifest by file hash.
func (idx *Index) GetManifest(fileHash string) (*chunk.Manifest, bool) {
	entry, ok := idx.Entries[fileHash]
	if !ok {
		return nil, false
	}
	return entry.Manifest, true
}

// RemoveManifest deletes a manifest entry from the index.
func (idx *Index) RemoveManifest(fileHash string) {
	delete(idx.Entries, fileHash)
}

// ChunksReferencedByOthers returns a set of chunk hashes
// that are referenced by manifests OTHER than the given file hash.
// Used by DeleteChunks to avoid deleting shared chunks.
func (idx *Index) ChunksReferencedByOthers(excludeFileHash string) map[string]bool {
	referenced := make(map[string]bool)
	for fileHash, entry := range idx.Entries {
		if fileHash == excludeFileHash {
			continue
		}
		for _, h := range entry.Manifest.ChunkHashes {
			referenced[h] = true
		}
	}
	return referenced
}

// ListComplete returns all manifests that are fully downloaded.
// Used by the seeder to know what it can serve.
func (idx *Index) ListComplete() []*chunk.Manifest {
	var manifests []*chunk.Manifest
	for _, entry := range idx.Entries {
		if entry.Complete {
			manifests = append(manifests, entry.Manifest)
		}
	}
	return manifests
}

// ListIncomplete returns all manifests with partial downloads.
// Used on startup to resume interrupted transfers.
func (idx *Index) ListIncomplete() []*chunk.Manifest {
	var manifests []*chunk.Manifest
	for _, entry := range idx.Entries {
		if !entry.Complete {
			manifests = append(manifests, entry.Manifest)
		}
	}
	return manifests
}