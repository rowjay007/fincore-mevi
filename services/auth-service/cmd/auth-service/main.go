package main

import (
	"context"
	"crypto/ed25519"
	"encoding/json"
	"log"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	authv1 "fincore/gen/go/auth/v1"
	"fincore/pkg/postgres"
	"fincore/pkg/security"
	authgrpc "fincore/services/auth-service/infrastructure/grpc"
)

func main() {
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	dsn := os.Getenv("AUTH_DB_DSN")
	jwtKid := strings.TrimSpace(os.Getenv("AUTH_JWT_KID"))
	jwtEd25519Priv := strings.TrimSpace(os.Getenv("AUTH_JWT_ED25519_PRIVATE_KEY"))

	grpcAddr := os.Getenv("AUTH_GRPC_ADDR")
	if grpcAddr == "" {
		grpcAddr = ":50055"
	}
	httpAddr := os.Getenv("AUTH_HTTP_ADDR")
	if httpAddr == "" {
		httpAddr = ":8082"
	}

	accessTTL := 15 * time.Minute
	refreshTTL := 30 * 24 * time.Hour

	pool, err := postgres.NewPool(ctx, postgres.Config{DSN: dsn})
	if err != nil {
		log.Fatalf("failed to connect to db: %v", err)
	}
	defer pool.Close()

	priv, err := security.ParseEd25519PrivateKeyBase64URL(jwtEd25519Priv)
	if err != nil {
		log.Fatalf("failed to parse ed25519 private key: %v", err)
	}
	tokens, err := security.NewEd25519JWTMaker(jwtKid, ed25519.PrivateKey(priv))
	if err != nil {
		log.Fatalf("failed to create token maker: %v", err)
	}
	pub := ed25519.PrivateKey(priv).Public().(ed25519.PublicKey)
	jwk, err := security.Ed25519PublicJWK(jwtKid, pub)
	if err != nil {
		log.Fatalf("failed to build jwk: %v", err)
	}
	jwks := security.JWKS{Keys: []security.JWK{jwk}}

	srv := authgrpc.NewServer(pool, tokens, accessTTL, refreshTTL)

	lis, err := net.Listen("tcp", grpcAddr)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	g := grpc.NewServer()
	authv1.RegisterAuthServiceServer(g, srv)

	go func() {
		log.Printf("Starting gRPC server on %s", grpcAddr)
		if err := g.Serve(lis); err != nil {
			log.Fatalf("failed to serve gRPC: %v", err)
		}
	}()

	mux := runtime.NewServeMux()
	opts := []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())}
	if err := authv1.RegisterAuthServiceHandlerFromEndpoint(ctx, mux, grpcAddr, opts); err != nil {
		log.Fatalf("failed to register gateway: %v", err)
	}

	h := http.NewServeMux()
	h.Handle("/", mux)
	h.HandleFunc("/jwks.json", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(jwks)
	})

	log.Printf("Starting HTTP gateway on %s", httpAddr)
	if err := http.ListenAndServe(httpAddr, h); err != nil {
		log.Fatalf("failed to serve HTTP: %v", err)
	}
}
