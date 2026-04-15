package main

import (
	"os"
	"testing"
)

func TestParsePublicPrefixesEnv_DefaultsDoNotIncludeAdmin(t *testing.T) {
	old := os.Getenv("GATEWAY_PUBLIC_PATH_PREFIXES")
	_ = os.Unsetenv("GATEWAY_PUBLIC_PATH_PREFIXES")
	t.Cleanup(func() {
		if old == "" {
			_ = os.Unsetenv("GATEWAY_PUBLIC_PATH_PREFIXES")
			return
		}
		_ = os.Setenv("GATEWAY_PUBLIC_PATH_PREFIXES", old)
	})

	prefixes := parsePublicPrefixesEnv()
	for _, p := range prefixes {
		if len(p) >= len("/v1/auth/admin") && p[:len("/v1/auth/admin")] == "/v1/auth/admin" {
			t.Fatalf("admin prefix must not be public: %q", p)
		}
	}
	foundLogin := false
	for _, p := range prefixes {
		if p == "/v1/auth/login" {
			foundLogin = true
			break
		}
	}
	if !foundLogin {
		t.Fatalf("expected /v1/auth/login to be public")
	}
}
