package transfer

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"os"

	"github.com/zenmakek/parcel/client/utils"
	"github.com/zenmakek/parcel/shared/protocol"
)

func SendFile(conn net.Conn, meta *Metadata) error {
	reader := bufio.NewReader(conn)

	hello, err := protocol.Encode(protocol.PacketHello, protocol.HelloPayload{
		Version: "1.0.0",
		Role:    "sender",
	})
	if err != nil {
		return fmt.Errorf("failed to encode hello: %w", err)
	}

	if _, err := fmt.Fprint(conn, hello); err != nil {
		return fmt.Errorf("failed to send hello: %w", err)
	}

	ackRaw, err := reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("failed to read OTP ack: %w", err)
	}

	ackPacket, err := protocol.Decode(ackRaw)
	if err != nil {
		return fmt.Errorf("failed to decode OTP ack: %w", err)
	}

	var ackPayload protocol.AckPayload
	if err := protocol.DecodePayload(ackPacket.Payload, &ackPayload); err != nil {
		return fmt.Errorf("failed to decode ack payload: %w", err)
	}

	otp := ackPayload.Message

	fmt.Println()
	fmt.Println("  ┌─────────────────────────────┐")
	fmt.Println("  │                             │")
	fmt.Printf("  │   Your OTP:   %s        │\n", otp)
	fmt.Println("  │                             │")
	fmt.Println("  └─────────────────────────────┘")
	fmt.Println()
	fmt.Println("  Share this code with the receiver.")
	fmt.Println("  Waiting for them to connect...")

	transferInit, err := protocol.Encode(protocol.PacketTransferInit, protocol.TransferInitPayload{
		OTP:       otp,
		Filename:  meta.Filename,
		Size:      meta.Size,
		IsArchive: meta.IsArchive,
	})
	if err != nil {
		return fmt.Errorf("failed to encode transfer init: %w", err)
	}

	if _, err := fmt.Fprint(conn, transferInit); err != nil {
		return fmt.Errorf("failed to send transfer init: %w", err)
	}

	readyRaw, err := reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("failed to read transfer ready: %w", err)
	}

	readyPacket, err := protocol.Decode(readyRaw)
	if err != nil {
		return fmt.Errorf("failed to decode transfer ready: %w", err)
	}

	if readyPacket.Type == protocol.PacketOTPExpired {
		return fmt.Errorf("OTP expired before receiver joined")
	}

	if readyPacket.Type != protocol.PacketTransferReady {
		return fmt.Errorf("unexpected packet type: %s", readyPacket.Type)
	}

	fmt.Println("  Receiver connected. Transferring...")
	fmt.Println()

	file, err := os.Open(meta.OriginalPath)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	progress := utils.NewProgressReader(file, meta.Size, "Sending")

	written, err := io.Copy(conn, progress)
	if err != nil {
		return fmt.Errorf("failed to stream file: %w", err)
	}

	fmt.Printf("  Sent %d bytes.\n", written)
	return nil
}
