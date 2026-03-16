package security

import (
	"context"
	"crypto/ed25519"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"
)

func TestJWKSVerifier_VerifiesTokenFromJWKS(t *testing.T) {
	pub, priv, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatalf("GenerateKey: %v", err)
	}
	maker, err := NewEd25519JWTMaker("k1", priv)
	if err != nil {
		t.Fatalf("NewEd25519JWTMaker: %v", err)
	}

	jwk, err := Ed25519PublicJWK("k1", pub)
	if err != nil {
		t.Fatalf("Ed25519PublicJWK: %v", err)
	}
	jwks := JWKS{Keys: []JWK{jwk}}

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(jwks)
	}))
	defer ts.Close()

	verifier, err := NewJWKSVerifier(ts.URL, 5*time.Minute)
	if err != nil {
		t.Fatalf("NewJWKSVerifier: %v", err)
	}

	now := time.Now().UTC()
	tok, err := maker.CreateToken(TokenPayload{
		UserID:    "user-1",
		IssuedAt:  now,
		ExpiredAt: now.Add(1 * time.Minute),
	})
	if err != nil {
		t.Fatalf("CreateToken: %v", err)
	}

	p, err := verifier.VerifyToken(tok)
	if err != nil {
		t.Fatalf("VerifyToken: %v", err)
	}
	if p.UserID != "user-1" {
		t.Fatalf("user id mismatch: got %q", p.UserID)
	}
}

func TestJWKSVerifier_Rotation(t *testing.T) {
	pub1, priv1, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatalf("GenerateKey: %v", err)
	}
	pub2, _, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatalf("GenerateKey: %v", err)
	}

	maker, err := NewEd25519JWTMaker("k1", priv1)
	if err != nil {
		t.Fatalf("NewEd25519JWTMaker: %v", err)
	}

	jwk1, err := Ed25519PublicJWK("k1", pub1)
	if err != nil {
		t.Fatalf("Ed25519PublicJWK: %v", err)
	}
	jwk2, err := Ed25519PublicJWK("k2", pub2)
	if err != nil {
		t.Fatalf("Ed25519PublicJWK: %v", err)
	}

	var (
		mu   sync.RWMutex
		jwks = JWKS{Keys: []JWK{jwk1}}
	)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.RLock()
		cur := jwks
		mu.RUnlock()
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(cur)
	}))
	defer ts.Close()

	verifier, err := NewJWKSVerifier(ts.URL, 50*time.Millisecond)
	if err != nil {
		t.Fatalf("NewJWKSVerifier: %v", err)
	}

	now := time.Now().UTC()
	tok1, err := maker.CreateToken(TokenPayload{UserID: "user-1", IssuedAt: now, ExpiredAt: now.Add(1 * time.Minute)})
	if err != nil {
		t.Fatalf("CreateToken: %v", err)
	}

	// Initial verify succeeds with k1.
	if _, err := verifier.VerifyToken(tok1); err != nil {
		t.Fatalf("VerifyToken (k1): %v", err)
	}

	// Rotate: publish both keys. Old token should still verify.
	mu.Lock()
	jwks = JWKS{Keys: []JWK{jwk1, jwk2}}
	mu.Unlock()

	time.Sleep(60 * time.Millisecond)
	if _, err := verifier.VerifyToken(tok1); err != nil {
		t.Fatalf("VerifyToken after adding k2: %v", err)
	}

	// Rotate: remove old key. After cache expiry, old token should fail.
	mu.Lock()
	jwks = JWKS{Keys: []JWK{jwk2}}
	mu.Unlock()

	deadline := time.Now().Add(500 * time.Millisecond)
	for {
		_, err := verifier.VerifyToken(tok1)
		if err != nil {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("expected verification error after removing old key")
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func TestJWKSVerifier_InvalidJWKS(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"keys":[]}`))
	}))
	defer ts.Close()

	verifier, err := NewJWKSVerifier(ts.URL, 10*time.Millisecond)
	if err != nil {
		t.Fatalf("NewJWKSVerifier: %v", err)
	}

	_, err = verifier.lookupKey(context.Background(), "k1")
	if err == nil {
		t.Fatalf("expected error")
	}
}
