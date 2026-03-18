package main

import (
	"context"
	"log"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	accountv1 "fincore/gen/go/account/v1"
	ledgerv1 "fincore/gen/go/ledger/v1"
	"fincore/pkg/postgres"
	"fincore/pkg/security"
	"fincore/pkg/security/middleware"
	"fincore/services/account-service/application/commands"
	accountgrpc "fincore/services/account-service/infrastructure/grpc"
	accountpg "fincore/services/account-service/infrastructure/postgres"
)

func main() {
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	dsn := os.Getenv("ACCOUNT_DB_DSN")
	ledgerAddr := os.Getenv("LEDGER_ADDR")
	grpcAddr := os.Getenv("ACCOUNT_GRPC_ADDR")
	if grpcAddr == "" {
		grpcAddr = ":50051"
	}
	httpAddr := os.Getenv("ACCOUNT_HTTP_ADDR")
	if httpAddr == "" {
		httpAddr = ":8080"
	}

	jwksURL := os.Getenv("AUTH_JWKS_URL")
	tokens, err := security.NewJWKSVerifier(jwksURL, 5*time.Minute)
	if err != nil {
		log.Fatalf("failed to create token maker: %v", err)
	}

	// 1. Database & Infrastructure
	pool, err := postgres.NewPool(ctx, postgres.Config{DSN: dsn})
	if err != nil {
		log.Fatalf("failed to connect to db: %v", err)
	}
	defer pool.Close()

	// 2. Ledger Client (Sync gRPC)
	dialCreds := grpc.WithTransportCredentials(insecure.NewCredentials())
	if creds, closeSrc, err := security.NewSpiffeMTLSClientCredentials(ctx); err == nil {
		defer closeSrc()
		dialCreds = grpc.WithTransportCredentials(creds)
		log.Printf("SPIFFE mTLS enabled for ledger dial")
	}
	lconn, err := grpc.NewClient(ledgerAddr, dialCreds)
	if err != nil {
		log.Fatalf("failed to connect to ledger: %v", err)
	}
	defer lconn.Close()
	ledgerClient, err := accountgrpc.NewLedgerClient(ledgerv1.NewLedgerServiceClient(lconn))
	if err != nil {
		log.Fatalf("failed to create ledger client: %v", err)
	}

	// 3. Handlers
	uow := accountpg.NewUnitOfWork(pool)
	openHandler := commands.NewOpenAccountHandler(uow)
	depositHandler := commands.NewDepositMoneyHandler(uow, ledgerClient)
	withdrawHandler := commands.NewWithdrawMoneyHandler(uow, ledgerClient)

	// 4. Start gRPC Server
	lis, err := net.Listen("tcp", grpcAddr)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	authInterceptor := middleware.UnaryAuthzInterceptor(tokens, map[string]string{
		"/OpenAccount": "account:write",
		"/Deposit":     "account:write",
		"/Withdraw":    "account:write",
		"/GetAccount":  "account:read",
	})

	s := grpc.NewServer(grpc.UnaryInterceptor(authInterceptor))
	accountv1.RegisterAccountServiceServer(s, accountgrpc.NewServer(
		openHandler,
		depositHandler,
		withdrawHandler,
		ledgerClient,
		uow,
	))

	go func() {
		log.Printf("Starting gRPC server on %s", grpcAddr)
		if err := s.Serve(lis); err != nil {
			log.Fatalf("failed to serve gRPC: %v", err)
		}
	}()

	// 5. Start HTTP Gateway
	mux := runtime.NewServeMux(middleware.GatewayAuthHeaderForwarder())
	opts := []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())}
	err = accountv1.RegisterAccountServiceHandlerFromEndpoint(ctx, mux, grpcAddr, opts)
	if err != nil {
		log.Fatalf("failed to register gateway: %v", err)
	}

	log.Printf("Starting HTTP gateway on %s", httpAddr)
	if err := http.ListenAndServe(httpAddr, mux); err != nil {
		log.Fatalf("failed to serve HTTP: %v", err)
	}
}
