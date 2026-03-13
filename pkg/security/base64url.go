package security

import (
	"encoding/base64"
)

func encodeB64URL(b []byte) string {
	return base64.RawURLEncoding.EncodeToString(b)
}

func decodeB64URL(s string) ([]byte, error) {
	return base64.RawURLEncoding.DecodeString(s)
}
