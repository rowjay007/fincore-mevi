package vault

import (
	"context"
	"fmt"

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

func (c *Client) Encrypt(ctx context.Context, path string, data string) (string, error) {
	// Uses Vault Transit Secret Engine for FPE/Tokenization
	secret, err := c.client.Logical().WriteWithContext(ctx, fmt.Sprintf("transit/encrypt/%s", path), map[string]interface{}{
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

func (c *Client) Decrypt(ctx context.Context, path string, ciphertext string) (string, error) {
	secret, err := c.client.Logical().WriteWithContext(ctx, fmt.Sprintf("transit/decrypt/%s", path), map[string]interface{}{
		"ciphertext": ciphertext,
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
