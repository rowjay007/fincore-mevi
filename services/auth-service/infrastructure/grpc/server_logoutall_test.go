package grpc

import (
	"context"
	"testing"
	"time"

	authv1 "fincore/gen/go/auth/v1"

	"github.com/pashagolub/pgxmock/v4"
	"google.golang.org/grpc/metadata"
)

func TestLogoutAll_UsesAuthorizationHeaderWhenBodyEmpty(t *testing.T) {
	db, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock: %v", err)
	}
	defer db.Close()

	s := NewServer(db, fixedTokenMaker{}, 15*time.Minute, 30*24*time.Hour)

	db.ExpectExec("update auth_refresh_sessions set revoked_at").WithArgs("user-1", pgxmock.AnyArg()).WillReturnResult(pgxmock.NewResult("UPDATE", 2))

	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs("authorization", "Bearer access"))
	res, err := s.LogoutAll(ctx, &authv1.LogoutAllRequest{})
	if err != nil {
		t.Fatalf("LogoutAll: %v", err)
	}
	if !res.Success {
		t.Fatalf("expected success")
	}
	if res.RevokedSessions != 2 {
		t.Fatalf("expected revoked_sessions=2, got %d", res.RevokedSessions)
	}

	if err := db.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations: %v", err)
	}
}

func TestLogoutAll_BackwardCompat_UsesBodyAccessToken(t *testing.T) {
	db, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock: %v", err)
	}
	defer db.Close()

	s := NewServer(db, fixedTokenMaker{}, 15*time.Minute, 30*24*time.Hour)

	db.ExpectExec("update auth_refresh_sessions set revoked_at").WithArgs("user-1", pgxmock.AnyArg()).WillReturnResult(pgxmock.NewResult("UPDATE", 1))

	res, err := s.LogoutAll(context.Background(), &authv1.LogoutAllRequest{AccessToken: "access"})
	if err != nil {
		t.Fatalf("LogoutAll: %v", err)
	}
	if !res.Success {
		t.Fatalf("expected success")
	}
	if res.RevokedSessions != 1 {
		t.Fatalf("expected revoked_sessions=1, got %d", res.RevokedSessions)
	}

	if err := db.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations: %v", err)
	}
}
