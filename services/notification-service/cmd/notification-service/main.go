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
	"fincore/services/notification-service/domain"
	"fincore/services/notification-service/infrastructure/messaging"

	"github.com/nats-io/nats.go"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/reflection"
	grpcstatus "google.golang.org/grpc/status"
)

type notificationServer struct {
	notificationv1.UnimplementedNotificationServiceServer
	nc       *nats.Conn
	notifier domain.NotificationPort
}

func (s *notificationServer) SendNotification(ctx context.Context, req *notificationv1.SendNotificationRequest) (*notificationv1.SendNotificationResponse, error) {
	// Sync path: High priority direct send
	id, err := s.notifier.Send(ctx, domain.Notification{
		UserID:     req.UserId,
		TemplateID: req.TemplateId,
		Channel:    req.Channel,
		Data:       req.Data.AsMap(),
	})
	if err != nil {
		return nil, grpcstatus.Errorf(codes.Internal, "failed to send notification: %v", err)
	}

	return &notificationv1.SendNotificationResponse{
		NotificationId: id,
		Status:         "sent",
	}, nil
}

func (s *notificationServer) GetNotificationStatus(ctx context.Context, req *notificationv1.GetNotificationStatusRequest) (*notificationv1.GetNotificationStatusResponse, error) {
	status, err := s.notifier.GetStatus(ctx, req.NotificationId)
	if err != nil {
		return nil, grpcstatus.Errorf(codes.NotFound, "notification not found: %v", err)
	}

	return &notificationv1.GetNotificationStatusResponse{
		NotificationId: status.ID,
		Status:         status.Status,
		ErrorMessage:   status.Error,
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

	notifier := messaging.NewDummyNotifier()

	listenAddr := os.Getenv("LISTEN_ADDR")
	if listenAddr == "" {
		listenAddr = ":50062"
	}

	lis, err := net.Listen("tcp", listenAddr)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	s := grpc.NewServer()
	notificationv1.RegisterNotificationServiceServer(s, &notificationServer{nc: nc, notifier: notifier})
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
