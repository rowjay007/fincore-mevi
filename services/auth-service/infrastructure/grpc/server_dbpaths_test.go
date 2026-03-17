package grpc

import (
	"context"
	"errors"
	"strings"
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

type stubTokenMakerNonEmpty struct {
	payload *security.TokenPayload
	err     error
}

func (s stubTokenMakerNonEmpty) CreateToken(payload security.TokenPayload) (string, error) {
	if s.err != nil {
		return "", s.err
	}
	return "access-token", nil
}

func (s stubTokenMakerNonEmpty) VerifyToken(token string) (*security.TokenPayload, error) {
	if s.err != nil {
		return nil, s.err
	}
	return s.payload, nil
}

type stubRow struct {
	scan func(dest ...any) error
}

func (r stubRow) Scan(dest ...any) error {
	if r.scan == nil {
		return errors.New("no scan")
	}
	return r.scan(dest...)
}

type stubRows struct {
	cols [][]any
	i    int
	err  error
}

func (r *stubRows) Close()     {}
func (r *stubRows) Err() error { return r.err }
func (r *stubRows) CommandTag() pgconn.CommandTag {
	var tag pgconn.CommandTag
	return tag
}
func (r *stubRows) FieldDescriptions() []pgconn.FieldDescription { return nil }
func (r *stubRows) Next() bool {
	if r.i >= len(r.cols) {
		return false
	}
	r.i++
	return true
}
func (r *stubRows) Scan(dest ...any) error {
	row := r.cols[r.i-1]
	if len(dest) != len(row) {
		return errors.New("scan arity mismatch")
	}
	for i := range dest {
		switch d := dest[i].(type) {
		case *string:
			*d = row[i].(string)
		default:
			return errors.New("unsupported dest type")
		}
	}
	return nil
}
func (r *stubRows) Values() ([]any, error) { return nil, errors.New("not implemented") }
func (r *stubRows) RawValues() [][]byte    { return nil }
func (r *stubRows) Conn() *pgx.Conn        { return nil }

type stubTx struct {
	exec func(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
}

func (t *stubTx) Begin(ctx context.Context) (pgx.Tx, error) {
	return nil, errors.New("not implemented")
}
func (t *stubTx) Commit(ctx context.Context) error   { return nil }
func (t *stubTx) Rollback(ctx context.Context) error { return nil }
func (t *stubTx) CopyFrom(ctx context.Context, tableName pgx.Identifier, columnNames []string, rowSrc pgx.CopyFromSource) (int64, error) {
	return 0, errors.New("not implemented")
}
func (t *stubTx) SendBatch(ctx context.Context, b *pgx.Batch) pgx.BatchResults { return nil }
func (t *stubTx) LargeObjects() pgx.LargeObjects                               { return pgx.LargeObjects{} }
func (t *stubTx) Prepare(ctx context.Context, name, sql string) (*pgconn.StatementDescription, error) {
	return nil, errors.New("not implemented")
}
func (t *stubTx) Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error) {
	if t.exec != nil {
		return t.exec(ctx, sql, arguments...)
	}
	var tag pgconn.CommandTag
	return tag, nil
}
func (t *stubTx) Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
	return nil, errors.New("not implemented")
}
func (t *stubTx) QueryRow(ctx context.Context, sql string, args ...any) pgx.Row {
	return stubRow{scan: func(dest ...any) error { return errors.New("not implemented") }}
}
func (t *stubTx) Conn() *pgx.Conn { return nil }

type stubAuthDB struct {
	begin    func(ctx context.Context) (pgx.Tx, error)
	exec     func(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
	query    func(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	queryRow func(ctx context.Context, sql string, args ...any) pgx.Row
}

func (s stubAuthDB) Begin(ctx context.Context) (pgx.Tx, error) {
	if s.begin != nil {
		return s.begin(ctx)
	}
	return &stubTx{}, nil
}
func (s stubAuthDB) Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error) {
	if s.exec != nil {
		return s.exec(ctx, sql, arguments...)
	}
	var tag pgconn.CommandTag
	return tag, nil
}
func (s stubAuthDB) Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
	if s.query != nil {
		return s.query(ctx, sql, args...)
	}
	return &stubRows{}, nil
}
func (s stubAuthDB) QueryRow(ctx context.Context, sql string, args ...any) pgx.Row {
	if s.queryRow != nil {
		return s.queryRow(ctx, sql, args...)
	}
	return stubRow{scan: func(dest ...any) error { return errors.New("no row") }}
}

