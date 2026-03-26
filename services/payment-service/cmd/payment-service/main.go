package main

import (
	"context"
	"log"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	paymentv1 "fincore/gen/go/payment/v1"
	"fincore/pkg/postgres"
	"fincore/pkg/secrets"
	"fincore/pkg/security"
	"fincore/pkg/security/middleware"
	"fincore/services/payment-service/application/commands"
	paymentgrpc "fincore/services/payment-service/infrastructure/grpc"
	paymentpg "fincore/services/payment-service/infrastructure/postgres"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func maybeLoadPaymentDBDSNFromVault(ctx context.Context) (string, bool, error) {
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
	secretPath := strings.TrimSpace(os.Getenv("VAULT_PAYMENT_DB_DSN_SECRET_PATH"))
	if secretPath == "" {
		secretPath = "payment"
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

	dsn := os.Getenv("PAYMENT_DB_DSN")
	if strings.TrimSpace(dsn) == "" {
		if v, ok, err := maybeLoadPaymentDBDSNFromVault(ctx); err != nil {
			log.Fatalf("failed to load payment db dsn from vault: %v", err)
		} else if ok {
			dsn = v
			log.Printf("loaded payment db dsn from vault")
		}
	}

	grpcAddr := os.Getenv("PAYMENT_LISTEN_ADDR")
	if grpcAddr == "" {
		grpcAddr = ":50055"
	}
	httpAddr := os.Getenv("PAYMENT_HTTP_ADDR")
	if httpAddr == "" {
		httpAddr = ":8085"
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

	uow := paymentpg.NewUnitOfWork(pool)
	initiate := commands.NewInitiatePaymentHandler(uow)
	authorize := commands.NewAuthorizePaymentHandler(uow)
	settle := commands.NewSettlePaymentHandler(uow)
	fail := commands.NewFailPaymentHandler(uow)
	q := paymentpg.NewPaymentQuery(pool)

	lis, err := net.Listen("tcp", grpcAddr)
	if err != nil {
		panic(err)
	}

	authInterceptor := middleware.UnaryAuthzInterceptor(tokens, map[string]string{
		"/InitiatePayment":  "payment:write",
		"/AuthorizePayment": "payment:write",
		"/SettlePayment":    "payment:write",
		"/FailPayment":      "payment:write",
		"/GetPayment":       "payment:read",
	})
	serverOpts := []grpc.ServerOption{grpc.UnaryInterceptor(authInterceptor)}
	if creds, closeSrc, err := security.NewSpiffeMTLSServerCredentials(ctx); err == nil {
		defer closeSrc()
		serverOpts = append(serverOpts, grpc.Creds(creds))
		log.Printf("SPIFFE mTLS enabled for gRPC server")
	}

	gs := grpc.NewServer(serverOpts...)
	paymentv1.RegisterPaymentServiceServer(gs, paymentgrpc.NewServer(initiate, authorize, settle, fail, q))

	go func() {
		log.Printf("Starting gRPC server on %s", grpcAddr)
		if err := gs.Serve(lis); err != nil {
			log.Fatalf("failed to serve gRPC: %v", err)
		}
	}()

	mux := runtime.NewServeMux(middleware.GatewayAuthHeaderForwarder())
	opts := []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())}
	if err := paymentv1.RegisterPaymentServiceHandlerFromEndpoint(ctx, mux, grpcAddr, opts); err != nil {
		log.Fatalf("failed to register gateway: %v", err)
	}

	h := http.NewServeMux()
	h.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	h.HandleFunc("/readyz", func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()
		if err := pool.Ping(ctx); err != nil {
			http.Error(w, "not ready", http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	h.Handle("/", mux)

	log.Printf("Starting HTTP gateway on %s", httpAddr)
	if err := http.ListenAndServe(httpAddr, h); err != nil {
		log.Fatalf("failed to serve HTTP: %v", err)
	}
}
