package otp

import (
	"crypto/rand"
	"fmt"
	"math/big"
)

const otpLength = 6

func Generate() (string, error) {
	max := big.NewInt(900000)
	n, err := rand.Int(rand.Reader, max)
	if err != nil {
		return "", fmt.Errorf("failed to generate OTP: %w", err)
	}
	otp := n.Int64() + 100000
	return fmt.Sprintf(("%06d"), otp), nil
}
