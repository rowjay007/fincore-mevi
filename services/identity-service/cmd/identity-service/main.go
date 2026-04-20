package main

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"html/template"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/jackc/pgx/v5/pgxpool"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	authv1 "fincore/gen/go/auth/v1"
	"fincore/pkg/postgres"
	"fincore/pkg/secrets"
	"fincore/pkg/security"
	"fincore/pkg/security/middleware"
	authgrpc "fincore/services/auth-service/infrastructure/grpc"
)

type openIDConfiguration struct {
	Issuer                           string   `json:"issuer"`
	JWKSURI                          string   `json:"jwks_uri"`
	AuthorizationEndpoint            string   `json:"authorization_endpoint"`
	TokenEndpoint                    string   `json:"token_endpoint"`
	GrantTypesSupported              []string `json:"grant_types_supported,omitempty"`
	CodeChallengeMethodsSupported    []string `json:"code_challenge_methods_supported,omitempty"`
	ResponseTypesSupported           []string `json:"response_types_supported"`
	SubjectTypesSupported            []string `json:"subject_types_supported"`
	IDTokenSigningAlgValuesSupported []string `json:"id_token_signing_alg_values_supported"`
}

type authorizePageData struct {
	Error               string
	ClientID            string
	RedirectURI         string
	Scope               string
	State               string
	CodeChallenge       string
	CodeChallengeMethod string
	CSRFToken           string
	LoggedIn            bool
	WebAuthnEnabled     bool
}

var authorizePageTmpl = template.Must(template.New("authorize").Parse(`<!doctype html>
<html lang="en">
  <head>
    <meta charset="utf-8" />
    <meta name="viewport" content="width=device-width, initial-scale=1" />
    <title>Authorize | FinCore Identity</title>
    <script src="https://unpkg.com/@passwordless-id/webauthn@1.2.2/dist/bundle.iife.js"></script>
    <style>
      body { font-family: system-ui, -apple-system, sans-serif; background: #0a0a0a; color: #fff; display: flex; align-items: center; justify-content: center; min-height: 100vh; margin: 0; }
      .card { background: #111; border: 1px solid #333; padding: 2rem; border-radius: 1rem; width: 100%; max-width: 400px; box-shadow: 0 25px 50px -12px rgba(0, 0, 0, 0.5); }
      h1 { font-size: 1.5rem; margin-bottom: 0.5rem; font-weight: 800; }
      .subtitle { color: #888; font-size: 0.875rem; margin-bottom: 2rem; }
      label { display: block; font-size: 0.75rem; font-weight: 700; text-transform: uppercase; color: #666; margin-bottom: 0.5rem; }
      input[type="email"], input[type="password"] { width: 100%; background: #000; border: 1px solid #333; padding: 0.75rem; border-radius: 0.5rem; color: #fff; margin-bottom: 1.5rem; box-sizing: border-box; }
      button { width: 100%; background: #3b82f6; color: #fff; border: none; padding: 0.75rem; border-radius: 0.5rem; font-weight: 600; cursor: pointer; transition: background 0.2s; }
      button:hover { background: #2563eb; }
      .webauthn-btn { background: #10b981; margin-top: 1rem; }
      .webauthn-btn:hover { background: #059669; }
      .error { color: #ef4444; background: rgba(239, 68, 68, 0.1); padding: 0.75rem; border-radius: 0.5rem; font-size: 0.875rem; margin-bottom: 1.5rem; border: 1px solid rgba(239, 68, 68, 0.2); }
      .approve-label { display: flex; items-center: center; gap: 0.5rem; font-size: 0.875rem; color: #aaa; text-transform: none; margin-bottom: 1.5rem; cursor: pointer; }
    </style>
  </head>
  <body>
    <div class="card">
      <h1>FinCore Identity</h1>
      <p class="subtitle">Securely authorize <code>{{.ClientID}}</code></p>
      
      {{if .Error}}<div class="error">{{.Error}}</div>{{end}}
      
      <form id="auth-form" method="post" action="/oauth/authorize">
        <input type="hidden" name="client_id" value="{{.ClientID}}" />
        <input type="hidden" name="redirect_uri" value="{{.RedirectURI}}" />
        <input type="hidden" name="scope" value="{{.Scope}}" />
        <input type="hidden" name="state" value="{{.State}}" />
        <input type="hidden" name="code_challenge" value="{{.CodeChallenge}}" />
        <input type="hidden" name="code_challenge_method" value="{{.CodeChallengeMethod}}" />
        <input type="hidden" name="csrf_token" value="{{.CSRFToken}}" />

        {{if not .LoggedIn}}
        <label for="email">Email</label>
        <input type="email" id="email" name="email" autocomplete="username" placeholder="name@company.com" required />
        
        <label for="password">Password</label>
        <input type="password" id="password" name="password" autocomplete="current-password" placeholder="••••••••" required />
        {{end}}

        <label class="approve-label">
          <input type="checkbox" name="approve" value="yes" checked required />
          I approve this authorization request
        </label>

        <button type="submit">Continue with Password</button>
        
        <button type="button" id="webauthn-login" class="webauthn-btn">
          Sign in with Passkey
        </button>
      </form>
    </div>

    <script>
      const webauthnBtn = document.getElementById('webauthn-login');
      webauthnBtn.addEventListener('click', async () => {
        const email = document.getElementById('email')?.value;
        if (!email) {
          alert('Please enter your email first to use a Passkey');
          return;
        }

        try {
          // 1. Get options from server
          const resp = await fetch('/webauthn/login/begin?email=' + encodeURIComponent(email));
          if (!resp.ok) throw new Error('Failed to begin WebAuthn');
          const options = await resp.json();

          // 2. Authenticate with hardware/biometrics
          const assertion = await window.WebAuthn.authenticate(options.publicKey);

          // 3. Finish on server
          const finishResp = await fetch('/webauthn/login/finish', {
            method: 'POST',
            body: JSON.stringify(assertion)
          });
          
          if (finishResp.ok) {
            window.location.reload(); // Session cookie set, reload to authorize
          } else {
            alert('Passkey verification failed');
          }
        } catch (err) {
          console.error(err);
          alert('WebAuthn Error: ' + err.message);
        }
      });
    </script>
  </body>
</html>`))

