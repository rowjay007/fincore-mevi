package security

import (
	"context"
	"crypto/ed25519"
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var (
	ErrInvalidToken = errors.New("invalid token")
	ErrExpiredToken = errors.New("expired token")
)

type contextKey string

const (
	LSNContextKey contextKey = "lsn"
)

type TokenPayload struct {
	UserID      string    `json:"user_id"`
	Roles       []string  `json:"roles"`
	Permissions []string  `json:"permissions"`
	LSN         uint64    `json:"lsn,omitempty"`
	IssuedAt    time.Time `json:"iat"`
	ExpiredAt   time.Time `json:"exp"`
}

type TokenMaker interface {
	CreateToken(payload TokenPayload) (string, error)
	VerifyToken(token string) (*TokenPayload, error)
}

type Ed25519JWTMaker struct {
	kid  string
	priv ed25519.PrivateKey
}

type jwtClaims struct {
	Roles       []string `json:"roles"`
	Permissions []string `json:"permissions"`
	LSN         uint64   `json:"lsn,omitempty"`
	jwt.RegisteredClaims
}

func NewEd25519JWTMaker(kid string, priv ed25519.PrivateKey) (*Ed25519JWTMaker, error) {
	if kid == "" {
		return nil, errors.New("kid required")
	}
	if len(priv) != ed25519.PrivateKeySize {
		return nil, errors.New("invalid ed25519 private key")
	}
	return &Ed25519JWTMaker{kid: kid, priv: priv}, nil
}

func (maker *Ed25519JWTMaker) CreateToken(payload TokenPayload) (string, error) {
	claims := jwtClaims{
		Roles:       payload.Roles,
		Permissions: payload.Permissions,
		LSN:         payload.LSN,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   payload.UserID,
			IssuedAt:  jwt.NewNumericDate(payload.IssuedAt),
			ExpiresAt: jwt.NewNumericDate(payload.ExpiredAt),
		},
	}

	t := jwt.NewWithClaims(jwt.SigningMethodEdDSA, claims)
	t.Header["kid"] = maker.kid
	return t.SignedString(maker.priv)
}

func (maker *Ed25519JWTMaker) VerifyToken(tokenStr string) (*TokenPayload, error) {
	pub := maker.priv.Public().(ed25519.PublicKey)
	return verifyEd25519JWT(tokenStr, func(_ context.Context, kid string) (ed25519.PublicKey, error) {
		if kid != maker.kid {
			return nil, ErrInvalidToken
		}
		return pub, nil
	})
}

var _ TokenMaker = (*Ed25519JWTMaker)(nil)
