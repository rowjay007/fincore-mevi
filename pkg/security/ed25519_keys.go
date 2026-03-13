package security

import (
	"crypto/ed25519"
	"errors"
)

func ParseEd25519PrivateKeyBase64URL(s string) (ed25519.PrivateKey, error) {
	if s == "" {
		return nil, errors.New("ed25519 private key required")
	}
	b, err := decodeB64URL(s)
	if err != nil {
		return nil, err
	}
	if len(b) != ed25519.PrivateKeySize {
		return nil, errors.New("invalid ed25519 private key")
	}
	return ed25519.PrivateKey(b), nil
}

func Ed25519PublicJWK(kid string, pub ed25519.PublicKey) (JWK, error) {
	if kid == "" {
		return JWK{}, errors.New("kid required")
	}
	if len(pub) != ed25519.PublicKeySize {
		return JWK{}, errors.New("invalid ed25519 public key")
	}
	return JWK{
		Kty: "OKP",
		Crv: "Ed25519",
		Kid: kid,
		Use: "sig",
		Alg: "EdDSA",
		X:   encodeB64URL(pub),
	}, nil
}
