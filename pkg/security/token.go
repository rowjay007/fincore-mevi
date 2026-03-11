package security

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var (
	ErrInvalidToken = errors.New("invalid token")
	ErrExpiredToken = errors.New("expired token")
)

type TokenPayload struct {
	UserID      string    `json:"user_id"`
	Roles       []string  `json:"roles"`
	Permissions []string  `json:"permissions"`
	IssuedAt    time.Time `json:"iat"`
	ExpiredAt   time.Time `json:"exp"`
}

type TokenMaker interface {
	CreateToken(payload TokenPayload) (string, error)
	VerifyToken(token string) (*TokenPayload, error)
}

type JWTMaker struct {
	secret []byte
}

type jwtClaims struct {
	Roles       []string `json:"roles"`
	Permissions []string `json:"permissions"`
	jwt.RegisteredClaims
}

func NewJWTMaker(secret string) (*JWTMaker, error) {
	if strings.TrimSpace(secret) == "" {
		return nil, errors.New("secret required")
	}
	if len(secret) < 32 {
		return nil, fmt.Errorf("secret too short: got %d, want >= 32", len(secret))
	}
	return &JWTMaker{secret: []byte(secret)}, nil
}

func (maker *JWTMaker) CreateToken(payload TokenPayload) (string, error) {
	claims := jwtClaims{
		Roles:       payload.Roles,
		Permissions: payload.Permissions,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   payload.UserID,
			IssuedAt:  jwt.NewNumericDate(payload.IssuedAt),
			ExpiresAt: jwt.NewNumericDate(payload.ExpiredAt),
		},
	}

	t := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return t.SignedString(maker.secret)
}

func (maker *JWTMaker) VerifyToken(tokenStr string) (*TokenPayload, error) {
	t, err := jwt.ParseWithClaims(tokenStr, &jwtClaims{}, func(token *jwt.Token) (any, error) {
		if token.Method != jwt.SigningMethodHS256 {
			return nil, ErrInvalidToken
		}
		return maker.secret, nil
	})
	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, ErrExpiredToken
		}
		return nil, ErrInvalidToken
	}

	claims, ok := t.Claims.(*jwtClaims)
	if !ok || !t.Valid {
		return nil, ErrInvalidToken
	}
	if claims.Subject == "" {
		return nil, ErrInvalidToken
	}
	if claims.IssuedAt == nil || claims.ExpiresAt == nil {
		return nil, ErrInvalidToken
	}

	return &TokenPayload{
		UserID:      claims.Subject,
		Roles:       claims.Roles,
		Permissions: claims.Permissions,
		IssuedAt:    claims.IssuedAt.Time,
		ExpiredAt:   claims.ExpiresAt.Time,
	}, nil
}

var _ TokenMaker = (*JWTMaker)(nil)
