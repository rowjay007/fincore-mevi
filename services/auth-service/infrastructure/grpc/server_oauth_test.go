package grpc

import (
	"context"
	"encoding/base64"
	"testing"
	"time"

	authv1 "fincore/gen/go/auth/v1"
	"fincore/pkg/security"

	"github.com/jackc/pgx/v5"
	"github.com/pashagolub/pgxmock/v4"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type oauthAdminTokenMaker struct{}

func TestOAuthAuthorize_RejectsRedirectURIFragment(t *testing.T) {
	db, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock: %v", err)
	}
	defer db.Close()

	s := NewServer(db, oauthUserTokenMaker{}, 15*time.Minute, 30*24*time.Hour)

	redirect := "https://app.example/cb#frag"
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs("authorization", "Bearer access"))
	_, err = s.OAuthAuthorize(ctx, &authv1.OAuthAuthorizeRequest{ResponseType: "code", ClientId: "c1", RedirectUri: redirect, Scope: "openid", State: "s", CodeChallenge: "challenge", CodeChallengeMethod: "S256"})
	if err == nil {
		t.Fatalf("expected error")
	}
}

func TestOAuthToken_ConfidentialClient_BasicAuthOK(t *testing.T) {
	db, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock: %v", err)
	}
	defer db.Close()

	// server token maker isn't used for client auth
	s := NewServer(db, oauthUserTokenMaker{}, 15*time.Minute, 30*24*time.Hour)

	secretHash, err := security.HashPassword("sec")
	if err != nil {
		t.Fatalf("hash: %v", err)
	}
	secretHashPtr := &secretHash

	db.ExpectQuery("select id, name, type, secret_hash, redirect_uris, allowed_scopes from oauth_clients").WithArgs("c1").
		WillReturnRows(pgxmock.NewRows([]string{"id", "name", "type", "secret_hash", "redirect_uris", "allowed_scopes"}).AddRow("c1", "client", "confidential", secretHashPtr, []string{"https://app.example/cb"}, []string{}))

	db.ExpectBegin()
	exp := time.Now().UTC().Add(2 * time.Minute)
	storedChallenge := hashB64URLSHA256("ver")
	db.ExpectQuery("select user_id, redirect_uri, scopes, code_challenge").WithArgs(pgxmock.AnyArg(), "c1").
		WillReturnRows(pgxmock.NewRows([]string{"user_id", "redirect_uri", "scopes", "code_challenge", "code_challenge_method", "expires_at", "consumed_at"}).AddRow("user-1", "https://app.example/cb", []string{"openid"}, storedChallenge, "S256", exp, nil))
	db.ExpectExec("update oauth_authorization_codes set consumed_at").WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg()).WillReturnResult(pgxmock.NewResult("UPDATE", 1))

	db.ExpectQuery("select r.name").WithArgs("user-1").WillReturnRows(pgxmock.NewRows([]string{"name"}).AddRow("customer"))
	db.ExpectQuery("select distinct p.name").WithArgs("user-1").WillReturnRows(pgxmock.NewRows([]string{"name"}).AddRow("account:read"))

	db.ExpectExec("insert into auth_refresh_sessions").WithArgs(pgxmock.AnyArg(), "user-1", pgxmock.AnyArg(), pgxmock.AnyArg(), nil, nil).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))
	db.ExpectCommit()

	basic := base64.StdEncoding.EncodeToString([]byte("c1:sec"))
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs("authorization", "Basic "+basic))
	res, err := s.OAuthToken(ctx, &authv1.OAuthTokenRequest{GrantType: "authorization_code", Code: "code", RedirectUri: "https://app.example/cb", ClientId: "c1", CodeVerifier: "ver"})
	if err != nil {
		t.Fatalf("OAuthToken: %v", err)
	}
	if res.AccessToken == "" || res.RefreshToken == "" {
		t.Fatalf("expected tokens")
	}

	if err := db.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations: %v", err)
	}
}

