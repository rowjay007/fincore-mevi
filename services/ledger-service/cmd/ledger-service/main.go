package main

import (
	"context"
	"log"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	ledgerv1 "fincore/gen/go/ledger/v1"
	"fincore/pkg/postgres"
	"fincore/pkg/secrets"
	"fincore/pkg/security"
	"fincore/pkg/security/middleware"
	"fincore/services/ledger-service/application/commands"
	ledgergrpc "fincore/services/ledger-service/infrastructure/grpc"
	ledgerpg "fincore/services/ledger-service/infrastructure/postgres"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func maybeLoadLedgerDBDSNFromVault(ctx context.Context) (string, bool, error) {
	addr := strings.TrimSpace(os.Getenv("VAULT_ADDR"))
	token, ok, err := secrets.VaultTokenFromEnvOrFile()
	if err != nil {
		return "", false, err
	}
	if addr == "" || !ok {
		return "", false, nil
	}

	mount := strings.TrimSpace(os.Getenv("VAULT_KV_MOUNT"))
	if mount == "" {
		mount = "secret"
	}
	secretPath := strings.TrimSpace(os.Getenv("VAULT_LEDGER_DB_DSN_SECRET_PATH"))
	if secretPath == "" {
		secretPath = "ledger"
	}

	c, err := secrets.NewVaultKVClient(secrets.VaultKVClientConfig{Addr: addr, Token: token, KVV2Mount: mount})
	if err != nil {
		return "", false, err
	}
	data, err := c.ReadKVV2(ctx, secretPath)
	if err != nil {
		return "", false, err
	}

	dsn, _ := data["dsn"].(string)
	if strings.TrimSpace(dsn) == "" {
		return "", false, nil
	}
	return dsn, true, nil
}

func main() {
	ctx := context.Background()

	dsn := os.Getenv("LEDGER_DB_DSN")
	if strings.TrimSpace(dsn) == "" {
		if v, ok, err := maybeLoadLedgerDBDSNFromVault(ctx); err != nil {
			log.Fatalf("failed to load ledger db dsn from vault: %v", err)
		} else if ok {
			dsn = v
			log.Printf("loaded ledger db dsn from vault")
		}
	}
	addr := os.Getenv("LEDGER_LISTEN_ADDR")
	if addr == "" {
		addr = ":50053"
	}
	httpAddr := os.Getenv("LEDGER_HTTP_ADDR")
	if httpAddr == "" {
		httpAddr = ":8083"
	}

	jwksURL := os.Getenv("AUTH_JWKS_URL")
	tokens, err := security.NewJWKSVerifier(jwksURL, 5*time.Minute)
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

	serverOpts := []grpc.ServerOption{grpc.UnaryInterceptor(authInterceptor)}
	if creds, closeSrc, err := security.NewSpiffeMTLSServerCredentials(ctx); err == nil {
		defer closeSrc()
		serverOpts = append(serverOpts, grpc.Creds(creds))
		log.Printf("SPIFFE mTLS enabled for gRPC server")
	}

	s := grpc.NewServer(serverOpts...)
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
