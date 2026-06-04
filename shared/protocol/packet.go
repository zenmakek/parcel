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
)