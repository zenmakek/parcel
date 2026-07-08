package p2p

import (
	"fmt"
	"net"
	"time"

	"github.com/zenmakek/parcel/client/identity"
)

const dialTimeout = 10 * time.Second

// Dial attempts a direct TCP connection to a peer address
// and performs the P2P handshake.
func Dial(address string, id *identity.Identity) (*PeerConn, error) {
	conn, err := net.DialTimeout("tcp", address, dialTimeout)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to peer at %s: %w", address, err)
	}

	pc, err := handshake(conn, id)
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("handshake failed with %s: %w", address, err)
	}

	fmt.Printf("[p2p] connected to peer %s at %s\n", pc.RemotePeerID[:12], address)
	return pc, nil
}

// DialWithFallback tries a direct connection first.
// If it fails, returns an error so the caller can fall back to relay.
func DialWithFallback(address string, id *identity.Identity) (*PeerConn, error) {
	pc, err := Dial(address, id)
	if err != nil {
		fmt.Printf("[p2p] direct connection failed: %v — will fall back to relay\n", err)
		return nil, err
	}
	return pc, nil
}