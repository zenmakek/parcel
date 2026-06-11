package archive

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCompressAndExtract(t *testing.T) {
	srcDir, err := os.MkdirTemp("", "parcel-src-*")
	if err != nil {
		t.Fatalf("failed to create source dir: %v", err)
	}
	defer os.RemoveAll(srcDir)

	os.WriteFile(filepath.Join(srcDir, "hello.txt"), []byte("hello parcel"), 0644)
	os.WriteFile(filepath.Join(srcDir, "world.txt"), []byte("world parcel"), 0644)

	subDir := filepath.Join(srcDir, "subdir")
	os.MkdirAll(subDir, 0755)
	os.WriteFile(filepath.Join(subDir, "nested.txt"), []byte("nested file"), 0644)

	archivePath := srcDir + ".tar.gz"
	defer os.Remove(archivePath)

	if err := CompressFolder(srcDir, archivePath); err != nil {
		t.Fatalf("CompressFolder failed: %v", err)
	}

	info, err := os.Stat(archivePath)
	if err != nil {
		t.Fatalf("archive file not created: %v", err)
	}
	t.Logf("archive size: %d bytes", info.Size())

	destDir, err := os.MkdirTemp("", "parcel-dest-*")
	if err != nil {
		t.Fatalf("failed to create dest dir: %v", err)
	}
	defer os.RemoveAll(destDir)

	if err := ExtractArchive(archivePath, destDir); err != nil {
		t.Fatalf("ExtractArchive failed: %v", err)
	}

	baseName := filepath.Base(srcDir)

	checkFile := func(relPath string, expectedContent string) {
		fullPath := filepath.Join(destDir, baseName, relPath)
		data, err := os.ReadFile(fullPath)
		if err != nil {
			t.Errorf("expected file not found: %s — %v", fullPath, err)
			return
		}
		if string(data) != expectedContent {
			t.Errorf("file %s: expected %q got %q", relPath, expectedContent, string(data))
		}
	}

	checkFile("hello.txt", "hello parcel")
	checkFile("world.txt", "world parcel")
	checkFile(filepath.Join("subdir", "nested.txt"), "nested file")

	t.Log("compress and extract round-trip successful")
}

func TestCleanupArchive(t *testing.T) {
	tmp, err := os.CreateTemp("", "parcel-cleanup-*.tar.gz")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	tmp.Close()

	if err := CleanupArchive(tmp.Name()); err != nil {
		t.Fatalf("CleanupArchive failed: %v", err)
	}

	if _, err := os.Stat(tmp.Name()); !os.IsNotExist(err) {
		t.Fatal("expected file to be deleted")
	}

	t.Log("cleanup successful")
}

func TestCleanupRefusesNonArchive(t *testing.T) {
	err := CleanupArchive("/tmp/somefile.txt")
	if err == nil {
		t.Fatal("expected error when deleting non-archive file")
	}
	t.Logf("correctly refused: %v", err)
}

func TestValidateTarPath(t *testing.T) {
	if err := validateTarPath("../etc/passwd"); err == nil {
		t.Fatal("expected error for path traversal")
	}

	if err := validateTarPath("/etc/passwd"); err == nil {
		t.Fatal("expected error for absolute path")
	}

	if err := validateTarPath("subdir/file.txt"); err != nil {
		t.Errorf("expected no error for safe path, got: %v", err)
	}

	t.Log("path validation works correctly")
}