func TestListOAuthConsentHistory_DefaultLimitAndOrdering(t *testing.T) {
	db, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock: %v", err)
	}
	defer db.Close()

	s := NewServer(db, oauthUserTokenMaker{}, 15*time.Minute, 30*24*time.Hour)

	now := time.Now().UTC()
	rows := pgxmock.NewRows([]string{"scopes", "created_at"}).
		AddRow([]string{"openid"}, now).
		AddRow([]string{"openid", "profile"}, now.Add(-time.Minute))

	db.ExpectQuery("select scopes, created_at").WithArgs("u1", "c1", 50).WillReturnRows(rows)

	res, err := s.ListOAuthConsentHistory(context.Background(), &authv1.ListOAuthConsentHistoryRequest{UserId: "u1", ClientId: "c1"})
	if err != nil {
		t.Fatalf("ListOAuthConsentHistory: %v", err)
	}
	if res == nil || len(res.Entries) != 2 {
		t.Fatalf("expected 2 entries")
	}
	if res.Entries[0].CreatedAt == nil || res.Entries[1].CreatedAt == nil {
		t.Fatalf("expected timestamps")
	}
	if res.Entries[0].CreatedAt.AsTime() != now {
		t.Fatalf("expected first entry time %v, got %v", now, res.Entries[0].CreatedAt.AsTime())
	}

	if err := db.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations: %v", err)
	}
}

func TestStoreOAuthConsent_AppendsHistoryOnCreate(t *testing.T) {
	db, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock: %v", err)
	}
	defer db.Close()

	s := NewServer(db, oauthUserTokenMaker{}, 15*time.Minute, 30*24*time.Hour)

	db.ExpectBegin()
	db.ExpectQuery("select scopes from oauth_consents").WithArgs("u1", "c1").WillReturnError(pgx.ErrNoRows)
	db.ExpectExec("insert into oauth_consents").WithArgs("u1", "c1", []string{"openid"}).WillReturnResult(pgxmock.NewResult("INSERT", 1))
	db.ExpectExec("insert into oauth_consent_history").WithArgs("u1", "c1", []string{"openid"}).WillReturnResult(pgxmock.NewResult("INSERT", 1))
	db.ExpectCommit()

	_, err = s.StoreOAuthConsent(context.Background(), &authv1.StoreOAuthConsentRequest{UserId: "u1", ClientId: "c1", Scopes: []string{"openid"}})
	if err != nil {
		t.Fatalf("StoreOAuthConsent: %v", err)
	}

	if err := db.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations: %v", err)
	}
}

func TestStoreOAuthConsent_AppendsHistoryOnChange(t *testing.T) {
	db, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock: %v", err)
	}
	defer db.Close()

	s := NewServer(db, oauthUserTokenMaker{}, 15*time.Minute, 30*24*time.Hour)

	db.ExpectBegin()
	db.ExpectQuery("select scopes from oauth_consents").WithArgs("u1", "c1").
		WillReturnRows(pgxmock.NewRows([]string{"scopes"}).AddRow([]string{"openid"}))
	db.ExpectExec("insert into oauth_consents").WithArgs("u1", "c1", []string{"openid", "profile"}).WillReturnResult(pgxmock.NewResult("INSERT", 1))
	db.ExpectExec("insert into oauth_consent_history").WithArgs("u1", "c1", []string{"openid", "profile"}).WillReturnResult(pgxmock.NewResult("INSERT", 1))
	db.ExpectCommit()

	_, err = s.StoreOAuthConsent(context.Background(), &authv1.StoreOAuthConsentRequest{UserId: "u1", ClientId: "c1", Scopes: []string{"openid", "profile"}})
	if err != nil {
		t.Fatalf("StoreOAuthConsent: %v", err)
	}

	if err := db.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations: %v", err)
	}
}

func TestStoreOAuthConsent_DoesNotAppendHistoryOnNoop(t *testing.T) {
	db, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock: %v", err)
	}
	defer db.Close()

	s := NewServer(db, oauthUserTokenMaker{}, 15*time.Minute, 30*24*time.Hour)

	db.ExpectBegin()
	db.ExpectQuery("select scopes from oauth_consents").WithArgs("u1", "c1").
		WillReturnRows(pgxmock.NewRows([]string{"scopes"}).AddRow([]string{"openid", "profile"}))
	db.ExpectExec("insert into oauth_consents").WithArgs("u1", "c1", []string{"profile", "openid"}).WillReturnResult(pgxmock.NewResult("INSERT", 1))
	db.ExpectCommit()

	_, err = s.StoreOAuthConsent(context.Background(), &authv1.StoreOAuthConsentRequest{UserId: "u1", ClientId: "c1", Scopes: []string{"profile", "openid"}})
	if err != nil {
		t.Fatalf("StoreOAuthConsent: %v", err)
	}

	if err := db.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations: %v", err)
	}
}

