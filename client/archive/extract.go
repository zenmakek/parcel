package archive

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

const maxFileSize = 100 * 1024 * 1024 // 100MB

func ExtractArchive(archivePath string, destDir string) error {
	if !strings.HasSuffix(archivePath, ".tar.gz") {
		return fmt.Errorf("not a .tar.gz file: %s", archivePath)
	}

	inFile, err := os.Open(archivePath)
	if err != nil {
		return fmt.Errorf("failed to open archive: %w", err)
	}
	defer inFile.Close()

	gzReader, err := gzip.NewReader(inFile)
	if err != nil {
		return fmt.Errorf("failed to create gzip reader: %w", err)
	}
	defer gzReader.Close()

	tarReader := tar.NewReader(gzReader)

	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("failed to read tar entry: %w", err)
		}

		if err := validateTarPath(header.Name); err != nil {
			return fmt.Errorf("unsafe tar entry rejected: %w", err)
		}

		destPath := filepath.Join(destDir, header.Name)

		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(destPath, 0755); err != nil {
				return fmt.Errorf("failed to create directory %s: %w", destPath, err)
			}

		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
				return fmt.Errorf("failed to create parent dir for %s: %w", destPath, err)
			}

			if header.Size > maxFileSize {
				return fmt.Errorf("file too large: %s (%d bytes)", header.Name, header.Size)
			}

			outFile, err := os.Create(destPath)
			if err != nil {
				return fmt.Errorf("failed to create file %s: %w", destPath, err)
			}

			if _, err := io.CopyN(outFile, tarReader, header.Size); err != nil && err != io.EOF {
				outFile.Close()
				return fmt.Errorf("failed to write file %s: %w", destPath, err)
			}

			outFile.Close()

		default:
			fmt.Printf("  [archive] skipping unsupported entry type: %s\n", header.Name)
		}
	}

	fmt.Printf("  [archive] extracted: %s → %s\n", archivePath, destDir)
	return nil
}

func validateTarPath(path string) error {
	if strings.Contains(path, "..") {
		return fmt.Errorf("path traversal detected: %s", path)
	}
	if filepath.IsAbs(path) {
		return fmt.Errorf("absolute path rejected: %s", path)
	}
	return nil
}