type browserSessionStore struct {
	db  *pgxpool.Pool
	ttl time.Duration
}

type browserSession struct {
	UserID      string
	AccessToken string
	ExpiresAt   time.Time
}

func newBrowserSessionStore(db *pgxpool.Pool, ttl time.Duration) *browserSessionStore {
	if ttl <= 0 {
		ttl = 30 * time.Minute
	}
	return &browserSessionStore{db: db, ttl: ttl}
}

func (s *browserSessionStore) get(ctx context.Context, sessionID string) (browserSession, bool) {
	if s == nil || s.db == nil {
		return browserSession{}, false
	}
	if strings.TrimSpace(sessionID) == "" {
		return browserSession{}, false
	}
	var ss browserSession
	err := s.db.QueryRow(ctx, `select user_id, access_token, expires_at from browser_sessions where id = $1 and expires_at > now()`, sessionID).
		Scan(&ss.UserID, &ss.AccessToken, &ss.ExpiresAt)
	if err != nil {
		return browserSession{}, false
	}
	return ss, true
}

func (s *browserSessionStore) put(ctx context.Context, userID string, accessToken string) (string, browserSession, bool) {
	if s == nil || s.db == nil {
		return "", browserSession{}, false
	}
	accessToken = strings.TrimSpace(accessToken)
	if accessToken == "" {
		return "", browserSession{}, false
	}
	id, err := randomB64URL(32)
	if err != nil {
		return "", browserSession{}, false
	}
	expiresAt := time.Now().UTC().Add(s.ttl)
	_, err = s.db.Exec(ctx, `insert into browser_sessions (id, user_id, access_token, expires_at) values ($1, $2, $3, $4)`, id, userID, accessToken, expiresAt)
	if err != nil {
		return "", browserSession{}, false
	}
	return id, browserSession{UserID: userID, AccessToken: accessToken, ExpiresAt: expiresAt}, true
}

