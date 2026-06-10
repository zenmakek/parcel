package relay

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"strings"
	"time"

	"github.com/zenmakek/parcel/server/otp"
	"github.com/zenmakek/parcel/server/session"
	"github.com/zenmakek/parcel/shared/protocol"
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
		fmt.Printf("[relay] failed to read hello: %v\n", err)
		return
	}

	helloPacket, err := protocol.Decode(strings.TrimSpace(helloRaw))
	if err != nil {
		fmt.Printf("[relay] failed to decode hello: %v\n", err)
		return
	}

	if helloPacket.Type != protocol.PacketHello {
		fmt.Printf("[relay] unexpected packet, expected HELLO got %s\n", helloPacket.Type)
		return
	}

	var helloPayload protocol.HelloPayload
	if err := protocol.DecodePayload(helloPacket.Payload, &helloPayload); err != nil {
		fmt.Printf("[relay] failed to decode hello payload: %v\n", err)
		return
	}

	fmt.Printf("[relay] client identified as: %s\n", helloPayload.Role)

	switch helloPayload.Role {
	case "sender":
		s.handleSender(conn, reader)
	case "receiver":
		s.handleReceiver(conn, reader)
	default:
		fmt.Printf("[relay] unknown role: %s\n", helloPayload.Role)
	}
}

func (s *Server) handleSender(conn net.Conn, reader *bufio.Reader) {
	generatedOTP, err := otp.Generate()
	if err != nil {
		fmt.Printf("[relay] failed to generate OTP: %v\n", err)
		return
	}

	sess, err := s.registry.Create(generatedOTP, conn)
	if err != nil {
		fmt.Printf("[relay] failed to create session: %v\n", err)
		return
	}

	ack, err := protocol.Encode(protocol.PacketAck, protocol.AckPayload{
		Message: generatedOTP,
	})
	if err != nil {
		fmt.Printf("[relay] failed to encode OTP ack: %v\n", err)
		return
	}

	if _, err := fmt.Fprint(conn, ack); err != nil {
		fmt.Printf("[relay] failed to send OTP ack: %v\n", err)
		return
	}

	fmt.Printf("[relay] OTP issued: %s, waiting for transfer init\n", generatedOTP)

	initRaw, err := reader.ReadString('\n')
	if err != nil {
		fmt.Printf("[relay] failed to read transfer init: %v\n", err)
		s.registry.Destroy(generatedOTP)
		return
	}

	initPacket, err := protocol.Decode(strings.TrimSpace(initRaw))
	if err != nil {
		fmt.Printf("[relay] failed to decode transfer init: %v\n", err)
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
		fmt.Printf("[relay] failed to decode transfer init payload: %v\n", err)
		s.registry.Destroy(generatedOTP)
		return
	}

	sess.Filename = initPayload.Filename
	sess.Size = initPayload.Size
	sess.IsArchive = initPayload.IsArchive

	fmt.Printf("[relay] transfer init received: %s (%d bytes)\n", initPayload.Filename, initPayload.Size)
	fmt.Printf("[relay] waiting for receiver to join session: %s\n", generatedOTP)

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
	fmt.Printf("[relay] piping %d bytes from sender to receiver\n", initPayload.Size)

	written, err := io.CopyN(sess.ReceiverConn, conn, initPayload.Size)
	if err != nil && err != io.EOF {
		fmt.Printf("[relay] pipe error: %v\n", err)
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
		fmt.Printf("[relay] failed to decode OTP join: %v\n", err)
		return
	}

	if joinPacket.Type != protocol.PacketOTPJoin {
		fmt.Printf("[relay] expected OTP_JOIN got %s\n", joinPacket.Type)
		return
	}

	var joinPayload protocol.OTPJoinPayload
	if err := protocol.DecodePayload(joinPacket.Payload, &joinPayload); err != nil {
		fmt.Printf("[relay] failed to decode OTP join payload: %v\n", err)
		return
	}

	fmt.Printf("[relay] receiver attempting to join session: %s\n", joinPayload.OTP)

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

	fmt.Printf("[relay] receiver is ready, waiting for pipe to open\n")
	fmt.Printf("[relay] receiver is ready, waiting for pipe to open\n")

	<-sess.Done

	fmt.Printf("[relay] receiver goroutine exiting cleanly\n")
}
