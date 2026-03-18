package main

import (
	"context"
	"crypto/ed25519"
	"encoding/json"
	"log"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"

	authv1 "fincore/gen/go/auth/v1"
	"fincore/pkg/postgres"
	"fincore/pkg/secrets"
	"fincore/pkg/security"
	"fincore/pkg/security/middleware"
	authgrpc "fincore/services/auth-service/infrastructure/grpc"
)

type openIDConfiguration struct {
	Issuer                           string `json:"issuer"`
	JWKSURI                          string `json:"jwks_uri"`
	ResponseTypesSupported           []string
	SubjectTypesSupported            []string
	IDTokenSigningAlgValuesSupported []string
}

func newHTTPHandler(gw http.Handler, cfg openIDConfiguration, jwks security.JWKS, jwksPath string) *http.ServeMux {
	h := http.NewServeMux()
	h.Handle("/", gw)
	h.HandleFunc("/.well-known/openid-configuration", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(cfg)
	})
	h.HandleFunc(jwksPath, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(jwks)
	})
	h.HandleFunc("/jwks.json", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(jwks)
	})
	return h
}

func getenv(key string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return ""
}

func getenvAny(keys ...string) string {
	for _, k := range keys {
		if v := getenv(k); v != "" {
			return v
		}
	}
	return ""
}

type vaultIdentityJWTSecret struct {
	Kid               string
	Ed25519PrivateKey string
}

func maybeLoadIdentityJWTFromVault(ctx context.Context) (vaultIdentityJWTSecret, bool, error) {
	addr := strings.TrimSpace(os.Getenv("VAULT_ADDR"))
	token, ok, err := secrets.VaultTokenFromEnvOrFile()
	if err != nil {
		return vaultIdentityJWTSecret{}, false, err
	}
	if addr == "" || !ok {
		return vaultIdentityJWTSecret{}, false, nil
	}

	mount := strings.TrimSpace(os.Getenv("VAULT_KV_MOUNT"))
	if mount == "" {
		mount = "secret"
	}
	secretPath := strings.TrimSpace(os.Getenv("VAULT_IDENTITY_JWT_SECRET_PATH"))
	if secretPath == "" {
		secretPath = "identity"
	}

	c, err := secrets.NewVaultKVClient(secrets.VaultKVClientConfig{Addr: addr, Token: token, KVV2Mount: mount})
	if err != nil {
		return vaultIdentityJWTSecret{}, false, err
	}
	data, err := c.ReadKVV2(ctx, secretPath)
	if err != nil {
		return vaultIdentityJWTSecret{}, false, err
	}

	kid, _ := data["kid"].(string)
	priv, _ := data["jwt_ed25519_private_key"].(string)
	if strings.TrimSpace(kid) == "" || strings.TrimSpace(priv) == "" {
		return vaultIdentityJWTSecret{}, false, nil
	}
	return vaultIdentityJWTSecret{Kid: kid, Ed25519PrivateKey: priv}, true, nil
}

