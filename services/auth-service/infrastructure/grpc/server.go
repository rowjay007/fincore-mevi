package grpc

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"time"

	authv1 "fincore/gen/go/auth/v1"
	"fincore/pkg/security"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type Server struct {
	authv1.UnimplementedAuthServiceServer
	db         DB
	tokens     security.TokenMaker
	accessTTL  time.Duration
	refreshTTL time.Duration
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

func (s *Server) Register(ctx context.Context, req *authv1.RegisterRequest) (*authv1.RegisterResponse, error) {
	email := strings.TrimSpace(strings.ToLower(req.Email))
	if email == "" {
		return nil, errors.New("email required")
	}
	if strings.TrimSpace(req.Password) == "" {
		return nil, errors.New("password required")
	}
	if strings.TrimSpace(req.FullName) == "" {
		return nil, errors.New("full_name required")
	}

	pwHash, err := security.HashPassword(req.Password)
	if err != nil {
		return nil, err
	}

	id := uuid.NewString()

	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	_, err = tx.Exec(ctx, `insert into auth_users (id, email, full_name, password_hash) values ($1,$2,$3,$4)`, id, email, req.FullName, pwHash)
	if err != nil {
		return nil, err
	}
	_, err = tx.Exec(ctx, `insert into auth_user_roles (user_id, role_id) values ($1,$2) on conflict do nothing`, id, "role_customer")
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return &authv1.RegisterResponse{UserId: id}, nil
}

func (s *Server) Login(ctx context.Context, req *authv1.LoginRequest) (*authv1.LoginResponse, error) {
	email := strings.TrimSpace(strings.ToLower(req.Email))
	if email == "" {
		return nil, errors.New("email required")
	}
	if strings.TrimSpace(req.Password) == "" {
		return nil, errors.New("password required")
	}

	var userID string
	var pwHash string
	err := s.db.QueryRow(ctx, `select id, password_hash from auth_users where email = $1`, email).Scan(&userID, &pwHash)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, errors.New("invalid credentials")
		}
		return nil, err
	}

	ok, err := security.VerifyPassword(req.Password, pwHash)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, errors.New("invalid credentials")
	}

	roles, perms, err := s.getRolesAndPermissions(ctx, userID)
	if err != nil {
		return nil, err
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
		return nil, err
	}

	refresh, err := newRefreshToken()
	if err != nil {
		return nil, err
	}
	refreshExp := accessNow.Add(s.refreshTTL)
	refreshHash := security.HashRefreshToken(refresh)
	_, err = s.db.Exec(ctx, `insert into auth_refresh_sessions (token_hash, user_id, expires_at) values ($1,$2,$3)`, refreshHash, userID, refreshExp)
	if err != nil {
		return nil, err
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
		return nil, errors.New("access_token required")
	}
	payload, err := s.tokens.VerifyToken(req.AccessToken)
	if err != nil {
		return nil, err
	}
	return &authv1.ValidateTokenResponse{
		UserId:      payload.UserID,
		Roles:       payload.Roles,
		Permissions: payload.Permissions,
	}, nil
}

func (s *Server) RefreshToken(ctx context.Context, req *authv1.RefreshTokenRequest) (*authv1.RefreshTokenResponse, error) {
	refreshToken := strings.TrimSpace(req.RefreshToken)
	if refreshToken == "" {
		return nil, errors.New("refresh_token required")
	}
	refreshHash := security.HashRefreshToken(refreshToken)

	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	var userID string
	var expiresAt time.Time
	var revokedAt *time.Time
	var replacedBy *string
	err = tx.QueryRow(ctx, `
		select user_id, expires_at, revoked_at, replaced_by_hash
		from auth_refresh_sessions
		where token_hash = $1
		`, refreshHash).Scan(&userID, &expiresAt, &revokedAt, &replacedBy)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, errors.New("invalid refresh token")
		}
		return nil, err
	}

	now := time.Now().UTC()
	if revokedAt != nil || replacedBy != nil {
		_, _ = tx.Exec(ctx, `update auth_refresh_sessions set revoked_at = coalesce(revoked_at, $2) where token_hash = $1`, refreshHash, now)
		_, _ = tx.Exec(ctx, `update auth_refresh_sessions set revoked_at = $2 where user_id = $1 and revoked_at is null`, userID, now)
		return nil, errors.New("refresh token reuse detected")
	}
	if now.After(expiresAt.UTC()) {
		_, _ = tx.Exec(ctx, `update auth_refresh_sessions set revoked_at = $2 where token_hash = $1`, refreshHash, now)
		return nil, errors.New("expired refresh token")
	}

	newRefresh, err := newRefreshToken()
	if err != nil {
		return nil, err
	}
	newHash := security.HashRefreshToken(newRefresh)
	newRefreshExp := now.Add(s.refreshTTL)

	_, err = tx.Exec(ctx, `update auth_refresh_sessions set last_used_at = $2, revoked_at = $2, replaced_by_hash = $3 where token_hash = $1`, refreshHash, now, newHash)
	if err != nil {
		return nil, err
	}
	_, err = tx.Exec(ctx, `insert into auth_refresh_sessions (token_hash, user_id, expires_at) values ($1,$2,$3)`, newHash, userID, newRefreshExp)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	roles, perms, err := s.getRolesAndPermissions(ctx, userID)
	if err != nil {
		return nil, err
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
		return nil, err
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
		return nil, errors.New("refresh_token required")
	}
	h := security.HashRefreshToken(refreshToken)
	_, err := s.db.Exec(ctx, `update auth_refresh_sessions set revoked_at = $2 where token_hash = $1 and revoked_at is null`, h, time.Now().UTC())
	if err != nil {
		return nil, err
	}
	return &authv1.LogoutResponse{Success: true}, nil
}

func (s *Server) LogoutAll(ctx context.Context, req *authv1.LogoutAllRequest) (*authv1.LogoutAllResponse, error) {
	accessToken := strings.TrimSpace(req.AccessToken)
	if accessToken == "" {
		return nil, errors.New("access_token required")
	}
	payload, err := s.tokens.VerifyToken(accessToken)
	if err != nil {
		return nil, err
	}
	userID := strings.TrimSpace(payload.UserID)
	if userID == "" {
		return nil, errors.New("invalid access token")
	}

	res, err := s.db.Exec(ctx, `update auth_refresh_sessions set revoked_at = $2 where user_id = $1 and revoked_at is null`, userID, time.Now().UTC())
	if err != nil {
		return nil, err
	}

	return &authv1.LogoutAllResponse{Success: true, RevokedSessions: res.RowsAffected()}, nil
}

func newRefreshToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}
