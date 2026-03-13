package main

import (
	"context"
	"net"
	"os"
	"strings"

	ledgerv1 "fincore/gen/go/ledger/v1"
	"fincore/pkg/postgres"
	"fincore/pkg/security"
	"fincore/services/ledger-service/application/commands"
	ledgergrpc "fincore/services/ledger-service/infrastructure/grpc"
	ledgerpg "fincore/services/ledger-service/infrastructure/postgres"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

func main() {
	ctx := context.Background()

	dsn := os.Getenv("LEDGER_DB_DSN")
	addr := os.Getenv("LEDGER_LISTEN_ADDR")
	if addr == "" {
		addr = ":50053"
	}

	jwts := os.Getenv("AUTH_JWT_SECRET")
	tokens, err := security.NewJWTMaker(jwts)
	if err != nil {
		panic(err)
	}

	pool, err := postgres.NewPool(ctx, postgres.Config{DSN: dsn})
	if err != nil {
		panic(err)
	}
	defer pool.Close()

	uow := ledgerpg.NewUnitOfWork(pool)
	post := commands.NewPostEntryHandler(uow)

	balQuery := ledgerpg.NewBalanceQuery(pool)

	l, err := net.Listen("tcp", addr)
	if err != nil {
		panic(err)
	}

	authInterceptor := func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		requiresAuth := false
		requiredPerm := ""
		switch {
		case strings.HasSuffix(info.FullMethod, "/PostEntry"):
			requiresAuth = true
			requiredPerm = "ledger:write"
		case strings.HasSuffix(info.FullMethod, "/GetBalance"):
			requiresAuth = true
			requiredPerm = "ledger:read"
		}
		if !requiresAuth {
			return handler(ctx, req)
		}

		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			return nil, security.ErrInvalidToken
		}
		vals := md.Get("authorization")
		if len(vals) == 0 {
			return nil, security.ErrInvalidToken
		}
		v := strings.TrimSpace(vals[0])
		const prefix = "Bearer "
		if !strings.HasPrefix(v, prefix) {
			return nil, security.ErrInvalidToken
		}
		tok := strings.TrimSpace(strings.TrimPrefix(v, prefix))
		if tok == "" {
			return nil, security.ErrInvalidToken
		}
		payload, err := tokens.VerifyToken(tok)
		if err != nil {
			return nil, err
		}
		if requiredPerm != "" {
			allowed := false
			for _, p := range payload.Permissions {
				if p == requiredPerm {
					allowed = true
					break
				}
			}
			if !allowed {
				return nil, security.ErrInvalidToken
			}
		}
		return handler(ctx, req)
	}

	s := grpc.NewServer(grpc.UnaryInterceptor(authInterceptor))
	ledgerv1.RegisterLedgerServiceServer(s, ledgergrpc.NewServer(post, balQuery))
	if err := s.Serve(l); err != nil {
		panic(err)
	}
}
