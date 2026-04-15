package main

import (
	"context"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	auditv1 "fincore/gen/go/audit/v1"
	"fincore/pkg/postgres"
	audit_messaging "fincore/services/audit-service/infrastructure/messaging"
	repo "fincore/services/audit-service/infrastructure/postgres"

	"github.com/nats-io/nats.go"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/reflection"
	"google.golang.org/grpc/status"
)

type auditServer struct {
	auditv1.UnimplementedAuditServiceServer
	repo *repo.AuditRepository
}

func (s *auditServer) ListAuditLogs(ctx context.Context, req *auditv1.ListAuditLogsRequest) (*auditv1.ListAuditLogsResponse, error) {
	entries, err := s.repo.List(ctx, req)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list audit logs: %v", err)
	}
	return &auditv1.ListAuditLogsResponse{Entries: entries}, nil
}

func (s *auditServer) GetAuditLog(ctx context.Context, req *auditv1.GetAuditLogRequest) (*auditv1.GetAuditLogResponse, error) {
	entry, err := s.repo.Get(ctx, req.Id)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "audit log not found: %v", err)
	}
	return &auditv1.GetAuditLogResponse{Entry: entry}, nil
}

func (s *auditServer) ValidateIntegrity(ctx context.Context, req *auditv1.ValidateIntegrityRequest) (*auditv1.ValidateIntegrityResponse, error) {
	isValid, count, failingID, err := s.repo.ValidateIntegrity(ctx, req.StartId, req.EndId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to validate integrity: %v", err)
	}
	return &auditv1.ValidateIntegrityResponse{
		IsValid:        isValid,
		ProcessedCount: count,
		FailingId:      failingID,
	}, nil
}

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = "postgres://postgres:postgres@localhost:5432/fincore_audit?sslmode=disable"
	}

	pool, err := postgres.NewPool(ctx, postgres.Config{DSN: dsn})
	if err != nil {
		log.Fatalf("failed to connect to postgres: %v", err)
	}
	defer pool.Close()

	repository := repo.NewAuditRepository(pool)
	reportingRepo := repo.NewReportingProjectionRepo(pool)

	natsURL := os.Getenv("NATS_URL")
	if natsURL == "" {
		natsURL = nats.DefaultURL
	}
	nc, err := nats.Connect(natsURL)
	if err != nil {
		log.Fatalf("failed to connect to nats: %v", err)
	}
	defer nc.Close()

	consumer := audit_messaging.NewAuditConsumer(nc, repository)
	if err := consumer.Start(ctx); err != nil {
		log.Fatalf("failed to start audit consumer: %v", err)
	}

	repConsumer := audit_messaging.NewReportingConsumer(nc, reportingRepo)
	if err := repConsumer.Start(ctx); err != nil {
		log.Fatalf("failed to start reporting consumer: %v", err)
	}

	listenAddr := os.Getenv("LISTEN_ADDR")
	if listenAddr == "" {
		listenAddr = ":50058"
	}

	lis, err := net.Listen("tcp", listenAddr)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	s := grpc.NewServer()
	auditv1.RegisterAuditServiceServer(s, &auditServer{repo: repository})
	reflection.Register(s)

	log.Printf("audit-service listening on %s", listenAddr)

	go func() {
		if err := s.Serve(lis); err != nil {
			log.Fatalf("failed to serve: %v", err)
		}
	}()

	<-ctx.Done()
	log.Println("shutting down audit-service")
	s.GracefulStop()
}
