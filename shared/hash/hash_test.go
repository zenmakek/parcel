package hash

import (
	"os"
	"testing"
)

func TestHashBytes(t *testing.T) {
	// SHA256 of "hello" is well-known — good sanity check
	h := HashBytes([]byte("hello"))
	expected := "2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824"
	if h != expected {
		t.Errorf("expected %s got %s", expected, h)
	}
	t.Logf("HashBytes: %s", h)
}

func TestHashFile(t *testing.T) {
	tmp, err := os.CreateTemp("", "parcel-hash-*.txt")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer os.Remove(tmp.Name())

	tmp.WriteString("hello parcel")
	tmp.Close()

	h, err := HashFile(tmp.Name())
	if err != nil {
		t.Fatalf("HashFile failed: %v", err)
	}

	if len(h) != HashLength {
		t.Errorf("expected hash length %d got %d", HashLength, len(h))
	}

	t.Logf("HashFile: %s", h)
}

func TestHashFileConsistency(t *testing.T) {
	tmp, err := os.CreateTemp("", "parcel-hash-*.txt")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer os.Remove(tmp.Name())

	tmp.WriteString("consistent content")
	tmp.Close()

	h1, err := HashFile(tmp.Name())
	if err != nil {
		t.Fatalf("first hash failed: %v", err)
	}

	h2, err := HashFile(tmp.Name())
	if err != nil {
		t.Fatalf("second hash failed: %v", err)
	}

	if h1 != h2 {
		t.Errorf("same file produced different hashes: %s vs %s", h1, h2)
	}

	t.Log("consistency check passed")
}

func TestVerify(t *testing.T) {
	tmp, err := os.CreateTemp("", "parcel-verify-*.txt")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer os.Remove(tmp.Name())

	tmp.WriteString("verify me")
	tmp.Close()

	h, err := HashFile(tmp.Name())
	if err != nil {
		t.Fatalf("HashFile failed: %v", err)
	}

	ok, err := Verify(tmp.Name(), h)
	if err != nil {
		t.Fatalf("Verify failed: %v", err)
	}
	if !ok {
		t.Error("expected verification to pass")
	}

	ok, err = Verify(tmp.Name(), "wronghashwronghashwronghashwronghashwronghashwronghashwronghashwro")
	if err != nil {
		t.Fatalf("Verify with wrong hash failed unexpectedly: %v", err)
	}
	if ok {
		t.Error("expected verification to fail with wrong hash")
	}

	t.Log("Verify works correctly")
}

func TestValidate(t *testing.T) {
	validHash := "2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824"
	if err := Validate(validHash); err != nil {
		t.Errorf("expected valid hash to pass: %v", err)
	}

	if err := Validate("tooshort"); err == nil {
		t.Error("expected short hash to fail")
	}

	if err := Validate("zzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzz"); err == nil {
		t.Error("expected non-hex hash to fail")
	}

	t.Log("Validate works correctly")
}

func TestShort(t *testing.T) {
	h := "2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824"
	s := Short(h)
	if s != "2cf24dba5fb0" {
		t.Errorf("expected 2cf24dba5fb0 got %s", s)
	}
	t.Logf("Short: %s", s)
}