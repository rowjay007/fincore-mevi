package grpc

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"strings"
	"time"

	authv1 "fincore/gen/go/auth/v1"
	"fincore/pkg/security"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Server struct {
	authv1.UnimplementedAuthServiceServer
	pool       *pgxpool.Pool
	tokens     security.TokenMaker
	accessTTL  time.Duration
	refreshTTL time.Duration
}

func NewServer(pool *pgxpool.Pool, tokens security.TokenMaker, accessTTL time.Duration, refreshTTL time.Duration) *Server {
	return &Server{pool: pool, tokens: tokens, accessTTL: accessTTL, refreshTTL: refreshTTL}
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
	_, err = s.pool.Exec(ctx, `insert into auth_users (id, email, full_name, password_hash) values ($1,$2,$3,$4)`, id, email, req.FullName, pwHash)
	if err != nil {
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
	err := s.pool.QueryRow(ctx, `select id, password_hash from auth_users where email = $1`, email).Scan(&userID, &pwHash)
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

	now := time.Now().UTC()
	accessExp := now.Add(s.accessTTL)
	access, err := s.tokens.CreateToken(security.TokenPayload{
		UserID:      userID,
		Roles:       []string{},
		Permissions: []string{},
		IssuedAt:    now,
		ExpiredAt:   accessExp,
	})
	if err != nil {
		return nil, err
	}

	refresh, err := newRefreshToken()
	if err != nil {
		return nil, err
	}
	refreshExp := now.Add(s.refreshTTL)
	_, err = s.pool.Exec(ctx, `insert into auth_refresh_tokens (token, user_id, expires_at) values ($1,$2,$3)`, refresh, userID, refreshExp)
	if err != nil {
		return nil, err
	}

	return &authv1.LoginResponse{
		AccessToken:  access,
		RefreshToken: refresh,
		ExpiresIn:    int64(s.accessTTL.Seconds()),
	}, nil
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

func newRefreshToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}
