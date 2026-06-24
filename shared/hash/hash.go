package hash

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
)

const HashLength = 64

func HashFile(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("failed to open file for hashing: %w", err)
	}
	defer f.Close()

	return HashReader(f)
}

func HashReader(r io.Reader) (string, error) {
	h := sha256.New()
	if _, err := io.Copy(h, r); err != nil {
		return "", fmt.Errorf("failed to hash reader: %w", err)
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func HashBytes(data []byte) string {
	h := sha256.Sum256(data)
	return hex.EncodeToString(h[:])
}

func Verify(path string, expected string) (bool, error) {
	actual, err := HashFile(path)
	if err != nil {
		return false, fmt.Errorf("failed to hash file for verification: %w", err)
	}
	return actual == expected, nil
}

func Validate(h string) error {
	if len(h) != HashLength {
		return fmt.Errorf("invalid hash length: expected %d got %d", HashLength, len(h))
	}
	if _, err := hex.DecodeString(h); err != nil {
		return fmt.Errorf("invalid hash encoding: must be hex: %w", err)
	}
	return nil
}

func Short(h string) string {
	if len(h) <= 12 {
		return h
	}
	return h[:12]
}
