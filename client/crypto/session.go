package crypto

import (
	"encoding/binary"
	"fmt"
	"io"
	"net"

	"github.com/flynn/noise"
)

// Session wraps a net.Conn with Noise encryption.
// All reads and writes are transparently encrypted/decrypted.
type Session struct {
	conn net.Conn
	send *noise.CipherState
	recv *noise.CipherState
}

func newSession(conn net.Conn, send, recv *noise.CipherState) *Session {
	return &Session{conn: conn, send: send, recv: recv}
}

// Write encrypts data and writes it to the connection.
// Frames: 2-byte length prefix + encrypted payload.
func (s *Session) Write(data []byte) (int, error) {
	encrypted, err := s.send.Encrypt(nil, nil, data)
	if err != nil {
		return 0, fmt.Errorf("encrypt failed: %w", err)
	}

	if err := writeFrame(s.conn, encrypted); err != nil {
		return 0, err
	}

	return len(data), nil
}

// Read decrypts the next frame from the connection.
func (s *Session) Read(buf []byte) (int, error) {
	frame, err := readFrame(s.conn)
	if err != nil {
		return 0, err
	}

	decrypted, err := s.recv.Decrypt(nil, nil, frame)
	if err != nil {
		return 0, fmt.Errorf("decrypt failed: %w", err)
	}

	n := copy(buf, decrypted)
	return n, nil
}

// Close closes the underlying connection.
func (s *Session) Close() error {
	return s.conn.Close()
}

// RemoteAddr returns the remote address of the connection.
func (s *Session) RemoteAddr() net.Addr {
	return s.conn.RemoteAddr()
}

// writeFrame writes a length-prefixed frame to conn.
func writeFrame(conn net.Conn, data []byte) error {
	length := make([]byte, 2)
	binary.BigEndian.PutUint16(length, uint16(len(data)))

	if _, err := conn.Write(length); err != nil {
		return fmt.Errorf("failed to write frame length: %w", err)
	}
	if _, err := conn.Write(data); err != nil {
		return fmt.Errorf("failed to write frame data: %w", err)
	}
	return nil
}

// readFrame reads a length-prefixed frame from conn.
func readFrame(conn net.Conn) ([]byte, error) {
	length := make([]byte, 2)
	if _, err := io.ReadFull(conn, length); err != nil {
		return nil, fmt.Errorf("failed to read frame length: %w", err)
	}

	size := binary.BigEndian.Uint16(length)
	data := make([]byte, size)
	if _, err := io.ReadFull(conn, data); err != nil {
		return nil, fmt.Errorf("failed to read frame data: %w", err)
	}

	return data, nil
}