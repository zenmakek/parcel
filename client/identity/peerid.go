package identity

import (
	"crypto/ed25519"
	"encoding/hex"
	"fmt"

	"github.com/zenmakek/parcel/shared/hash"
)

// Identity holds a keypair and the derived PeerID.
type Identity struct {
	*KeyPair
	PeerID string
}

// New loads or generates a keypair and derives the PeerID.
func New() (*Identity, error) {
	kp, err := Load()
	if err != nil {
		return nil, fmt.Errorf("failed to load keypair: %w", err)
	}
	return &Identity{
		KeyPair: kp,
		PeerID:  derivePeerID(kp.Public),
	}, nil
}

// derivePeerID computes PeerID = SHA256(public key), hex encoded.
func derivePeerID(pub ed25519.PublicKey) string {
	return hash.HashBytes(pub)
}

// Sign signs a message with the private key.
func (id *Identity) Sign(message []byte) []byte {
	return ed25519.Sign(id.Private, message)
}

// Verify checks a signature against a public key and message.
func Verify(pub ed25519.PublicKey, message, sig []byte) bool {
	return ed25519.Verify(pub, message, sig)
}

// PublicKeyHex returns the public key as a hex string for wire transfer.
func (id *Identity) PublicKeyHex() string {
	return hex.EncodeToString(id.Public)
}

// PublicKeyFromHex decodes a hex public key back to ed25519.PublicKey.
func PublicKeyFromHex(s string) (ed25519.PublicKey, error) {
	b, err := hex.DecodeString(s)
	if err != nil {
		return nil, fmt.Errorf("invalid public key hex: %w", err)
	}
	if len(b) != ed25519.PublicKeySize {
		return nil, fmt.Errorf("invalid public key length: %d", len(b))
	}
	return ed25519.PublicKey(b), nil
}

// Short returns the first 12 chars of the PeerID for display.
func (id *Identity) Short() string {
	if len(id.PeerID) <= 12 {
		return id.PeerID
	}
	return id.PeerID[:12]
}