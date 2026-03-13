package security

import (
	"context"
	"crypto/ed25519"
	"errors"

	"github.com/golang-jwt/jwt/v5"
)

func verifyEd25519JWT(tokenStr string, lookup func(ctx context.Context, kid string) (ed25519.PublicKey, error)) (*TokenPayload, error) {
	ctx := context.Background()
	t, err := jwt.ParseWithClaims(tokenStr, &jwtClaims{}, func(token *jwt.Token) (any, error) {
		if token.Method != jwt.SigningMethodEdDSA {
			return nil, ErrInvalidToken
		}
		kid, _ := token.Header["kid"].(string)
		pub, err := lookup(ctx, kid)
		if err != nil {
			return nil, ErrInvalidToken
		}
		return pub, nil
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
