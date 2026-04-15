package main

import (
	"context"
	"encoding/json"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	notificationv1 "fincore/gen/go/notification/v1"
	"fincore/pkg/security"
	"github.com/nats-io/nats.go"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

type notificationServer struct {
	notificationv1.UnimplementedNotificationServiceServer
	nc *nats.Conn
}

func (s *notificationServer) SendNotification(ctx context.Context, req *notificationv1.SendNotificationRequest) (*notificationv1.SendNotificationResponse, error) {
	// Sync path: High priority direct send
	log.Printf("NOTIFICATION: Sending %s to user %s via %s", req.TemplateId, req.UserId, req.Channel)
	
	return &notificationv1.SendNotificationResponse{
		NotificationId: "notif_sync_" + req.UserId,
		Status:         "sent",
	}, nil
}

func (s *notificationServer) GetNotificationStatus(ctx context.Context, req *notificationv1.GetNotificationStatusRequest) (*notificationv1.GetNotificationStatusResponse, error) {
	return &notificationv1.GetNotificationStatusResponse{
		NotificationId: req.NotificationId,
		Status:         "sent",
	}, nil
}

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// Initialize OTel tracing
	shutdown, err := security.InitTracer(ctx, "notification-service")
	if err != nil {
		log.Printf("failed to init tracer: %v", err)
	} else {
		defer shutdown(ctx)
	}

	natsURL := os.Getenv("NATS_URL")
	if natsURL == "" {
		natsURL = nats.DefaultURL
	}
	nc, err := nats.Connect(natsURL)
	if err != nil {
		log.Fatalf("failed to connect to nats: %v", err)
	}
	defer nc.Close()

	// Async path: Listen for domain events and send alerts (e.g. login_alert, payment_settled)
	nc.Subscribe("auth.user.login", func(m *nats.Msg) {
		var data map[string]interface{}
		json.Unmarshal(m.Data, &data)
		log.Printf("NOTIFICATION: Async login alert for user %v", data["user_id"])
	})

	listenAddr := os.Getenv("LISTEN_ADDR")
	if listenAddr == "" {
		listenAddr = ":50062"
	}

	lis, err := net.Listen("tcp", listenAddr)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	s := grpc.NewServer()
	notificationv1.RegisterNotificationServiceServer(s, &notificationServer{nc: nc})
	reflection.Register(s)

	log.Printf("notification-service listening on %s", listenAddr)

	go func() {
		if err := s.Serve(lis); err != nil {
			log.Fatalf("failed to serve: %v", err)
		}
	}()

	<-ctx.Done()
	log.Println("shutting down notification-service")
	s.GracefulStop()
}
