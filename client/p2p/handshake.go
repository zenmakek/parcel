package p2p

import (
	"bufio"
	"crypto/ed25519"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/zenmakek/parcel/client/identity"
	"github.com/zenmakek/parcel/shared/hash"
	"github.com/zenmakek/parcel/shared/protocol"
)

// PeerConn represents an authenticated connection to a remote peer.
type PeerConn struct {
	conn interface {
		Read([]byte) (int, error)
		Write([]byte) (int, error)
		Close() error
	}
	reader          *bufio.Reader
	RemotePeerID    string
	RemotePublicKey ed25519.PublicKey
	LocalIdentity   *identity.Identity
}

// handshake performs mutual identity exchange over a connection.
// Both peers send their PeerID and public key, verify each other,
// then sign a challenge to prove key ownership.
func handshake(conn interface {
	Read([]byte) (int, error)
	Write([]byte) (int, error)
	Close() error
}, id *identity.Identity) (*PeerConn, error) {
	reader := bufio.NewReader(conn)

	// send our hello with public key
	hello, err := protocol.Encode(protocol.PacketHello, protocol.HelloPayload{
		Version: "2.0.0",
		Role:    id.PublicKeyHex(),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to encode hello: %w", err)
	}
	if _, err := fmt.Fprint(conn, hello); err != nil {
		return nil, fmt.Errorf("failed to send hello: %w", err)
	}

	// read remote hello
	line, err := reader.ReadString('\n')
	if err != nil {
		return nil, fmt.Errorf("failed to read remote hello: %w", err)
	}

	packet, err := protocol.Decode(strings.TrimSpace(line))
	if err != nil {
		return nil, fmt.Errorf("failed to decode remote hello: %w", err)
	}

	if packet.Type != protocol.PacketHello {
		return nil, fmt.Errorf("expected HELLO got %s", packet.Type)
	}

	var remoteHello protocol.HelloPayload
	if err := protocol.DecodePayload(packet.Payload, &remoteHello); err != nil {
		return nil, fmt.Errorf("failed to decode remote hello payload: %w", err)
	}

	remotePub, err := identity.PublicKeyFromHex(remoteHello.Role)
	if err != nil {
		return nil, fmt.Errorf("invalid remote public key: %w", err)
	}

	remotePeerID := derivePeerID(remotePub)
	// sign our PeerID as proof of key ownership
	sig := id.Sign([]byte(id.PeerID))
	sigPacket, err := protocol.Encode(protocol.PacketAck, protocol.AckPayload{
		Message: fmt.Sprintf("%x", sig),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to encode signature: %w", err)
	}
	if _, err := fmt.Fprint(conn, sigPacket); err != nil {
		return nil, fmt.Errorf("failed to send signature: %w", err)
	}

	// read remote signature
	sigLine, err := reader.ReadString('\n')
	if err != nil {
		return nil, fmt.Errorf("failed to read remote signature: %w", err)
	}

	sigPkt, err := protocol.Decode(strings.TrimSpace(sigLine))
	if err != nil {
		return nil, fmt.Errorf("failed to decode remote signature: %w", err)
	}

	var sigPayload protocol.AckPayload
	if err := protocol.DecodePayload(sigPkt.Payload, &sigPayload); err != nil {
		return nil, fmt.Errorf("failed to decode signature payload: %w", err)
	}

	remoteSig, err := hex.DecodeString(sigPayload.Message)
	if err != nil {
		return nil, fmt.Errorf("failed to decode remote signature: %w", err)
	}

	if !identity.Verify(remotePub, []byte(remotePeerID), remoteSig) {
		return nil, fmt.Errorf("remote peer failed signature verification")
	}

	return &PeerConn{
		conn:            conn,
		reader:          reader,
		RemotePeerID:    remotePeerID,
		RemotePublicKey: remotePub,
		LocalIdentity:   id,
	}, nil
}

// Send encodes and sends a packet to the remote peer.
func (pc *PeerConn) Send(packetType string, payload any) error {
	encoded, err := protocol.Encode(packetType, payload)
	if err != nil {
		return fmt.Errorf("failed to encode packet: %w", err)
	}
	if _, err := fmt.Fprint(pc.conn, encoded); err != nil {
		return fmt.Errorf("failed to send packet: %w", err)
	}
	return nil
}

// Receive reads and decodes the next packet from the remote peer.
func (pc *PeerConn) Receive() (*protocol.Packet, error) {
	line, err := pc.reader.ReadString('\n')
	if err != nil {
		return nil, fmt.Errorf("failed to read packet: %w", err)
	}
	return protocol.Decode(strings.TrimSpace(line))
}

// Close closes the underlying connection.
func (pc *PeerConn) Close() {
	pc.conn.Close()
}

func derivePeerID(pub ed25519.PublicKey) string {
	return hash.HashBytes(pub)
}
