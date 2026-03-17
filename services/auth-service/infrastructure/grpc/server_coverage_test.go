package grpc

import (
	"context"
	"errors"
	"testing"
	"time"

	authv1 "fincore/gen/go/auth/v1"
	"fincore/pkg/security"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type stubDB struct {
	exec func(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
}

func (s stubDB) Begin(ctx context.Context) (pgx.Tx, error) { return nil, nil }
func (s stubDB) Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error) {
	if s.exec != nil {
		return s.exec(ctx, sql, arguments...)
	}
	var tag pgconn.CommandTag
	return tag, nil
}
func (s stubDB) Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
	return nil, nil
}
func (s stubDB) QueryRow(ctx context.Context, sql string, args ...any) pgx.Row { return nil }

func TestErrorHelpers(t *testing.T) {
	if status.Code(invalidArg("x")) != codes.InvalidArgument {
		t.Fatalf("expected InvalidArgument")
	}
	if status.Code(unauth("x")) != codes.Unauthenticated {
		t.Fatalf("expected Unauthenticated")
	}
	if status.Code(forbidden("x")) != codes.PermissionDenied {
		t.Fatalf("expected PermissionDenied")
	}
	if status.Code(notFound("x")) != codes.NotFound {
		t.Fatalf("expected NotFound")
	}
	if status.Code(rateLimited("x")) != codes.ResourceExhausted {
		t.Fatalf("expected ResourceExhausted")
	}
	if status.Code(internal(nil)) != codes.Internal {
		t.Fatalf("expected Internal")
	}
	if status.Code(internal(errors.New("boom"))) != codes.Internal {
		t.Fatalf("expected Internal")
	}
}

func TestNewServerWithLoginLimiter(t *testing.T) {
	s := NewServerWithLoginLimiter(stubDB{}, stubTokenMaker{payload: &security.TokenPayload{}}, time.Minute, time.Hour, 3, time.Minute, time.Minute)
	if s.limiter == nil {
		t.Fatalf("expected limiter")
	}
}

