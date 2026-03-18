package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"fincore/pkg/security"
)

func TestWellKnownEndpoints(t *testing.T) {
	cfg := openIDConfiguration{Issuer: "http://example", JWKSURI: "http://example/.well-known/jwks.json"}
	jwks := security.JWKS{Keys: []security.JWK{{Kty: "OKP", Crv: "Ed25519", Kid: "kid1", X: "x"}}}

	h := newHTTPHandler(http.NewServeMux(), cfg, jwks, "/.well-known/jwks.json")

	{
		req := httptest.NewRequest(http.MethodGet, "http://example/.well-known/openid-configuration", nil)
		w := httptest.NewRecorder()
		h.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", w.Code)
		}
		var got openIDConfiguration
		if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if got.JWKSURI != cfg.JWKSURI {
			t.Fatalf("expected jwks_uri %q, got %q", cfg.JWKSURI, got.JWKSURI)
		}
	}

	{
		req := httptest.NewRequest(http.MethodGet, "http://example/.well-known/jwks.json", nil)
		w := httptest.NewRecorder()
		h.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", w.Code)
		}
		var got security.JWKS
		if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if len(got.Keys) != 1 || got.Keys[0].Kid != "kid1" {
			t.Fatalf("unexpected jwks: %+v", got)
		}
	}

	{
		req := httptest.NewRequest(http.MethodGet, "http://example/jwks.json", nil)
		w := httptest.NewRecorder()
		h.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", w.Code)
		}
		var got security.JWKS
		if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if len(got.Keys) != 1 || got.Keys[0].Kid != "kid1" {
			t.Fatalf("unexpected jwks alias: %+v", got)
		}
	}
}
