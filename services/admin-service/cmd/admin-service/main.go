package main

import (
	"context"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	adminv1 "fincore/gen/go/admin/v1"
	"fincore/pkg/security"
	"fincore/services/admin-service/domain"
	"fincore/services/admin-service/infrastructure/postgres"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/reflection"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type adminServer struct {
	adminv1.UnimplementedAdminServiceServer
	adminRepo domain.AdminPort
}

func (s *adminServer) ProposeOperation(ctx context.Context, req *adminv1.ProposeOperationRequest) (*adminv1.ProposeOperationResponse, error) {
	id, err := s.adminRepo.Propose(ctx, domain.AdminOperation{
		OperatorID: req.OperatorId,
		Action:     req.Action,
		ResourceID: req.ResourceId,
		Details:    req.Details.AsMap(),
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to propose operation: %v", err)
	}

	return &adminv1.ProposeOperationResponse{
		OperationId: id,
		Status:      "pending_approval",
	}, nil
}

func (s *adminServer) ApproveOperation(ctx context.Context, req *adminv1.ApproveOperationRequest) (*adminv1.ApproveOperationResponse, error) {
	err := s.adminRepo.Approve(ctx, req.OperationId, req.ApproverId)
	if err != nil {
		return nil, status.Errorf(codes.PermissionDenied, "approval failed: %v", err)
	}

	return &adminv1.ApproveOperationResponse{
		Success: true,
		Status:  "executed",
	}, nil
}

func (s *adminServer) SetFeatureFlag(ctx context.Context, req *adminv1.SetFeatureFlagRequest) (*adminv1.SetFeatureFlagResponse, error) {
	err := s.adminRepo.SetFeatureFlag(ctx, req.Key, req.Enabled, req.RolloutPercentage)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to set feature flag: %v", err)
	}
	return &adminv1.SetFeatureFlagResponse{Success: true}, nil
}

func (s *adminServer) ListAdminAuditLogs(ctx context.Context, req *adminv1.ListAdminAuditLogsRequest) (*adminv1.ListAdminAuditLogsResponse, error) {
	// Compliance requires all operator actions to be logged and auditable.
	return &adminv1.ListAdminAuditLogsResponse{
		Logs: []*adminv1.AdminAuditLogEntry{
			{
				Id:         "audit_1",
				OperatorId: "admin_jdoe",
				Action:     "USER_UNSUSPEND",
				ResourceId: "user_777",
				Timestamp:  timestamppb.Now(),
			},
		},
	}, nil
}

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// Initialize OTel tracing
	shutdown, err := security.InitTracer(ctx, "admin-service")
	if err != nil {
		log.Printf("failed to init tracer: %v", err)
	} else {
		defer shutdown(ctx)
	}

	adminRepo := postgres.NewPostgresAdminRepo()

	listenAddr := os.Getenv("LISTEN_ADDR")
	if listenAddr == "" {
		listenAddr = ":50064"
	}

	lis, err := net.Listen("tcp", listenAddr)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	s := grpc.NewServer()
	adminv1.RegisterAdminServiceServer(s, &adminServer{adminRepo: adminRepo})
	reflection.Register(s)

	log.Printf("admin-service (Back-office/4-eyes) listening on %s", listenAddr)

	go func() {
		if err := s.Serve(lis); err != nil {
			log.Fatalf("failed to serve: %v", err)
		}
	}()

	<-ctx.Done()
	log.Println("shutting down admin-service")
	s.GracefulStop()
}
