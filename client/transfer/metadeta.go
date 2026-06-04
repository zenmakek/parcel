package transfer

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type FileType int

const (
	TypeFile   FileType = iota
	TypeFolder FileType = iota
)

type Metadata struct {
	OriginalPath string
	Filename     string
	Size         int64
	Type         FileType
	IsArchive    bool
}

func Inspect(path string) (*Metadata, error) {
	cleaned := filepath.Clean(path)

	info, err := os.Stat(cleaned)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("path does not exist: %s", cleaned)
		}
		return nil, fmt.Errorf("failed to stat path: %w", err)
	}

	if info.IsDir() {
		size, err := dirSize(cleaned)
		if err != nil {
			return nil, fmt.Errorf("failed to calculate folder size: %w", err)
		}

		return &Metadata{
			OriginalPath: cleaned,
			Filename:     filepath.Base(cleaned) + ".tar.gz",
			Size:         size,
			Type:         TypeFolder,
			IsArchive:    true,
		}, nil
	}

	return &Metadata{
		OriginalPath: cleaned,
		Filename:     filepath.Base(cleaned),
		Size:         info.Size(),
		Type:         TypeFile,
		IsArchive:    false,
	}, nil
}

func dirSize(path string) (int64, error) {
	var total int64

	err := filepath.WalkDir(path, func(entry string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		total += info.Size()
		return nil
	})

	return total, err
}

func ValidatePath(path string) error {
	if strings.TrimSpace(path) == "" {
		return fmt.Errorf("path cannot be empty")
	}

	cleaned := filepath.Clean(path)
	_, err := os.Stat(cleaned)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("path does not exist: %s", cleaned)
		}
		return fmt.Errorf("cannot access path: %w", err)
	}

	return nil
}

func (m *Metadata) HumanSize() string {
	const (
		KB = 1024
		MB = 1024 * KB
	)

	switch {
	case m.Size >= MB:
		return fmt.Sprintf("%.2f MB", float64(m.Size)/MB)
	case m.Size >= KB:
		return fmt.Sprintf("%.2f KB", float64(m.Size)/KB)
	default:
		return fmt.Sprintf("%d bytes", m.Size)
	}
}
