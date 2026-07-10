package identity

import (
	"os"
	"testing"
)

func TestGenerateAndLoad(t *testing.T) {
	dir, err := os.MkdirTemp("", "parcel-identity-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(dir)

	// override identity dir for test
	privPath := dir + "/identity.key"
	pubPath := dir + "/identity.pub"

	kp, err := generate(privPath, pubPath)
	if err != nil {
		t.Fatalf("generate failed: %v", err)
	}

	if len(kp.Private) == 0 || len(kp.Public) == 0 {
		t.Fatal("keypair has empty keys")
	}

	loaded, err := load(privPath, pubPath)
	if err != nil {
		t.Fatalf("load failed: %v", err)
	}

	if derivePeerID(kp.Public) != derivePeerID(loaded.Public) {
		t.Error("PeerID mismatch after reload")
	}

	t.Logf("PeerID: %s", derivePeerID(kp.Public)[:12])
}

func TestSignAndVerify(t *testing.T) {
	dir, err := os.MkdirTemp("", "parcel-sign-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(dir)

	kp, err := generate(dir+"/identity.key", dir+"/identity.pub")
	if err != nil {
		t.Fatalf("generate failed: %v", err)
	}

	id := &Identity{KeyPair: kp, PeerID: derivePeerID(kp.Public)}

	message := []byte("parcel p2p handshake")
	sig := id.Sign(message)

	if !Verify(kp.Public, message, sig) {
		t.Fatal("signature verification failed")
	}

	tampered := []byte("parcel p2p handshake tampered")
	if Verify(kp.Public, tampered, sig) {
		t.Fatal("expected verification to fail for tampered message")
	}

	t.Logf("sign/verify passed for peer %s", id.Short())
}

func TestPeerIDConsistency(t *testing.T) {
	dir, err := os.MkdirTemp("", "parcel-peerid-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(dir)

	kp, _ := generate(dir+"/identity.key", dir+"/identity.pub")

	id1 := derivePeerID(kp.Public)
	id2 := derivePeerID(kp.Public)

	if id1 != id2 {
		t.Error("PeerID is not deterministic")
	}

	t.Logf("PeerID consistent: %s", id1[:12])
}

func TestPublicKeyHexRoundTrip(t *testing.T) {
	dir, err := os.MkdirTemp("", "parcel-hex-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(dir)

	kp, _ := generate(dir+"/identity.key", dir+"/identity.pub")
	id := &Identity{KeyPair: kp, PeerID: derivePeerID(kp.Public)}

	hexKey := id.PublicKeyHex()
	decoded, err := PublicKeyFromHex(hexKey)
	if err != nil {
		t.Fatalf("PublicKeyFromHex failed: %v", err)
	}

	if string(decoded) != string(kp.Public) {
		t.Error("public key mismatch after hex round-trip")
	}

	t.Logf("public key hex round-trip passed: %s...", hexKey[:16])
}
