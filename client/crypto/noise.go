package crypto

import (
	"crypto/ed25519"
	"crypto/rand"
	"fmt"
	"net"

	"github.com/flynn/noise"
)

// Config holds the local keypair for the Noise handshake.
type Config struct {
	StaticKey noise.DHKey
}

// NewConfig generates a Noise static keypair from an Ed25519 private key.
// We derive a Curve25519 keypair from the Ed25519 seed — both use
// the same underlying curve arithmetic.
func NewConfig(priv ed25519.PrivateKey) (*Config, error) {
	// Ed25519 private key = 64 bytes: seed (32) + public (32)
	// Noise uses Curve25519 — derive from the same 32-byte seed
	seed := []byte(priv)[:32]

	cs := noise.NewCipherSuite(noise.DH25519, noise.CipherChaChaPoly, noise.HashSHA256)
	dhKey, err := cs.GenerateKeypair(newDeterministicReader(seed))
	if err != nil {
		return nil, fmt.Errorf("failed to derive noise keypair: %w", err)
	}

	return &Config{StaticKey: dhKey}, nil
}

// Handshake performs the Noise_XX pattern handshake over conn.
// XX means both sides authenticate each other — mutual auth.
// Returns an encrypted Session on success.
func Handshake(conn net.Conn, cfg *Config, initiator bool) (*Session, error) {
	cs := noise.NewCipherSuite(noise.DH25519, noise.CipherChaChaPoly, noise.HashSHA256)

	hs, err := noise.NewHandshakeState(noise.Config{
		CipherSuite:   cs,
		Random:        rand.Reader,
		Pattern:       noise.HandshakeXX,
		Initiator:     initiator,
		StaticKeypair: cfg.StaticKey,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create handshake state: %w", err)
	}

	var send, recv *noise.CipherState

	if initiator {
		// initiator sends first
		send, recv, err = runInitiatorHandshake(conn, hs)
	} else {
		send, recv, err = runResponderHandshake(conn, hs)
	}

	if err != nil {
		return nil, fmt.Errorf("noise handshake failed: %w", err)
	}

	return newSession(conn, send, recv), nil
}

func runInitiatorHandshake(conn net.Conn, hs *noise.HandshakeState) (*noise.CipherState, *noise.CipherState, error) {
	// message 1: initiator → responder
	msg, _, _, err := hs.WriteMessage(nil, nil)
	if err != nil {
		return nil, nil, err
	}
	if err := writeFrame(conn, msg); err != nil {
		return nil, nil, err
	}

	// message 2: responder → initiator
	frame, err := readFrame(conn)
	if err != nil {
		return nil, nil, err
	}
	if _, _, _, err := hs.ReadMessage(nil, frame); err != nil {
		return nil, nil, err
	}

	// message 3: initiator → responder (final)
	msg, send, recv, err := hs.WriteMessage(nil, nil)
	if err != nil {
		return nil, nil, err
	}
	if err := writeFrame(conn, msg); err != nil {
		return nil, nil, err
	}

	return send, recv, nil
}

func runResponderHandshake(conn net.Conn, hs *noise.HandshakeState) (*noise.CipherState, *noise.CipherState, error) {
	// message 1: initiator → responder
	frame, err := readFrame(conn)
	if err != nil {
		return nil, nil, err
	}
	if _, _, _, err := hs.ReadMessage(nil, frame); err != nil {
		return nil, nil, err
	}

	// message 2: responder → initiator
	msg, _, _, err := hs.WriteMessage(nil, nil)
	if err != nil {
		return nil, nil, err
	}
	if err := writeFrame(conn, msg); err != nil {
		return nil, nil, err
	}

	// message 3: initiator → responder (final)
	frame, err = readFrame(conn)
	if err != nil {
		return nil, nil, err
	}
	_, send, recv, err := hs.ReadMessage(nil, frame)
	if err != nil {
		return nil, nil, err
	}

	return recv, send, nil
}

// deterministic reader derives Curve25519 key from Ed25519 seed.
type deterministicReader struct {
	seed []byte
	pos  int
}

func newDeterministicReader(seed []byte) *deterministicReader {
	return &deterministicReader{seed: seed}
}

func (r *deterministicReader) Read(p []byte) (int, error) {
	n := copy(p, r.seed[r.pos:])
	r.pos += n
	if n < len(p) {
		// pad with zeros if seed exhausted
		for i := n; i < len(p); i++ {
			p[i] = 0
		}
		return len(p), nil
	}
	return n, nil
}