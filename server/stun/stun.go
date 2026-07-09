package stun

import (
	"bufio"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/zenmakek/parcel/shared/protocol"
)

const STUNPort = "3478"

// Server listens for STUN requests and responds with
// the caller's observed public IP and port.
type Server struct {
	listener net.Listener
}

func New() *Server {
	return &Server{}
}

func (s *Server) Start() error {
	address := fmt.Sprintf("0.0.0.0:%s", STUNPort)
	ln, err := net.Listen("tcp", address)
	if err != nil {
		return fmt.Errorf("failed to start STUN server: %w", err)
	}
	s.listener = ln
	fmt.Printf("[stun] started on %s\n", address)

	for {
		conn, err := ln.Accept()
		if err != nil {
			fmt.Printf("[stun] accept error: %v\n", err)
			continue
		}
		go s.handle(conn)
	}
}

func (s *Server) handle(conn net.Conn) {
	defer conn.Close()
	conn.SetDeadline(time.Now().Add(10 * time.Second))
	reader := bufio.NewReader(conn)

	line, err := reader.ReadString('\n')
	if err != nil {
		return
	}

	packet, err := protocol.Decode(strings.TrimSpace(line))
	if err != nil || packet.Type != protocol.PacketSTUNRequest {
		return
	}

	// The observed address is what the OS reports as the remote address.
	// For a peer behind NAT this is their public IP:port.
	observedAddr := conn.RemoteAddr().String()

	resp, err := protocol.Encode(protocol.PacketSTUNResponse, protocol.STUNResponsePayload{
		ObservedAddress: observedAddr,
	})
	if err != nil {
		return
	}

	fmt.Fprint(conn, resp)
	fmt.Printf("[stun] %s observed as %s\n", conn.RemoteAddr(), observedAddr)
}