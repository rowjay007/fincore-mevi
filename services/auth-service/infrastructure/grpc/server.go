package grpc

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"net/mail"
	"net/url"
	"strings"
	"time"

	authv1 "fincore/gen/go/auth/v1"
	"fincore/pkg/security"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

func invalidArg(msg string) error  { return status.Error(codes.InvalidArgument, msg) }
func unauth(msg string) error      { return status.Error(codes.Unauthenticated, msg) }
func forbidden(msg string) error   { return status.Error(codes.PermissionDenied, msg) }
func notFound(msg string) error    { return status.Error(codes.NotFound, msg) }
func rateLimited(msg string) error { return status.Error(codes.ResourceExhausted, msg) }
func internal(err error) error {
	if err == nil {
		return status.Error(codes.Internal, "internal error")
	}
	return status.Error(codes.Internal, err.Error())
}

func (s *Server) userIDFromAuthHeader(ctx context.Context) (string, error) {
	accessToken := ""
	if md, ok := metadata.FromIncomingContext(ctx); ok {
		vals := md.Get("authorization")
		if len(vals) == 0 {
			vals = md.Get("Authorization")
		}
		if len(vals) > 0 {
			v := strings.TrimSpace(vals[0])
			lv := strings.ToLower(v)
			if strings.HasPrefix(lv, "bearer ") {
				accessToken = strings.TrimSpace(v[len("bearer "):])
			}
		}
	}
	if accessToken == "" {
		return "", unauth("access token required")
	}
	payload, err := s.tokens.VerifyToken(accessToken)
	if err != nil {
		if errors.Is(err, security.ErrExpiredToken) || errors.Is(err, security.ErrInvalidToken) {
			return "", unauth("invalid access token")
		}
		return "", internal(err)
	}
	userID := strings.TrimSpace(payload.UserID)
	if userID == "" {
		return "", unauth("invalid access token")
	}
	return userID, nil
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

func hashB64URLSHA256(s string) string {
	h := sha256.Sum256([]byte(s))
	return base64.RawURLEncoding.EncodeToString(h[:])
}

func normalizeClientType(v string) (string, error) {
	v = strings.TrimSpace(strings.ToLower(v))
	if v == "" {
		return "public", nil
	}
	if v != "public" && v != "confidential" {
		return "", invalidArg("invalid client type")
	}
	return v, nil
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

func (s *Server) fetchOAuthClient(ctx context.Context, clientID string) (oauthClientRecord, error) {
	clientID = strings.TrimSpace(clientID)
	if clientID == "" {
		return oauthClientRecord{}, invalidArg("client_id required")
	}

	var rec oauthClientRecord
	err := s.db.QueryRow(ctx, `select id, name, type, secret_hash, redirect_uris, allowed_scopes from oauth_clients where id = $1`, clientID).
		Scan(&rec.ID, &rec.Name, &rec.Type, &rec.SecretHash, &rec.RedirectURIs, &rec.AllowedScopes)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return oauthClientRecord{}, notFound("unknown client")
		}
		return oauthClientRecord{}, internal(err)
	}
	return rec, nil
}

