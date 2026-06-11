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

func CompressFolder(sourcePath string, destPath string) error {
	sourceInfo, err := os.Stat(sourcePath)
	if err != nil {
		return fmt.Errorf("failed to stat source: %w", err)
	}

	if !sourceInfo.IsDir() {
		return fmt.Errorf("source is not a directory: %s", sourcePath)
	}

	outFile, err := os.Create(destPath)
	if err != nil {
		return fmt.Errorf("failed to create archive file: %w", err)
	}
	defer outFile.Close()

	gzWriter := gzip.NewWriter(outFile)
	defer gzWriter.Close()

	tarWriter := tar.NewWriter(gzWriter)
	defer tarWriter.Close()

	baseDir := filepath.Base(sourcePath)

	err = filepath.WalkDir(sourcePath, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}

		info, err := d.Info()
		if err != nil {
			return fmt.Errorf("failed to get info for %s: %w", path, err)
		}

		header, err := tar.FileInfoHeader(info, "")
		if err != nil {
			return fmt.Errorf("failed to create tar header for %s: %w", path, err)
		}

		relativePath, err := filepath.Rel(filepath.Dir(sourcePath), path)
		if err != nil {
			return fmt.Errorf("failed to compute relative path: %w", err)
		}

		header.Name = filepath.ToSlash(relativePath)

		if d.IsDir() {
			header.Name += "/"
		}

		if err := tarWriter.WriteHeader(header); err != nil {
			return fmt.Errorf("failed to write tar header: %w", err)
		}

		if d.IsDir() {
			return nil
		}

		file, err := os.Open(path)
		if err != nil {
			return fmt.Errorf("failed to open file %s: %w", path, err)
		}
		defer file.Close()

		if _, err := io.Copy(tarWriter, file); err != nil {
			return fmt.Errorf("failed to write file to archive: %w", err)
		}

		return nil
	})

	if err != nil {
		return fmt.Errorf("failed to walk directory: %w", err)
	}

	_ = baseDir
	fmt.Printf("  [archive] compressed: %s → %s\n", sourcePath, destPath)
	return nil
}

func ArchivePath(sourcePath string) string {
	base := filepath.Base(sourcePath)
	dir := filepath.Dir(sourcePath)
	return filepath.Join(dir, base+".tar.gz")
}

func CleanupArchive(path string) error {
	if !strings.HasSuffix(path, ".tar.gz") {
		return fmt.Errorf("refusing to delete non-archive file: %s", path)
	}
	if err := os.Remove(path); err != nil {
		return fmt.Errorf("failed to delete archive: %w", err)
	}
	fmt.Printf("  [archive] cleaned up: %s\n", path)
	return nil
}