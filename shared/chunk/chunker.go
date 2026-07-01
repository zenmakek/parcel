package chunk

import (
	"fmt"
	"io"
	"os"

	"github.com/zenmakek/parcel/shared/hash"
)

const DefaultChunkSize = 256 * 1024 // 256KB

type Chunk struct {
	Index int
	Hash  string
	Data  []byte
	Size  int
}

// ChunkFile splits a file into fixed-size chunks.
// Returns a slice of Chunk structs, each with its index, hash, and data.
// The last chunk may be smaller than ChunkSize.
func ChunkFile(path string, chunkSize int) ([]*Chunk, error) {
	if chunkSize <= 0 {
		chunkSize = DefaultChunkSize
	}

	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open file for chunking: %w", err)
	}
	defer f.Close()

	var chunks []*Chunk
	buf := make([]byte, chunkSize)
	index := 0

	for {
		n, err := io.ReadFull(f, buf)
		if n > 0 {
			data := make([]byte, n)
			copy(data, buf[:n])

			chunk := &Chunk{
				Index: index,
				Hash:  hash.HashBytes(data),
				Data:  data,
				Size:  n,
			}
			chunks = append(chunks, chunk)
			index++
		}

		if err == io.EOF || err == io.ErrUnexpectedEOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to read chunk %d: %w", index, err)
		}
	}

	return chunks, nil
}

// ChunkReader splits an io.Reader into chunks.
// Used when the source is a network stream or buffer, not a file.
func ChunkReader(r io.Reader, chunkSize int) ([]*Chunk, error) {
	if chunkSize <= 0 {
		chunkSize = DefaultChunkSize
	}

	var chunks []*Chunk
	buf := make([]byte, chunkSize)
	index := 0

	for {
		n, err := io.ReadFull(r, buf)
		if n > 0 {
			data := make([]byte, n)
			copy(data, buf[:n])

			chunk := &Chunk{
				Index: index,
				Hash:  hash.HashBytes(data),
				Data:  data,
				Size:  n,
			}
			chunks = append(chunks, chunk)
			index++
		}

		if err == io.EOF || err == io.ErrUnexpectedEOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to read chunk %d: %w", index, err)
		}
	}

	return chunks, nil
}
