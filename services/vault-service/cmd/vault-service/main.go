package main

import (
	"context"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	vaultv1 "fincore/gen/go/vault/v1"
	"fincore/pkg/security"
	"fincore/services/vault-service/domain"
	"fincore/services/vault-service/infrastructure/vault"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/reflection"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type vaultServer struct {
	vaultv1.UnimplementedVaultServiceServer
	vault domain.VaultPort
}

func (s *vaultServer) Tokenize(ctx context.Context, req *vaultv1.TokenizeRequest) (*vaultv1.TokenizeResponse, error) {
	token, err := s.vault.Tokenize(ctx, req.Category, req.Data)
	if err != nil {
		log.Printf("vault tokenize error: %v", err)
		return nil, status.Errorf(codes.Internal, "failed to tokenize: %v", err)
	}

	return &vaultv1.TokenizeResponse{
		Token:     token,
		CreatedAt: timestamppb.Now(),
	}, nil
}

func (s *vaultServer) Detokenize(ctx context.Context, req *vaultv1.DetokenizeRequest) (*vaultv1.DetokenizeResponse, error) {
	if req.Reason == "" {
		return nil, status.Error(codes.InvalidArgument, "detokenize reason is required for audit")
	}

	plaintext, err := s.vault.Detokenize(ctx, req.Token, req.Reason)
	if err != nil {
		log.Printf("vault detokenize error: %v", err)
		return nil, status.Errorf(codes.PermissionDenied, "failed to detokenize")
	}

	return &vaultv1.DetokenizeResponse{
		Data: plaintext,
	}, nil
}

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// Initialize OTel tracing
	shutdown, err := security.InitTracer(ctx, "vault-service")
	if err != nil {
		log.Printf("failed to init tracer: %v", err)
	} else {
		defer shutdown(ctx)
	}

	vaultAddr := os.Getenv("VAULT_ADDR")
	if vaultAddr == "" {
		vaultAddr = "http://localhost:8200"
	}
	vaultToken := os.Getenv("VAULT_TOKEN") // In prod, use AppRole or Kubernetes auth

	vClient, err := vault.NewClient(vaultAddr, vaultToken)
	if err != nil {
		log.Fatalf("failed to connect to vault: %v", err)
	}

	listenAddr := os.Getenv("LISTEN_ADDR")
	if listenAddr == "" {
		listenAddr = ":50059"
	}

	lis, err := net.Listen("tcp", listenAddr)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	s := grpc.NewServer()
	vaultv1.RegisterVaultServiceServer(s, &vaultServer{vault: vClient})
	reflection.Register(s)

	log.Printf("vault-service (PCI-DSS) listening on %s", listenAddr)

	go func() {
		if err := s.Serve(lis); err != nil {
			log.Fatalf("failed to serve: %v", err)
		}
	}()

	<-ctx.Done()
	log.Println("shutting down vault-service")
	s.GracefulStop()
}
