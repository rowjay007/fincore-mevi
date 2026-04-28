package main

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/go-webauthn/webauthn/webauthn"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type WebAuthnHandler struct {
	w       *webauthn.WebAuthn
	db      *pgxpool.Pool
	session *browserSessionStore
}

func NewWebAuthnHandler(db *pgxpool.Pool, session *browserSessionStore, issuer string) (*WebAuthnHandler, error) {
	if db == nil {
		return nil, fmt.Errorf("db required")
	}
	if session == nil {
		return nil, fmt.Errorf("session store required")
	}

	rpID := "localhost"
	origins := []string{"http://localhost:8084"}
	if u, err := url.Parse(strings.TrimSpace(issuer)); err == nil && u != nil {
		if host := strings.TrimSpace(u.Hostname()); host != "" {
			rpID = host
		}
		if scheme := strings.TrimSpace(u.Scheme); scheme != "" && strings.TrimSpace(u.Host) != "" {
			origins = []string{scheme + "://" + u.Host}
		}
	}

	w, err := webauthn.New(&webauthn.Config{
		RPDisplayName: "FinCore OS",
		RPID:          rpID,
		RPOrigins:     origins,
	})
	if err != nil {
		return nil, err
	}

	ctx, cancel := contextWithTimeout()
	defer cancel()
	if err := ensureWebAuthnTables(ctx, db); err != nil {
		return nil, err
	}

	return &WebAuthnHandler{w: w, db: db, session: session}, nil
}

func contextWithTimeout() (context.Context, context.CancelFunc) {
	// local helper to avoid threading context through constructors
	ctx := context.Background()
	return context.WithTimeout(ctx, 5*time.Second)
}

// User implementation for WebAuthn
type webauthnUser struct {
	id          []byte
	name        string
	displayName string
	credentials []webauthn.Credential
}

func (u *webauthnUser) WebAuthnID() []byte                         { return u.id }
func (u *webauthnUser) WebAuthnName() string                       { return u.name }
func (u *webauthnUser) WebAuthnDisplayName() string                { return u.displayName }
func (u *webauthnUser) WebAuthnCredentials() []webauthn.Credential { return u.credentials }
func (u *webauthnUser) WebAuthnIcon() string                       { return "" }

type webAuthnSessionKind string

const (
	webAuthnSessionKindLogin    webAuthnSessionKind = "login"
	webAuthnSessionKindRegister webAuthnSessionKind = "register"
)

const webAuthnCookieName = "fincore_webauthn_session"

func ensureWebAuthnTables(ctx context.Context, db *pgxpool.Pool) error {
	_, err := db.Exec(ctx, `
		create table if not exists webauthn_credentials (
		  id text primary key,
		  user_id text not null references auth_users(id) on delete cascade,
		  credential_id bytea not null unique,
		  credential jsonb not null,
		  sign_count bigint not null default 0,
		  created_at timestamptz not null default now(),
		  updated_at timestamptz not null default now()
		);
		create index if not exists webauthn_credentials_user_idx on webauthn_credentials(user_id);
		create table if not exists webauthn_sessions (
		  id text primary key,
		  kind text not null,
		  user_id text not null references auth_users(id) on delete cascade,
		  session_data jsonb not null,
		  expires_at timestamptz not null,
		  created_at timestamptz not null default now()
		);
		create index if not exists webauthn_sessions_expires_idx on webauthn_sessions(expires_at);
	`)
	return err
}

