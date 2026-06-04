package session

import (
	"testing"
	"time"
)

func TestCreateAndGetSession(t *testing.T) {
	r := NewRegistry()

	session, err := r.Create("483921", nil)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	if session.OTP != "483921" {
		t.Errorf("expected OTP 483921, got %s", session.OTP)
	}

	if session.Status != StatusWaiting {
		t.Errorf("expected status WAITING, got %s", session.Status)
	}

	got, exists := r.Get("483921")
	if !exists {
		t.Fatal("expected session to exist")
	}

	if got.OTP != "483921" {
		t.Errorf("expected OTP 483921, got %s", got.OTP)
	}
}

func TestDuplicateOTP(t *testing.T) {
	r := NewRegistry()

	_, err := r.Create("111111", nil)
	if err != nil {
		t.Fatalf("first Create failed: %v", err)
	}

	_, err = r.Create("111111", nil)
	if err == nil {
		t.Fatal("expected error for duplicate OTP, got nil")
	}

	t.Logf("Correctly rejected duplicate OTP: %v", err)
}

func TestSessionExpiry(t *testing.T) {
	r := NewRegistry()

	session, err := r.Create("999999", nil)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	session.ExpiresAt = time.Now().Add(-1 * time.Second)

	_, exists := r.Get("999999")
	if exists {
		t.Fatal("expected expired session to not be found")
	}

	t.Log("Correctly rejected expired session")
}

func TestDestroySession(t *testing.T) {
	r := NewRegistry()

	_, err := r.Create("777777", nil)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	r.Destroy("777777")

	_, exists := r.Get("777777")
	if exists {
		t.Fatal("expected session to be destroyed")
	}

	t.Log("Session destroyed successfully")
}