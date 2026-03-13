package grpc

import (
	"context"
	"testing"
	"time"

	authv1 "fincore/gen/go/auth/v1"
	"fincore/pkg/security"

	"github.com/pashagolub/pgxmock/v4"
)

type fixedTokenMaker struct{}

func (fixedTokenMaker) CreateToken(payload security.TokenPayload) (string, error) {
	return "access", nil
}
func (fixedTokenMaker) VerifyToken(token string) (*security.TokenPayload, error) {
	now := time.Now().UTC()
	return &security.TokenPayload{UserID: "user-1", Roles: []string{"customer"}, Permissions: []string{"account:read"}, IssuedAt: now, ExpiredAt: now.Add(time.Minute)}, nil
}

func TestRefreshToken_RotatesSession(t *testing.T) {
	db, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock: %v", err)
	}
	defer db.Close()

	s := NewServer(db, fixedTokenMaker{}, 15*time.Minute, 30*24*time.Hour)

	refresh := "refresh-token"
	h := security.HashRefreshToken(refresh)

	exp := time.Now().UTC().Add(1 * time.Hour)

	db.ExpectBegin()
	db.ExpectQuery("select user_id, expires_at, revoked_at, replaced_by_hash").WithArgs(h).
		WillReturnRows(pgxmock.NewRows([]string{"user_id", "expires_at", "revoked_at", "replaced_by_hash"}).AddRow("user-1", exp, nil, nil))
	db.ExpectExec("update auth_refresh_sessions set last_used_at").WithArgs(h, pgxmock.AnyArg(), pgxmock.AnyArg()).WillReturnResult(pgxmock.NewResult("UPDATE", 1))
	db.ExpectExec("insert into auth_refresh_sessions").WithArgs(pgxmock.AnyArg(), "user-1", pgxmock.AnyArg()).WillReturnResult(pgxmock.NewResult("INSERT", 1))
	db.ExpectCommit()

	db.ExpectQuery("select r.name").WithArgs("user-1").
		WillReturnRows(pgxmock.NewRows([]string{"name"}).AddRow("customer"))
	db.ExpectQuery("select distinct p.name").WithArgs("user-1").
		WillReturnRows(pgxmock.NewRows([]string{"name"}).AddRow("account:read"))

	_, err = s.RefreshToken(context.Background(), &authv1.RefreshTokenRequest{RefreshToken: refresh})
	if err != nil {
		t.Fatalf("RefreshToken: %v", err)
	}

	if err := db.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations: %v", err)
	}
}

func TestRefreshToken_ReuseDetectionRevokesAll(t *testing.T) {
	db, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock: %v", err)
	}
	defer db.Close()

	s := NewServer(db, fixedTokenMaker{}, 15*time.Minute, 30*24*time.Hour)

	refresh := "refresh-token"
	h := security.HashRefreshToken(refresh)

	exp := time.Now().UTC().Add(1 * time.Hour)
	revoked := time.Now().UTC().Add(-1 * time.Minute)

	db.ExpectBegin()
	db.ExpectQuery("select user_id, expires_at, revoked_at, replaced_by_hash").WithArgs(h).
		WillReturnRows(pgxmock.NewRows([]string{"user_id", "expires_at", "revoked_at", "replaced_by_hash"}).AddRow("user-1", exp, &revoked, nil))
	db.ExpectExec("update auth_refresh_sessions set revoked_at = coalesce").WithArgs(h, pgxmock.AnyArg()).WillReturnResult(pgxmock.NewResult("UPDATE", 1))
	db.ExpectExec("update auth_refresh_sessions set revoked_at =").WithArgs("user-1", pgxmock.AnyArg()).WillReturnResult(pgxmock.NewResult("UPDATE", 2))
	db.ExpectRollback()

	_, err = s.RefreshToken(context.Background(), &authv1.RefreshTokenRequest{RefreshToken: refresh})
	if err == nil {
		t.Fatalf("expected error")
	}

	if err := db.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations: %v", err)
	}
}
