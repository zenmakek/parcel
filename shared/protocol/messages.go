package protocol

import (
	"encoding/json"
	"fmt"
)

type Packet struct {
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload,omitempty"`
}

type HelloPayload struct {
	Version string `json:"version"`
	Role    string `json:"role"`
}

type AckPayload struct {
	Message string `json:"message"`
}

type TransferInitPayload struct {
	OTP       string `json:"otp"`
	Filename  string `json:"filename"`
	Size      int64  `json:"size"`
	IsArchive bool   `json:"isArchive"`
}

type TransferReadyPayload struct {
	OTP string `json:"otp"`
}

type TransferBeginPayload struct {
	OTP string `json:"otp"`
}

type TransferCompletePayload struct {
	OTP      string `json:"otp"`
	Filename string `json:"filename"`
	Size     int64  `json:"size"`
}

type TransferErrorPayload struct {
	OTP     string `json:"otp"`
	Message string `json:"message"`
}

type OTPJoinPayload struct {
	OTP string `json:"otp"`
}

type OTPExpiredPayload struct {
	OTP string `json:"otp"`
}

type OTPInvalidPayload struct {
	OTP string `json:"otp"`
}

func Encode(packetType string, payload any) (string, error) {
	raw, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("failed to marshal payload: %w", err)
	}

	packet := Packet{
		Type:    packetType,
		Payload: json.RawMessage(raw),
	}

	data, err := json.Marshal(packet)
	if err != nil {
		return "", fmt.Errorf("failed to marshal packet: %w", err)
	}

	return string(data) + "\n", nil
}

func Decode(raw string) (*Packet, error) {
	var packet Packet
	if err := json.Unmarshal([]byte(raw), &packet); err != nil {
		return nil, fmt.Errorf("failed to unmarshal packet: %w", err)
	}
	return &packet, nil
}

func DecodePayload(raw json.RawMessage, target any) error {
	if err := json.Unmarshal(raw, target); err != nil {
		return fmt.Errorf("failed to unmarshal payload: %w", err)
	}
	return nil
}
