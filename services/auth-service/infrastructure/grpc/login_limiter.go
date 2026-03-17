package grpc

import (
	"sync"
	"time"
)

type loginLimiter struct {
	mu          sync.Mutex
	maxAttempts int
	window      time.Duration
	lockout     time.Duration
	states      map[string]*loginState
}

type loginState struct {
	fails        []time.Time
	lockedUntil  time.Time
	lastSeenTime time.Time
}

func newLoginLimiter(maxAttempts int, window time.Duration, lockout time.Duration) *loginLimiter {
	if maxAttempts <= 0 {
		maxAttempts = 5
	}
	if window <= 0 {
		window = 5 * time.Minute
	}
	if lockout < 0 {
		lockout = 0
	}
	return &loginLimiter{maxAttempts: maxAttempts, window: window, lockout: lockout, states: make(map[string]*loginState)}
}

func (l *loginLimiter) allow(now time.Time, key string) (allowed bool, lockedUntil time.Time) {
	l.mu.Lock()
	defer l.mu.Unlock()

	s := l.states[key]
	if s == nil {
		s = &loginState{}
		l.states[key] = s
	}
	s.lastSeenTime = now

	if !s.lockedUntil.IsZero() && now.Before(s.lockedUntil) {
		return false, s.lockedUntil
	}

	cutoff := now.Add(-l.window)
	filtered := s.fails[:0]
	for _, t := range s.fails {
		if t.After(cutoff) {
			filtered = append(filtered, t)
		}
	}
	s.fails = filtered

	if len(s.fails) >= l.maxAttempts {
		if l.lockout > 0 {
			s.lockedUntil = now.Add(l.lockout)
			return false, s.lockedUntil
		}
		return false, time.Time{}
	}

	return true, time.Time{}
}

func (l *loginLimiter) onFailure(now time.Time, key string) {
	l.mu.Lock()
	defer l.mu.Unlock()

	s := l.states[key]
	if s == nil {
		s = &loginState{}
		l.states[key] = s
	}
	s.lastSeenTime = now

	cutoff := now.Add(-l.window)
	filtered := s.fails[:0]
	for _, t := range s.fails {
		if t.After(cutoff) {
			filtered = append(filtered, t)
		}
	}
	s.fails = append(filtered, now)

	if len(s.fails) >= l.maxAttempts && l.lockout > 0 {
		s.lockedUntil = now.Add(l.lockout)
	}
}

func (l *loginLimiter) onSuccess(key string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	delete(l.states, key)
}
