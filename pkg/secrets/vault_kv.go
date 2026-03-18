package secrets

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"path"
	"strings"
	"time"
)

type VaultKVClient struct {
	addr  string
	token string
	mount string
	c     *http.Client
}

type VaultKVClientConfig struct {
	Addr        string
	Token       string
	KVV2Mount   string
	HTTPTimeout time.Duration
}

func NewVaultKVClient(cfg VaultKVClientConfig) (*VaultKVClient, error) {
	addr := strings.TrimSpace(cfg.Addr)
	token := strings.TrimSpace(cfg.Token)
	mount := strings.TrimSpace(cfg.KVV2Mount)
	if addr == "" {
		return nil, errors.New("vault addr required")
	}
	if token == "" {
		return nil, errors.New("vault token required")
	}
	if mount == "" {
		mount = "secret"
	}
	if _, err := url.Parse(addr); err != nil {
		return nil, err
	}
	t := cfg.HTTPTimeout
	if t <= 0 {
		t = 5 * time.Second
	}
	return &VaultKVClient{
		addr:  strings.TrimRight(addr, "/"),
		token: token,
		mount: mount,
		c: &http.Client{
			Timeout: t,
		},
	}, nil
}

type vaultKVV2ReadResponse struct {
	Data struct {
		Data map[string]any `json:"data"`
	} `json:"data"`
}

func (v *VaultKVClient) ReadKVV2(ctx context.Context, secretPath string) (map[string]any, error) {
	p := strings.TrimLeft(strings.TrimSpace(secretPath), "/")
	if p == "" {
		return nil, errors.New("secret path required")
	}

	u := v.addr + path.Join("/v1/", v.mount, "data", p)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("X-Vault-Token", v.token)

	resp, err := v.c.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("vault read failed: status=%d", resp.StatusCode)
	}

	var out vaultKVV2ReadResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	return out.Data.Data, nil
}