func TestValidateToken(t *testing.T) {
	s := &Server{tokens: stubTokenMaker{payload: &security.TokenPayload{UserID: "u1", Roles: []string{"r"}, Permissions: []string{"p"}}}}
	_, err := s.ValidateToken(context.Background(), &authv1.ValidateTokenRequest{AccessToken: " "})
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("expected InvalidArgument")
	}

	s = &Server{tokens: stubTokenMaker{err: security.ErrInvalidToken}}
	_, err = s.ValidateToken(context.Background(), &authv1.ValidateTokenRequest{AccessToken: "t"})
	if status.Code(err) != codes.Unauthenticated {
		t.Fatalf("expected Unauthenticated")
	}

	s = &Server{tokens: stubTokenMaker{err: errors.New("db down")}}
	_, err = s.ValidateToken(context.Background(), &authv1.ValidateTokenRequest{AccessToken: "t"})
	if status.Code(err) != codes.Internal {
		t.Fatalf("expected Internal")
	}

	s = &Server{tokens: stubTokenMaker{payload: &security.TokenPayload{UserID: "u1", Roles: []string{"r"}, Permissions: []string{"p"}}}}
	res, err := s.ValidateToken(context.Background(), &authv1.ValidateTokenRequest{AccessToken: "t"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.UserId != "u1" {
		t.Fatalf("expected user u1")
	}
}

func TestLogoutAndLogoutAll_EarlyValidationAndHappyPaths(t *testing.T) {
	s := &Server{db: stubDB{}, tokens: stubTokenMaker{payload: &security.TokenPayload{UserID: "u1"}}}
	_, err := s.Logout(context.Background(), &authv1.LogoutRequest{RefreshToken: " "})
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("expected InvalidArgument")
	}

	s = &Server{db: stubDB{exec: func(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
		return pgconn.CommandTag{}, errors.New("fail")
	}}, tokens: stubTokenMaker{payload: &security.TokenPayload{UserID: "u1"}}}
	_, err = s.Logout(context.Background(), &authv1.LogoutRequest{RefreshToken: "rt"})
	if status.Code(err) != codes.Internal {
		t.Fatalf("expected Internal")
	}

	called := false
	s = &Server{db: stubDB{exec: func(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
		called = true
		var tag pgconn.CommandTag
		return tag, nil
	}}, tokens: stubTokenMaker{payload: &security.TokenPayload{UserID: "u1"}}}
	_, err = s.Logout(context.Background(), &authv1.LogoutRequest{RefreshToken: "rt"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !called {
		t.Fatalf("expected exec called")
	}

	called = false
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs("authorization", "Bearer at"))
	s = &Server{db: stubDB{exec: func(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
		called = true
		var tag pgconn.CommandTag
		return tag, nil
	}}, tokens: stubTokenMaker{payload: &security.TokenPayload{UserID: "u1"}}}
	res, err := s.LogoutAll(ctx, &authv1.LogoutAllRequest{AccessToken: ""})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !called {
		t.Fatalf("expected exec called")
	}
	_ = res

	_, err = s.LogoutAll(context.Background(), &authv1.LogoutAllRequest{AccessToken: ""})
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("expected InvalidArgument")
	}

	s = &Server{db: stubDB{exec: func(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
		return pgconn.CommandTag{}, errors.New("fail")
	}}, tokens: stubTokenMaker{payload: &security.TokenPayload{UserID: "u1"}}}
	_, err = s.LogoutAll(ctx, &authv1.LogoutAllRequest{AccessToken: "at"})
	if status.Code(err) != codes.Internal {
		t.Fatalf("expected Internal")
	}

	s = &Server{db: stubDB{}, tokens: stubTokenMaker{payload: &security.TokenPayload{UserID: ""}}}
	_, err = s.LogoutAll(ctx, &authv1.LogoutAllRequest{AccessToken: "at"})
	if status.Code(err) != codes.Unauthenticated {
		t.Fatalf("expected Unauthenticated")
	}
}

func TestHandlers_EarlyValidationBranches(t *testing.T) {
	s := &Server{tokens: stubTokenMaker{payload: &security.TokenPayload{UserID: "u1", Permissions: []string{"x"}}}}

	_, err := s.Register(context.Background(), &authv1.RegisterRequest{Email: "bad", Password: "password123", FullName: "n"})
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("expected InvalidArgument")
	}
	_, err = s.Register(context.Background(), &authv1.RegisterRequest{Email: "a@b.com", Password: "short", FullName: "n"})
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("expected InvalidArgument")
	}
	_, err = s.Register(context.Background(), &authv1.RegisterRequest{Email: "a@b.com", Password: "password123", FullName: " "})
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("expected InvalidArgument")
	}

	_, err = s.Login(context.Background(), &authv1.LoginRequest{Email: "bad", Password: "x"})
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("expected InvalidArgument")
	}
	_, err = s.Login(context.Background(), &authv1.LoginRequest{Email: "a@b.com", Password: " "})
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("expected InvalidArgument")
	}

	// Permission-gated endpoints should fail before DB access when missing auth header.
	_, err = s.GrantRole(context.Background(), &authv1.GrantRoleRequest{UserId: "u", RoleName: "r"})
	if status.Code(err) != codes.Unauthenticated {
		t.Fatalf("expected Unauthenticated")
	}
	_, err = s.RevokeRole(context.Background(), &authv1.RevokeRoleRequest{UserId: "u", RoleName: "r"})
	if status.Code(err) != codes.Unauthenticated {
		t.Fatalf("expected Unauthenticated")
	}
	_, err = s.ListUserRoles(context.Background(), &authv1.ListUserRolesRequest{UserId: "u"})
	if status.Code(err) != codes.Unauthenticated {
		t.Fatalf("expected Unauthenticated")
	}

	_, err = s.RefreshToken(context.Background(), &authv1.RefreshTokenRequest{RefreshToken: " "})
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("expected InvalidArgument")
	}
}
