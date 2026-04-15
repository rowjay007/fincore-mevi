package domain

import (
	"context"
	"time"
)

// Token represents a sensitive data token reference.
type Token struct {
	ID        string
	Category  string
	CreatedAt time.Time
}

// VaultPort defines the interface for secure data operations.
type VaultPort interface {
	Tokenize(ctx context.Context, category string, data string) (string, error)
	Detokenize(ctx context.Context, token string, reason string) (string, error)
}
