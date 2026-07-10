package dht

import (
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/zenmakek/parcel/shared/protocol"
)

// BootstrapNode is a well-known node used to join the DHT network.
// Your DigitalOcean/Oracle server acts as the bootstrap node.
var DefaultBootstrapNodes = []string{
	"139.59.60.82:9090",
}

// Bootstrap contacts known bootstrap nodes and populates
// the routing table with their peers.
func (d *DHT) Bootstrap(bootstrapAddrs []string) error {
	if len(bootstrapAddrs) == 0 {
		bootstrapAddrs = DefaultBootstrapNodes
	}

	joined := 0
	for _, addr := range bootstrapAddrs {
		if err := d.pingNode(addr); err != nil {
			fmt.Printf("[dht] bootstrap node unreachable: %s (%v)\n", addr, err)
			continue
		}
		joined++
		fmt.Printf("[dht] bootstrapped via %s\n", addr)
	}

	if joined == 0 {
		return fmt.Errorf("could not reach any bootstrap nodes")
	}

	fmt.Printf("[dht] routing table has %d nodes\n", d.table.Size())
	return nil
}

// pingNode sends a HELLO to a node and adds it to the routing table
// if it responds.
func (d *DHT) pingNode(address string) error {
	conn, err := net.DialTimeout("tcp", address, 5*time.Second)
	if err != nil {
		return fmt.Errorf("dial failed: %w", err)
	}
	defer conn.Close()
	conn.SetDeadline(time.Now().Add(5 * time.Second))

	ping, err := protocol.Encode(protocol.PacketHello, protocol.HelloPayload{
		Version: "2.0.0",
		Role:    d.self.Address,
	})
	if err != nil {
		return err
	}

	fmt.Fprint(conn, ping)

	buf := make([]byte, 512)
	n, err := conn.Read(buf)
	if err != nil {
		return fmt.Errorf("no response: %w", err)
	}

	packet, err := protocol.Decode(strings.TrimSpace(string(buf[:n])))
	if err != nil {
		return fmt.Errorf("bad response: %w", err)
	}

	if packet.Type != protocol.PacketAck {
		return fmt.Errorf("expected ACK got %s", packet.Type)
	}

	var ack protocol.AckPayload
	if err := protocol.DecodePayload(packet.Payload, &ack); err != nil {
		return err
	}

	// add bootstrap node to routing table
	nodeID, err := nodeIDFromHex(ack.Message)
	if err == nil {
		d.table.Add(&Node{
			ID:       nodeID,
			Address:  address,
			LastSeen: time.Now(),
		})
	}

	return nil
}

// Serve starts a simple DHT responder on the given port.
// Responds to LOOKUP requests with known peers for a file hash.
func (d *DHT) Serve(port string) error {
	ln, err := net.Listen("tcp", ":"+port)
	if err != nil {
		return fmt.Errorf("failed to listen: %w", err)
	}
	defer ln.Close()

	fmt.Printf("[dht] serving on :%s\n", port)

	for {
		conn, err := ln.Accept()
		if err != nil {
			return err
		}
		go d.handleConn(conn)
	}
}

func (d *DHT) handleConn(conn net.Conn) {
	defer conn.Close()
	conn.SetDeadline(time.Now().Add(10 * time.Second))

	buf := make([]byte, 4096)
	n, err := conn.Read(buf)
	if err != nil {
		return
	}

	packet, err := protocol.Decode(strings.TrimSpace(string(buf[:n])))
	if err != nil {
		return
	}

	switch packet.Type {
	case protocol.PacketLookup:
		var payload protocol.LookupPayload
		if err := protocol.DecodePayload(packet.Payload, &payload); err != nil {
			return
		}

		d.mu.RLock()
		addrs := d.announces[payload.FileHash]
		d.mu.RUnlock()

		peers := make([]protocol.PeerInfo, len(addrs))
		for i, addr := range addrs {
			peers[i] = protocol.PeerInfo{Address: addr}
		}

		resp, _ := protocol.Encode(protocol.PacketPeerList, protocol.PeerListPayload{
			FileHash: payload.FileHash,
			Peers:    peers,
		})
		fmt.Fprint(conn, resp)

	case protocol.PacketHello:
		// ping response — send our node ID
		ack, _ := protocol.Encode(protocol.PacketAck, protocol.AckPayload{
			Message: fmt.Sprintf("%x", d.self.ID),
		})
		fmt.Fprint(conn, ack)
	}
}
