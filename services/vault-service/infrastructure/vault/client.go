package vault

import (
	"context"
	"fmt"
	"log"

	"fincore/services/vault-service/domain"

	"github.com/hashicorp/vault/api"
)

type Client struct {
	client *api.Client
}

func NewClient(address, token string) (*Client, error) {
	config := api.DefaultConfig()
	config.Address = address
	client, err := api.NewClient(config)
	if err != nil {
		return nil, err
	}
	client.SetToken(token)
	return &Client{client: client}, nil
}

func (c *Client) Tokenize(ctx context.Context, category string, data string) (string, error) {
	// Uses Vault Transit Secret Engine for FPE/Tokenization
	secret, err := c.client.Logical().WriteWithContext(ctx, fmt.Sprintf("transit/encrypt/%s", category), map[string]interface{}{
		"plaintext": data, // In real prod, this should be base64 encoded
	})
	if err != nil {
		return "", err
	}
	ciphertext, ok := secret.Data["ciphertext"].(string)
	if !ok {
		return "", fmt.Errorf("invalid response from vault")
	}
	return ciphertext, nil
}

func (c *Client) Detokenize(ctx context.Context, token string, reason string) (string, error) {
	// Security: Log detokenization for PCI-DSS/SOC2 compliance.
	log.Printf("AUDIT: Detokenize request for token %s, reason: %s", token, reason)

	secret, err := c.client.Logical().WriteWithContext(ctx, "transit/decrypt/card_pan", map[string]interface{}{
		"ciphertext": token,
	})
	if err != nil {
		return "", err
	}
	plaintext, ok := secret.Data["plaintext"].(string)
	if !ok {
		return "", fmt.Errorf("invalid response from vault")
	}
	return plaintext, nil
}

// Ensure Client implements domain.VaultPort
var _ domain.VaultPort = (*Client)(nil)
