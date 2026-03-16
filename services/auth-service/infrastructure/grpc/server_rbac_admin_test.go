package grpc

import (
	"context"
	"testing"
	"time"

	authv1 "fincore/gen/go/auth/v1"
	"fincore/pkg/security"

	"github.com/pashagolub/pgxmock/v4"
	"google.golang.org/grpc/metadata"
)

type adminTokenMaker struct{}

func (adminTokenMaker) CreateToken(payload security.TokenPayload) (string, error) {
	return "access", nil
}

func (adminTokenMaker) VerifyToken(token string) (*security.TokenPayload, error) {
	now := time.Now().UTC()
	return &security.TokenPayload{UserID: "admin-1", Permissions: []string{"auth:admin"}, IssuedAt: now, ExpiredAt: now.Add(time.Minute)}, nil
}

func TestGrantRole_AdminAllowed(t *testing.T) {
	db, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock: %v", err)
	}
	defer db.Close()

	s := NewServer(db, adminTokenMaker{}, 15*time.Minute, 30*24*time.Hour)

	db.ExpectQuery("select id from auth_roles").WithArgs("admin").
		WillReturnRows(pgxmock.NewRows([]string{"id"}).AddRow("role_admin"))
	db.ExpectExec("insert into auth_user_roles").WithArgs("user-1", "role_admin").
		WillReturnResult(pgxmock.NewResult("INSERT", 1))

	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs("authorization", "Bearer access"))
	res, err := s.GrantRole(ctx, &authv1.GrantRoleRequest{UserId: "user-1", RoleName: "admin"})
	if err != nil {
		t.Fatalf("GrantRole: %v", err)
	}
	if !res.Success {
		t.Fatalf("expected success")
	}

	if err := db.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations: %v", err)
	}
}

func TestGrantRole_ForbiddenWithoutPermission(t *testing.T) {
	db, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock: %v", err)
	}
	defer db.Close()

	s := NewServer(db, fixedTokenMaker{}, 15*time.Minute, 30*24*time.Hour)

	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs("authorization", "Bearer access"))
	_, err = s.GrantRole(ctx, &authv1.GrantRoleRequest{UserId: "user-1", RoleName: "admin"})
	if err == nil {
		t.Fatalf("expected error")
	}
}