func (s *browserSessionStore) del(ctx context.Context, sessionID string) {
	if s == nil || s.db == nil {
		return
	}
	if strings.TrimSpace(sessionID) == "" {
		return
	}
	_, _ = s.db.Exec(ctx, `delete from browser_sessions where id = $1`, sessionID)
}

func randomB64URL(n int) (string, error) {
	if n <= 0 {
		n = 32
	}
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func splitScopes(scope string) []string {
	scope = strings.TrimSpace(scope)
	if scope == "" {
		return nil
	}
	parts := strings.Fields(scope)
	out := make([]string, 0, len(parts))
	seen := map[string]struct{}{}
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		if _, ok := seen[p]; ok {
			continue
		}
		seen[p] = struct{}{}
		out = append(out, p)
	}
	return out
}

func oauth2ErrorRedirect(redirectURI string, state string, code string, desc string) (string, bool) {
	u, err := url.Parse(strings.TrimSpace(redirectURI))
	if err != nil {
		return "", false
	}
	if strings.TrimSpace(u.Fragment) != "" {
		return "", false
	}
	q := u.Query()
	q.Set("error", code)
	if strings.TrimSpace(desc) != "" {
		q.Set("error_description", desc)
	}
	if strings.TrimSpace(state) != "" {
		q.Set("state", state)
	}
	u.RawQuery = q.Encode()
	return u.String(), true
}

func oauth2ErrorFromGRPC(err error) (code string, desc string) {
	if err == nil {
		return "server_error", ""
	}
	st, ok := status.FromError(err)
	if !ok {
		return "server_error", ""
	}
	msg := strings.TrimSpace(st.Message())
	switch st.Code() {
	case codes.InvalidArgument:
		if strings.Contains(strings.ToLower(msg), "scope") {
			return "invalid_scope", msg
		}
		return "invalid_request", msg
	case codes.PermissionDenied:
		return "access_denied", msg
	case codes.Unauthenticated:
		return "unauthorized_client", msg
	case codes.NotFound:
		return "unauthorized_client", msg
	default:
		if msg == "" {
			msg = "server error"
		}
		return "server_error", msg
	}
}

type oauthTokenErrorResponse struct {
	Error            string `json:"error"`
	ErrorDescription string `json:"error_description,omitempty"`
}

func writeOAuthTokenError(w http.ResponseWriter, statusCode int, code string, desc string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	_ = json.NewEncoder(w).Encode(oauthTokenErrorResponse{Error: code, ErrorDescription: strings.TrimSpace(desc)})
}

func oauthTokenErrorFromGRPC(err error) (code string, desc string, httpStatus int) {
	if err == nil {
		return "server_error", "", http.StatusInternalServerError
	}
	st, ok := status.FromError(err)
	if !ok {
		return "server_error", "", http.StatusInternalServerError
	}
	msg := strings.TrimSpace(st.Message())

	switch st.Code() {
	case codes.Unauthenticated:
		if msg == "invalid_client" {
			return "invalid_client", "", http.StatusUnauthorized
		}
		return "unauthorized_client", msg, http.StatusUnauthorized
	case codes.InvalidArgument:
		if msg == "invalid_grant" {
			return "invalid_grant", "", http.StatusBadRequest
		}
		if strings.Contains(strings.ToLower(msg), "unsupported") && strings.Contains(strings.ToLower(msg), "grant_type") {
			return "unsupported_grant_type", "", http.StatusBadRequest
		}
		return "invalid_request", msg, http.StatusBadRequest
	case codes.PermissionDenied:
		return "access_denied", msg, http.StatusForbidden
	default:
		if msg == "" {
			msg = "server error"
		}
		return "server_error", msg, http.StatusInternalServerError
	}
}