func (s *Server) CreateOAuthClient(ctx context.Context, req *authv1.CreateOAuthClientRequest) (*authv1.CreateOAuthClientResponse, error) {
	if err := s.requirePermissionFromAuthHeader(ctx, "auth:admin"); err != nil {
		return nil, err
	}
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return nil, invalidArg("name required")
	}
	ct, err := normalizeClientType(req.Type)
	if err != nil {
		return nil, err
	}
	if len(req.RedirectUris) == 0 {
		return nil, invalidArg("redirect_uris required")
	}
	redirects := make([]string, 0, len(req.RedirectUris))
	for _, r := range req.RedirectUris {
		r = strings.TrimSpace(r)
		if r == "" {
			continue
		}
		u, err := url.Parse(r)
		if err != nil || u.Scheme == "" || u.Host == "" {
			return nil, invalidArg("invalid redirect_uri")
		}
		redirects = append(redirects, r)
	}
	if len(redirects) == 0 {
		return nil, invalidArg("redirect_uris required")
	}
	allowedScopes := make([]string, 0, len(req.AllowedScopes))
	for _, s := range req.AllowedScopes {
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}
		allowedScopes = append(allowedScopes, s)
	}

	clientID := uuid.NewString()
	var clientSecret string
	var secretHash *string
	if ct == "confidential" {
		clientSecret, err = randomB64URL(32)
		if err != nil {
			return nil, internal(err)
		}
		h, err := security.HashPassword(clientSecret)
		if err != nil {
			return nil, internal(err)
		}
		secretHash = &h
	}

	var secretArg any
	if secretHash != nil {
		secretArg = *secretHash
	}
	_, err = s.db.Exec(ctx, `insert into oauth_clients (id, name, type, secret_hash, redirect_uris, allowed_scopes) values ($1,$2,$3,$4,$5,$6)`, clientID, name, ct, secretArg, redirects, allowedScopes)
	if err != nil {
		return nil, internal(err)
	}

	res := &authv1.CreateOAuthClientResponse{
		Client: &authv1.OAuthClient{ClientId: clientID, Name: name, Type: ct, RedirectUris: redirects, AllowedScopes: allowedScopes},
	}
	if clientSecret != "" {
		res.ClientSecret = clientSecret
	}
	return res, nil
}

func (s *Server) GetOAuthClient(ctx context.Context, req *authv1.GetOAuthClientRequest) (*authv1.GetOAuthClientResponse, error) {
	if err := s.requirePermissionFromAuthHeader(ctx, "auth:admin"); err != nil {
		return nil, err
	}
	rec, err := s.fetchOAuthClient(ctx, req.ClientId)
	if err != nil {
		return nil, err
	}
	return &authv1.GetOAuthClientResponse{Client: &authv1.OAuthClient{ClientId: rec.ID, Name: rec.Name, Type: rec.Type, RedirectUris: rec.RedirectURIs, AllowedScopes: rec.AllowedScopes}}, nil
}

func (s *Server) ListOAuthClients(ctx context.Context, _ *authv1.ListOAuthClientsRequest) (*authv1.ListOAuthClientsResponse, error) {
	if err := s.requirePermissionFromAuthHeader(ctx, "auth:admin"); err != nil {
		return nil, err
	}
	rows, err := s.db.Query(ctx, `select id, name, type, redirect_uris, allowed_scopes from oauth_clients order by created_at desc`)
	if err != nil {
		return nil, internal(err)
	}
	defer rows.Close()

	var out []*authv1.OAuthClient
	for rows.Next() {
		var id, name, typ string
		var redirects []string
		var scopes []string
		if err := rows.Scan(&id, &name, &typ, &redirects, &scopes); err != nil {
			return nil, internal(err)
		}
		out = append(out, &authv1.OAuthClient{ClientId: id, Name: name, Type: typ, RedirectUris: redirects, AllowedScopes: scopes})
	}
	if err := rows.Err(); err != nil {
		return nil, internal(err)
	}
	return &authv1.ListOAuthClientsResponse{Clients: out}, nil
}

func (s *Server) DeleteOAuthClient(ctx context.Context, req *authv1.DeleteOAuthClientRequest) (*authv1.DeleteOAuthClientResponse, error) {
	if err := s.requirePermissionFromAuthHeader(ctx, "auth:admin"); err != nil {
		return nil, err
	}
	clientID := strings.TrimSpace(req.ClientId)
	if clientID == "" {
		return nil, invalidArg("client_id required")
	}
	_, err := s.db.Exec(ctx, `delete from oauth_clients where id = $1`, clientID)
	if err != nil {
		return nil, internal(err)
	}
	return &authv1.DeleteOAuthClientResponse{Success: true}, nil
}

func (s *Server) RotateOAuthClientSecret(ctx context.Context, req *authv1.RotateOAuthClientSecretRequest) (*authv1.RotateOAuthClientSecretResponse, error) {
	if err := s.requirePermissionFromAuthHeader(ctx, "auth:admin"); err != nil {
		return nil, err
	}
	rec, err := s.fetchOAuthClient(ctx, req.ClientId)
	if err != nil {
		return nil, err
	}
	if rec.Type != "confidential" {
		return nil, invalidArg("client is not confidential")
	}
	secret, err := randomB64URL(32)
	if err != nil {
		return nil, internal(err)
	}
	h, err := security.HashPassword(secret)
	if err != nil {
		return nil, internal(err)
	}
	_, err = s.db.Exec(ctx, `update oauth_clients set secret_hash = $2 where id = $1`, rec.ID, h)
	if err != nil {
		return nil, internal(err)
	}
	return &authv1.RotateOAuthClientSecretResponse{ClientSecret: secret}, nil
}

