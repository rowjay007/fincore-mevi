package postgres

import (
	"context"
	"testing"

	"fincore/pkg/ids"
	"fincore/pkg/postgres"
)

func TestBalanceRepo_ApplyDelta(t *testing.T) {
	// This test requires a running postgres instance.
	// For CI/CD, we'd use testcontainers-go or a mock.
	// For now, this serves as a template for integration tests.
	t.Skip("skipping integration test needing postgres")

	ctx := context.Background()
	pool, _ := postgres.NewPool(ctx, postgres.Config{DSN: "postgres://user:password@localhost:5432/fincore?sslmode=disable"})
	defer pool.Close()

	tx, _ := pool.Begin(ctx)
	defer tx.Rollback(ctx)

	repo := NewBalanceRepo(tx)
	accountID := ids.New().String()

	// Initial deposit
	err := repo.ApplyDelta(ctx, accountID, 1000)
	if err != nil {
		t.Fatalf("ApplyDelta failed: %v", err)
	}

	bal, _ := repo.GetBalanceKobo(ctx, accountID)
	if bal != 1000 {
		t.Errorf("expected 1000, got %d", bal)
	}

	// Update
	repo.ApplyDelta(ctx, accountID, -200)
	bal, _ = repo.GetBalanceKobo(ctx, accountID)
	if bal != 800 {
		t.Errorf("expected 800, got %d", bal)
	}
}
