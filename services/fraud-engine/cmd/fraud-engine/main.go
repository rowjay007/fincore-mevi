package main

import (
	"context"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	fraudv1 "fincore/gen/go/fraud/v1"
	"fincore/pkg/security"

	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type fraudServer struct {
	fraudv1.UnimplementedFraudServiceServer
}

func (s *fraudServer) ScoreTransaction(ctx context.Context, req *fraudv1.ScoreTransactionRequest) (*fraudv1.ScoreTransactionResponse, error) {
	// In production, this would use ONNX runtime or a rules engine.
	// We'll simulate a deterministic but simplistic rule: amounts > 10k are riskier.
	// For this skeleton, we use a sample score.
	score := float32(0.05)
	decision := "approve"
	var reasons []string

	// Simple heuristic simulation
	if req.Amount != "" && len(req.Amount) > 4 {
		score = 0.85
		decision = "review"
		reasons = append(reasons, "large_amount_threshold_exceeded")
	}

	return &fraudv1.ScoreTransactionResponse{
		Score:    score,
		Decision: decision,
		Reasons:  reasons,
		ScoredAt: timestamppb.Now(),
	}, nil
}

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// Initialize OTel tracing
	shutdown, err := security.InitTracer(ctx, "fraud-engine")
	if err != nil {
		log.Printf("failed to init tracer: %v", err)
	} else {
		defer shutdown(ctx)
	}

	listenAddr := os.Getenv("LISTEN_ADDR")
	if listenAddr == "" {
		listenAddr = ":50061"
	}

	lis, err := net.Listen("tcp", listenAddr)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	s := grpc.NewServer()
	fraudv1.RegisterFraudServiceServer(s, &fraudServer{})
	reflection.Register(s)

	log.Printf("fraud-engine (ML Scoring) listening on %s", listenAddr)

	go func() {
		if err := s.Serve(lis); err != nil {
			log.Fatalf("failed to serve: %v", err)
		}
	}()

	<-ctx.Done()
	log.Println("shutting down fraud-engine")
	s.GracefulStop()
}
