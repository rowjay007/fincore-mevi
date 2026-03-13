package middleware

import (
	"context"
	"testing"
	"time"

	"fincore/pkg/security"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

type stubMaker struct {
	payload security.TokenPayload
	err     error
}

func (s stubMaker) CreateToken(payload security.TokenPayload) (string, error) { return "", nil }
func (s stubMaker) VerifyToken(token string) (*security.TokenPayload, error) {
	if s.err != nil {
		return nil, s.err
	}
	return &s.payload, nil
}

func TestUnaryAuthzInterceptor_AllowsWithPermission(t *testing.T) {
	m := stubMaker{payload: security.TokenPayload{UserID: "u1", Permissions: []string{"account:read"}, IssuedAt: time.Now(), ExpiredAt: time.Now().Add(time.Minute)}}
	interceptor := UnaryAuthzInterceptor(m, map[string]string{"/GetAccount": "account:read"})

	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs("authorization", "Bearer token"))
	info := &grpc.UnaryServerInfo{FullMethod: "/fincore.account.v1.AccountService/GetAccount"}

	called := false
	_, err := interceptor(ctx, nil, info, func(ctx context.Context, req any) (any, error) {
		called = true
		return "ok", nil
	})
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if !called {
		t.Fatalf("expected handler called")
	}
}

func TestUnaryAuthzInterceptor_DeniesWithoutPermission(t *testing.T) {
	m := stubMaker{payload: security.TokenPayload{UserID: "u1", Permissions: []string{"account:read"}, IssuedAt: time.Now(), ExpiredAt: time.Now().Add(time.Minute)}}
	interceptor := UnaryAuthzInterceptor(m, map[string]string{"/Deposit": "account:write"})

	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs("authorization", "Bearer token"))
	info := &grpc.UnaryServerInfo{FullMethod: "/fincore.account.v1.AccountService/Deposit"}

	_, err := interceptor(ctx, nil, info, func(ctx context.Context, req any) (any, error) {
		return "ok", nil
	})
	if err == nil {
		t.Fatalf("expected error")
	}
	if err != security.ErrInvalidToken {
		t.Fatalf("expected ErrInvalidToken, got %v", err)
	}
}
