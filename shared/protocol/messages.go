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

// AnnouncePayload is sent by a peer to the tracker
// to declare that it has a file available for download.
type AnnouncePayload struct {
	FileHash string `json:"file_hash"`
	PeerID   string `json:"peer_id"`
	Address  string `json:"address"`
	Filename string `json:"filename"`
	Size     int64  `json:"size"`
}

// LookupPayload is sent by a peer to the tracker
// to find which peers have a given file hash.
type LookupPayload struct {
	FileHash string `json:"file_hash"`
}

// PeerInfo describes a single peer that has a file.
type PeerInfo struct {
	PeerID  string `json:"peer_id"`
	Address string `json:"address"`
}

// PeerListPayload is the tracker's response to a LOOKUP.
// Contains all known peers that have announced the requested file hash.
type PeerListPayload struct {
	FileHash string     `json:"file_hash"`
	Peers    []PeerInfo `json:"peers"`
}

// ManifestRequestPayload is sent by a receiver to a seeder
// to request the manifest for a given file hash.
type ManifestRequestPayload struct {
	FileHash string `json:"file_hash"`
}

// ManifestResponsePayload is the seeder's response.
// Contains everything the receiver needs to know to fetch all chunks.
type ManifestResponsePayload struct {
	FileHash    string   `json:"file_hash"`
	Filename    string   `json:"filename"`
	TotalSize   int64    `json:"total_size"`
	ChunkSize   int      `json:"chunk_size"`
	ChunkCount  int      `json:"chunk_count"`
	ChunkHashes []string `json:"chunk_hashes"`
	IsArchive   bool     `json:"is_archive"`
}

// ChunkRequestPayload is sent by a receiver to a seeder
// to request a specific chunk by file hash and chunk index.
type ChunkRequestPayload struct {
	FileHash   string `json:"file_hash"`
	ChunkIndex int    `json:"chunk_index"`
}

// ChunkResponsePayload carries the actual chunk data.
// The receiver verifies Data against Hash before storing.
type ChunkResponsePayload struct {
	FileHash   string `json:"file_hash"`
	ChunkIndex int    `json:"chunk_index"`
	Hash       string `json:"hash"`
	Data       []byte `json:"data"`
}

// ChunkVerifyFailPayload is sent by the receiver when a chunk
// fails hash verification — asking the seeder to resend it.
type ChunkVerifyFailPayload struct {
	FileHash   string `json:"file_hash"`
	ChunkIndex int    `json:"chunk_index"`
	Expected   string `json:"expected"`
	Got        string `json:"got"`
}

// HavePayload is sent by a peer to announce it has received
// a specific chunk — used in multi-peer swarms so peers
// can update their view of who has what.
type HavePayload struct {
	FileHash   string `json:"file_hash"`
	ChunkIndex int    `json:"chunk_index"`
}

// BitfieldPayload is sent at connection start to declare
// which chunks a peer already has. More efficient than
// sending individual HAVE packets for each chunk.
type BitfieldPayload struct {
	FileHash string `json:"file_hash"`
	Bitfield []bool `json:"bitfield"`
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
