package stun

import (
	"fmt"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/zenmakek/parcel/shared/protocol"
)

func TestSTUNObservesAddress(t *testing.T) {
	// start STUN on a random port
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to listen: %v", err)
	}

	s := &Server{listener: ln}
	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			go s.handle(conn)
		}
	}()
	defer ln.Close()

	addr := ln.Addr().String()
	time.Sleep(20 * time.Millisecond)

	conn, err := net.Dial("tcp", addr)
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer conn.Close()

	req, _ := protocol.Encode(protocol.PacketSTUNRequest, protocol.STUNRequestPayload{
		PeerID: "test-peer",
	})
	fmt.Fprint(conn, req)

	conn.SetDeadline(time.Now().Add(3 * time.Second))
	buf := make([]byte, 512)
	n, err := conn.Read(buf)
	if err != nil {
		t.Fatalf("failed to read response: %v", err)
	}

	packet, err := protocol.Decode(strings.TrimSpace(string(buf[:n])))
	if err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if packet.Type != protocol.PacketSTUNResponse {
		t.Fatalf("expected STUN_RESPONSE got %s", packet.Type)
	}

	var resp protocol.STUNResponsePayload
	if err := protocol.DecodePayload(packet.Payload, &resp); err != nil {
		t.Fatalf("failed to decode payload: %v", err)
	}

	if resp.ObservedAddress == "" {
		t.Fatal("expected non-empty observed address")
	}

	if !strings.Contains(resp.ObservedAddress, "127.0.0.1") {
		t.Errorf("expected loopback address, got %s", resp.ObservedAddress)
	}

	t.Logf("observed address: %s", resp.ObservedAddress)
}