func TestOAuthToken_UnknownClient_ReturnsInvalidClient(t *testing.T) {
	db, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock: %v", err)
	}
	defer db.Close()

	s := NewServer(db, oauthUserTokenMaker{}, 15*time.Minute, 30*24*time.Hour)

	db.ExpectQuery("select id, name, type, secret_hash, redirect_uris, allowed_scopes from oauth_clients").WithArgs("c-unknown").
		WillReturnError(pgx.ErrNoRows)

	_, err = s.OAuthToken(context.Background(), &authv1.OAuthTokenRequest{GrantType: "authorization_code", Code: "code", RedirectUri: "https://app.example/cb", ClientId: "c-unknown", CodeVerifier: "ver"})
	if err == nil {
		t.Fatalf("expected error")
	}
	if st, ok := status.FromError(err); ok {
		if st.Message() != "invalid_client" {
			t.Fatalf("expected invalid_client, got %q", st.Message())
		}
	}

	if err := db.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations: %v", err)
	}
}

func TestOAuthToken_ConfidentialClient_UsesBasicAuthClientID(t *testing.T) {
	db, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock: %v", err)
	}
	defer db.Close()

	s := NewServer(db, oauthUserTokenMaker{}, 15*time.Minute, 30*24*time.Hour)

	secretHash, err := security.HashPassword("sec")
	if err != nil {
		t.Fatalf("hash: %v", err)
	}
	secretHashPtr := &secretHash

	db.ExpectQuery("select id, name, type, secret_hash, redirect_uris, allowed_scopes from oauth_clients").WithArgs("c1").
		WillReturnRows(pgxmock.NewRows([]string{"id", "name", "type", "secret_hash", "redirect_uris", "allowed_scopes"}).AddRow("c1", "client", "confidential", secretHashPtr, []string{"https://app.example/cb"}, []string{}))

	basic := base64.StdEncoding.EncodeToString([]byte("c1:sec"))
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs("authorization", "Basic "+basic))

	db.ExpectBegin()
	db.ExpectQuery("select user_id, redirect_uri, scopes, code_challenge").WithArgs(pgxmock.AnyArg(), "c1").
		WillReturnError(pgx.ErrNoRows)
	db.ExpectRollback()

	_, err = s.OAuthToken(ctx, &authv1.OAuthTokenRequest{GrantType: "authorization_code", Code: "code", RedirectUri: "https://app.example/cb", CodeVerifier: "ver"})
	if err == nil {
		t.Fatalf("expected error")
	}
	if st, ok := status.FromError(err); ok {
		if st.Message() != "invalid_grant" {
			t.Fatalf("expected invalid_grant, got %q", st.Message())
		}
	}

	if err := db.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations: %v", err)
	}
}

func TestOAuthAuthorize_RejectsDisallowedScope(t *testing.T) {
	db, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock: %v", err)
	}
	defer db.Close()

	s := NewServer(db, oauthUserTokenMaker{}, 15*time.Minute, 30*24*time.Hour)

	db.ExpectQuery("select id, name, type, secret_hash, redirect_uris, allowed_scopes from oauth_clients").WithArgs("c1").
		WillReturnRows(pgxmock.NewRows([]string{"id", "name", "type", "secret_hash", "redirect_uris", "allowed_scopes"}).AddRow("c1", "client", "public", nil, []string{"https://app.example/cb"}, []string{"openid"}))

	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs("authorization", "Bearer access"))
	_, err = s.OAuthAuthorize(ctx, &authv1.OAuthAuthorizeRequest{ResponseType: "code", ClientId: "c1", RedirectUri: "https://app.example/cb", Scope: "openid profile", State: "s", CodeChallenge: "challenge", CodeChallengeMethod: "S256"})
	if err == nil {
		t.Fatalf("expected error")
	}

	if err := db.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations: %v", err)
	}
}

