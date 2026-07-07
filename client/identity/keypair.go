package identity

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/pem"
	"fmt"
	"os"
	"path/filepath"
)

const (
	privateKeyFile = "identity.key"
	publicKeyFile  = "identity.pub"
)

type KeyPair struct {
	Private ed25519.PrivateKey
	Public  ed25519.PublicKey
}

// Load loads an existing keypair or generates a new one.
func Load() (*KeyPair, error) {
	dir, err := identityDir()
	if err != nil {
		return nil, err
	}

	privPath := filepath.Join(dir, privateKeyFile)
	pubPath := filepath.Join(dir, publicKeyFile)

	if _, err := os.Stat(privPath); os.IsNotExist(err) {
		return generate(privPath, pubPath)
	}

	return load(privPath, pubPath)
}

func generate(privPath, pubPath string) (*KeyPair, error) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("failed to generate keypair: %w", err)
	}

	if err := writeKey(privPath, "ED25519 PRIVATE KEY", priv); err != nil {
		return nil, err
	}
	if err := writeKey(pubPath, "ED25519 PUBLIC KEY", pub); err != nil {
		return nil, err
	}

	fmt.Println("  [identity] new keypair generated")
	return &KeyPair{Private: priv, Public: pub}, nil
}

func load(privPath, pubPath string) (*KeyPair, error) {
	priv, err := readKey(privPath)
	if err != nil {
		return nil, err
	}
	pub, err := readKey(pubPath)
	if err != nil {
		return nil, err
	}
	return &KeyPair{
		Private: ed25519.PrivateKey(priv),
		Public:  ed25519.PublicKey(pub),
	}, nil
}

func writeKey(path, keyType string, data []byte) error {
	block := &pem.Block{Type: keyType, Bytes: data}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0600)
	if err != nil {
		return fmt.Errorf("failed to create key file: %w", err)
	}
	defer f.Close()
	return pem.Encode(f, block)
}

func readKey(path string) ([]byte, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read key file: %w", err)
	}
	block, _ := pem.Decode(data)
	if block == nil {
		return nil, fmt.Errorf("failed to decode PEM block from %s", path)
	}
	return block.Bytes, nil
}

func identityDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}
	dir := filepath.Join(home, ".parcel", "identity")
	if err := os.MkdirAll(dir, 0700); err != nil {
		return "", fmt.Errorf("failed to create identity directory: %w", err)
	}
	return dir, nil
}