func (s *Server) OAuthAuthorize(ctx context.Context, req *authv1.OAuthAuthorizeRequest) (*authv1.OAuthAuthorizeResponse, error) {
	if strings.TrimSpace(req.ResponseType) != "code" {
		return nil, invalidArg("unsupported response_type")
	}
	clientID := strings.TrimSpace(req.ClientId)
	redirectURI := strings.TrimSpace(req.RedirectUri)
	if clientID == "" {
		return nil, invalidArg("client_id required")
	}
	if redirectURI == "" {
		return nil, invalidArg("redirect_uri required")
	}

	rec, err := s.fetchOAuthClient(ctx, clientID)
	if err != nil {
		return nil, err
	}
	allowedRedirect := false
	for _, r := range rec.RedirectURIs {
		if r == redirectURI {
			allowedRedirect = true
			break
		}
	}
	if !allowedRedirect {
		return nil, invalidArg("invalid redirect_uri")
	}

	challenge := strings.TrimSpace(req.CodeChallenge)
	method := strings.TrimSpace(req.CodeChallengeMethod)
	if challenge == "" {
		return nil, invalidArg("code_challenge required")
	}
	if strings.ToUpper(method) != "S256" {
		return nil, invalidArg("code_challenge_method must be S256")
	}

	userID, err := s.userIDFromAuthHeader(ctx)
	if err != nil {
		return nil, err
	}

	code, err := randomB64URL(32)
	if err != nil {
		return nil, internal(err)
	}
	codeHash := hashB64URLSHA256(code)

	exp := time.Now().UTC().Add(5 * time.Minute)
	scopes := splitScopes(req.Scope)
	_, err = s.db.Exec(ctx, `insert into oauth_authorization_codes (code_hash, client_id, user_id, redirect_uri, scopes, code_challenge, code_challenge_method, expires_at) values ($1,$2,$3,$4,$5,$6,$7,$8)`,
		codeHash, rec.ID, userID, redirectURI, scopes, challenge, "S256", exp)
	if err != nil {
		return nil, internal(err)
	}

	u, err := url.Parse(redirectURI)
	if err != nil {
		return nil, internal(err)
	}
	q := u.Query()
	q.Set("code", code)
	if strings.TrimSpace(req.State) != "" {
		q.Set("state", strings.TrimSpace(req.State))
	}
	u.RawQuery = q.Encode()

	return &authv1.OAuthAuthorizeResponse{Code: code, State: strings.TrimSpace(req.State), RedirectUrl: u.String()}, nil
}