func TestOAuthToken_ConfidentialClient_MissingSecretDenied(t *testing.T) {
	db, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock: %v", err)
	}
	defer db.Close()

	s := NewServer(db, oauthUserTokenMaker{}, 15*time.Minute, 30*24*time.Hour)

	secretHash, err := security.HashPassword("sec")
	if err != nil {
		t.Fatalf("hash: %v", err)
	}
	secretHashPtr := &secretHash

	db.ExpectQuery("select id, name, type, secret_hash, redirect_uris, allowed_scopes from oauth_clients").WithArgs("c1").
		WillReturnRows(pgxmock.NewRows([]string{"id", "name", "type", "secret_hash", "redirect_uris", "allowed_scopes"}).AddRow("c1", "client", "confidential", secretHashPtr, []string{"https://app.example/cb"}, []string{}))

	_, err = s.OAuthToken(context.Background(), &authv1.OAuthTokenRequest{GrantType: "authorization_code", Code: "code", RedirectUri: "https://app.example/cb", ClientId: "c1", CodeVerifier: "ver"})
	if err == nil {
		t.Fatalf("expected error")
	}
	if st, ok := status.FromError(err); ok {
		if st.Message() != "invalid_client" {
			t.Fatalf("expected invalid_client, got %q", st.Message())
		}
	}

	if err := db.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations: %v", err)
	}
}

func (oauthAdminTokenMaker) CreateToken(payload security.TokenPayload) (string, error) {
	return "access", nil
}
func (oauthAdminTokenMaker) VerifyToken(token string) (*security.TokenPayload, error) {
	now := time.Now().UTC()
	return &security.TokenPayload{UserID: "admin-1", Permissions: []string{"auth:admin"}, IssuedAt: now, ExpiredAt: now.Add(time.Minute)}, nil
}

type oauthUserTokenMaker struct{}

func (oauthUserTokenMaker) CreateToken(payload security.TokenPayload) (string, error) {
	return "access", nil
}
func (oauthUserTokenMaker) VerifyToken(token string) (*security.TokenPayload, error) {
	now := time.Now().UTC()
	return &security.TokenPayload{UserID: "user-1", Roles: []string{"customer"}, Permissions: []string{"account:read"}, IssuedAt: now, ExpiredAt: now.Add(time.Minute)}, nil
}

func TestCreateOAuthClient_AdminAllowed(t *testing.T) {
	db, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock: %v", err)
	}
	defer db.Close()

	s := NewServer(db, oauthAdminTokenMaker{}, 15*time.Minute, 30*24*time.Hour)

	db.ExpectExec("insert into oauth_clients").WithArgs(pgxmock.AnyArg(), "client", "public", nil, []string{"https://app.example/cb"}, []string{}).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))

	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs("authorization", "Bearer access"))
	res, err := s.CreateOAuthClient(ctx, &authv1.CreateOAuthClientRequest{Name: "client", Type: "public", RedirectUris: []string{"https://app.example/cb"}})
	if err != nil {
		t.Fatalf("CreateOAuthClient: %v", err)
	}
	if res.Client == nil || res.Client.ClientId == "" {
		t.Fatalf("expected client id")
	}

	if err := db.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations: %v", err)
	}
}

func TestOAuthAuthorize_IssuesCodeWithPKCE(t *testing.T) {
	db, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock: %v", err)
	}
	defer db.Close()

	s := NewServer(db, oauthUserTokenMaker{}, 15*time.Minute, 30*24*time.Hour)

	db.ExpectQuery("select id, name, type, secret_hash, redirect_uris, allowed_scopes from oauth_clients").WithArgs("c1").
		WillReturnRows(pgxmock.NewRows([]string{"id", "name", "type", "secret_hash", "redirect_uris", "allowed_scopes"}).AddRow("c1", "client", "public", nil, []string{"https://app.example/cb"}, []string{}))
	// Insert code
	db.ExpectExec("insert into oauth_authorization_codes").WithArgs(pgxmock.AnyArg(), "c1", "user-1", "https://app.example/cb", []string{"openid"}, "challenge", "S256", pgxmock.AnyArg()).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))

	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs("authorization", "Bearer access"))
	res, err := s.OAuthAuthorize(ctx, &authv1.OAuthAuthorizeRequest{ResponseType: "code", ClientId: "c1", RedirectUri: "https://app.example/cb", Scope: "openid", State: "s", CodeChallenge: "challenge", CodeChallengeMethod: "S256"})
	if err != nil {
		t.Fatalf("OAuthAuthorize: %v", err)
	}
	if res.Code == "" || res.RedirectUrl == "" {
		t.Fatalf("expected code and redirect_url")
	}

	if err := db.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations: %v", err)
	}
}

