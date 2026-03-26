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

func TestPaymentLifecycle_AuthorizeSettle(t *testing.T) {
	amt, _ := money.New(100, money.NGN)
	p, err := NewPayment(ids.New(), ids.New(), amt, "")
	if err != nil {
		t.Fatalf("new payment: %v", err)
	}
	if err := p.Authorize(); err != nil {
		t.Fatalf("authorize: %v", err)
	}
	if p.Status() != StatusAuthorized {
		t.Fatalf("expected authorized")
	}
	if err := p.Settle(); err != nil {
		t.Fatalf("settle: %v", err)
	}
	if p.Status() != StatusSettled {
		t.Fatalf("expected settled")
	}
}

func TestPaymentLifecycle_Fail(t *testing.T) {
	amt, _ := money.New(100, money.NGN)
	p, err := NewPayment(ids.New(), ids.New(), amt, "")
	if err != nil {
		t.Fatalf("new payment: %v", err)
	}
	if err := p.Fail("declined"); err != nil {
		t.Fatalf("fail: %v", err)
	}
	if p.Status() != StatusFailed {
		t.Fatalf("expected failed")
	}
}
