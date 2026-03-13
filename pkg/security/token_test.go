package security

import (
	"testing"
	"time"
)

func TestJWTMaker_CreateAndVerify(t *testing.T) {
	maker, err := NewJWTMaker("0123456789abcdef0123456789abcdef")
	if err != nil {
		t.Fatalf("NewJWTMaker: %v", err)
	}

	now := time.Now().UTC()
	tok, err := maker.CreateToken(TokenPayload{
		UserID:      "user-1",
		Roles:       []string{"customer"},
		Permissions: []string{"account:read"},
		IssuedAt:    now,
		ExpiredAt:   now.Add(1 * time.Minute),
	})
	if err != nil {
		t.Fatalf("CreateToken: %v", err)
	}

	p, err := maker.VerifyToken(tok)
	if err != nil {
		t.Fatalf("VerifyToken: %v", err)
	}
	if p.UserID != "user-1" {
		t.Fatalf("user id mismatch: got %q", p.UserID)
	}
	if len(p.Permissions) != 1 || p.Permissions[0] != "account:read" {
		t.Fatalf("permissions mismatch: %#v", p.Permissions)
	}
}

func TestJWTMaker_VerifyExpired(t *testing.T) {
	maker, err := NewJWTMaker("0123456789abcdef0123456789abcdef")
	if err != nil {
		t.Fatalf("NewJWTMaker: %v", err)
	}

	now := time.Now().UTC()
	tok, err := maker.CreateToken(TokenPayload{
		UserID:    "user-1",
		IssuedAt:  now.Add(-2 * time.Minute),
		ExpiredAt: now.Add(-1 * time.Minute),
	})
	if err != nil {
		t.Fatalf("CreateToken: %v", err)
	}

	_, err = maker.VerifyToken(tok)
	if err == nil {
		t.Fatalf("expected error")
	}
	if err != ErrExpiredToken {
		t.Fatalf("expected ErrExpiredToken, got %v", err)
	}
}
