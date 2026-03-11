package money

import (
	"testing"
)

func TestMoney_Add(t *testing.T) {
	m1 := MustNew(100, NGN)
	m2 := MustNew(50, NGN)

	res, err := m1.Add(m2)
	if err != nil {
		t.Fatalf("Add failed: %v", err)
	}
	if res.AmountKobo() != 150 {
		t.Errorf("expected 150, got %d", res.AmountKobo())
	}
}

func TestMoney_Sub(t *testing.T) {
	m1 := MustNew(100, NGN)
	m2 := MustNew(30, NGN)

	res, err := m1.Sub(m2)
	if err != nil {
		t.Fatalf("Sub failed: %v", err)
	}
	if res.AmountKobo() != 70 {
		t.Errorf("expected 70, got %d", res.AmountKobo())
	}
}

func TestMoney_CurrencyMismatch(t *testing.T) {
	m1 := MustNew(100, NGN)
	m2, _ := New(100, "USD")

	_, err := m1.Add(m2)
	if err == nil {
		t.Error("expected error for currency mismatch in Add")
	}

	_, err = m1.Sub(m2)
	if err == nil {
		t.Error("expected error for currency mismatch in Sub")
	}
}

func TestMoney_Negative(t *testing.T) {
	m := MustNew(-10, NGN)
	if !m.IsNegative() {
		t.Error("expected IsNegative to be true")
	}
	if !m.IsNegativeOrZero() {
		t.Error("expected IsNegativeOrZero to be true")
	}
}
