package security

import (
	"crypto/sha256"
	"encoding/base64"
	"strings"
)

func HashRefreshToken(token string) string {
	t := strings.TrimSpace(token)
	sum := sha256.Sum256([]byte(t))
	return base64.RawURLEncoding.EncodeToString(sum[:])
}