func (s *Server) OAuthToken(ctx context.Context, req *authv1.OAuthTokenRequest) (*authv1.OAuthTokenResponse, error) {
	if strings.TrimSpace(req.GrantType) != "authorization_code" {
		return nil, invalidArg("unsupported grant_type")
	}
	clientID := strings.TrimSpace(req.ClientId)
	if clientID == "" {
		return nil, invalidArg("client_id required")
	}
	code := strings.TrimSpace(req.Code)
	if code == "" {
		return nil, invalidArg("code required")
	}
	redirectURI := strings.TrimSpace(req.RedirectUri)
	if redirectURI == "" {
		return nil, invalidArg("redirect_uri required")
	}
	verifier := strings.TrimSpace(req.CodeVerifier)
	if verifier == "" {
		return nil, invalidArg("code_verifier required")
	}

	rec, err := s.fetchOAuthClient(ctx, clientID)
	if err != nil {
		return nil, err
	}
	if rec.Type == "confidential" {
		if rec.SecretHash == nil {
			return nil, internal(errors.New("confidential client missing secret"))
		}
		sec := strings.TrimSpace(req.ClientSecret)
		if sec == "" {
			return nil, unauth("client_secret required")
		}
		ok, err := security.VerifyPassword(sec, *rec.SecretHash)
		if err != nil {
			return nil, internal(err)
		}
		if !ok {
			return nil, unauth("invalid client_secret")
		}
	}

	codeHash := hashB64URLSHA256(code)

	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, internal(err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var userID string
	var storedRedirect string
	var scopes []string
	var storedChallenge string
	var method string
	var expiresAt time.Time
	var consumedAt *time.Time
	row := tx.QueryRow(ctx, `select user_id, redirect_uri, scopes, code_challenge, code_challenge_method, expires_at, consumed_at from oauth_authorization_codes where code_hash = $1 and client_id = $2`, codeHash, rec.ID)
	if err := row.Scan(&userID, &storedRedirect, &scopes, &storedChallenge, &method, &expiresAt, &consumedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, unauth("invalid code")
		}
		return nil, internal(err)
	}

	now := time.Now().UTC()
	if consumedAt != nil {
		return nil, unauth("code already used")
	}
	if now.After(expiresAt.UTC()) {
		return nil, unauth("code expired")
	}
	if storedRedirect != redirectURI {
		return nil, unauth("invalid redirect_uri")
	}
	if strings.ToUpper(strings.TrimSpace(method)) != "S256" {
		return nil, unauth("invalid code")
	}
	expected := strings.TrimSpace(storedChallenge)
	got := hashB64URLSHA256(verifier)
	if expected == "" || got == "" || expected != got {
		return nil, unauth("invalid code_verifier")
	}

	_, err = tx.Exec(ctx, `update oauth_authorization_codes set consumed_at = $2 where code_hash = $1 and consumed_at is null`, codeHash, now)
	if err != nil {
		return nil, internal(err)
	}

	roles, perms, err := s.getRolesAndPermissions(ctx, userID)
	if err != nil {
		return nil, internal(err)
	}
	accessNow := time.Now().UTC()
	accessExp := accessNow.Add(s.accessTTL)
	access, err := s.tokens.CreateToken(security.TokenPayload{UserID: userID, Roles: roles, Permissions: perms, IssuedAt: accessNow, ExpiredAt: accessExp})
	if err != nil {
		return nil, internal(err)
	}

	refresh, err := newRefreshToken()
	if err != nil {
		return nil, internal(err)
	}
	refreshExp := accessNow.Add(s.refreshTTL)
	absExp := refreshExp
	refreshHash := security.HashRefreshToken(refresh)
	_, err = tx.Exec(ctx, `insert into auth_refresh_sessions (token_hash, session_id, user_id, expires_at, absolute_expires_at, user_agent, ip) values ($1,$1,$2,$3,$4,$5,$6)`, refreshHash, userID, refreshExp, absExp, nil, nil)
	if err != nil {
		return nil, internal(err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, internal(err)
	}

	_ = scopes
	return &authv1.OAuthTokenResponse{AccessToken: access, TokenType: "bearer", ExpiresIn: int64(s.accessTTL.Seconds()), RefreshToken: refresh}, nil
}

func normalizeEmail(email string) (string, error) {
	e := strings.TrimSpace(strings.ToLower(email))
	if e == "" {
		return "", invalidArg("email required")
	}
	if _, err := mail.ParseAddress(e); err != nil {
		return "", invalidArg("invalid email")
	}
	return e, nil
}

type Server struct {
	authv1.UnimplementedAuthServiceServer
	db         DB
	tokens     security.TokenMaker
	accessTTL  time.Duration
	refreshTTL time.Duration
	limiter    *loginLimiter
}

type oauthClientRecord struct {
	ID            string
	Name          string
	Type          string
	SecretHash    *string
	RedirectURIs  []string
	AllowedScopes []string
}

type DB interface {
	Begin(ctx context.Context) (pgx.Tx, error)
	Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

func NewServer(db DB, tokens security.TokenMaker, accessTTL time.Duration, refreshTTL time.Duration) *Server {
	return &Server{db: db, tokens: tokens, accessTTL: accessTTL, refreshTTL: refreshTTL}
}

func NewServerWithLoginLimiter(db DB, tokens security.TokenMaker, accessTTL time.Duration, refreshTTL time.Duration, maxAttempts int, window time.Duration, lockout time.Duration) *Server {
	return &Server{db: db, tokens: tokens, accessTTL: accessTTL, refreshTTL: refreshTTL, limiter: newLoginLimiter(maxAttempts, window, lockout)}
}

func (s *Server) Register(ctx context.Context, req *authv1.RegisterRequest) (*authv1.RegisterResponse, error) {
	email, err := normalizeEmail(req.Email)
	if err != nil {
		return nil, err
	}
	password := strings.TrimSpace(req.Password)
	if password == "" {
		return nil, invalidArg("password required")
	}
	if len(password) < 8 {
		return nil, invalidArg("password too short")
	}
	if strings.TrimSpace(req.FullName) == "" {
		return nil, invalidArg("full_name required")
	}

	pwHash, err := security.HashPassword(password)
	if err != nil {
		return nil, internal(err)
	}

	id := uuid.NewString()

	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, internal(err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	_, err = tx.Exec(ctx, `insert into auth_users (id, email, full_name, password_hash) values ($1,$2,$3,$4)`, id, email, req.FullName, pwHash)
	if err != nil {
		return nil, internal(err)
	}
	_, err = tx.Exec(ctx, `insert into auth_user_roles (user_id, role_id) values ($1,$2) on conflict do nothing`, id, "role_customer")
	if err != nil {
		return nil, internal(err)
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, internal(err)
	}
	return &authv1.RegisterResponse{UserId: id}, nil
}

func (s *Server) Login(ctx context.Context, req *authv1.LoginRequest) (*authv1.LoginResponse, error) {
	email, err := normalizeEmail(req.Email)
	if err != nil {
		return nil, err
	}
	password := strings.TrimSpace(req.Password)
	if password == "" {
		return nil, invalidArg("password required")
	}
	ua, ip := clientMetaFromContext(ctx)
	if s.limiter != nil {
		key := email + "|" + ip
		allowed, _ := s.limiter.allow(time.Now().UTC(), key)
		if !allowed {
			return nil, rateLimited("too many login attempts")
		}
	}

	var userID string
	var pwHash string
	err = s.db.QueryRow(ctx, `select id, password_hash from auth_users where email = $1`, email).Scan(&userID, &pwHash)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			if s.limiter != nil {
				key := email + "|" + ip
				s.limiter.onFailure(time.Now().UTC(), key)
			}
			return nil, unauth("invalid credentials")
		}
		return nil, internal(err)
	}

	ok, err := security.VerifyPassword(password, pwHash)
	if err != nil {
		return nil, internal(err)
	}
	if !ok {
		if s.limiter != nil {
			key := email + "|" + ip
			s.limiter.onFailure(time.Now().UTC(), key)
		}
		return nil, unauth("invalid credentials")
	}
	if s.limiter != nil {
		key := email + "|" + ip
		s.limiter.onSuccess(key)
	}

	roles, perms, err := s.getRolesAndPermissions(ctx, userID)
	if err != nil {
		return nil, internal(err)
	}

	accessNow := time.Now().UTC()
	accessExp := accessNow.Add(s.accessTTL)
	access, err := s.tokens.CreateToken(security.TokenPayload{
		UserID:      userID,
		Roles:       roles,
		Permissions: perms,
		IssuedAt:    accessNow,
		ExpiredAt:   accessExp,
	})
	if err != nil {
		return nil, internal(err)
	}

	refresh, err := newRefreshToken()
	if err != nil {
		return nil, internal(err)
	}
	refreshExp := accessNow.Add(s.refreshTTL)
	absExp := refreshExp
	refreshHash := security.HashRefreshToken(refresh)
	_, err = s.db.Exec(ctx, `
		insert into auth_refresh_sessions (token_hash, session_id, user_id, expires_at, absolute_expires_at, user_agent, ip)
		values ($1,$1,$2,$3,$4,$5,$6)
	`, refreshHash, userID, refreshExp, absExp, nullIfEmpty(ua), nullIfEmpty(ip))
	if err != nil {
		return nil, internal(err)
	}

	return &authv1.LoginResponse{
		AccessToken:  access,
		RefreshToken: refresh,
		ExpiresIn:    int64(s.accessTTL.Seconds()),
	}, nil
}

func (s *Server) getRolesAndPermissions(ctx context.Context, userID string) ([]string, []string, error) {
	rows, err := s.db.Query(ctx, `
		select r.name
		from auth_roles r
		join auth_user_roles ur on ur.role_id = r.id
		where ur.user_id = $1
		order by r.name
	`, userID)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	var roles []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, nil, err
		}
		roles = append(roles, name)
	}
	if err := rows.Err(); err != nil {
		return nil, nil, err
	}

	permRows, err := s.db.Query(ctx, `
		select distinct p.name
		from auth_permissions p
		join auth_role_permissions rp on rp.permission_id = p.id
		join auth_user_roles ur on ur.role_id = rp.role_id
		where ur.user_id = $1
		order by p.name
	`, userID)
	if err != nil {
		return nil, nil, err
	}
	defer permRows.Close()

	var perms []string
	for permRows.Next() {
		var name string
		if err := permRows.Scan(&name); err != nil {
			return nil, nil, err
		}
		perms = append(perms, name)
	}
	if err := permRows.Err(); err != nil {
		return nil, nil, err
	}

	if len(roles) == 0 {
		return nil, nil, fmt.Errorf("no roles assigned for user %s", userID)
	}

	return roles, perms, nil
}

