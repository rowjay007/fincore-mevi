package main

import (
	"context"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"

	accountv1 "fincore/gen/go/account/v1"
	authv1 "fincore/gen/go/auth/v1"
	ledgerv1 "fincore/gen/go/ledger/v1"
	"fincore/pkg/security"
)

type tokenBucket struct {
	capacity float64
	rate     float64

	mu     sync.Mutex
	tokens float64
	last   time.Time
}

func newTokenBucket(reqPerSec float64, burst int) *tokenBucket {
	if reqPerSec <= 0 {
		reqPerSec = 20
	}
	if burst <= 0 {
		burst = 40
	}
	return &tokenBucket{capacity: float64(burst), rate: reqPerSec, tokens: float64(burst), last: time.Now().UTC()}
}

func (b *tokenBucket) allow() bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	now := time.Now().UTC()
	elapsed := now.Sub(b.last).Seconds()
	b.last = now
	b.tokens = min(b.capacity, b.tokens+(elapsed*b.rate))
	if b.tokens < 1 {
		return false
	}
	b.tokens -= 1
	return true
}

type rateLimiter struct {
	mu      sync.Mutex
	buckets map[string]*tokenBucket

	reqPerSec float64
	burst     int
}

func newRateLimiter(reqPerSec float64, burst int) *rateLimiter {
	return &rateLimiter{buckets: map[string]*tokenBucket{}, reqPerSec: reqPerSec, burst: burst}
}

func (r *rateLimiter) bucketFor(key string) *tokenBucket {
	r.mu.Lock()
	defer r.mu.Unlock()
	b, ok := r.buckets[key]
	if !ok {
		b = newTokenBucket(r.reqPerSec, r.burst)
		r.buckets[key] = b
	}
	return b
}

func withRateLimit(l *rateLimiter, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		key := strings.TrimSpace(r.Header.Get("X-Fincore-Subject"))
		if key == "" {
			key = strings.TrimSpace(r.RemoteAddr)
		}
		if key == "" {
			key = "anonymous"
		}
		if !l.bucketFor(key).allow() {
			w.Header().Set("Retry-After", "1")
			http.Error(w, "rate limit exceeded", http.StatusTooManyRequests)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func withRateLimitByPath(defaultLimiter *rateLimiter, strictLimiter *rateLimiter, strictPrefixes []string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		lim := defaultLimiter
		path := r.URL.Path
		for _, p := range strictPrefixes {
			if p != "" && strings.HasPrefix(path, p) {
				lim = strictLimiter
				break
			}
		}
		withRateLimit(lim, next).ServeHTTP(w, r)
	})
}

func stripUntrustedIdentityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.Header.Del("X-Fincore-Subject")
		r.Header.Del("X-Fincore-Roles")
		r.Header.Del("X-Fincore-Permissions")
		next.ServeHTTP(w, r)
	})
}

func withJWTAuth(verifier *security.JWKSVerifier, publicPrefixes []string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		for _, p := range publicPrefixes {
			if p != "" && strings.HasPrefix(path, p) {
				next.ServeHTTP(w, r)
				return
			}
		}

		v := strings.TrimSpace(r.Header.Get("Authorization"))
		const prefix = "Bearer "
		if !strings.HasPrefix(v, prefix) {
			http.Error(w, "missing authorization", http.StatusUnauthorized)
			return
		}
		tok := strings.TrimSpace(strings.TrimPrefix(v, prefix))
		if tok == "" {
			http.Error(w, "missing authorization", http.StatusUnauthorized)
			return
		}

		payload, err := verifier.VerifyToken(tok)
		if err != nil {
			http.Error(w, "invalid token", http.StatusUnauthorized)
			return
		}

		r.Header.Set("X-Fincore-Subject", payload.UserID)
		if len(payload.Roles) != 0 {
			r.Header.Set("X-Fincore-Roles", strings.Join(payload.Roles, ","))
		}
		if len(payload.Permissions) != 0 {
			r.Header.Set("X-Fincore-Permissions", strings.Join(payload.Permissions, ","))
		}

		next.ServeHTTP(w, r)
	})
}

func parsePublicPrefixesEnv() []string {
	v := strings.TrimSpace(os.Getenv("GATEWAY_PUBLIC_PATH_PREFIXES"))
	if v == "" {
		v = "/.well-known,/oauth/,/v1/auth/register,/v1/auth/login,/v1/auth/refresh,/v1/auth/logout,/v1/auth/logout_all"
	}
	parts := strings.Split(v, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		out = append(out, p)
	}
	return out
}

