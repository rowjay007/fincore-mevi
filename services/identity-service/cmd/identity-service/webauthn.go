package main

import (
	"encoding/json"
	"net/http"

	"github.com/go-webauthn/webauthn/webauthn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type WebAuthnHandler struct {
	w       *webauthn.WebAuthn
	db      *pgxpool.Pool
	session *browserSessionStore
}

func NewWebAuthnHandler(db *pgxpool.Pool, session *browserSessionStore, issuer string) (*WebAuthnHandler, error) {
	w, err := webauthn.New(&webauthn.Config{
		RPDisplayName: "FinCore OS",
		RPID:          "localhost",
		RPOrigins:     []string{"http://localhost:8080", "http://localhost:8084"},
	})
	if err != nil {
		return nil, err
	}
	return &WebAuthnHandler{w: w, db: db, session: session}, nil
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

func (h *WebAuthnHandler) BeginLogin(w http.ResponseWriter, r *http.Request) {
	email := r.URL.Query().Get("email")
	if email == "" {
		http.Error(w, "email required", http.StatusBadRequest)
		return
	}

	// 1. Fetch user and their stored credentials from DB
	// For demo/sim, we'll create a mock user. In production, this queries the 'users' and 'user_credentials' tables.
	user := &webauthnUser{
		id:          []byte(email),
		name:        email,
		displayName: email,
		credentials: []webauthn.Credential{}, // Load from DB here
	}

	options, _, err := h.w.BeginLogin(user)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// 2. Store session data in DB or encrypted cookie (simulated here)
	// h.storeWebAuthnSession(email, session)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(options)
}

func (h *WebAuthnHandler) FinishLogin(w http.ResponseWriter, r *http.Request) {
	// 1. Parse assertion from client
	// 2. Validate against stored session and user credentials
	// 3. If valid, create a browser session

	// Simulation for Demo:
	sessionID, _, ok := h.session.put(r.Context(), "user_123", "simulated_access_token")
	if !ok {
		http.Error(w, "failed to create session", http.StatusInternalServerError)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "fincore_authorize_session",
		Value:    sessionID,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})

	w.WriteHeader(http.StatusOK)
}
