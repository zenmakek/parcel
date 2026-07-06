package protocol

import (
	"testing"
)

func TestEncodeDecodeAnnounce(t *testing.T) {
	payload := AnnouncePayload{
		FileHash: "3a7bd3e2a7b9f1c2d3e4f5a6b7c8d9e0f1a2b3c4d5e6f7a8b9c0d1e2f3a4b5c6",
		PeerID:   "peer-abc123",
		Address:  "192.168.1.1:9000",
		Filename: "video.mp4",
		Size:     104857600,
	}

	encoded, err := Encode(PacketAnnounce, payload)
	if err != nil {
		t.Fatalf("Encode failed: %v", err)
	}

	packet, err := Decode(encoded)
	if err != nil {
		t.Fatalf("Decode failed: %v", err)
	}

	if packet.Type != PacketAnnounce {
		t.Errorf("expected type %s got %s", PacketAnnounce, packet.Type)
	}

	var decoded AnnouncePayload
	if err := DecodePayload(packet.Payload, &decoded); err != nil {
		t.Fatalf("DecodePayload failed: %v", err)
	}

	if decoded.FileHash != payload.FileHash {
		t.Errorf("file hash mismatch")
	}
	if decoded.Address != payload.Address {
		t.Errorf("address mismatch")
	}

	t.Logf("Announce: peer=%s file=%s", decoded.PeerID, decoded.FileHash[:12])
}

func TestEncodeDecodeChunkRequest(t *testing.T) {
	payload := ChunkRequestPayload{
		FileHash:   "3a7bd3e2a7b9f1c2d3e4f5a6b7c8d9e0f1a2b3c4d5e6f7a8b9c0d1e2f3a4b5c6",
		ChunkIndex: 7,
	}

	encoded, err := Encode(PacketChunkRequest, payload)
	if err != nil {
		t.Fatalf("Encode failed: %v", err)
	}

	packet, err := Decode(encoded)
	if err != nil {
		t.Fatalf("Decode failed: %v", err)
	}

	var decoded ChunkRequestPayload
	if err := DecodePayload(packet.Payload, &decoded); err != nil {
		t.Fatalf("DecodePayload failed: %v", err)
	}

	if decoded.ChunkIndex != 7 {
		t.Errorf("expected chunk index 7 got %d", decoded.ChunkIndex)
	}

	t.Logf("ChunkRequest: hash=%s index=%d", decoded.FileHash[:12], decoded.ChunkIndex)
}

func TestEncodeDecodeChunkResponse(t *testing.T) {
	data := []byte("chunk data bytes here")

	payload := ChunkResponsePayload{
		FileHash:   "3a7bd3e2a7b9f1c2d3e4f5a6b7c8d9e0f1a2b3c4d5e6f7a8b9c0d1e2f3a4b5c6",
		ChunkIndex: 3,
		Hash:       "abc123abc123abc123abc123abc123abc123abc123abc123abc123abc123abcd",
		Data:       data,
	}

	encoded, err := Encode(PacketChunkResponse, payload)
	if err != nil {
		t.Fatalf("Encode failed: %v", err)
	}

	packet, err := Decode(encoded)
	if err != nil {
		t.Fatalf("Decode failed: %v", err)
	}

	var decoded ChunkResponsePayload
	if err := DecodePayload(packet.Payload, &decoded); err != nil {
		t.Fatalf("DecodePayload failed: %v", err)
	}

	if string(decoded.Data) != string(data) {
		t.Errorf("data mismatch")
	}
	if decoded.ChunkIndex != 3 {
		t.Errorf("chunk index mismatch")
	}

	t.Logf("ChunkResponse: index=%d data=%d bytes", decoded.ChunkIndex, len(decoded.Data))
}

func TestEncodeDecodeBitfield(t *testing.T) {
	payload := BitfieldPayload{
		FileHash: "3a7bd3e2a7b9f1c2d3e4f5a6b7c8d9e0f1a2b3c4d5e6f7a8b9c0d1e2f3a4b5c6",
		Bitfield: []bool{true, false, true, true, false},
	}

	encoded, err := Encode(PacketBitfield, payload)
	if err != nil {
		t.Fatalf("Encode failed: %v", err)
	}

	packet, err := Decode(encoded)
	if err != nil {
		t.Fatalf("Decode failed: %v", err)
	}

	var decoded BitfieldPayload
	if err := DecodePayload(packet.Payload, &decoded); err != nil {
		t.Fatalf("DecodePayload failed: %v", err)
	}

	if len(decoded.Bitfield) != 5 {
		t.Errorf("expected bitfield length 5 got %d", len(decoded.Bitfield))
	}
	if decoded.Bitfield[0] != true || decoded.Bitfield[1] != false {
		t.Errorf("bitfield values mismatch")
	}

	t.Logf("Bitfield: %v", decoded.Bitfield)
}

func TestEncodeDecodePeerList(t *testing.T) {
	payload := PeerListPayload{
		FileHash: "3a7bd3e2a7b9f1c2d3e4f5a6b7c8d9e0f1a2b3c4d5e6f7a8b9c0d1e2f3a4b5c6",
		Peers: []PeerInfo{
			{PeerID: "peer-aaa", Address: "1.2.3.4:9000"},
			{PeerID: "peer-bbb", Address: "5.6.7.8:9000"},
		},
	}

	encoded, err := Encode(PacketPeerList, payload)
	if err != nil {
		t.Fatalf("Encode failed: %v", err)
	}

	packet, err := Decode(encoded)
	if err != nil {
		t.Fatalf("Decode failed: %v", err)
	}

	var decoded PeerListPayload
	if err := DecodePayload(packet.Payload, &decoded); err != nil {
		t.Fatalf("DecodePayload failed: %v", err)
	}

	if len(decoded.Peers) != 2 {
		t.Errorf("expected 2 peers got %d", len(decoded.Peers))
	}
	if decoded.Peers[0].PeerID != "peer-aaa" {
		t.Errorf("peer ID mismatch")
	}

	t.Logf("PeerList: %d peers for hash %s", len(decoded.Peers), decoded.FileHash[:12])
}
