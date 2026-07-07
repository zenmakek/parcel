package tracker

import (
	"bufio"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/zenmakek/parcel/shared/protocol"
)

const TrackerPort = "9090"

type Tracker struct {
	listener  net.Listener
	peerstore *PeerStore
}

func New() *Tracker {
	return &Tracker{
		peerstore: NewPeerStore(),
	}
}

func (t *Tracker) Start() error {
	host := "0.0.0.0"
	address := fmt.Sprintf("%s:%s", host, TrackerPort)

	listener, err := net.Listen("tcp", address)
	if err != nil {
		return fmt.Errorf("failed to start tracker: %w", err)
	}
	t.listener = listener
	fmt.Printf("[tracker] started on %s\n", address)

	for {
		conn, err := t.listener.Accept()
		if err != nil {
			fmt.Printf("[tracker] accept error: %v\n", err)
			continue
		}
		go t.handleConn(conn)
	}
}

func (t *Tracker) handleConn(conn net.Conn) {
	defer conn.Close()
	conn.SetDeadline(time.Now().Add(30 * time.Second))
	reader := bufio.NewReader(conn)

	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			return
		}

		packet, err := protocol.Decode(strings.TrimSpace(line))
		if err != nil {
			fmt.Printf("[tracker] malformed packet: %v\n", err)
			return
		}

		switch packet.Type {
		case protocol.PacketAnnounce:
			t.handleAnnounce(conn, packet)
		case protocol.PacketLookup:
			t.handleLookup(conn, packet)
		default:
			fmt.Printf("[tracker] unknown packet type: %s\n", packet.Type)
			return
		}
	}
}

func (t *Tracker) handleAnnounce(conn net.Conn, packet *protocol.Packet) {
	var payload protocol.AnnouncePayload
	if err := protocol.DecodePayload(packet.Payload, &payload); err != nil {
		fmt.Printf("[tracker] bad announce payload: %v\n", err)
		return
	}

	t.peerstore.Announce(&PeerEntry{
		PeerID:   payload.PeerID,
		Address:  payload.Address,
		FileHash: payload.FileHash,
		Filename: payload.Filename,
		Size:     payload.Size,
	})

	fmt.Printf("[tracker] announce: peer=%s hash=%s\n", payload.PeerID[:8], payload.FileHash[:12])

	ack, _ := protocol.Encode(protocol.PacketAck, protocol.AckPayload{
		Message: "announced",
	})
	fmt.Fprint(conn, ack)
}

func (t *Tracker) handleLookup(conn net.Conn, packet *protocol.Packet) {
	var payload protocol.LookupPayload
	if err := protocol.DecodePayload(packet.Payload, &payload); err != nil {
		fmt.Printf("[tracker] bad lookup payload: %v\n", err)
		return
	}

	entries := t.peerstore.Lookup(payload.FileHash)
	peers := make([]protocol.PeerInfo, len(entries))
	for i, e := range entries {
		peers[i] = protocol.PeerInfo{
			PeerID:  e.PeerID,
			Address: e.Address,
		}
	}

	fmt.Printf("[tracker] lookup: hash=%s found=%d peers\n", payload.FileHash[:12], len(peers))

	resp, _ := protocol.Encode(protocol.PacketPeerList, protocol.PeerListPayload{
		FileHash: payload.FileHash,
		Peers:    peers,
	})
	fmt.Fprint(conn, resp)
}