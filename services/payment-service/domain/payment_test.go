package domain

import (
	"testing"

	"fincore/pkg/ids"
	"fincore/pkg/money"
)

func TestNewPayment_ValidatesBasics(t *testing.T) {
	amt, err := money.New(100, money.NGN)
	if err != nil {
		t.Fatalf("money: %v", err)
	}

	from := ids.New()
	to := ids.New()
	if from == to {
		t.Fatalf("expected distinct ids")
	}

	p, err := NewPayment(from, to, amt, "test")
	if err != nil {
		t.Fatalf("expected success, got %v", err)
	}
	if p.ID() == "" {
		t.Fatalf("expected id")
	}
	if p.Status() != StatusInitiated {
		t.Fatalf("expected initiated, got %s", p.Status())
	}
	if p.Version() != 1 {
		t.Fatalf("expected version 1, got %d", p.Version())
	}
}

func TestNewPayment_RejectsSameAccount(t *testing.T) {
	amt, _ := money.New(100, money.NGN)
	id := ids.New()
	if _, err := NewPayment(id, id, amt, ""); err == nil {
		t.Fatalf("expected error")
	}
}