func (s *Server) ValidateToken(ctx context.Context, req *authv1.ValidateTokenRequest) (*authv1.ValidateTokenResponse, error) {
	if strings.TrimSpace(req.AccessToken) == "" {
		return nil, invalidArg("access_token required")
	}
	payload, err := s.tokens.VerifyToken(req.AccessToken)
	if err != nil {
		if errors.Is(err, security.ErrExpiredToken) || errors.Is(err, security.ErrInvalidToken) {
			return nil, unauth("invalid access token")
		}
		return nil, internal(err)
	}
	return &authv1.ValidateTokenResponse{
		UserId:      payload.UserID,
		Roles:       payload.Roles,
		Permissions: payload.Permissions,
	}, nil
}

func (s *Server) GrantRole(ctx context.Context, req *authv1.GrantRoleRequest) (*authv1.GrantRoleResponse, error) {
	if err := s.requirePermissionFromAuthHeader(ctx, "auth:admin"); err != nil {
		return nil, err
	}
	userID := strings.TrimSpace(req.UserId)
	roleName := strings.TrimSpace(req.RoleName)
	if userID == "" {
		return nil, invalidArg("user_id required")
	}
	if roleName == "" {
		return nil, invalidArg("role_name required")
	}

	var roleID string
	err := s.db.QueryRow(ctx, `select id from auth_roles where name = $1`, roleName).Scan(&roleID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, notFound("unknown role")
		}
		return nil, internal(err)
	}

	_, err = s.db.Exec(ctx, `insert into auth_user_roles (user_id, role_id) values ($1,$2) on conflict do nothing`, userID, roleID)
	if err != nil {
		return nil, internal(err)
	}
	return &authv1.GrantRoleResponse{Success: true}, nil
}

