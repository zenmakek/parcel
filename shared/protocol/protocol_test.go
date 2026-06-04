package protocol

import (
	"testing"
)

func TestEncodeDecodeHello(t *testing.T) {
	payload := HelloPayload{
		Version: "1.0.0",
		Role:    "sender",
	}

	encoded, err := Encode(PacketHello, payload)
	if err != nil {
		t.Fatalf("Encode failed: %v", err)
	}

	t.Logf("Encoded: %s", encoded)

	packet, err := Decode(encoded)
	if err != nil {
		t.Fatalf("Decode failed: %v", err)
	}

	if packet.Type != PacketHello {
		t.Errorf("expected type %s, got %s", PacketHello, packet.Type)
	}

	var decoded HelloPayload
	if err := DecodePayload(packet.Payload, &decoded); err != nil {
		t.Fatalf("DecodePayload failed: %v", err)
	}

	if decoded.Role != "sender" {
		t.Errorf("expected role sender, got %s", decoded.Role)
	}

	t.Logf("Decoded role: %s, version: %s", decoded.Role, decoded.Version)
}

func TestEncodeDecodeTransferInit(t *testing.T) {
	payload := TransferInitPayload{
		OTP:       "483921",
		Filename:  "photos.tar.gz",
		Size:      10485760,
		IsArchive: true,
	}

	encoded, err := Encode(PacketTransferInit, payload)
	if err != nil {
		t.Fatalf("Encode failed: %v", err)
	}

	t.Logf("Encoded: %s", encoded)

	packet, err := Decode(encoded)
	if err != nil {
		t.Fatalf("Decode failed: %v", err)
	}

	var decoded TransferInitPayload
	if err := DecodePayload(packet.Payload, &decoded); err != nil {
		t.Fatalf("DecodePayload failed: %v", err)
	}

	if decoded.OTP != "483921" {
		t.Errorf("expected OTP 483921, got %s", decoded.OTP)
	}

	if decoded.Size != 10485760 {
		t.Errorf("expected size 10485760, got %d", decoded.Size)
	}

	t.Logf("Decoded OTP: %s, filename: %s, size: %d", decoded.OTP, decoded.Filename, decoded.Size)
}
