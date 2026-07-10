package crypto

import (
	"crypto/ed25519"
	"crypto/rand"
	"net"
	"testing"
)

func generateTestKey(t *testing.T) ed25519.PrivateKey {
	t.Helper()
	_, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("failed to generate key: %v", err)
	}
	return priv
}

func TestNoiseHandshakeAndEncryption(t *testing.T) {
	privA := generateTestKey(t)
	privB := generateTestKey(t)

	cfgA, err := NewConfig(privA)
	if err != nil {
		t.Fatalf("NewConfig A failed: %v", err)
	}
	cfgB, err := NewConfig(privB)
	if err != nil {
		t.Fatalf("NewConfig B failed: %v", err)
	}

	connA, connB := net.Pipe()

	type result struct {
		sess *Session
		err  error
	}

	chA := make(chan result, 1)
	chB := make(chan result, 1)

	go func() {
		sess, err := Handshake(connA, cfgA, true)
		chA <- result{sess, err}
	}()

	go func() {
		sess, err := Handshake(connB, cfgB, false)
		chB <- result{sess, err}
	}()

	rA := <-chA
	rB := <-chB

	if rA.err != nil {
		t.Fatalf("initiator handshake failed: %v", rA.err)
	}
	if rB.err != nil {
		t.Fatalf("responder handshake failed: %v", rB.err)
	}

	sessA, sessB := rA.sess, rB.sess

	message := []byte("hello encrypted parcel")

	done := make(chan struct{})
	go func() {
		buf := make([]byte, 256)
		n, err := sessB.Read(buf)
		if err != nil {
			t.Errorf("Read failed: %v", err)
			close(done)
			return
		}
		if string(buf[:n]) != string(message) {
			t.Errorf("message mismatch: expected %q got %q", message, buf[:n])
		}
		t.Logf("encrypted message received: %q", buf[:n])
		close(done)
	}()

	if _, err := sessA.Write(message); err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	<-done
}

func TestBidirectionalEncryption(t *testing.T) {
	privA := generateTestKey(t)
	privB := generateTestKey(t)

	cfgA, _ := NewConfig(privA)
	cfgB, _ := NewConfig(privB)

	connA, connB := net.Pipe()

	type result struct {
		sess *Session
		err  error
	}

	chA := make(chan result, 1)
	chB := make(chan result, 1)

	go func() {
		sess, err := Handshake(connA, cfgA, true)
		chA <- result{sess, err}
	}()

	go func() {
		sess, err := Handshake(connB, cfgB, false)
		chB <- result{sess, err}
	}()

	rA := <-chA
	rB := <-chB

	if rA.err != nil {
		t.Fatalf("A handshake failed: %v", rA.err)
	}
	if rB.err != nil {
		t.Fatalf("B handshake failed: %v", rB.err)
	}

	sessA, sessB := rA.sess, rB.sess
	buf := make([]byte, 64)

	// A → B
	done := make(chan string, 1)
	go func() {
		n, err := sessB.Read(buf)
		if err != nil {
			done <- ""
			return
		}
		done <- string(buf[:n])
	}()
	sessA.Write([]byte("from A"))
	if got := <-done; got != "from A" {
		t.Errorf("A→B failed: got %q", got)
	} else {
		t.Logf("A→B: %q", got)
	}

	// B → A
	go func() {
		n, err := sessA.Read(buf)
		if err != nil {
			done <- ""
			return
		}
		done <- string(buf[:n])
	}()
	sessB.Write([]byte("from B"))
	if got := <-done; got != "from B" {
		t.Errorf("B→A failed: got %q", got)
	} else {
		t.Logf("B→A: %q", got)
	}

	t.Log("bidirectional encryption works correctly")
}