func (s *Server) RevokeRole(ctx context.Context, req *authv1.RevokeRoleRequest) (*authv1.RevokeRoleResponse, error) {
	if err := s.requirePermissionFromAuthHeader(ctx, "auth:admin"); err != nil {
		return nil, err
	}
	userID := strings.TrimSpace(req.UserId)
	roleName := strings.TrimSpace(req.RoleName)
	if userID == "" {
		return nil, invalidArg("user_id required")
	}
	if roleName == "" {
		return nil, invalidArg("role_name required")
	}

	var roleID string
	err := s.db.QueryRow(ctx, `select id from auth_roles where name = $1`, roleName).Scan(&roleID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, notFound("unknown role")
		}
		return nil, internal(err)
	}

	_, err = s.db.Exec(ctx, `delete from auth_user_roles where user_id = $1 and role_id = $2`, userID, roleID)
	if err != nil {
		return nil, internal(err)
	}
	return &authv1.RevokeRoleResponse{Success: true}, nil
}

func (s *Server) ListUserRoles(ctx context.Context, req *authv1.ListUserRolesRequest) (*authv1.ListUserRolesResponse, error) {
	if err := s.requirePermissionFromAuthHeader(ctx, "auth:admin"); err != nil {
		return nil, err
	}
	userID := strings.TrimSpace(req.UserId)
	if userID == "" {
		return nil, invalidArg("user_id required")
	}

	rows, err := s.db.Query(ctx, `
		select r.name
		from auth_roles r
		join auth_user_roles ur on ur.role_id = r.id
		where ur.user_id = $1
		order by r.name
	`, userID)
	if err != nil {
		return nil, internal(err)
	}
	defer rows.Close()

	var roles []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, internal(err)
		}
		roles = append(roles, name)
	}
	if err := rows.Err(); err != nil {
		return nil, internal(err)
	}

	return &authv1.ListUserRolesResponse{Roles: roles}, nil
}

