package transfer

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"strings"

	"github.com/zenmakek/parcel/client/utils"
	"github.com/zenmakek/parcel/shared/protocol"
)

func ReceiveFile(conn net.Conn, otp string, downloadDir string) (string, bool, error) {
	reader := bufio.NewReader(conn)

	hello, err := protocol.Encode(protocol.PacketHello, protocol.HelloPayload{
		Version: "1.0.0",
		Role:    "receiver",
	})
	if err != nil {
		return "", false, fmt.Errorf("failed to encode hello: %w", err)
	}

	if _, err := fmt.Fprint(conn, hello); err != nil {
		return "", false, fmt.Errorf("failed to send hello: %w", err)
	}

	join, err := protocol.Encode(protocol.PacketOTPJoin, protocol.OTPJoinPayload{
		OTP: otp,
	})
	if err != nil {
		return "", false, fmt.Errorf("failed to encode OTP join: %w", err)
	}

	if _, err := fmt.Fprint(conn, join); err != nil {
		return "", false, fmt.Errorf("failed to send OTP join: %w", err)
	}

	metaRaw, err := reader.ReadString('\n')
	if err != nil {
		return "", false, fmt.Errorf("failed to read transfer init: %w", err)
	}

	metaPacket, err := protocol.Decode(metaRaw)
	if err != nil {
		return "", false, fmt.Errorf("failed to decode transfer init: %w", err)
	}

	switch metaPacket.Type {
	case protocol.PacketOTPInvalid:
		return "", false, fmt.Errorf("invalid OTP: %s", otp)
	case protocol.PacketOTPExpired:
		return "", false, fmt.Errorf("OTP expired: %s", otp)
	case protocol.PacketTransferError:
		var errPayload protocol.TransferErrorPayload
		protocol.DecodePayload(metaPacket.Payload, &errPayload)
		return "", false, fmt.Errorf("relay error: %s", errPayload.Message)
	case protocol.PacketTransferInit:
		// continue
	default:
		return "", false, fmt.Errorf("unexpected packet: %s", metaPacket.Type)
	}

	var initPayload protocol.TransferInitPayload
	if err := protocol.DecodePayload(metaPacket.Payload, &initPayload); err != nil {
		return "", false, fmt.Errorf("failed to decode transfer init payload: %w", err)
	}

	fmt.Printf("  Incoming: %s (%d bytes)\n", initPayload.Filename, initPayload.Size)

	destPath := resolveDestPath(downloadDir, initPayload.Filename)

	outFile, err := os.Create(destPath)
	if err != nil {
		return "", false, fmt.Errorf("failed to create destination file: %w", err)
	}
	defer outFile.Close()

	progress := utils.NewProgressReader(conn, initPayload.Size, "Receiving")

	received, err := io.CopyN(outFile, progress, initPayload.Size)
	if err != nil && err != io.EOF {
		return "", false, fmt.Errorf("failed to receive file: %w", err)
	}

	fmt.Printf("  Received %d bytes.\n", received)
	return destPath, initPayload.IsArchive, nil
}

func resolveDestPath(dir string, filename string) string {
	base := strings.TrimSuffix(filename, filepath.Ext(filename))
	ext := filepath.Ext(filename)

	if ext == ".gz" {
		base = strings.TrimSuffix(base, ".tar")
		ext = ".tar.gz"
	}

	dest := filepath.Join(dir, filename)
	if _, err := os.Stat(dest); os.IsNotExist(err) {
		return dest
	}

	for i := 1; ; i++ {
		candidate := filepath.Join(dir, fmt.Sprintf("%s(%d)%s", base, i, ext))
		if _, err := os.Stat(candidate); os.IsNotExist(err) {
			return candidate
		}
	}
}
