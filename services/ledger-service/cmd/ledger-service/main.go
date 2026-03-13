package main

import (
	"context"
	"log"
	"net"
	"net/http"
	"os"

	ledgerv1 "fincore/gen/go/ledger/v1"
	"fincore/pkg/postgres"
	"fincore/pkg/security"
	"fincore/pkg/security/middleware"
	"fincore/services/ledger-service/application/commands"
	ledgergrpc "fincore/services/ledger-service/infrastructure/grpc"
	ledgerpg "fincore/services/ledger-service/infrastructure/postgres"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
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

	authInterceptor := middleware.UnaryAuthzInterceptor(tokens, map[string]string{
		"/PostEntry":  "ledger:write",
		"/GetBalance": "ledger:read",
	})

	s := grpc.NewServer(grpc.UnaryInterceptor(authInterceptor))
	ledgerv1.RegisterLedgerServiceServer(s, ledgergrpc.NewServer(post, balQuery))

	go func() {
		log.Printf("Starting gRPC server on %s", addr)
		if err := s.Serve(l); err != nil {
			log.Fatalf("failed to serve gRPC: %v", err)
		}
	}()

	mux := runtime.NewServeMux(middleware.GatewayAuthHeaderForwarder())
	opts := []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())}
	if err := ledgerv1.RegisterLedgerServiceHandlerFromEndpoint(ctx, mux, addr, opts); err != nil {
		log.Fatalf("failed to register gateway: %v", err)
	}

	log.Printf("Starting HTTP gateway on %s", httpAddr)
	if err := http.ListenAndServe(httpAddr, mux); err != nil {
		log.Fatalf("failed to serve HTTP: %v", err)
	}
}
