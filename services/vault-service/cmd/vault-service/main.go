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
	"fincore/services/vault-service/infrastructure/vault"

	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type vaultServer struct {
	vaultv1.UnimplementedVaultServiceServer
	vault *vault.Client
}

func (s *vaultServer) Tokenize(ctx context.Context, req *vaultv1.TokenizeRequest) (*vaultv1.TokenizeResponse, error) {
	// In production, we'd use Format-Preserving Encryption (FPE) via Vault Transit
	// For this skeleton, we use standard Transit encryption path
	// The path corresponds to the 'category' (e.g., card_pan)
	ciphertext, err := s.vault.Encrypt(ctx, req.Category, req.Data)
	if err != nil {
		log.Printf("vault encrypt error: %v", err)
		return nil, err
	}

	return &vaultv1.TokenizeResponse{
		Token:     ciphertext,
		CreatedAt: timestamppb.Now(),
	}, nil
}

func (s *vaultServer) Detokenize(ctx context.Context, req *vaultv1.DetokenizeRequest) (*vaultv1.DetokenizeResponse, error) {
	// Security: Log detokenization reasons for PCI-DSS/SOC2 audit
	log.Printf("AUDIT: Detokenize request for token %s, reason: %s", req.Token, req.Reason)

	// Determine category from token prefix if needed, or require it in request.
	// Vault ciphertext usually starts with 'vault:v1:...'
	plaintext, err := s.vault.Decrypt(ctx, "card_pan", req.Token)
	if err != nil {
		return nil, err
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