func randID(n int) (string, error) {
	if n <= 0 {
		n = 32
	}
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func (h *WebAuthnHandler) userByEmail(ctx context.Context, email string) (userID string, normalizedEmail string, err error) {
	email = strings.TrimSpace(strings.ToLower(email))
	if email == "" {
		return "", "", fmt.Errorf("email required")
	}
	var id string
	var dbEmail string
	err = h.db.QueryRow(ctx, `select id, email from auth_users where email = $1`, email).Scan(&id, &dbEmail)
	if err != nil {
		return "", "", err
	}
	return id, dbEmail, nil
}

func (h *WebAuthnHandler) loadCredentials(ctx context.Context, userID string) ([]webauthn.Credential, error) {
	rows, err := h.db.Query(ctx, `select credential from webauthn_credentials where user_id = $1 order by created_at asc`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []webauthn.Credential
	for rows.Next() {
		var raw []byte
		if err := rows.Scan(&raw); err != nil {
			return nil, err
		}
		var cred webauthn.Credential
		if err := json.Unmarshal(raw, &cred); err != nil {
			return nil, err
		}
		out = append(out, cred)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func (h *WebAuthnHandler) putSession(ctx context.Context, kind webAuthnSessionKind, userID string, sd *webauthn.SessionData) (string, error) {
	if sd == nil {
		return "", fmt.Errorf("session required")
	}
	id, err := randID(32)
	if err != nil {
		return "", err
	}
	b, err := json.Marshal(sd)
	if err != nil {
		return "", err
	}
	exp := time.Now().UTC().Add(5 * time.Minute)
	_, err = h.db.Exec(ctx, `insert into webauthn_sessions (id, kind, user_id, session_data, expires_at) values ($1,$2,$3,$4,$5)`, id, string(kind), userID, b, exp)
	if err != nil {
		return "", err
	}
	return id, nil
}

func (h *WebAuthnHandler) popSession(ctx context.Context, id string, kind webAuthnSessionKind) (userID string, sd webauthn.SessionData, ok bool, err error) {
	if strings.TrimSpace(id) == "" {
		return "", webauthn.SessionData{}, false, nil
	}
	var uid string
	var raw []byte
	err = h.db.QueryRow(ctx, `
		delete from webauthn_sessions
		where id = $1 and kind = $2 and expires_at > now()
		returning user_id, session_data
	`, id, string(kind)).Scan(&uid, &raw)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", webauthn.SessionData{}, false, nil
		}
		return "", webauthn.SessionData{}, false, err
	}
	if err := json.Unmarshal(raw, &sd); err != nil {
		return "", webauthn.SessionData{}, false, err
	}
	return uid, sd, true, nil
}

func (h *WebAuthnHandler) setWebAuthnCookie(w http.ResponseWriter, r *http.Request, sessionID string) {
	http.SetCookie(w, &http.Cookie{
		Name:     webAuthnCookieName,
		Value:    sessionID,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   r.TLS != nil,
		MaxAge:   int((5 * time.Minute).Seconds()),
	})
}

func (h *WebAuthnHandler) clearWebAuthnCookie(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:     webAuthnCookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   r.TLS != nil,
		MaxAge:   -1,
	})
}

func (h *WebAuthnHandler) cookieSessionID(r *http.Request) string {
	c, err := r.Cookie(webAuthnCookieName)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(c.Value)
}

