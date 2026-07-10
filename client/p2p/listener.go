package p2p

import (
	"fmt"
	"net"

	"github.com/zenmakek/parcel/client/identity"
)

const DefaultP2PPort = "9000"

// Listener accepts incoming peer connections on a TCP port.
type Listener struct {
	listener net.Listener
	id       *identity.Identity
	Port     string
}

// NewListener creates a Listener on the given port.
func NewListener(port string, id *identity.Identity) (*Listener, error) {
	if port == "" {
		port = DefaultP2PPort
	}

	ln, err := net.Listen("tcp", ":"+port)
	if err != nil {
		return nil, fmt.Errorf("failed to listen on port %s: %w", port, err)
	}

	fmt.Printf("[p2p] listening for peers on :%s\n", port)
	return &Listener{listener: ln, id: id, Port: port}, nil
}

// Accept blocks until an incoming peer connection is established
// and the handshake completes. Returns an authenticated PeerConn.
func (l *Listener) Accept() (*PeerConn, error) {
	conn, err := l.listener.Accept()
	if err != nil {
		return nil, fmt.Errorf("failed to accept connection: %w", err)
	}

	pc, err := handshake(conn, l.id)
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("handshake failed: %w", err)
	}

	fmt.Printf("[p2p] accepted peer %s from %s\n", pc.RemotePeerID[:12], conn.RemoteAddr())
	return pc, nil
}

// Close shuts down the listener.
func (l *Listener) Close() error {
	return l.listener.Close()
}

// Address returns the full listen address including port.
func (l *Listener) Address() string {
	return l.listener.Addr().String()
}