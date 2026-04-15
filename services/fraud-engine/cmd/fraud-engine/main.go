package main

import (
	"context"
	"log"
	"math/big"
	"net"
	"os"
	"os/signal"
	"syscall"

	fraudv1 "fincore/gen/go/fraud/v1"
	"fincore/pkg/security"
	"fincore/services/fraud-engine/domain"
	"fincore/services/fraud-engine/infrastructure/rules"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/reflection"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type fraudServer struct {
	fraudv1.UnimplementedFraudServiceServer
	evaluator domain.Evaluator
}

func (s *fraudServer) ScoreTransaction(ctx context.Context, req *fraudv1.ScoreTransactionRequest) (*fraudv1.ScoreTransactionResponse, error) {
	amount, ok := new(big.Float).SetPrec(256).SetString(req.Amount)
	if !ok {
		return nil, status.Error(codes.InvalidArgument, "invalid amount format")
	}

	isNewDevice := false
	if req.Metadata != nil {
		fields := req.Metadata.GetFields()
		if val, ok := fields["is_new_device"]; ok {
			isNewDevice = val.GetBoolValue()
		}
	}

	txn := &domain.Transaction{
		ID:          req.TransactionId,
		UserID:      req.UserId,
		Amount:      amount,
		Currency:    req.Currency,
		DeviceID:    req.DeviceId,
		IPAddress:   req.IpAddress,
		IsNewDevice: isNewDevice,
	}

	res, err := s.evaluator.Evaluate(ctx, txn)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "fraud evaluation failed: %v", err)
	}

	log.Printf("FRAUD_SCORE: txn=%s user=%s score=%.2f decision=%s", req.TransactionId, req.UserId, res.Score, res.Decision)

	return &fraudv1.ScoreTransactionResponse{
		Score:    res.Score,
		Decision: res.Decision,
		Reasons:  res.Reasons,
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

	evaluator := rules.NewHeuristicEvaluator()

	listenAddr := os.Getenv("LISTEN_ADDR")
	if listenAddr == "" {
		listenAddr = ":50061"
	}

	lis, err := net.Listen("tcp", listenAddr)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	s := grpc.NewServer()
	fraudv1.RegisterFraudServiceServer(s, &fraudServer{evaluator: evaluator})
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
