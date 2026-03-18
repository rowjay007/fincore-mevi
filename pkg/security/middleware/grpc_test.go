package middleware

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"net/url"
	"testing"
	"time"

	"fincore/pkg/security"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
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
	st, ok := status.FromError(err)
	if !ok {
		t.Fatalf("expected status error, got %v", err)
	}
	if st.Code() != codes.PermissionDenied {
		t.Fatalf("expected PermissionDenied, got %v", st.Code())
	}
}

func TestUnaryAuthzInterceptor_AllowsWhenNoAuthRequired(t *testing.T) {
	m := stubMaker{err: security.ErrInvalidToken}
	interceptor := UnaryAuthzInterceptor(m, map[string]string{"/GetAccount": "account:read"})

	ctx := context.Background()
	info := &grpc.UnaryServerInfo{FullMethod: "/fincore.account.v1.AccountService/Health"}

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

func TestUnaryAuthzInterceptor_DeniesMissingAuthMetadata(t *testing.T) {
	m := stubMaker{payload: security.TokenPayload{UserID: "u1", Permissions: []string{"account:read"}, IssuedAt: time.Now(), ExpiredAt: time.Now().Add(time.Minute)}}
	interceptor := UnaryAuthzInterceptor(m, map[string]string{"/GetAccount": "account:read"})

	ctx := context.Background()
	info := &grpc.UnaryServerInfo{FullMethod: "/fincore.account.v1.AccountService/GetAccount"}

	_, err := interceptor(ctx, nil, info, func(ctx context.Context, req any) (any, error) {
		return "ok", nil
	})
	st, ok := status.FromError(err)
	if !ok {
		t.Fatalf("expected status error, got %v", err)
	}
	if st.Code() != codes.Unauthenticated {
		t.Fatalf("expected Unauthenticated, got %v", st.Code())
	}
}

func TestUnaryAuthzInterceptor_AllowsGatewayHeadersWhenPeerIsGateway(t *testing.T) {
	m := stubMaker{err: security.ErrInvalidToken}
	interceptor := UnaryAuthzInterceptor(m, map[string]string{"/GetAccount": "account:read"})

	spiffeURI, err := url.Parse("spiffe://fincore.local/ns/default/sa/api-gateway")
	if err != nil {
		t.Fatalf("failed to parse spiffe uri: %v", err)
	}
	cert := &x509.Certificate{URIs: []*url.URL{spiffeURI}}
	ctx := peer.NewContext(context.Background(), &peer.Peer{AuthInfo: credentials.TLSInfo{State: tls.ConnectionState{Version: tls.VersionTLS12, PeerCertificates: []*x509.Certificate{cert}}}})
	ctx = metadata.NewIncomingContext(ctx, metadata.Pairs(
		"x-fincore-subject", "u1",
		"x-fincore-permissions", "account:read",
	))
	info := &grpc.UnaryServerInfo{FullMethod: "/fincore.account.v1.AccountService/GetAccount"}

	called := false
	_, err = interceptor(ctx, nil, info, func(ctx context.Context, req any) (any, error) {
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

func TestUnaryAuthzInterceptor_DeniesGatewayHeadersWhenPeerIsNotGateway(t *testing.T) {
	m := stubMaker{err: security.ErrInvalidToken}
	interceptor := UnaryAuthzInterceptor(m, map[string]string{"/GetAccount": "account:read"})

	spiffeURI, err := url.Parse("spiffe://fincore.local/ns/default/sa/not-gateway")
	if err != nil {
		t.Fatalf("failed to parse spiffe uri: %v", err)
	}
	cert := &x509.Certificate{URIs: []*url.URL{spiffeURI}}
	ctx := peer.NewContext(context.Background(), &peer.Peer{AuthInfo: credentials.TLSInfo{State: tls.ConnectionState{Version: tls.VersionTLS12, PeerCertificates: []*x509.Certificate{cert}}}})
	ctx = metadata.NewIncomingContext(ctx, metadata.Pairs(
		"x-fincore-subject", "u1",
		"x-fincore-permissions", "account:read",
	))
	info := &grpc.UnaryServerInfo{FullMethod: "/fincore.account.v1.AccountService/GetAccount"}

	_, err = interceptor(ctx, nil, info, func(ctx context.Context, req any) (any, error) {
		return "ok", nil
	})
	st, ok := status.FromError(err)
	if !ok {
		t.Fatalf("expected status error, got %v", err)
	}
	if st.Code() != codes.Unauthenticated {
		t.Fatalf("expected Unauthenticated, got %v", st.Code())
	}
}

func TestUnaryAuthzInterceptor_DeniesInvalidAuthorizationFormat(t *testing.T) {
	m := stubMaker{payload: security.TokenPayload{UserID: "u1", Permissions: []string{"account:read"}, IssuedAt: time.Now(), ExpiredAt: time.Now().Add(time.Minute)}}
	interceptor := UnaryAuthzInterceptor(m, map[string]string{"/GetAccount": "account:read"})

	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs("authorization", "Token abc"))
	info := &grpc.UnaryServerInfo{FullMethod: "/fincore.account.v1.AccountService/GetAccount"}

	_, err := interceptor(ctx, nil, info, func(ctx context.Context, req any) (any, error) {
		return "ok", nil
	})
	st, ok := status.FromError(err)
	if !ok {
		t.Fatalf("expected status error, got %v", err)
	}
	if st.Code() != codes.Unauthenticated {
		t.Fatalf("expected Unauthenticated, got %v", st.Code())
	}
}

func TestUnaryAuthzInterceptor_DeniesInvalidToken(t *testing.T) {
	m := stubMaker{err: security.ErrInvalidToken}
	interceptor := UnaryAuthzInterceptor(m, map[string]string{"/GetAccount": "account:read"})

	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs("authorization", "Bearer token"))
	info := &grpc.UnaryServerInfo{FullMethod: "/fincore.account.v1.AccountService/GetAccount"}

	_, err := interceptor(ctx, nil, info, func(ctx context.Context, req any) (any, error) {
		return "ok", nil
	})
	st, ok := status.FromError(err)
	if !ok {
		t.Fatalf("expected status error, got %v", err)
	}
	if st.Code() != codes.Unauthenticated {
		t.Fatalf("expected Unauthenticated, got %v", st.Code())
	}
}
