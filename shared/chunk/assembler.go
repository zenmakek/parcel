package chunk

import (
	"fmt"
	"os"
	"sort"

	"github.com/zenmakek/parcel/shared/hash"
)

// Assembler collects chunks and writes the final file when complete.
type Assembler struct {
	manifest *Manifest
	chunks   map[int]*Chunk
}

// NewAssembler creates an Assembler for a given manifest.
func NewAssembler(m *Manifest) *Assembler {
	return &Assembler{
		manifest: m,
		chunks:   make(map[int]*Chunk),
	}
}

// AddChunk adds a verified chunk to the assembler.
// Returns an error if the chunk hash doesn't match the manifest.
func (a *Assembler) AddChunk(c *Chunk) error {
	if err := a.manifest.VerifyChunk(c.Index, c.Data); err != nil {
		return fmt.Errorf("chunk verification failed: %w", err)
	}
	a.chunks[c.Index] = c
	return nil
}

// IsComplete returns true when all chunks have been received and verified.
func (a *Assembler) IsComplete() bool {
	return len(a.chunks) == a.manifest.ChunkCount
}

// Progress returns how many chunks have been received out of total.
func (a *Assembler) Progress() (int, int) {
	return len(a.chunks), a.manifest.ChunkCount
}

// Assemble writes all chunks to destPath in order.
// Returns an error if any chunk is missing or the final file hash doesn't match.
func (a *Assembler) Assemble(destPath string) error {
	if !a.IsComplete() {
		have, total := a.Progress()
		return fmt.Errorf("cannot assemble: have %d of %d chunks", have, total)
	}

	indices := make([]int, 0, len(a.chunks))
	for i := range a.chunks {
		indices = append(indices, i)
	}
	sort.Ints(indices)

	outFile, err := os.Create(destPath)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer outFile.Close()

	for _, i := range indices {
		chunk := a.chunks[i]
		if _, err := outFile.Write(chunk.Data); err != nil {
			return fmt.Errorf("failed to write chunk %d: %w", i, err)
		}
	}

	ok, err := hash.Verify(destPath, a.manifest.FileHash)
	if err != nil {
		return fmt.Errorf("failed to verify assembled file: %w", err)
	}
	if !ok {
		return fmt.Errorf("assembled file hash mismatch — file may be corrupt")
	}

	return nil
}

// HaveChunks returns a map of which chunk indices have been received.
// Used by MissingChunks to compute what still needs fetching.
func (a *Assembler) HaveChunks() map[int]bool {
	have := make(map[int]bool)
	for i := range a.chunks {
		have[i] = true
	}
	return have
}