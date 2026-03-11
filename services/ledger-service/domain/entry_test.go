package domain

import (
	"testing"
	"time"
	"fincore/pkg/ids"
	"fincore/pkg/money"
)

func TestNewEntry_Valid(t *testing.T) {
	accountID := ids.New()
	amt := money.MustNew(1000, money.NGN)
	narration := "Test deposit"
	
	entry, err := NewEntry(accountID, EntryTypeDeposit, amt, narration, time.Now().UTC())
	if err != nil {
		t.Fatalf("failed to create entry: %v", err)
	}

	if entry.AccountID() != accountID {
		t.Error("account id mismatch")
	}
	if entry.Amount().AmountKobo() != 1000 {
		t.Error("amount mismatch")
	}
}

func TestNewEntry_InvalidCurrency(t *testing.T) {
	amt, _ := money.New(1000, "USD")
	_, err := NewEntry(ids.New(), EntryTypeDeposit, amt, "fail", time.Now().UTC())
	if err == nil || err != ErrCurrencyNotNGN {
		t.Errorf("expected ErrCurrencyNotNGN, got %v", err)
	}
}
