package main

import (
	"context"
	"log"
	"math/big"
	"net"
	"os"
	"os/signal"
	"syscall"

	fxv1 "fincore/gen/go/fx/v1"
	"fincore/pkg/security"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/reflection"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type fxServer struct {
	fxv1.UnimplementedFXServiceServer
}

func (s *fxServer) GetRate(ctx context.Context, req *fxv1.GetRateRequest) (*fxv1.GetRateResponse, error) {
	// In production, this would fetch from ECB, XE, or internal cache.
	// For this skeleton, we use a fixed sample rate for demonstration.
	rate := "0.92345678" // Sample USD/EUR rate

	return &fxv1.GetRateResponse{
		Rate:      rate,
		FetchedAt: timestamppb.Now(),
	}, nil
}

func (s *fxServer) Convert(ctx context.Context, req *fxv1.ConvertRequest) (*fxv1.ConvertResponse, error) {
	amount, ok := new(big.Float).SetPrec(256).SetString(req.Amount)
	if !ok {
		return nil, status.Error(codes.InvalidArgument, "invalid amount format")
	}

	// Fetch actual rate (mocked for skeleton)
	rateStr := "0.92345678"
	rate, _ := new(big.Float).SetPrec(256).SetString(rateStr)

	converted := new(big.Float).Mul(amount, rate)

	return &fxv1.ConvertResponse{
		ConvertedAmount: converted.Text('f', 8),
		Rate:            rateStr,
		ConvertedAt:     timestamppb.Now(),
	}, nil
}

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// Initialize OTel tracing
	shutdown, err := security.InitTracer(ctx, "fx-service")
	if err != nil {
		log.Printf("failed to init tracer: %v", err)
	} else {
		defer shutdown(ctx)
	}

	listenAddr := os.Getenv("LISTEN_ADDR")
	if listenAddr == "" {
		listenAddr = ":50060"
	}

	lis, err := net.Listen("tcp", listenAddr)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	s := grpc.NewServer()
	fxv1.RegisterFXServiceServer(s, &fxServer{})
	reflection.Register(s)

	log.Printf("fx-service (FX rates) listening on %s", listenAddr)

	go func() {
		if err := s.Serve(lis); err != nil {
			log.Fatalf("failed to serve: %v", err)
		}
	}()

	<-ctx.Done()
	log.Println("shutting down fx-service")
	s.GracefulStop()
}
