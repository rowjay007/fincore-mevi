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
	"fincore/services/reporting-service/domain"
	"fincore/services/reporting-service/infrastructure/clickhouse"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/reflection"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type reportingServer struct {
	reportingv1.UnimplementedReportingServiceServer
	reporter domain.ReportingPort
}

func (s *reportingServer) GetBaselIIIReport(ctx context.Context, req *reportingv1.GetBaselIIIReportRequest) (*reportingv1.GetBaselIIIReportResponse, error) {
	start := req.StartDate.AsTime()
	end := req.EndDate.AsTime()

	res, err := s.reporter.GetBaselIIIReport(ctx, start, end)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to generate Basel III report: %v", err)
	}

	return &reportingv1.GetBaselIIIReportResponse{
		CapitalRatio:           res.CapitalRatio,
		LiquidityCoverageRatio: res.LiquidityCoverageRatio,
		GeneratedAt:            timestamppb.New(res.GeneratedAt),
	}, nil
}

func (s *reportingServer) GetAMLMonitoringReport(ctx context.Context, req *reportingv1.GetAMLMonitoringReportRequest) (*reportingv1.GetAMLMonitoringReportResponse, error) {
	start := req.StartDate.AsTime()
	end := req.EndDate.AsTime()

	res, err := s.reporter.GetAMLMonitoringReport(ctx, start, end, req.RiskThreshold)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to generate AML report: %v", err)
	}

	var alerts []*reportingv1.AMLAlert
	for _, a := range res {
		alerts = append(alerts, &reportingv1.AMLAlert{
			UserId:        a.UserID,
			TransactionId: a.TransactionID,
			RiskScore:     a.RiskScore,
			Reason:        a.Reason,
		})
	}

	return &reportingv1.GetAMLMonitoringReportResponse{
		Alerts:      alerts,
		TotalAlerts: int32(len(alerts)),
	}, nil
}

func (s *reportingServer) GetDashboardStats(ctx context.Context, req *reportingv1.GetDashboardStatsRequest) (*reportingv1.GetDashboardStatsResponse, error) {
	res, err := s.reporter.GetDashboardStats(ctx, req.Period)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get dashboard stats: %v", err)
	}

	return &reportingv1.GetDashboardStatsResponse{
		TotalTransactions: res.TotalTransactions,
		TotalVolumeKobo:   res.TotalVolumeKobo,
		AvgFraudScore:     res.AvgFraudScore,
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

	reporter := clickhouse.NewClickHouseReporter()

	listenAddr := os.Getenv("LISTEN_ADDR")
	if listenAddr == "" {
		listenAddr = ":50063"
	}

	lis, err := net.Listen("tcp", listenAddr)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	s := grpc.NewServer()
	reportingv1.RegisterReportingServiceServer(s, &reportingServer{reporter: reporter})
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
