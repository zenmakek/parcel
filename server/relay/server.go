package relay

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"os"
	"strings"
	"time"

	"github.com/zenmakek/parcel/server/otp"
	"github.com/zenmakek/parcel/server/session"
	"github.com/zenmakek/parcel/shared/protocol"
)

func getEnv(key string, fallback string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return fallback
}

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
	host := getEnv("PARCEL_HOST", "0.0.0.0")
	port := getEnv("PARCEL_PORT", "8080")
	address := fmt.Sprintf("%s:%s", host, port)

	listener, err := net.Listen("tcp", address)
	if err != nil {
		return fmt.Errorf("failed to start relay server: %w", err)
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

	helloRaw, err := reader.ReadString('\n')
	if err != nil {
		fmt.Printf("[relay] failed to read hello from %s: %v\n", conn.RemoteAddr(), err)
		return
	}

	helloPacket, err := protocol.Decode(strings.TrimSpace(helloRaw))
	if err != nil {
		fmt.Printf("[relay] malformed hello from %s: %v\n", conn.RemoteAddr(), err)
		sendError(conn, "", "malformed hello packet")
		return
	}

	if helloPacket.Type != protocol.PacketHello {
		fmt.Printf("[relay] expected HELLO got %s from %s\n", helloPacket.Type, conn.RemoteAddr())
		sendError(conn, "", "expected HELLO packet")
		return
	}

	var helloPayload protocol.HelloPayload
	if err := protocol.DecodePayload(helloPacket.Payload, &helloPayload); err != nil {
		fmt.Printf("[relay] failed to decode hello payload from %s: %v\n", conn.RemoteAddr(), err)
		sendError(conn, "", "malformed hello payload")
		return
	}

	fmt.Printf("[relay] client identified as: %s from %s\n", helloPayload.Role, conn.RemoteAddr())

	switch helloPayload.Role {
	case "sender":
		s.handleSender(conn, reader)
	case "receiver":
		s.handleReceiver(conn, reader)
	default:
		fmt.Printf("[relay] unknown role: %s from %s\n", helloPayload.Role, conn.RemoteAddr())
		sendError(conn, "", "unknown role")
	}
}

func (s *Server) handleSender(conn net.Conn, reader *bufio.Reader) {
	generatedOTP, err := otp.Generate()
	if err != nil {
		fmt.Printf("[relay] failed to generate OTP: %v\n", err)
		sendError(conn, "", "server failed to generate OTP")
		return
	}

	sess, err := s.registry.Create(generatedOTP, conn)
	if err != nil {
		fmt.Printf("[relay] failed to create session: %v\n", err)
		sendError(conn, "", "server failed to create session")
		return
	}

	ack, err := protocol.Encode(protocol.PacketAck, protocol.AckPayload{
		Message: generatedOTP,
	})
	if err != nil {
		fmt.Printf("[relay] failed to encode OTP ack: %v\n", err)
		s.registry.Destroy(generatedOTP)
		return
	}

	if _, err := fmt.Fprint(conn, ack); err != nil {
		fmt.Printf("[relay] failed to send OTP ack: %v\n", err)
		s.registry.Destroy(generatedOTP)
		return
	}

	fmt.Printf("[relay] OTP issued: %s\n", generatedOTP)

	initRaw, err := reader.ReadString('\n')
	if err != nil {
		fmt.Printf("[relay] sender disconnected before transfer init: %v\n", err)
		s.registry.Destroy(generatedOTP)
		return
	}

	initPacket, err := protocol.Decode(strings.TrimSpace(initRaw))
	if err != nil {
		fmt.Printf("[relay] malformed transfer init: %v\n", err)
		s.registry.Destroy(generatedOTP)
		return
	}

	if initPacket.Type != protocol.PacketTransferInit {
		fmt.Printf("[relay] expected TRANSFER_INIT got %s\n", initPacket.Type)
		s.registry.Destroy(generatedOTP)
		return
	}

	var initPayload protocol.TransferInitPayload
	if err := protocol.DecodePayload(initPacket.Payload, &initPayload); err != nil {
		fmt.Printf("[relay] failed to decode transfer init: %v\n", err)
		s.registry.Destroy(generatedOTP)
		return
	}

	sess.Filename = initPayload.Filename
	sess.Size = initPayload.Size
	sess.IsArchive = initPayload.IsArchive

	fmt.Printf("[relay] transfer init: %s (%d bytes)\n", initPayload.Filename, initPayload.Size)

	for {
		if sess.Status == session.StatusConnected {
			break
		}
		if time.Now().After(sess.ExpiresAt) {
			fmt.Printf("[relay] OTP expired waiting for receiver: %s\n", generatedOTP)
			expired, _ := protocol.Encode(protocol.PacketOTPExpired, protocol.OTPExpiredPayload{
				OTP: generatedOTP,
			})
			fmt.Fprint(conn, expired)
			s.registry.Destroy(generatedOTP)
			return
		}
		time.Sleep(500 * time.Millisecond)
	}

	ready, err := protocol.Encode(protocol.PacketTransferReady, protocol.TransferReadyPayload{
		OTP: generatedOTP,
	})
	if err != nil {
		fmt.Printf("[relay] failed to encode transfer ready: %v\n", err)
		s.registry.Destroy(generatedOTP)
		return
	}

	if _, err := fmt.Fprint(conn, ready); err != nil {
		fmt.Printf("[relay] failed to send transfer ready: %v\n", err)
		s.registry.Destroy(generatedOTP)
		return
	}

	sess.Status = session.StatusTransferring
	fmt.Printf("[relay] piping %d bytes\n", initPayload.Size)

	conn.SetDeadline(time.Time{})

	written, err := io.CopyN(sess.ReceiverConn, conn, initPayload.Size)
	if err != nil && err != io.EOF {
		fmt.Printf("[relay] pipe error after %d bytes: %v\n", written, err)
		sendError(sess.ReceiverConn, generatedOTP, "transfer interrupted by sender disconnect")
		s.registry.Destroy(generatedOTP)
		return
	}

	fmt.Printf("[relay] transfer complete: %d bytes piped\n", written)
	sess.Status = session.StatusDone
	close(sess.Done)
	s.registry.Destroy(generatedOTP)
}

