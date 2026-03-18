package secrets

import (
	"os"
	"path/filepath"
	"testing"
)

func TestVaultTokenFromEnvOrFile_Env(t *testing.T) {
	t.Setenv("VAULT_TOKEN", "abc")
	t.Setenv("VAULT_TOKEN_FILE", "")
	v, ok, err := VaultTokenFromEnvOrFile()
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if !ok || v != "abc" {
		t.Fatalf("unexpected token")
	}
}

func TestVaultTokenFromEnvOrFile_File(t *testing.T) {
	t.Setenv("VAULT_TOKEN", "")
	d := t.TempDir()
	p := filepath.Join(d, "token")
	if err := os.WriteFile(p, []byte("root\n"), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}
	t.Setenv("VAULT_TOKEN_FILE", p)
	v, ok, err := VaultTokenFromEnvOrFile()
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if !ok || v != "root" {
		t.Fatalf("unexpected token")
	}
}

func TestVaultTokenFromEnvOrFile_None(t *testing.T) {
	t.Setenv("VAULT_TOKEN", "")
	t.Setenv("VAULT_TOKEN_FILE", "")
	_, ok, err := VaultTokenFromEnvOrFile()
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if ok {
		t.Fatalf("expected ok=false")
	}
}
