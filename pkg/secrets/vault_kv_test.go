package secrets

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestNewVaultKVClient_Validation(t *testing.T) {
	if _, err := NewVaultKVClient(VaultKVClientConfig{}); err == nil {
		t.Fatalf("expected error")
	}
	if _, err := NewVaultKVClient(VaultKVClientConfig{Addr: "http://vault", Token: "t"}); err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
}

func TestVaultKVClient_ReadKVV2(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/secret/data/identity", func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Vault-Token") != "root" {
			w.WriteHeader(http.StatusForbidden)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":{"data":{"kid":"dev","jwt_ed25519_private_key":"abc"}}}`))
	})
	s := httptest.NewServer(mux)
	defer s.Close()

	c, err := NewVaultKVClient(VaultKVClientConfig{Addr: s.URL, Token: "root", KVV2Mount: "secret", HTTPTimeout: 500 * time.Millisecond})
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}

	data, err := c.ReadKVV2(context.Background(), "identity")
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if data["kid"] != "dev" {
		t.Fatalf("unexpected kid: %v", data["kid"])
	}
	if data["jwt_ed25519_private_key"] != "abc" {
		t.Fatalf("unexpected key")
	}
}

func TestVaultKVClient_ReadKVV2_StatusError(t *testing.T) {
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer s.Close()

	c, err := NewVaultKVClient(VaultKVClientConfig{Addr: s.URL, Token: "root", KVV2Mount: "secret"})
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}

	if _, err := c.ReadKVV2(context.Background(), "identity"); err == nil {
		t.Fatalf("expected error")
	}
}
