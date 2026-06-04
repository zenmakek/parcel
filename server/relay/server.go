package relay

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"strings"
	"time"

	"github.com/zenmakek/parcel/server/session"
)

const (
	Host = "0.0.0.0"
	Port = "8080"
)

type Server struct {
	listener net.Listener
	registry *session.Registry
}

func New() *Server {
	return &Server{
		registry: session.NewRegistry(),
	}
}

func (s *Server) Start() error {
	address := fmt.Sprintf("%s:%s", Host, Port)
	listener, err := net.Listen("tcp", address)
	if err != nil {
		return fmt.Errorf("failed tos tart relay server: %w", err)
	}
	s.listener = listener
	fmt.Printf("[relay] server started on %s\n", address)
	s.acceptLoop()
	return nil
}

func (s *Server) acceptLoop() {
	for {
		conn, err := s.listener.Accept()
		if err != nil {
			fmt.Printf("[relay] error accepting connection: %v\n", err)
			continue
		}
		fmt.Printf("[relay] new connection from %s\n", conn.RemoteAddr())
		go s.handleConnection(conn)
	}
}

func (s *Server) handleConnection(conn net.Conn) {
	defer func() {
		conn.Close()
		fmt.Printf("[relay] connection closed: %s\n", conn.RemoteAddr())
	}()

	conn.SetDeadline(time.Now().Add(5 * time.Minute))
	reader := bufio.NewReader(conn)

	for {
		message, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				fmt.Printf("[relay] client disconnected cleanly: %s\n", conn.RemoteAddr())
			} else {
				fmt.Printf("[relay] read error from %s: %v\n", conn.RemoteAddr(), err)
			}
			return
		}

		message = strings.TrimSpace(message)
		fmt.Printf("[relay] received from %s: %s\n", conn.RemoteAddr(), message)

		response := fmt.Sprintf("[relay-ack] received: %s\n", message)
		conn.Write([]byte(response))
	}
}
