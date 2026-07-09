package protocol

const (
	PacketHello            = "HELLO"
	PacketAck              = "ACK"
	PacketTransferInit     = "TRANSFER_INIT"
	PacketTransferReady    = "TRANSFER_READY"
	PacketTransferBegin    = "TRANSFER_BEGIN"
	PacketTransferComplete = "TRANSFER_COMPLETE"
	PacketTransferError    = "TRANSFER_ERROR"
	PacketOTPExpired       = "OTP_EXPIRED"
	PacketOTPJoin          = "OTP_JOIN"
	PacketOTPInvalid       = "OTP_INVALID"

	PacketAnnounce         = "ANNOUNCE"
	PacketLookup           = "LOOKUP"
	PacketPeerList         = "PEER_LIST"
	PacketManifestRequest  = "MANIFEST_REQUEST"
	PacketManifestResponse = "MANIFEST_RESPONSE"
	PacketChunkRequest     = "CHUNK_REQUEST"
	PacketChunkResponse    = "CHUNK_RESPONSE"
	PacketChunkVerifyFail  = "CHUNK_VERIFY_FAIL"
	PacketHave             = "HAVE"
	PacketBitfield         = "BITFIELD"

	PacketSTUNRequest      = "STUN_REQUEST"
	PacketSTUNResponse     = "STUN_RESPONSE"
	PacketHolePunchReady   = "HOLE_PUNCH_READY"
	PacketHolePunchConnect = "HOLE_PUNCH_CONNECT"
)