func main() {
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	dsn := getenvAny("IDENTITY_DB_DSN", "AUTH_DB_DSN")
	jwtKid := getenvAny("IDENTITY_JWT_KID", "AUTH_JWT_KID")
	jwtEd25519Priv := getenvAny("IDENTITY_JWT_ED25519_PRIVATE_KEY", "AUTH_JWT_ED25519_PRIVATE_KEY")
	if jwtKid == "" || jwtEd25519Priv == "" {
		if v, ok, err := maybeLoadIdentityJWTFromVault(ctx); err != nil {
			log.Fatalf("failed to load identity jwt secret from vault: %v", err)
		} else if ok {
			if jwtKid == "" {
				jwtKid = v.Kid
			}
			if jwtEd25519Priv == "" {
				jwtEd25519Priv = v.Ed25519PrivateKey
			}
			log.Printf("loaded identity jwt signing material from vault")
		}
	}
	if strings.TrimSpace(dsn) == "" {
		log.Fatalf("IDENTITY_DB_DSN (or AUTH_DB_DSN) is required")
	}
	if jwtKid == "" {
		log.Fatalf("IDENTITY_JWT_KID (or AUTH_JWT_KID) is required")
	}
	if jwtEd25519Priv == "" {
		log.Fatalf("IDENTITY_JWT_ED25519_PRIVATE_KEY (or AUTH_JWT_ED25519_PRIVATE_KEY) is required")
	}

	grpcAddr := getenvAny("IDENTITY_GRPC_ADDR", "AUTH_GRPC_ADDR")
	if grpcAddr == "" {
		grpcAddr = ":50056"
	}
	httpAddr := getenvAny("IDENTITY_HTTP_ADDR", "AUTH_HTTP_ADDR")
	if httpAddr == "" {
		httpAddr = ":8084"
	}

	issuer := getenvAny("IDENTITY_ISSUER", "AUTH_ISSUER")
	if issuer == "" {
		issuer = "http://localhost" + httpAddr
	}

	accessTTL := 15 * time.Minute
	refreshTTL := 30 * 24 * time.Hour

	loginMaxAttempts := 5
	if v := strings.TrimSpace(getenvAny("IDENTITY_LOGIN_MAX_ATTEMPTS", "AUTH_LOGIN_MAX_ATTEMPTS")); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			loginMaxAttempts = n
		}
	}
	loginWindow := 5 * time.Minute
	if v := strings.TrimSpace(getenvAny("IDENTITY_LOGIN_WINDOW", "AUTH_LOGIN_WINDOW")); v != "" {
		if d, err := time.ParseDuration(v); err == nil && d > 0 {
			loginWindow = d
		}
	}
	loginLockout := 15 * time.Minute
	if v := strings.TrimSpace(getenvAny("IDENTITY_LOGIN_LOCKOUT", "AUTH_LOGIN_LOCKOUT")); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			loginLockout = d
		}
	}
	cleanupInterval := 5 * time.Minute
	if v := strings.TrimSpace(getenvAny("IDENTITY_REFRESH_CLEANUP_INTERVAL", "AUTH_REFRESH_CLEANUP_INTERVAL")); v != "" {
		if d, err := time.ParseDuration(v); err == nil && d > 0 {
			cleanupInterval = d
		}
	}
	revokedRetention := 30 * 24 * time.Hour
	if v := strings.TrimSpace(getenvAny("IDENTITY_REFRESH_REVOKED_RETENTION", "AUTH_REFRESH_REVOKED_RETENTION")); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			revokedRetention = d
		}
	}

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

	srv := authgrpc.NewServerWithLoginLimiter(pool, tokens, accessTTL, refreshTTL, loginMaxAttempts, loginWindow, loginLockout)

	lis, err := net.Listen("tcp", grpcAddr)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	serverOpts := []grpc.ServerOption{}
	if creds, closeSrc, err := security.NewSpiffeMTLSServerCredentials(ctx); err == nil {
		defer closeSrc()
		serverOpts = append(serverOpts, grpc.Creds(creds))
		log.Printf("SPIFFE mTLS enabled for gRPC server")
	}

	g := grpc.NewServer(serverOpts...)
	authv1.RegisterAuthServiceServer(g, srv)

	go func() {
		log.Printf("Starting gRPC server on %s", grpcAddr)
		if err := g.Serve(lis); err != nil {
			log.Fatalf("failed to serve gRPC: %v", err)
		}
	}()

	go func() {
		t := time.NewTicker(cleanupInterval)
		defer t.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-t.C:
				expired, revoked, err := authgrpc.CleanupRefreshSessions(ctx, pool, revokedRetention)
				if err != nil {
					log.Printf("refresh session cleanup error: %v", err)
					continue
				}
				if expired != 0 || revoked != 0 {
					log.Printf("refresh session cleanup deleted: expired=%d revoked=%d", expired, revoked)
				}
			}
		}
	}()

	mux := runtime.NewServeMux(middleware.GatewayAuthHeaderForwarder())
	var dialCreds credentials.TransportCredentials = insecure.NewCredentials()
	if creds, closeSrc, err := security.NewSpiffeMTLSClientCredentials(ctx); err == nil {
		defer closeSrc()
		dialCreds = creds
		log.Printf("SPIFFE mTLS enabled for grpc-gateway dial")
	}
	opts := []grpc.DialOption{grpc.WithTransportCredentials(dialCreds)}
	if err := authv1.RegisterAuthServiceHandlerFromEndpoint(ctx, mux, grpcAddr, opts); err != nil {
		log.Fatalf("failed to register gateway: %v", err)
	}

	jwksPath := "/.well-known/jwks.json"
	cfg := openIDConfiguration{
		Issuer:                           issuer,
		JWKSURI:                          strings.TrimRight(issuer, "/") + jwksPath,
		ResponseTypesSupported:           []string{"token"},
		SubjectTypesSupported:            []string{"public"},
		IDTokenSigningAlgValuesSupported: []string{"EdDSA"},
	}

	h := newHTTPHandler(mux, cfg, jwks, jwksPath)

	log.Printf("Starting HTTP gateway on %s", httpAddr)
	if err := http.ListenAndServe(httpAddr, h); err != nil {
		log.Fatalf("failed to serve HTTP: %v", err)
	}
}
