package grpc

import (
	"context"
	"testing"
	"time"

	"fincore/pkg/security"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type stubTokenMaker struct {
	payload *security.TokenPayload
	err     error
}

func (s stubTokenMaker) CreateToken(payload security.TokenPayload) (string, error) { return "", nil }
func (s stubTokenMaker) VerifyToken(token string) (*security.TokenPayload, error) {
	if s.err != nil {
		return nil, s.err
	}
	return s.payload, nil
}

func TestNormalizeEmail(t *testing.T) {
	_, err := normalizeEmail(" ")
	if err == nil {
		t.Fatalf("expected error")
	}
	_, err = normalizeEmail("not-an-email")
	if err == nil {
		t.Fatalf("expected error")
	}
	got, err := normalizeEmail("  TEST@Example.com  ")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "test@example.com" {
		t.Fatalf("expected normalized email, got %q", got)
	}
}

func TestClientMetaFromContext(t *testing.T) {
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(
		"user-agent", " ua ",
		"x-forwarded-for", " 1.2.3.4, 5.6.7.8 ",
	))
	ua, ip := clientMetaFromContext(ctx)
	if ua != "ua" {
		t.Fatalf("expected ua, got %q", ua)
	}
	if ip != "1.2.3.4" {
		t.Fatalf("expected ip, got %q", ip)
	}

	ua, ip = clientMetaFromContext(context.Background())
	if ua != "" || ip != "" {
		t.Fatalf("expected empty meta")
	}
}

func TestNullIfEmpty(t *testing.T) {
	if nullIfEmpty("  ") != nil {
		t.Fatalf("expected nil")
	}
	if got := nullIfEmpty("x"); got != "x" {
		t.Fatalf("expected x, got %v", got)
	}
}

func TestNewRefreshToken(t *testing.T) {
	a, err := newRefreshToken()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	b, err := newRefreshToken()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if a == "" || b == "" {
		t.Fatalf("expected non-empty tokens")
	}
	if a == b {
		t.Fatalf("expected tokens to differ")
	}
}

func TestRequirePermissionFromAuthHeader(t *testing.T) {
	s := &Server{tokens: stubTokenMaker{payload: &security.TokenPayload{UserID: "u1", Permissions: []string{"auth:admin"}, IssuedAt: time.Now(), ExpiredAt: time.Now().Add(time.Minute)}}}
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs("authorization", "Bearer t"))
	if err := s.requirePermissionFromAuthHeader(ctx, "auth:admin"); err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	s = &Server{tokens: stubTokenMaker{payload: &security.TokenPayload{UserID: "u1", Permissions: []string{"x"}, IssuedAt: time.Now(), ExpiredAt: time.Now().Add(time.Minute)}}}
	err := s.requirePermissionFromAuthHeader(ctx, "auth:admin")
	if err == nil {
		t.Fatalf("expected error")
	}
	st, ok := status.FromError(err)
	if !ok {
		t.Fatalf("expected status error")
	}
	if st.Code() != codes.PermissionDenied {
		t.Fatalf("expected PermissionDenied, got %v", st.Code())
	}

	s = &Server{tokens: stubTokenMaker{err: security.ErrInvalidToken}}
	err = s.requirePermissionFromAuthHeader(ctx, "auth:admin")
	if err == nil {
		t.Fatalf("expected error")
	}
	st, ok = status.FromError(err)
	if !ok {
		t.Fatalf("expected status error")
	}
	if st.Code() != codes.Unauthenticated {
		t.Fatalf("expected Unauthenticated, got %v", st.Code())
	}

	s = &Server{tokens: stubTokenMaker{payload: &security.TokenPayload{UserID: "u1", Permissions: []string{"auth:admin"}}}}
	err = s.requirePermissionFromAuthHeader(context.Background(), "auth:admin")
	if err == nil {
		t.Fatalf("expected error")
	}
	st, ok = status.FromError(err)
	if !ok {
		t.Fatalf("expected status error")
	}
	if st.Code() != codes.Unauthenticated {
		t.Fatalf("expected Unauthenticated, got %v", st.Code())
	}
}
