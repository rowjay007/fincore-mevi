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
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type adminServer struct {
	adminv1.UnimplementedAdminServiceServer
}

func (s *adminServer) ProposeOperation(ctx context.Context, req *adminv1.ProposeOperationRequest) (*adminv1.ProposeOperationResponse, error) {
	// Sensitive back-office actions like unfreezing large accounts
	// require a 2nd pair of eyes (the 4-eyes principle).
	log.Printf("ADMIN: Operator %s proposed %s on %s", req.OperatorId, req.Action, req.ResourceId)
	
	return &adminv1.ProposeOperationResponse{
		OperationId: "op_94302",
		Status:      "pending_approval",
	}, nil
}

func (s *adminServer) ApproveOperation(ctx context.Context, req *adminv1.ApproveOperationRequest) (*adminv1.ApproveOperationResponse, error) {
	// 2nd pair of eyes approval. In real prod, operator_id != approver_id.
	log.Printf("ADMIN: Approver %s approved operation %s", req.ApproverId, req.OperationId)

	return &adminv1.ApproveOperationResponse{
		Success: true,
		Status:  "executed",
	}, nil
}

func (s *adminServer) SetFeatureFlag(ctx context.Context, req *adminv1.SetFeatureFlagRequest) (*adminv1.SetFeatureFlagResponse, error) {
	log.Printf("ADMIN: Set feature flag %s to %v (rollout: %v)", req.Key, req.Enabled, req.RolloutPercentage)
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

	listenAddr := os.Getenv("LISTEN_ADDR")
	if listenAddr == "" {
		listenAddr = ":50064"
	}

	lis, err := net.Listen("tcp", listenAddr)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	s := grpc.NewServer()
	adminv1.RegisterAdminServiceServer(s, &adminServer{})
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
