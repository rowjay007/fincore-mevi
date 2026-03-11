package security

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"

	"golang.org/x/crypto/argon2"
)

const (
	argon2idTime    uint32 = 3
	argon2idMemory  uint32 = 64 * 1024
	argon2idThreads uint8  = 2
	argon2idKeyLen  uint32 = 32
	saltLen                = 16
)

func HashPassword(password string) (string, error) {
	if strings.TrimSpace(password) == "" {
		return "", errors.New("password required")
	}

	salt := make([]byte, saltLen)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("read salt: %w", err)
	}

	hash := argon2.IDKey([]byte(password), salt, argon2idTime, argon2idMemory, argon2idThreads, argon2idKeyLen)

	b64Salt := base64.RawStdEncoding.EncodeToString(salt)
	b64Hash := base64.RawStdEncoding.EncodeToString(hash)

	encoded := fmt.Sprintf("$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s", argon2.Version, argon2idMemory, argon2idTime, argon2idThreads, b64Salt, b64Hash)
	return encoded, nil
}

func VerifyPassword(password string, encodedHash string) (bool, error) {
	if strings.TrimSpace(password) == "" {
		return false, errors.New("password required")
	}
	if strings.TrimSpace(encodedHash) == "" {
		return false, errors.New("encoded hash required")
	}

	params, salt, expectedHash, err := parseArgon2idHash(encodedHash)
	if err != nil {
		return false, err
	}

	actualHash := argon2.IDKey([]byte(password), salt, params.time, params.memory, params.threads, uint32(len(expectedHash)))
	if subtle.ConstantTimeCompare(actualHash, expectedHash) == 1 {
		return true, nil
	}
	return false, nil
}

type argon2idParams struct {
	time    uint32
	memory  uint32
	threads uint8
}

func parseArgon2idHash(encoded string) (argon2idParams, []byte, []byte, error) {
	parts := strings.Split(encoded, "$")
	// Expect: "", "argon2id", "v=19", "m=65536,t=3,p=2", "<salt>", "<hash>"
	if len(parts) != 6 {
		return argon2idParams{}, nil, nil, errors.New("invalid hash format")
	}
	if parts[1] != "argon2id" {
		return argon2idParams{}, nil, nil, errors.New("invalid hash algorithm")
	}

	var version int
	if _, err := fmt.Sscanf(parts[2], "v=%d", &version); err != nil {
		return argon2idParams{}, nil, nil, errors.New("invalid hash version")
	}
	if version != argon2.Version {
		return argon2idParams{}, nil, nil, errors.New("incompatible hash version")
	}

	var memory, time uint32
	var threads uint8
	if _, err := fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &memory, &time, &threads); err != nil {
		return argon2idParams{}, nil, nil, errors.New("invalid hash params")
	}

	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return argon2idParams{}, nil, nil, errors.New("invalid salt encoding")
	}
	hash, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil {
		return argon2idParams{}, nil, nil, errors.New("invalid hash encoding")
	}
	if len(hash) == 0 {
		return argon2idParams{}, nil, nil, errors.New("invalid hash length")
	}

	return argon2idParams{time: time, memory: memory, threads: threads}, salt, hash, nil
}
