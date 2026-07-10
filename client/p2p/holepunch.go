package p2p

import (
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/zenmakek/parcel/client/identity"
	"github.com/zenmakek/parcel/shared/protocol"
)

const stunServer = "139.59.60.82:3478"

// DiscoverPublicAddress connects to the STUN server and returns
// our observed public IP:port — what the internet sees us as.
func DiscoverPublicAddress(id *identity.Identity) (string, error) {
	conn, err := net.DialTimeout("tcp", stunServer, 5*time.Second)
	if err != nil {
		return "", fmt.Errorf("failed to connect to STUN server: %w", err)
	}
	defer conn.Close()

	req, err := protocol.Encode(protocol.PacketSTUNRequest, protocol.STUNRequestPayload{
		PeerID: id.PeerID,
	})
	if err != nil {
		return "", err
	}

	if _, err := fmt.Fprint(conn, req); err != nil {
		return "", fmt.Errorf("failed to send STUN request: %w", err)
	}

	conn.SetDeadline(time.Now().Add(5 * time.Second))
	buf := make([]byte, 512)
	n, err := conn.Read(buf)
	if err != nil {
		return "", fmt.Errorf("failed to read STUN response: %w", err)
	}

	packet, err := protocol.Decode(strings.TrimSpace(string(buf[:n])))
	if err != nil {
		return "", fmt.Errorf("failed to decode STUN response: %w", err)
	}

	var resp protocol.STUNResponsePayload
	if err := protocol.DecodePayload(packet.Payload, &resp); err != nil {
		return "", fmt.Errorf("failed to decode STUN payload: %w", err)
	}

	fmt.Printf("[p2p] public address: %s\n", resp.ObservedAddress)
	return resp.ObservedAddress, nil
}

// HolePunch attempts to establish a direct connection to a peer
// by having both sides connect simultaneously at a coordinated time.
// Returns a PeerConn on success, error if hole punching fails.
func HolePunch(peerAddress string, connectAt time.Time, id *identity.Identity) (*PeerConn, error) {
	// wait until the coordinated connect time
	waitDuration := time.Until(connectAt)
	if waitDuration > 0 {
		fmt.Printf("[p2p] hole punch in %dms...\n", waitDuration.Milliseconds())
		time.Sleep(waitDuration)
	}

	fmt.Printf("[p2p] attempting hole punch to %s\n", peerAddress)

	// attempt connection — both peers do this simultaneously
	conn, err := net.DialTimeout("tcp", peerAddress, 5*time.Second)
	if err != nil {
		return nil, fmt.Errorf("hole punch failed: %w", err)
	}

	pc, err := handshake(conn, id)
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("hole punch handshake failed: %w", err)
	}

	fmt.Printf("[p2p] hole punch succeeded to %s\n", pc.RemotePeerID[:12])
	return pc, nil
}

// Connect attempts to reach a peer using the following strategy:
//  1. Direct TCP connection (works if peer has public IP)
//  2. Hole punching (works for most home NAT)
//  3. Returns error → caller falls back to relay
func Connect(peerAddress string, id *identity.Identity) (*PeerConn, error) {
	// try direct first
	pc, err := Dial(peerAddress, id)
	if err == nil {
		return pc, nil
	}

	fmt.Printf("[p2p] direct failed, attempting hole punch\n")

	// hole punch — coordinate via tracker in production
	// for now attempt with 500ms delay as coordination
	connectAt := time.Now().Add(500 * time.Millisecond)
	pc, err = HolePunch(peerAddress, connectAt, id)
	if err != nil {
		return nil, fmt.Errorf("all direct connection methods failed: %w", err)
	}

	return pc, nil
}
