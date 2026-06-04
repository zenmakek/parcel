package session

import (
	"fmt"
	"net"
	"sync"
	"time"
)

const (
	SessionExpiry      = 5 * time.Minute
	StatusWaiting      = "WAITING"
	StatusConnected    = "CONNECTED"
	StatusTransferring = "TRANSFERRING"
	StatusDone         = "DONE"
)

type Session struct {
	OTP          string
	SenderConn   net.Conn
	ReceiverConn net.Conn
	CreatedAt    time.Time
	ExpiresAt    time.Time
	Status       string
}

type Registry struct {
	mu       sync.RWMutex
	sessions map[string]*Session
}

func NewRegistry() *Registry {
	r := &Registry{
		sessions: make(map[string]*Session),
	}
	go r.cleanupLoop()
	return r
}

func (r *Registry) Create(otp string, sender net.Conn) (*Session, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.sessions[otp]; exists {
		return nil, fmt.Errorf("session with OTP %s already exists", otp)
	}

	session := &Session{
		OTP:        otp,
		SenderConn: sender,
		CreatedAt:  time.Now(),
		ExpiresAt:  time.Now().Add(SessionExpiry),
		Status:     StatusWaiting,
	}

	r.sessions[otp] = session
	fmt.Printf("[registry] session created: %s (expires in 5 minutes)\n", otp)
	return session, nil
}

func (r *Registry) Get(otp string) (*Session, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	session, exists := r.sessions[otp]
	if !exists {
		return nil, false
	}

	if time.Now().After(session.ExpiresAt) {
		return nil, false
	}

	return session, true
}

func (r *Registry) JoinReceiver(otp string, receiver net.Conn) (*Session, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	session, exists := r.sessions[otp]
	if !exists {
		return nil, fmt.Errorf("session not found: %s", otp)
	}

	if time.Now().After(session.ExpiresAt) {
		delete(r.sessions, otp)
		return nil, fmt.Errorf("session expired: %s", otp)
	}

	if session.Status != StatusWaiting {
		return nil, fmt.Errorf("session not available: %s (status: %s)", otp, session.Status)
	}

	session.ReceiverConn = receiver
	session.Status = StatusConnected
	fmt.Printf("[registry] receiver joined session: %s\n", otp)
	return session, nil
}

func (r *Registry) Destroy(otp string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	session, exists := r.sessions[otp]
	if !exists {
		return
	}

	if session.SenderConn != nil {
		session.SenderConn.Close()
	}
	if session.ReceiverConn != nil {
		session.ReceiverConn.Close()
	}

	delete(r.sessions, otp)
	fmt.Printf("[registry] session destroyed: %s\n", otp)
}

func (r *Registry) cleanupLoop() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		r.mu.Lock()
		now := time.Now()
		for otp, session := range r.sessions {
			if now.After(session.ExpiresAt) {
				if session.SenderConn != nil {
					session.SenderConn.Close()
				}
				if session.ReceiverConn != nil {
					session.ReceiverConn.Close()
				}
				delete(r.sessions, otp)
				fmt.Printf("[registry] session expired and cleaned up: %s\n", otp)
			}
		}
		r.mu.Unlock()
	}
}