func newHTTPHandler(gw http.Handler, authClient authv1.AuthServiceClient, cfg openIDConfiguration, jwks security.JWKS, jwksPath string, pool *pgxpool.Pool, issuer string) *http.ServeMux {
	h := http.NewServeMux()
	h.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	h.HandleFunc("/readyz", func(w http.ResponseWriter, r *http.Request) {
		if pool == nil {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("ok"))
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()
		if err := pool.Ping(ctx); err != nil {
			http.Error(w, "not ready", http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	store := newBrowserSessionStore(pool, 30*time.Minute)
	const sessionCookieName = "fincore_authorize_session"
	const csrfCookieName = "fincore_authorize_csrf"
	h.HandleFunc("/oauth/token", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.Header().Set("Allow", http.MethodPost)
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		if err := r.ParseForm(); err != nil {
			writeOAuthTokenError(w, http.StatusBadRequest, "invalid_request", "invalid form")
			return
		}
		req := &authv1.OAuthTokenRequest{
			GrantType:    strings.TrimSpace(r.FormValue("grant_type")),
			Code:         strings.TrimSpace(r.FormValue("code")),
			RedirectUri:  strings.TrimSpace(r.FormValue("redirect_uri")),
			ClientId:     strings.TrimSpace(r.FormValue("client_id")),
			ClientSecret: strings.TrimSpace(r.FormValue("client_secret")),
			CodeVerifier: strings.TrimSpace(r.FormValue("code_verifier")),
		}

		ctx := r.Context()
		authHeader := strings.TrimSpace(r.Header.Get("Authorization"))
		if authHeader != "" {
			ctx = metadata.AppendToOutgoingContext(ctx, "authorization", authHeader)
		}

		var hdr metadata.MD
		res, err := authClient.OAuthToken(ctx, req, grpc.Header(&hdr))
		if err != nil {
			if w.Header().Get("WWW-Authenticate") == "" {
				vals := hdr.Get("www-authenticate")
				if len(vals) != 0 {
					if wa := strings.TrimSpace(vals[0]); wa != "" {
						w.Header().Set("WWW-Authenticate", wa)
					}
				}
			}
			code, desc, st := oauthTokenErrorFromGRPC(err)
			writeOAuthTokenError(w, st, code, desc)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(res)
	})
	h.Handle("/", gw)
	h.HandleFunc("/oauth/logout", func(w http.ResponseWriter, r *http.Request) {
		c, err := r.Cookie(sessionCookieName)
		if err == nil {
			store.del(r.Context(), c.Value)
		}
		http.SetCookie(w, &http.Cookie{
			Name:     sessionCookieName,
			Value:    "",
			Path:     "/",
			MaxAge:   -1,
			HttpOnly: true,
			SameSite: http.SameSiteLaxMode,
			Secure:   r.TLS != nil,
		})
		http.SetCookie(w, &http.Cookie{
			Name:     csrfCookieName,
			Value:    "",
			Path:     "/",
			MaxAge:   -1,
			HttpOnly: true,
			SameSite: http.SameSiteLaxMode,
			Secure:   r.TLS != nil,
		})
		w.WriteHeader(http.StatusNoContent)
	})
	h.HandleFunc("/oauth/authorize", func(w http.ResponseWriter, r *http.Request) {
		accept := strings.ToLower(r.Header.Get("Accept"))
		wantsHTML := strings.Contains(accept, "text/html") || accept == "" || accept == "*/*"
		if !wantsHTML {
			gw.ServeHTTP(w, r)
			return
		}

		var sessionID string
		if c, err := r.Cookie(sessionCookieName); err == nil {
			sessionID = strings.TrimSpace(c.Value)
		}
		sess, hasSession := store.get(r.Context(), sessionID)

		q := r.URL.Query()
		data := authorizePageData{
			ClientID:            strings.TrimSpace(q.Get("client_id")),
			RedirectURI:         strings.TrimSpace(q.Get("redirect_uri")),
			Scope:               strings.TrimSpace(q.Get("scope")),
			State:               strings.TrimSpace(q.Get("state")),
			CodeChallenge:       strings.TrimSpace(q.Get("code_challenge")),
			CodeChallengeMethod: strings.TrimSpace(q.Get("code_challenge_method")),
			LoggedIn:            hasSession,
		}

		csrfCookieVal := ""
		if c, err := r.Cookie(csrfCookieName); err == nil {
			csrfCookieVal = strings.TrimSpace(c.Value)
		}
		if csrfCookieVal == "" {
			v, err := randomB64URL(32)
			if err == nil {
				csrfCookieVal = v
				http.SetCookie(w, &http.Cookie{
					Name:     csrfCookieName,
					Value:    csrfCookieVal,
					Path:     "/",
					HttpOnly: true,
					SameSite: http.SameSiteLaxMode,
					Secure:   r.TLS != nil,
				})
			}
		}
		data.CSRFToken = csrfCookieVal
		if data.CodeChallengeMethod == "" {
			data.CodeChallengeMethod = "S256"
		}

		switch r.Method {
		case http.MethodGet:
			if strings.TrimSpace(q.Get("response_type")) != "code" {
				data.Error = "unsupported response_type"
			}
			if data.ClientID == "" || data.RedirectURI == "" || data.CodeChallenge == "" {
				if data.Error == "" {
					data.Error = "missing required parameters"
				}
			}
			if strings.ToUpper(strings.TrimSpace(data.CodeChallengeMethod)) != "S256" {
				if data.Error == "" {
					data.Error = "code_challenge_method must be S256"
				}
			}

			// If logged in, check for existing consent.
			if hasSession && data.Error == "" {
				res, err := authClient.GetOAuthConsent(r.Context(), &authv1.GetOAuthConsentRequest{
					UserId:   sess.UserID,
					ClientId: data.ClientID,
				})
				if err == nil && res != nil && len(res.Scopes) > 0 {
					requested := splitScopes(data.Scope)
					granted := map[string]bool{}
					for _, s := range res.Scopes {
						granted[s] = true
					}
					allGranted := true
					for _, s := range requested {
						if !granted[s] {
							allGranted = false
							break
						}
					}

					if allGranted {
						resp, err := authClient.OAuthAuthorize(r.Context(), &authv1.OAuthAuthorizeRequest{
							ResponseType:        "code",
							ClientId:            data.ClientID,
							RedirectUri:         data.RedirectURI,
							Scope:               data.Scope,
							State:               data.State,
							CodeChallenge:       data.CodeChallenge,
							CodeChallengeMethod: data.CodeChallengeMethod,
						})
						if err != nil {
							code, desc := oauth2ErrorFromGRPC(err)
							if loc, ok := oauth2ErrorRedirect(data.RedirectURI, data.State, code, desc); ok {
								http.Redirect(w, r, loc, http.StatusFound)
								return
							}
						} else {
							http.Redirect(w, r, resp.RedirectUrl, http.StatusFound)
							return
						}
					}
				}
			}

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
			data.CSRFToken = strings.TrimSpace(r.Form.Get("csrf_token"))
			if data.CodeChallengeMethod == "" {
				data.CodeChallengeMethod = "S256"
			}

			cookieToken := ""
			if c, err := r.Cookie(csrfCookieName); err == nil {
				cookieToken = strings.TrimSpace(c.Value)
			}
			if cookieToken == "" || data.CSRFToken == "" || cookieToken != data.CSRFToken {
				if loc, ok := oauth2ErrorRedirect(data.RedirectURI, data.State, "invalid_request", "csrf required"); ok {
					http.Redirect(w, r, loc, http.StatusFound)
					return
				}
				w.Header().Set("Content-Type", "text/html; charset=utf-8")
				data.Error = "csrf required"
				_ = authorizePageTmpl.Execute(w, data)
				return
			}

			if r.FormValue("approve") != "yes" {
				if loc, ok := oauth2ErrorRedirect(data.RedirectURI, data.State, "access_denied", "user rejected request"); ok {
					http.Redirect(w, r, loc, http.StatusFound)
					return
				}
				http.Error(w, "access denied", http.StatusForbidden)
				return
			}

			email := strings.TrimSpace(r.FormValue("email"))
			password := strings.TrimSpace(r.FormValue("password"))

			var userID string
			if hasSession {
				userID = sess.UserID
			} else {
				loginRes, err := authClient.Login(r.Context(), &authv1.LoginRequest{
					Email:    email,
					Password: password,
				})
				if err != nil {
					data.Error = "Invalid email or password"
					_ = authorizePageTmpl.Execute(w, data)
					return
				}
				// Extract userID from the token without verification if we trust the auth-service.
				parts := strings.Split(loginRes.AccessToken, ".")
				if len(parts) == 3 {
					payloadBytes, err := base64.RawURLEncoding.DecodeString(parts[1])
					if err == nil {
						var p struct {
							UserID string `json:"user_id"`
						}
						_ = json.Unmarshal(payloadBytes, &p)
						userID = p.UserID
					}
				}

				if userID == "" {
					data.Error = "Login failed"
					_ = authorizePageTmpl.Execute(w, data)
					return
				}

				if strings.TrimSpace(sessionID) != "" {
					store.del(r.Context(), sessionID)
				}
				sessionID, _, _ = store.put(r.Context(), userID, loginRes.AccessToken)
				http.SetCookie(w, &http.Cookie{
					Name:     sessionCookieName,
					Value:    sessionID,
					Path:     "/",
					HttpOnly: true,
					SameSite: http.SameSiteLaxMode,
					Secure:   r.TLS != nil,
				})
			}

			// Store consent
			_, _ = authClient.StoreOAuthConsent(r.Context(), &authv1.StoreOAuthConsentRequest{
				UserId:   userID,
				ClientId: data.ClientID,
				Scopes:   splitScopes(data.Scope),
			})

			// Issue code
			resp, err := authClient.OAuthAuthorize(r.Context(), &authv1.OAuthAuthorizeRequest{
				ResponseType:        "code",
				ClientId:            data.ClientID,
				RedirectUri:         data.RedirectURI,
				Scope:               data.Scope,
				State:               data.State,
				CodeChallenge:       data.CodeChallenge,
				CodeChallengeMethod: data.CodeChallengeMethod,
			})
			if err != nil {
				code, desc := oauth2ErrorFromGRPC(err)
				if loc, ok := oauth2ErrorRedirect(data.RedirectURI, data.State, code, desc); ok {
					http.Redirect(w, r, loc, http.StatusFound)
					return
				}
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			http.Redirect(w, r, resp.RedirectUrl, http.StatusFound)
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

	// WebAuthn Handlers
	wa, err := NewWebAuthnHandler(pool, store, issuer)
	if err != nil {
		log.Printf("failed to initialize webauthn: %v", err)
	} else {
		h.HandleFunc("/webauthn/login/begin", wa.BeginLogin)
		h.HandleFunc("/webauthn/login/finish", wa.FinishLogin)
	}

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
	conn, err := grpc.NewClient(grpcAddr, opts...)
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
		GrantTypesSupported:              []string{"authorization_code"},
		CodeChallengeMethodsSupported:    []string{"S256"},
		ResponseTypesSupported:           []string{"code"},
		SubjectTypesSupported:            []string{"public"},
		IDTokenSigningAlgValuesSupported: []string{"EdDSA"},
	}

	h := newHTTPHandler(mux, authClient, cfg, jwks, jwksPath, pool, issuer)

	log.Printf("Starting HTTP gateway on %s", httpAddr)
	if err := http.ListenAndServe(httpAddr, h); err != nil {
		log.Fatalf("failed to serve HTTP: %v", err)
	}
}
