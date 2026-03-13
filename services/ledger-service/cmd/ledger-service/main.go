package main

import (
	"context"
	"log"
	"net"
	"net/http"
	"os"
	"strings"

	ledgerv1 "fincore/gen/go/ledger/v1"
	"fincore/pkg/postgres"
	"fincore/pkg/security"
	"fincore/services/ledger-service/application/commands"
	ledgergrpc "fincore/services/ledger-service/infrastructure/grpc"
	ledgerpg "fincore/services/ledger-service/infrastructure/postgres"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
)

func main() {
	ctx := context.Background()

	dsn := os.Getenv("LEDGER_DB_DSN")
	addr := os.Getenv("LEDGER_LISTEN_ADDR")
	if addr == "" {
		addr = ":50053"
	}
	httpAddr := os.Getenv("LEDGER_HTTP_ADDR")
	if httpAddr == "" {
		httpAddr = ":8083"
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

	go func() {
		log.Printf("Starting gRPC server on %s", addr)
		if err := s.Serve(l); err != nil {
			log.Fatalf("failed to serve gRPC: %v", err)
		}
	}()

	mux := runtime.NewServeMux(runtime.WithIncomingHeaderMatcher(func(key string) (string, bool) {
		if strings.EqualFold(key, "Authorization") {
			return "authorization", true
		}
		return runtime.DefaultHeaderMatcher(key)
	}))
	opts := []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())}
	if err := ledgerv1.RegisterLedgerServiceHandlerFromEndpoint(ctx, mux, addr, opts); err != nil {
		log.Fatalf("failed to register gateway: %v", err)
	}

	log.Printf("Starting HTTP gateway on %s", httpAddr)
	if err := http.ListenAndServe(httpAddr, mux); err != nil {
		log.Fatalf("failed to serve HTTP: %v", err)
	}
}
