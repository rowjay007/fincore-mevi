package main

import (
	"context"
	"crypto/ed25519"
	"encoding/json"
	"html/template"
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
	"google.golang.org/grpc/metadata"

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
	AuthorizationEndpoint            string `json:"authorization_endpoint"`
	TokenEndpoint                    string `json:"token_endpoint"`
	ResponseTypesSupported           []string
	SubjectTypesSupported            []string
	IDTokenSigningAlgValuesSupported []string
}

type authorizePageData struct {
	Error               string
	ClientID            string
	RedirectURI         string
	Scope               string
	State               string
	CodeChallenge       string
	CodeChallengeMethod string
}

var authorizePageTmpl = template.Must(template.New("authorize").Parse(`<!doctype html>
<html lang="en">
  <head>
    <meta charset="utf-8" />
    <meta name="viewport" content="width=device-width, initial-scale=1" />
    <title>Authorize</title>
  </head>
  <body>
    <h1>Authorize</h1>
    {{if .Error}}<p style="color:#b00020">{{.Error}}</p>{{end}}
    <p>Client: <code>{{.ClientID}}</code></p>
    <p>Scope: <code>{{.Scope}}</code></p>
    <form method="post" action="/oauth/authorize">
      <input type="hidden" name="client_id" value="{{.ClientID}}" />
      <input type="hidden" name="redirect_uri" value="{{.RedirectURI}}" />
      <input type="hidden" name="scope" value="{{.Scope}}" />
      <input type="hidden" name="state" value="{{.State}}" />
      <input type="hidden" name="code_challenge" value="{{.CodeChallenge}}" />
      <input type="hidden" name="code_challenge_method" value="{{.CodeChallengeMethod}}" />

      <label>Email<br />
        <input type="email" name="email" autocomplete="username" required />
      </label>
      <br />
      <label>Password<br />
        <input type="password" name="password" autocomplete="current-password" required />
      </label>
      <br />
      <label>
        <input type="checkbox" name="approve" value="yes" required />
        Approve
      </label>
      <br />
      <button type="submit">Continue</button>
    </form>
  </body>
</html>`))

func newHTTPHandler(gw http.Handler, authClient authv1.AuthServiceClient, cfg openIDConfiguration, jwks security.JWKS, jwksPath string) *http.ServeMux {
	h := http.NewServeMux()
	h.Handle("/", gw)
	h.HandleFunc("/oauth/authorize", func(w http.ResponseWriter, r *http.Request) {
		accept := strings.ToLower(r.Header.Get("Accept"))
		wantsHTML := strings.Contains(accept, "text/html") || accept == "" || accept == "*/*"
		if !wantsHTML {
			gw.ServeHTTP(w, r)
			return
		}

		q := r.URL.Query()
		data := authorizePageData{
			ClientID:            strings.TrimSpace(q.Get("client_id")),
			RedirectURI:         strings.TrimSpace(q.Get("redirect_uri")),
			Scope:               strings.TrimSpace(q.Get("scope")),
			State:               strings.TrimSpace(q.Get("state")),
			CodeChallenge:       strings.TrimSpace(q.Get("code_challenge")),
			CodeChallengeMethod: strings.TrimSpace(q.Get("code_challenge_method")),
		}
		if data.CodeChallengeMethod == "" {
			data.CodeChallengeMethod = "S256"
		}

		switch r.Method {
		case http.MethodGet:
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			_ = authorizePageTmpl.Execute(w, data)
			return
		case http.MethodPost:
			if err := r.ParseForm(); err != nil {
				w.Header().Set("Content-Type", "text/html; charset=utf-8")
				data.Error = "invalid form"
				_ = authorizePageTmpl.Execute(w, data)
				return
			}
			data.ClientID = strings.TrimSpace(r.Form.Get("client_id"))
			data.RedirectURI = strings.TrimSpace(r.Form.Get("redirect_uri"))
			data.Scope = strings.TrimSpace(r.Form.Get("scope"))
			data.State = strings.TrimSpace(r.Form.Get("state"))
			data.CodeChallenge = strings.TrimSpace(r.Form.Get("code_challenge"))
			data.CodeChallengeMethod = strings.TrimSpace(r.Form.Get("code_challenge_method"))
			if data.CodeChallengeMethod == "" {
				data.CodeChallengeMethod = "S256"
			}

			email := strings.TrimSpace(r.Form.Get("email"))
			password := r.Form.Get("password")
			approve := strings.TrimSpace(r.Form.Get("approve"))
			if approve != "yes" {
				w.Header().Set("Content-Type", "text/html; charset=utf-8")
				data.Error = "approval required"
				_ = authorizePageTmpl.Execute(w, data)
				return
			}
			if email == "" || password == "" {
				w.Header().Set("Content-Type", "text/html; charset=utf-8")
				data.Error = "email and password required"
				_ = authorizePageTmpl.Execute(w, data)
				return
			}

			loginRes, err := authClient.Login(r.Context(), &authv1.LoginRequest{Email: email, Password: password})
			if err != nil {
				w.Header().Set("Content-Type", "text/html; charset=utf-8")
				data.Error = "login failed"
				_ = authorizePageTmpl.Execute(w, data)
				return
			}
			authCtx := metadata.NewOutgoingContext(r.Context(), metadata.Pairs("authorization", "Bearer "+loginRes.AccessToken))
			res, err := authClient.OAuthAuthorize(authCtx, &authv1.OAuthAuthorizeRequest{
				ResponseType:        "code",
				ClientId:            data.ClientID,
				RedirectUri:         data.RedirectURI,
				Scope:               data.Scope,
				State:               data.State,
				CodeChallenge:       data.CodeChallenge,
				CodeChallengeMethod: data.CodeChallengeMethod,
			})
			if err != nil {
				w.Header().Set("Content-Type", "text/html; charset=utf-8")
				data.Error = "authorize failed"
				_ = authorizePageTmpl.Execute(w, data)
				return
			}
			http.Redirect(w, r, res.RedirectUrl, http.StatusFound)
			return
		default:
			w.Header().Set("Allow", "GET, POST")
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
	})
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
	conn, err := grpc.DialContext(ctx, grpcAddr, opts...)
	if err != nil {
		log.Fatalf("failed to dial local grpc: %v", err)
	}
	defer conn.Close()
	authClient := authv1.NewAuthServiceClient(conn)

	jwksPath := "/.well-known/jwks.json"
	authzPath := "/oauth/authorize"
	tokenPath := "/oauth/token"
	cfg := openIDConfiguration{
		Issuer:                           issuer,
		JWKSURI:                          strings.TrimRight(issuer, "/") + jwksPath,
		AuthorizationEndpoint:            strings.TrimRight(issuer, "/") + authzPath,
		TokenEndpoint:                    strings.TrimRight(issuer, "/") + tokenPath,
		ResponseTypesSupported:           []string{"token"},
		SubjectTypesSupported:            []string{"public"},
		IDTokenSigningAlgValuesSupported: []string{"EdDSA"},
	}

	h := newHTTPHandler(mux, authClient, cfg, jwks, jwksPath)

	log.Printf("Starting HTTP gateway on %s", httpAddr)
	if err := http.ListenAndServe(httpAddr, h); err != nil {
		log.Fatalf("failed to serve HTTP: %v", err)
	}
}