func TestRegister_Login_AndAdminRoleOps_WithStubDB(t *testing.T) {
	pwHash, err := security.HashPassword("password123")
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}

	tx := &stubTx{exec: func(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
		var tag pgconn.CommandTag
		return tag, nil
	}}

	db := stubAuthDB{
		begin: func(ctx context.Context) (pgx.Tx, error) { return tx, nil },
		queryRow: func(ctx context.Context, sql string, args ...any) pgx.Row {
			if strings.Contains(sql, "from auth_users") {
				return stubRow{scan: func(dest ...any) error {
					*dest[0].(*string) = "u1"
					*dest[1].(*string) = pwHash
					return nil
				}}
			}
			if strings.Contains(sql, "select id from auth_roles") {
				return stubRow{scan: func(dest ...any) error {
					*dest[0].(*string) = "role_admin"
					return nil
				}}
			}
			return stubRow{scan: func(dest ...any) error { return errors.New("unexpected query") }}
		},
		query: func(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
			if strings.Contains(sql, "select r.name") {
				return &stubRows{cols: [][]any{{"role_customer"}}}, nil
			}
			if strings.Contains(sql, "select distinct p.name") {
				return &stubRows{cols: [][]any{{"auth:admin"}}}, nil
			}
			return &stubRows{}, nil
		},
		exec: func(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
			var tag pgconn.CommandTag
			return tag, nil
		},
	}

	tokens := stubTokenMakerNonEmpty{payload: &security.TokenPayload{UserID: "admin", Permissions: []string{"auth:admin"}, IssuedAt: time.Now(), ExpiredAt: time.Now().Add(time.Minute)}}
	s := NewServer(db, tokens, time.Minute, time.Hour)

	// Register happy path.
	_, err = s.Register(context.Background(), &authv1.RegisterRequest{Email: "a@b.com", Password: "password123", FullName: "A"})
	if err != nil {
		t.Fatalf("register: %v", err)
	}

	// Login happy path.
	res, err := s.Login(context.Background(), &authv1.LoginRequest{Email: "a@b.com", Password: "password123"})
	if err != nil {
		t.Fatalf("login: %v", err)
	}
	if strings.TrimSpace(res.AccessToken) == "" {
		t.Fatalf("expected access token")
	}
	if strings.TrimSpace(res.RefreshToken) == "" {
		t.Fatalf("expected refresh token")
	}

	adminCtx := metadata.NewIncomingContext(context.Background(), metadata.Pairs("authorization", "Bearer t"))
	_, err = s.GrantRole(adminCtx, &authv1.GrantRoleRequest{UserId: "u1", RoleName: "admin"})
	if err != nil {
		t.Fatalf("grant role: %v", err)
	}
	_, err = s.RevokeRole(adminCtx, &authv1.RevokeRoleRequest{UserId: "u1", RoleName: "admin"})
	if err != nil {
		t.Fatalf("revoke role: %v", err)
	}
	_, err = s.ListUserRoles(adminCtx, &authv1.ListUserRolesRequest{UserId: "u1"})
	if err != nil {
		t.Fatalf("list roles: %v", err)
	}

	// Admin endpoints should be forbidden if token lacks permission.
	s2 := NewServer(db, stubTokenMakerNonEmpty{payload: &security.TokenPayload{UserID: "u1", Permissions: []string{"x"}}}, time.Minute, time.Hour)
	_, err = s2.GrantRole(adminCtx, &authv1.GrantRoleRequest{UserId: "u1", RoleName: "admin"})
	if status.Code(err) != codes.PermissionDenied {
		t.Fatalf("expected PermissionDenied, got %v", status.Code(err))
	}
}
