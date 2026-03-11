package domain

import (
	"testing"
	"fincore/pkg/ids"
)

func TestNewAccount(t *testing.T) {
	customerID := ids.New()
	acc, err := NewAccount(customerID)
	if err != nil {
		t.Fatalf("failed to create account: %v", err)
	}

	if acc.ID() == "" {
		t.Error("expected account id, got empty")
	}
	if acc.CustomerID() != customerID {
		t.Errorf("expected customer id %v, got %v", customerID, acc.CustomerID())
	}
	if acc.Status() != StatusActive {
		t.Errorf("expected status %v, got %v", StatusActive, acc.Status())
	}
	if acc.Version() != 1 {
		t.Errorf("expected version 1, got %v", acc.Version())
	}
}

func TestAccount_FreezeClose(t *testing.T) {
	acc, _ := NewAccount(ids.New())

	if err := acc.Freeze(); err != nil {
		t.Errorf("freeze failed: %v", err)
	}
	if acc.Status() != StatusFrozen {
		t.Error("expected status frozen")
	}

	// Cannot freeze frozen
	if err := acc.Freeze(); err == nil {
		t.Error("expected error when freezing frozen account")
	}

	// Close from frozen not allowed in our current domain rules (must be active)
	if err := acc.Close(); err == nil {
		t.Error("expected error when closing frozen account")
	}
}