func (s *Server) handleReceiver(conn net.Conn, reader *bufio.Reader) {
	joinRaw, err := reader.ReadString('\n')
	if err != nil {
		fmt.Printf("[relay] failed to read OTP join: %v\n", err)
		return
	}

	joinPacket, err := protocol.Decode(strings.TrimSpace(joinRaw))
	if err != nil {
		fmt.Printf("[relay] malformed OTP join: %v\n", err)
		sendError(conn, "", "malformed OTP join packet")
		return
	}

	if joinPacket.Type != protocol.PacketOTPJoin {
		fmt.Printf("[relay] expected OTP_JOIN got %s\n", joinPacket.Type)
		sendError(conn, "", "expected OTP_JOIN packet")
		return
	}

	var joinPayload protocol.OTPJoinPayload
	if err := protocol.DecodePayload(joinPacket.Payload, &joinPayload); err != nil {
		fmt.Printf("[relay] failed to decode OTP join payload: %v\n", err)
		sendError(conn, "", "malformed OTP join payload")
		return
	}

	fmt.Printf("[relay] receiver attempting to join: %s\n", joinPayload.OTP)

	sess, err := s.registry.JoinReceiver(joinPayload.OTP, conn)
	if err != nil {
		fmt.Printf("[relay] join failed: %v\n", err)
		invalid, _ := protocol.Encode(protocol.PacketOTPInvalid, protocol.OTPInvalidPayload{
			OTP: joinPayload.OTP,
		})
		fmt.Fprint(conn, invalid)
		return
	}

	transferInit, err := protocol.Encode(protocol.PacketTransferInit, protocol.TransferInitPayload{
		OTP:       joinPayload.OTP,
		Filename:  sess.Filename,
		Size:      sess.Size,
		IsArchive: sess.IsArchive,
	})
	if err != nil {
		fmt.Printf("[relay] failed to encode transfer init for receiver: %v\n", err)
		s.registry.Destroy(joinPayload.OTP)
		return
	}

	if _, err := fmt.Fprint(conn, transferInit); err != nil {
		fmt.Printf("[relay] failed to send transfer init to receiver: %v\n", err)
		s.registry.Destroy(joinPayload.OTP)
		return
	}

	conn.SetDeadline(time.Time{})

	fmt.Printf("[relay] receiver ready, pipe opening\n")
	<-sess.Done
	fmt.Printf("[relay] receiver goroutine exiting cleanly\n")
}

func sendError(conn net.Conn, otp string, message string) {
	errPacket, err := protocol.Encode(protocol.PacketTransferError, protocol.TransferErrorPayload{
		OTP:     otp,
		Message: message,
	})
	if err != nil {
		return
	}
	fmt.Fprint(conn, errPacket)
}