func gatewayHeaderForwarder() runtime.ServeMuxOption {
	return runtime.WithIncomingHeaderMatcher(func(key string) (string, bool) {
		if strings.EqualFold(key, "Authorization") {
			return "authorization", true
		}
		if strings.EqualFold(key, "X-Fincore-Subject") {
			return "x-fincore-subject", true
		}
		if strings.EqualFold(key, "X-Fincore-Roles") {
			return "x-fincore-roles", true
		}
		if strings.EqualFold(key, "X-Fincore-Permissions") {
			return "x-fincore-permissions", true
		}
		return runtime.DefaultHeaderMatcher(key)
	})
}

func spiffeDialCreds(ctx context.Context) (credentials.TransportCredentials, func()) {
	creds, closeSrc, err := security.NewSpiffeMTLSClientCredentials(ctx)
	if err == nil {
		log.Printf("SPIFFE mTLS enabled for api-gateway upstream dials")
		return creds, closeSrc
	}
	return insecure.NewCredentials(), func() {}
}

func min(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

func main() {
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	httpAddr := strings.TrimSpace(os.Getenv("GATEWAY_HTTP_ADDR"))
	if httpAddr == "" {
		httpAddr = ":8080"
	}

	jwksURL := strings.TrimSpace(os.Getenv("AUTH_JWKS_URL"))
	verifier, err := security.NewJWKSVerifier(jwksURL, 5*time.Minute)
	if err != nil {
		log.Fatalf("failed to create jwks verifier: %v", err)
	}

	accountGRPC := strings.TrimSpace(os.Getenv("ACCOUNT_GRPC_ADDR"))
	if accountGRPC == "" {
		accountGRPC = ":50051"
	}
	ledgerGRPC := strings.TrimSpace(os.Getenv("LEDGER_LISTEN_ADDR"))
	if ledgerGRPC == "" {
		ledgerGRPC = ":50053"
	}
	identityGRPC := strings.TrimSpace(os.Getenv("IDENTITY_GRPC_ADDR"))
	if identityGRPC == "" {
		identityGRPC = ":50056"
	}
	identityHTTP := strings.TrimSpace(os.Getenv("IDENTITY_HTTP_ADDR"))
	if identityHTTP == "" {
		identityHTTP = ":8084"
	}

	mux := runtime.NewServeMux(gatewayHeaderForwarder())
	creds, closeSrc := spiffeDialCreds(ctx)
	defer closeSrc()
	opts := []grpc.DialOption{grpc.WithTransportCredentials(creds)}

	if err := accountv1.RegisterAccountServiceHandlerFromEndpoint(ctx, mux, accountGRPC, opts); err != nil {
		log.Fatalf("failed to register account handler: %v", err)
	}
	if err := ledgerv1.RegisterLedgerServiceHandlerFromEndpoint(ctx, mux, ledgerGRPC, opts); err != nil {
		log.Fatalf("failed to register ledger handler: %v", err)
	}
	if err := authv1.RegisterAuthServiceHandlerFromEndpoint(ctx, mux, identityGRPC, opts); err != nil {
		log.Fatalf("failed to register identity(auth) handler: %v", err)
	}

	identityBase := "http://localhost" + identityHTTP
	proxy := http.NewServeMux()
	proxy.Handle("/", mux)
	proxy.HandleFunc("/.well-known/", func(w http.ResponseWriter, r *http.Request) {
		url := strings.TrimRight(identityBase, "/") + r.URL.Path
		if r.URL.RawQuery != "" {
			url += "?" + r.URL.RawQuery
		}
		req, err := http.NewRequestWithContext(r.Context(), r.Method, url, r.Body)
		if err != nil {
			http.Error(w, "bad gateway", http.StatusBadGateway)
			return
		}
		req.Header = r.Header.Clone()
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			http.Error(w, "bad gateway", http.StatusBadGateway)
			return
		}
		defer resp.Body.Close()
		for k, vals := range resp.Header {
			for _, v := range vals {
				w.Header().Add(k, v)
			}
		}
		w.WriteHeader(resp.StatusCode)
		_, _ = io.Copy(w, resp.Body)
	})

	publicPrefixes := parsePublicPrefixesEnv()
	lim := newRateLimiter(20, 40)
	strict := newRateLimiter(5, 10)
	strictPrefixes := []string{"/oauth/token", "/v1/auth/login", "/v1/auth/register"}

	h := http.Handler(proxy)
	h = stripUntrustedIdentityHeaders(h)
	h = withJWTAuth(verifier, publicPrefixes, h)
	h = withRateLimitByPath(lim, strict, strictPrefixes, h)

	root := http.NewServeMux()
	root.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	root.Handle("/", h)

	log.Printf("Starting api-gateway on %s", httpAddr)
	if err := http.ListenAndServe(httpAddr, root); err != nil {
		log.Fatalf("failed to serve http: %v", err)
	}
}
