package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	authv1 "fincore/gen/go/auth/v1"
	"fincore/pkg/security"
)

func TestWellKnownEndpoints(t *testing.T) {
	cfg := openIDConfiguration{
		Issuer:                           "http://example",
		JWKSURI:                          "http://example/.well-known/jwks.json",
		AuthorizationEndpoint:            "http://example/oauth/authorize",
		TokenEndpoint:                    "http://example/oauth/token",
		GrantTypesSupported:              []string{"authorization_code"},
		CodeChallengeMethodsSupported:    []string{"S256"},
		ResponseTypesSupported:           []string{"code"},
		SubjectTypesSupported:            []string{"public"},
		IDTokenSigningAlgValuesSupported: []string{"EdDSA"},
	}
	jwks := security.JWKS{Keys: []security.JWK{{Kty: "OKP", Crv: "Ed25519", Kid: "kid1", X: "x"}}}

	h := newHTTPHandler(http.NewServeMux(), authv1.AuthServiceClient(nil), cfg, jwks, "/.well-known/jwks.json")

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
		if len(got.ResponseTypesSupported) != 1 || got.ResponseTypesSupported[0] != "code" {
			t.Fatalf("expected response_types_supported [code], got %+v", got.ResponseTypesSupported)
		}
		if len(got.GrantTypesSupported) != 1 || got.GrantTypesSupported[0] != "authorization_code" {
			t.Fatalf("expected grant_types_supported [authorization_code], got %+v", got.GrantTypesSupported)
		}
		if len(got.CodeChallengeMethodsSupported) != 1 || got.CodeChallengeMethodsSupported[0] != "S256" {
			t.Fatalf("expected code_challenge_methods_supported [S256], got %+v", got.CodeChallengeMethodsSupported)
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