func (h *WebAuthnHandler) BeginLogin(w http.ResponseWriter, r *http.Request) {
	email := r.URL.Query().Get("email")
	if email == "" {
		http.Error(w, "email required", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	userID, dbEmail, err := h.userByEmail(ctx, email)
	if err != nil {
		http.Error(w, "unknown user", http.StatusNotFound)
		return
	}
	creds, err := h.loadCredentials(ctx, userID)
	if err != nil {
		http.Error(w, "failed to load credentials", http.StatusInternalServerError)
		return
	}
	if len(creds) == 0 {
		http.Error(w, "no passkeys registered", http.StatusNotFound)
		return
	}

	user := &webauthnUser{id: []byte(userID), name: dbEmail, displayName: dbEmail, credentials: creds}
	options, sd, err := h.w.BeginLogin(user)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	sid, err := h.putSession(ctx, webAuthnSessionKindLogin, userID, sd)
	if err != nil {
		http.Error(w, "failed to persist login session", http.StatusInternalServerError)
		return
	}
	h.setWebAuthnCookie(w, r, sid)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(options)
}

func (h *WebAuthnHandler) FinishLogin(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	sid := h.cookieSessionID(r)
	userID, sd, ok, err := h.popSession(ctx, sid, webAuthnSessionKindLogin)
	if err != nil {
		http.Error(w, "failed to load login session", http.StatusInternalServerError)
		return
	}
	if !ok {
		http.Error(w, "login session expired", http.StatusUnauthorized)
		return
	}
	h.clearWebAuthnCookie(w, r)

	creds, err := h.loadCredentials(ctx, userID)
	if err != nil {
		http.Error(w, "failed to load credentials", http.StatusInternalServerError)
		return
	}
	user := &webauthnUser{id: []byte(userID), name: userID, displayName: userID, credentials: creds}

	cred, err := h.w.FinishLogin(user, sd, r)
	if err != nil {
		http.Error(w, "passkey verification failed", http.StatusUnauthorized)
		return
	}

	rawCred, err := json.Marshal(cred)
	if err != nil {
		http.Error(w, "failed to marshal credential", http.StatusInternalServerError)
		return
	}
	_, _ = h.db.Exec(ctx, `
		update webauthn_credentials
		set credential = $1, sign_count = $2, updated_at = now()
		where user_id = $3 and credential_id = $4
	`, rawCred, int64(cred.Authenticator.SignCount), userID, cred.ID)

	browserToken, err := randomB64URL(32)
	if err != nil {
		http.Error(w, "failed to create session", http.StatusInternalServerError)
		return
	}
	sessionID, _, ok2 := h.session.put(ctx, userID, browserToken)
	if !ok2 {
		http.Error(w, "failed to create session", http.StatusInternalServerError)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "fincore_authorize_session",
		Value:    sessionID,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   r.TLS != nil,
	})

	w.WriteHeader(http.StatusOK)
}

func (h *WebAuthnHandler) BeginRegister(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	// registration requires existing browser session (password-authenticated)
	c, err := r.Cookie("fincore_authorize_session")
	if err != nil || strings.TrimSpace(c.Value) == "" {
		http.Error(w, "login required", http.StatusUnauthorized)
		return
	}
	sess, ok := h.session.get(ctx, strings.TrimSpace(c.Value))
	if !ok {
		http.Error(w, "login required", http.StatusUnauthorized)
		return
	}

	creds, err := h.loadCredentials(ctx, sess.UserID)
	if err != nil {
		http.Error(w, "failed to load credentials", http.StatusInternalServerError)
		return
	}
	user := &webauthnUser{id: []byte(sess.UserID), name: sess.UserID, displayName: sess.UserID, credentials: creds}

	options, sd, err := h.w.BeginRegistration(user)
	if err != nil {
		http.Error(w, "failed to begin registration", http.StatusInternalServerError)
		return
	}
	sid, err := h.putSession(ctx, webAuthnSessionKindRegister, sess.UserID, sd)
	if err != nil {
		http.Error(w, "failed to persist registration session", http.StatusInternalServerError)
		return
	}
	h.setWebAuthnCookie(w, r, sid)

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(options)
}

func (h *WebAuthnHandler) FinishRegister(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	sid := h.cookieSessionID(r)
	userID, sd, ok, err := h.popSession(ctx, sid, webAuthnSessionKindRegister)
	if err != nil {
		http.Error(w, "failed to load registration session", http.StatusInternalServerError)
		return
	}
	if !ok {
		http.Error(w, "registration session expired", http.StatusUnauthorized)
		return
	}
	h.clearWebAuthnCookie(w, r)

	creds, err := h.loadCredentials(ctx, userID)
	if err != nil {
		http.Error(w, "failed to load credentials", http.StatusInternalServerError)
		return
	}
	user := &webauthnUser{id: []byte(userID), name: userID, displayName: userID, credentials: creds}

	cred, err := h.w.FinishRegistration(user, sd, r)
	if err != nil {
		http.Error(w, "passkey registration failed", http.StatusBadRequest)
		return
	}

	credID := cred.ID
	rawCred, err := json.Marshal(cred)
	if err != nil {
		http.Error(w, "failed to marshal credential", http.StatusInternalServerError)
		return
	}
	rowID, err := randomB64URL(16)
	if err != nil {
		http.Error(w, "server error", http.StatusInternalServerError)
		return
	}
	_, err = h.db.Exec(ctx, `
		insert into webauthn_credentials (id, user_id, credential_id, credential, sign_count)
		values ($1,$2,$3,$4,$5)
		on conflict (credential_id) do update set
		  credential = excluded.credential,
		  sign_count = excluded.sign_count,
		  updated_at = now()
	`, rowID, userID, credID, rawCred, int64(cred.Authenticator.SignCount))
	if err != nil {
		http.Error(w, "failed to store credential", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (h *WebAuthnHandler) CleanupSessions(ctx context.Context) (int64, error) {
	res, err := h.db.Exec(ctx, `delete from webauthn_sessions where expires_at <= now()`)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected(), nil
}

type webAuthnCredentialInfo struct {
	ID        string    `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	SignCount int64     `json:"sign_count"`
}

func (h *WebAuthnHandler) ListCredentials(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	c, err := r.Cookie("fincore_authorize_session")
	if err != nil || strings.TrimSpace(c.Value) == "" {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	sess, ok := h.session.get(ctx, strings.TrimSpace(c.Value))
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	rows, err := h.db.Query(ctx, `
		select id, sign_count, created_at, updated_at
		from webauthn_credentials
		where user_id = $1
		order by created_at desc
	`, sess.UserID)
	if err != nil {
		http.Error(w, "server error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var out []webAuthnCredentialInfo
	for rows.Next() {
		var info webAuthnCredentialInfo
		if err := rows.Scan(&info.ID, &info.SignCount, &info.CreatedAt, &info.UpdatedAt); err != nil {
			http.Error(w, "server error", http.StatusInternalServerError)
			return
		}
		out = append(out, info)
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(out)
}

func (h *WebAuthnHandler) DeleteCredential(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	id := r.URL.Query().Get("id")
	if id == "" {
		http.Error(w, "id required", http.StatusBadRequest)
		return
	}

	c, err := r.Cookie("fincore_authorize_session")
	if err != nil || strings.TrimSpace(c.Value) == "" {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	sess, ok := h.session.get(ctx, strings.TrimSpace(c.Value))
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	res, err := h.db.Exec(ctx, `delete from webauthn_credentials where id = $1 and user_id = $2`, id, sess.UserID)
	if err != nil {
		http.Error(w, "server error", http.StatusInternalServerError)
		return
	}

	if res.RowsAffected() == 0 {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
