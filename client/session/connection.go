package session

import (
	"bufio"
	"fmt"
	"net"
	"strings"
)

const RelayAddress = "localhost:8080"

type Connection struct {
	conn   net.Conn
	reader *bufio.Reader
}

func Dial(address string) (*Connection, error) {
	conn, err := net.Dial("tcp", address)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to relay: %w", err)
	}
	fmt.Println("  [parcel] connected to relay at", address)

	return &Connection{
		conn:   conn,
		reader: bufio.NewReader(conn),
	}, nil

}

func (c *Connection) Send(message string) error {
	_, err := fmt.Fprintf(c.conn, "%s\n", message)
	if err != nil {
		return fmt.Errorf("failed to send message: %w", err)
	}
	return nil
}

func (c *Connection) Receive() (string, error) {
	message, err := c.reader.ReadString('\n')
	if err != nil {
		return "", fmt.Errorf("failed to receive message: %w", err)
	}
	return strings.TrimSpace(message), nil
}

func (c *Connection) Close() {
	c.conn.Close()
	fmt.Println("  [parcel] disconnected from relay")
}
