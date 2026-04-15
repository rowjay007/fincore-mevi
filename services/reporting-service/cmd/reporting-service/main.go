package main

import (
	"context"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	reportingv1 "fincore/gen/go/reporting/v1"
	"fincore/pkg/security"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type reportingServer struct {
	reportingv1.UnimplementedReportingServiceServer
}

func (s *reportingServer) GetBaselIIIReport(ctx context.Context, req *reportingv1.GetBaselIIIReportRequest) (*reportingv1.GetBaselIIIReportResponse, error) {
	// In production, this would query ClickHouse for denormalized regulatory metrics.
	// Basel III requires Tier 1 capital ratios and LCR (Liquidity Coverage Ratio).
	return &reportingv1.GetBaselIIIReportResponse{
		CapitalRatio:           "12.5%",
		LiquidityCoverageRatio: "115.2%",
		GeneratedAt:            timestamppb.Now(),
	}, nil
}

func (s *reportingServer) GetAMLMonitoringReport(ctx context.Context, req *reportingv1.GetAMLMonitoringReportRequest) (*reportingv1.GetAMLMonitoringReportResponse, error) {
	// Query ClickHouse for suspicious patterns (e.g. structuring, rapid movement).
	alerts := []*reportingv1.AMLAlert{
		{
			UserId:        "user_123",
			TransactionId: "tx_456",
			RiskScore:     0.88,
			Reason:        "Rapid succession of high-value transfers (structuring)",
		},
	}
	return &reportingv1.GetAMLMonitoringReportResponse{
		Alerts:      alerts,
		TotalAlerts: 1,
	}, nil
}

func (s *reportingServer) GetDashboardStats(ctx context.Context, req *reportingv1.GetDashboardStatsRequest) (*reportingv1.GetDashboardStatsResponse, error) {
	// Aggregated metrics from ClickHouse
	return &reportingv1.GetDashboardStatsResponse{
		TotalTransactions: 1540230,
		TotalVolumeKobo:   "943029432043",
		AvgFraudScore:     0.04,
	}, nil
}

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// Initialize OTel tracing
	shutdown, err := security.InitTracer(ctx, "reporting-service")
	if err != nil {
		log.Printf("failed to init tracer: %v", err)
	} else {
		defer shutdown(ctx)
	}

	listenAddr := os.Getenv("LISTEN_ADDR")
	if listenAddr == "" {
		listenAddr = ":50063"
	}

	lis, err := net.Listen("tcp", listenAddr)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	s := grpc.NewServer()
	reportingv1.RegisterReportingServiceServer(s, &reportingServer{})
	reflection.Register(s)

	log.Printf("reporting-service (OLAP/Basel III) listening on %s", listenAddr)

	go func() {
		if err := s.Serve(lis); err != nil {
			log.Fatalf("failed to serve: %v", err)
		}
	}()

	<-ctx.Done()
	log.Println("shutting down reporting-service")
	s.GracefulStop()
}
