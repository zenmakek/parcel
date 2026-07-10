package p2p

import (
	"os"
	"testing"
	"time"

	"github.com/zenmakek/parcel/client/identity"
	"github.com/zenmakek/parcel/shared/protocol"
)

func makeIdentity(t *testing.T) *identity.Identity {
	t.Helper()
	dir, err := os.MkdirTemp("", "parcel-p2p-identity-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	t.Cleanup(func() { os.RemoveAll(dir) })

	kp, err := identity.LoadFrom(dir)
	if err != nil {
		t.Fatalf("failed to load identity: %v", err)
	}
	return kp
}

func TestDirectConnectionAndHandshake(t *testing.T) {
	idA := makeIdentity(t)
	idB := makeIdentity(t)

	ln, err := NewListener("19001", idA)
	if err != nil {
		t.Fatalf("failed to create listener: %v", err)
	}
	defer ln.Close()

	errCh := make(chan error, 1)
	connCh := make(chan *PeerConn, 1)

	go func() {
		pc, err := ln.Accept()
		if err != nil {
			errCh <- err
			return
		}
		connCh <- pc
	}()

	time.Sleep(50 * time.Millisecond)

	dialedConn, err := Dial("localhost:19001", idB)
	if err != nil {
		t.Fatalf("Dial failed: %v", err)
	}
	defer dialedConn.Close()

	select {
	case err := <-errCh:
		t.Fatalf("listener error: %v", err)
	case acceptedConn := <-connCh:
		defer acceptedConn.Close()

		if acceptedConn.RemotePeerID != dialedConn.LocalIdentity.PeerID {
			t.Errorf("peer ID mismatch")
		}

		t.Logf("A accepted B: %s", acceptedConn.RemotePeerID[:12])
		t.Logf("B dialed A: %s", dialedConn.RemotePeerID[:12])
	case <-time.After(3 * time.Second):
		t.Fatal("timeout waiting for connection")
	}
}

func TestSendAndReceivePacket(t *testing.T) {
	idA := makeIdentity(t)
	idB := makeIdentity(t)

	ln, err := NewListener("19002", idA)
	if err != nil {
		t.Fatalf("failed to create listener: %v", err)
	}
	defer ln.Close()

	receivedCh := make(chan *protocol.Packet, 1)

	go func() {
		pc, err := ln.Accept()
		if err != nil {
			return
		}
		defer pc.Close()
		pkt, err := pc.Receive()
		if err != nil {
			return
		}
		receivedCh <- pkt
	}()

	time.Sleep(50 * time.Millisecond)

	dialedConn, err := Dial("localhost:19002", idB)
	if err != nil {
		t.Fatalf("Dial failed: %v", err)
	}
	defer dialedConn.Close()

	if err := dialedConn.Send(protocol.PacketLookup, protocol.LookupPayload{
		FileHash: "3a7bd3e2a7b9f1c2d3e4f5a6b7c8d9e0f1a2b3c4d5e6f7a8b9c0d1e2f3a4b5c6",
	}); err != nil {
		t.Fatalf("Send failed: %v", err)
	}

	select {
	case pkt := <-receivedCh:
		if pkt.Type != protocol.PacketLookup {
			t.Errorf("expected LOOKUP got %s", pkt.Type)
		}
		t.Logf("packet received: %s", pkt.Type)
	case <-time.After(3 * time.Second):
		t.Fatal("timeout waiting for packet")
	}
}
