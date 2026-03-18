package secrets

import (
	"errors"
	"os"
	"strings"
)

func VaultTokenFromEnvOrFile() (string, bool, error) {
	if v := strings.TrimSpace(os.Getenv("VAULT_TOKEN")); v != "" {
		return v, true, nil
	}
	p := strings.TrimSpace(os.Getenv("VAULT_TOKEN_FILE"))
	if p == "" {
		return "", false, nil
	}
	b, err := os.ReadFile(p)
	if err != nil {
		return "", false, err
	}
	tok := strings.TrimSpace(string(b))
	if tok == "" {
		return "", false, errors.New("vault token file is empty")
	}
	return tok, true, nil
}