func (s *Server) RefreshToken(ctx context.Context, req *authv1.RefreshTokenRequest) (*authv1.RefreshTokenResponse, error) {
	refreshToken := strings.TrimSpace(req.RefreshToken)
	if refreshToken == "" {
		return nil, invalidArg("refresh_token required")
	}
	refreshHash := security.HashRefreshToken(refreshToken)

	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, internal(err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	var userID string
	var expiresAt time.Time
	var absExpiresAt time.Time
	var revokedAt *time.Time
	var replacedBy *string
	var sessionID string
	err = tx.QueryRow(ctx, `
		select user_id, expires_at, absolute_expires_at, revoked_at, replaced_by_hash, session_id
		from auth_refresh_sessions
		where token_hash = $1
		`, refreshHash).Scan(&userID, &expiresAt, &absExpiresAt, &revokedAt, &replacedBy, &sessionID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, unauth("invalid refresh token")
		}
		return nil, internal(err)
	}

	now := time.Now().UTC()
	if revokedAt != nil || replacedBy != nil {
		_, _ = tx.Exec(ctx, `update auth_refresh_sessions set revoked_at = coalesce(revoked_at, $2) where token_hash = $1`, refreshHash, now)
		_, _ = tx.Exec(ctx, `update auth_refresh_sessions set revoked_at = $2 where user_id = $1 and revoked_at is null`, userID, now)
		return nil, unauth("refresh token reuse detected")
	}
	if now.After(expiresAt.UTC()) {
		_, _ = tx.Exec(ctx, `update auth_refresh_sessions set revoked_at = $2 where token_hash = $1`, refreshHash, now)
		return nil, unauth("expired refresh token")
	}
	if now.After(absExpiresAt.UTC()) {
		_, _ = tx.Exec(ctx, `update auth_refresh_sessions set revoked_at = $2 where token_hash = $1`, refreshHash, now)
		return nil, unauth("expired refresh token")
	}

	newRefresh, err := newRefreshToken()
	if err != nil {
		return nil, internal(err)
	}
	newHash := security.HashRefreshToken(newRefresh)
	newRefreshExp := now.Add(s.refreshTTL)
	if newRefreshExp.After(absExpiresAt.UTC()) {
		newRefreshExp = absExpiresAt.UTC()
	}
	ua, ip := clientMetaFromContext(ctx)

	_, err = tx.Exec(ctx, `
		update auth_refresh_sessions
		set last_used_at = $2,
			revoked_at = $2,
			replaced_by_hash = $3,
			user_agent = coalesce($4, user_agent),
			ip = coalesce($5, ip)
		where token_hash = $1
	`, refreshHash, now, newHash, nullIfEmpty(ua), nullIfEmpty(ip))
	if err != nil {
		return nil, internal(err)
	}
	_, err = tx.Exec(ctx, `
		insert into auth_refresh_sessions (token_hash, session_id, user_id, expires_at, absolute_expires_at, user_agent, ip)
		values ($1,$2,$3,$4,$5,$6,$7)
	`, newHash, sessionID, userID, newRefreshExp, absExpiresAt.UTC(), nullIfEmpty(ua), nullIfEmpty(ip))
	if err != nil {
		return nil, internal(err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, internal(err)
	}

	roles, perms, err := s.getRolesAndPermissions(ctx, userID)
	if err != nil {
		return nil, internal(err)
	}

	accessNow := time.Now().UTC()
	accessExp := accessNow.Add(s.accessTTL)
	access, err := s.tokens.CreateToken(security.TokenPayload{
		UserID:      userID,
		Roles:       roles,
		Permissions: perms,
		IssuedAt:    accessNow,
		ExpiredAt:   accessExp,
	})
	if err != nil {
		return nil, internal(err)
	}

	return &authv1.RefreshTokenResponse{
		AccessToken:  access,
		RefreshToken: newRefresh,
		ExpiresIn:    int64(s.accessTTL.Seconds()),
	}, nil
}

func (s *Server) Logout(ctx context.Context, req *authv1.LogoutRequest) (*authv1.LogoutResponse, error) {
	refreshToken := strings.TrimSpace(req.RefreshToken)
	if refreshToken == "" {
		return nil, invalidArg("refresh_token required")
	}
	h := security.HashRefreshToken(refreshToken)
	_, err := s.db.Exec(ctx, `update auth_refresh_sessions set revoked_at = $2 where token_hash = $1 and revoked_at is null`, h, time.Now().UTC())
	if err != nil {
		return nil, internal(err)
	}
	return &authv1.LogoutResponse{Success: true}, nil
}

func (s *Server) LogoutAll(ctx context.Context, req *authv1.LogoutAllRequest) (*authv1.LogoutAllResponse, error) {
	accessToken := strings.TrimSpace(req.AccessToken)
	if accessToken == "" {
		if md, ok := metadata.FromIncomingContext(ctx); ok {
			vals := md.Get("authorization")
			if len(vals) == 0 {
				vals = md.Get("Authorization")
			}
			if len(vals) > 0 {
				v := strings.TrimSpace(vals[0])
				lv := strings.ToLower(v)
				if strings.HasPrefix(lv, "bearer ") {
					accessToken = strings.TrimSpace(v[len("bearer "):])
				}
			}
		}
	}
	if accessToken == "" {
		return nil, invalidArg("access_token required")
	}
	payload, err := s.tokens.VerifyToken(accessToken)
	if err != nil {
		if errors.Is(err, security.ErrExpiredToken) || errors.Is(err, security.ErrInvalidToken) {
			return nil, unauth("invalid access token")
		}
		return nil, internal(err)
	}
	userID := strings.TrimSpace(payload.UserID)
	if userID == "" {
		return nil, unauth("invalid access token")
	}

	res, err := s.db.Exec(ctx, `update auth_refresh_sessions set revoked_at = $2 where user_id = $1 and revoked_at is null`, userID, time.Now().UTC())
	if err != nil {
		return nil, internal(err)
	}

	return &authv1.LogoutAllResponse{Success: true, RevokedSessions: res.RowsAffected()}, nil
}

func (s *Server) requirePermissionFromAuthHeader(ctx context.Context, required string) error {
	accessToken := ""
	if md, ok := metadata.FromIncomingContext(ctx); ok {
		vals := md.Get("authorization")
		if len(vals) == 0 {
			vals = md.Get("Authorization")
		}
		if len(vals) > 0 {
			v := strings.TrimSpace(vals[0])
			lv := strings.ToLower(v)
			if strings.HasPrefix(lv, "bearer ") {
				accessToken = strings.TrimSpace(v[len("bearer "):])
			}
		}
	}
	if accessToken == "" {
		return unauth("access token required")
	}
	payload, err := s.tokens.VerifyToken(accessToken)
	if err != nil {
		if errors.Is(err, security.ErrExpiredToken) || errors.Is(err, security.ErrInvalidToken) {
			return unauth("invalid access token")
		}
		return internal(err)
	}
	for _, p := range payload.Permissions {
		if p == required {
			return nil
		}
	}
	return forbidden("forbidden")
}

func clientMetaFromContext(ctx context.Context) (userAgent string, ip string) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return "", ""
	}
	ua := md.Get("user-agent")
	if len(ua) > 0 {
		userAgent = strings.TrimSpace(ua[0])
	}
	xff := md.Get("x-forwarded-for")
	if len(xff) > 0 {
		parts := strings.Split(xff[0], ",")
		if len(parts) > 0 {
			ip = strings.TrimSpace(parts[0])
		}
	}
	return userAgent, ip
}

func nullIfEmpty(s string) any {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	return s
}

func newRefreshToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}