func TestOAuthToken_ExchangeConsumesCode(t *testing.T) {
	db, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock: %v", err)
	}
	defer db.Close()

	s := NewServer(db, oauthUserTokenMaker{}, 15*time.Minute, 30*24*time.Hour)

	db.ExpectQuery("select id, name, type, secret_hash, redirect_uris, allowed_scopes from oauth_clients").WithArgs("c1").
		WillReturnRows(pgxmock.NewRows([]string{"id", "name", "type", "secret_hash", "redirect_uris", "allowed_scopes"}).AddRow("c1", "client", "public", nil, []string{"https://app.example/cb"}, []string{}))

	db.ExpectBegin()
	exp := time.Now().UTC().Add(2 * time.Minute)
	// stored challenge equals sha256(verifier) where verifier="ver"
	storedChallenge := hashB64URLSHA256("ver")
	db.ExpectQuery("select user_id, redirect_uri, scopes, code_challenge").WithArgs(pgxmock.AnyArg(), "c1").
		WillReturnRows(pgxmock.NewRows([]string{"user_id", "redirect_uri", "scopes", "code_challenge", "code_challenge_method", "expires_at", "consumed_at"}).AddRow("user-1", "https://app.example/cb", []string{"openid"}, storedChallenge, "S256", exp, nil))
	db.ExpectExec("update oauth_authorization_codes set consumed_at").WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg()).WillReturnResult(pgxmock.NewResult("UPDATE", 1))

	// roles + permissions are queried on s.db (not tx)
	db.ExpectQuery("select r.name").WithArgs("user-1").WillReturnRows(pgxmock.NewRows([]string{"name"}).AddRow("customer"))
	db.ExpectQuery("select distinct p.name").WithArgs("user-1").WillReturnRows(pgxmock.NewRows([]string{"name"}).AddRow("account:read"))

	db.ExpectExec("insert into auth_refresh_sessions").WithArgs(pgxmock.AnyArg(), "user-1", pgxmock.AnyArg(), pgxmock.AnyArg(), nil, nil).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))
	db.ExpectCommit()

	res, err := s.OAuthToken(context.Background(), &authv1.OAuthTokenRequest{GrantType: "authorization_code", Code: "code", RedirectUri: "https://app.example/cb", ClientId: "c1", CodeVerifier: "ver"})
	if err != nil {
		t.Fatalf("OAuthToken: %v", err)
	}
	if res.AccessToken == "" || res.RefreshToken == "" {
		t.Fatalf("expected tokens")
	}

	if err := db.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations: %v", err)
	}
}

func TestOAuthToken_DeniesReplay(t *testing.T) {
	db, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock: %v", err)
	}
	defer db.Close()

	s := NewServer(db, oauthUserTokenMaker{}, 15*time.Minute, 30*24*time.Hour)

	db.ExpectQuery("select id, name, type, secret_hash, redirect_uris, allowed_scopes from oauth_clients").WithArgs("c1").
		WillReturnRows(pgxmock.NewRows([]string{"id", "name", "type", "secret_hash", "redirect_uris", "allowed_scopes"}).AddRow("c1", "client", "public", nil, []string{"https://app.example/cb"}, []string{}))

	db.ExpectBegin()
	exp := time.Now().UTC().Add(2 * time.Minute)
	used := time.Now().UTC().Add(-time.Minute)
	db.ExpectQuery("select user_id, redirect_uri, scopes, code_challenge").WithArgs(pgxmock.AnyArg(), "c1").
		WillReturnRows(pgxmock.NewRows([]string{"user_id", "redirect_uri", "scopes", "code_challenge", "code_challenge_method", "expires_at", "consumed_at"}).AddRow("user-1", "https://app.example/cb", []string{"openid"}, "x", "S256", exp, &used))
	db.ExpectRollback()

	_, err = s.OAuthToken(context.Background(), &authv1.OAuthTokenRequest{GrantType: "authorization_code", Code: "code", RedirectUri: "https://app.example/cb", ClientId: "c1", CodeVerifier: "ver"})
	if err == nil {
		t.Fatalf("expected error")
	}

	if err := db.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations: %v", err)
	}
}
