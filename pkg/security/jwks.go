package security

import (
	"context"
	"crypto/ed25519"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sync"
	"time"
)

type JWK struct {
	Kty string `json:"kty"`
	Crv string `json:"crv,omitempty"`
	Kid string `json:"kid,omitempty"`
	Use string `json:"use,omitempty"`
	Alg string `json:"alg,omitempty"`
	X   string `json:"x,omitempty"`
}

type JWKS struct {
	Keys []JWK `json:"keys"`
}

type JWKSVerifier struct {
	url      string
	client   *http.Client
	cacheTTL time.Duration

	mu         sync.RWMutex
	cached     JWKS
	cachedAt   time.Time
	cachedKeys map[string]ed25519.PublicKey
}

func NewJWKSVerifier(jwksURL string, cacheTTL time.Duration) (*JWKSVerifier, error) {
	if jwksURL == "" {
		return nil, errors.New("jwks url required")
	}
	if cacheTTL <= 0 {
		cacheTTL = 5 * time.Minute
	}
	return &JWKSVerifier{
		url:      jwksURL,
		client:   &http.Client{Timeout: 5 * time.Second},
		cacheTTL: cacheTTL,
	}, nil
}

func (v *JWKSVerifier) CreateToken(TokenPayload) (string, error) {
	return "", errors.New("CreateToken not supported")
}

func (v *JWKSVerifier) VerifyToken(token string) (*TokenPayload, error) {
	return verifyEd25519JWT(token, v.lookupKey)
}

func (v *JWKSVerifier) lookupKey(ctx context.Context, kid string) (ed25519.PublicKey, error) {
	if kid == "" {
		return nil, ErrInvalidToken
	}

	v.mu.RLock()
	stale := v.cachedKeys == nil || time.Since(v.cachedAt) >= v.cacheTTL
	v.mu.RUnlock()
	if stale {
		if err := v.refresh(ctx); err != nil {
			// If refresh failed, try stale cache as a last resort.
			if key, ok := v.getCachedKey(kid); ok {
				return key, nil
			}
			return nil, err
		}
	}

	if key, ok := v.getCachedKey(kid); ok {
		return key, nil
	}
	return nil, ErrInvalidToken
}

func (v *JWKSVerifier) getCachedKey(kid string) (ed25519.PublicKey, bool) {
	v.mu.RLock()
	defer v.mu.RUnlock()
	if v.cachedKeys == nil {
		return nil, false
	}
	k, ok := v.cachedKeys[kid]
	return k, ok
}

func (v *JWKSVerifier) refresh(ctx context.Context) error {
	v.mu.RLock()
	stale := time.Since(v.cachedAt) < v.cacheTTL && v.cachedKeys != nil
	v.mu.RUnlock()
	if stale {
		return nil
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, v.url, nil)
	if err != nil {
		return err
	}
	resp, err := v.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("jwks fetch failed: %s", resp.Status)
	}

	var jwks JWKS
	dec := json.NewDecoder(resp.Body)
	if err := dec.Decode(&jwks); err != nil {
		return err
	}

	keys := make(map[string]ed25519.PublicKey)
	for _, k := range jwks.Keys {
		if k.Kty != "OKP" || k.Crv != "Ed25519" || k.Kid == "" || k.X == "" {
			continue
		}
		pub, err := decodeB64URL(k.X)
		if err != nil {
			continue
		}
		if len(pub) != ed25519.PublicKeySize {
			continue
		}
		keys[k.Kid] = ed25519.PublicKey(pub)
	}
	if len(keys) == 0 {
		return errors.New("jwks contains no usable keys")
	}

	v.mu.Lock()
	v.cached = jwks
	v.cachedAt = time.Now().UTC()
	v.cachedKeys = keys
	v.mu.Unlock()
	return nil
}

var _ TokenMaker = (*JWKSVerifier)(nil)
