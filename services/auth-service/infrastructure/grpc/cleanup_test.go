package grpc

import (
	"context"
	"testing"
	"time"

	"github.com/pashagolub/pgxmock/v4"
)

func TestCleanupRefreshSessions_DeletesExpiredAndRevoked(t *testing.T) {
	db, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock: %v", err)
	}
	defer db.Close()

	db.ExpectExec("delete from auth_refresh_sessions where expires_at").WithArgs(pgxmock.AnyArg()).
		WillReturnResult(pgxmock.NewResult("DELETE", 3))
	db.ExpectExec("delete from auth_refresh_sessions where revoked_at").WithArgs(pgxmock.AnyArg()).
		WillReturnResult(pgxmock.NewResult("DELETE", 2))

	expired, revoked, err := CleanupRefreshSessions(context.Background(), db, 7*24*time.Hour)
	if err != nil {
		t.Fatalf("CleanupRefreshSessions: %v", err)
	}
	if expired != 3 {
		t.Fatalf("expected expiredDeleted=3, got %d", expired)
	}
	if revoked != 2 {
		t.Fatalf("expected revokedDeleted=2, got %d", revoked)
	}

	if err := db.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations: %v", err)
	}
}

func TestCleanupRefreshSessions_SkipsRevokedWhenRetentionZero(t *testing.T) {
	db, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock: %v", err)
	}
	defer db.Close()

	db.ExpectExec("delete from auth_refresh_sessions where expires_at").WithArgs(pgxmock.AnyArg()).
		WillReturnResult(pgxmock.NewResult("DELETE", 1))

	expired, revoked, err := CleanupRefreshSessions(context.Background(), db, 0)
	if err != nil {
		t.Fatalf("CleanupRefreshSessions: %v", err)
	}
	if expired != 1 {
		t.Fatalf("expected expiredDeleted=1, got %d", expired)
	}
	if revoked != 0 {
		t.Fatalf("expected revokedDeleted=0, got %d", revoked)
	}

	if err := db.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations: %v", err)
	}
}
