package grpc

import (
	"testing"
	"time"
)

func TestLoginLimiter_LockoutAfterMaxAttempts(t *testing.T) {
	l := newLoginLimiter(3, 10*time.Minute, 15*time.Minute)
	key := "alice@example.com|1.2.3.4"
	now := time.Now().UTC()

	allowed, _ := l.allow(now, key)
	if !allowed {
		t.Fatalf("expected allowed")
	}

	l.onFailure(now, key)
	l.onFailure(now.Add(1*time.Second), key)
	l.onFailure(now.Add(2*time.Second), key)

	allowed, until := l.allow(now.Add(3*time.Second), key)
	if allowed {
		t.Fatalf("expected denied")
	}
	if until.IsZero() {
		t.Fatalf("expected lockout until")
	}

	allowed, _ = l.allow(until.Add(-1*time.Second), key)
	if allowed {
		t.Fatalf("expected still locked")
	}

	allowed, _ = l.allow(until.Add(1*time.Second), key)
	if !allowed {
		t.Fatalf("expected allowed after lockout expires")
	}
}

func TestLoginLimiter_WindowDropsOldFailures(t *testing.T) {
	l := newLoginLimiter(3, 2*time.Minute, 15*time.Minute)
	key := "alice@example.com|1.2.3.4"
	now := time.Now().UTC()

	l.onFailure(now.Add(-3*time.Minute), key)
	l.onFailure(now.Add(-2*time.Minute-1*time.Second), key)
	l.onFailure(now.Add(-1*time.Minute), key)

	allowed, _ := l.allow(now, key)
	if !allowed {
		t.Fatalf("expected allowed because old failures dropped")
	}
}

func TestLoginLimiter_SuccessResetsState(t *testing.T) {
	l := newLoginLimiter(2, 10*time.Minute, 15*time.Minute)
	key := "alice@example.com|1.2.3.4"
	now := time.Now().UTC()

	l.onFailure(now, key)
	l.onSuccess(key)

	allowed, _ := l.allow(now.Add(1*time.Second), key)
	if !allowed {
		t.Fatalf("expected allowed")
	}
}
