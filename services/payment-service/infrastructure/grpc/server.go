package grpc

import (
	"context"
	"errors"
	"strings"

	commonv1 "fincore/gen/go/common/v1"
	paymentv1 "fincore/gen/go/payment/v1"
	"fincore/pkg/ids"
	"fincore/pkg/money"
	"fincore/services/payment-service/application/commands"
	"fincore/services/payment-service/application/ports"
	"fincore/services/payment-service/domain"
)

type InitiateHandler interface {
	Handle(ctx context.Context, cmd commands.InitiatePayment) (*commands.InitiatePaymentResult, error)
}

type AuthorizeHandler interface {
	Handle(ctx context.Context, cmd commands.AuthorizePayment) (*commands.AuthorizePaymentResult, error)
}

type SettleHandler interface {
	Handle(ctx context.Context, cmd commands.SettlePayment) (*commands.SettlePaymentResult, error)
}

type FailHandler interface {
	Handle(ctx context.Context, cmd commands.FailPayment) (*commands.FailPaymentResult, error)
}

type PaymentQuery interface {
	GetByID(ctx context.Context, paymentID string) (ports.PaymentProjection, bool, error)
}

type Server struct {
	paymentv1.UnimplementedPaymentServiceServer
	initiate  InitiateHandler
	authorize AuthorizeHandler
	settle    SettleHandler
	fail      FailHandler
	q         PaymentQuery
}

func NewServer(initiate InitiateHandler, authorize AuthorizeHandler, settle SettleHandler, fail FailHandler, q PaymentQuery) *Server {
	return &Server{initiate: initiate, authorize: authorize, settle: settle, fail: fail, q: q}
}

func (s *Server) InitiatePayment(ctx context.Context, req *paymentv1.InitiatePaymentRequest) (*paymentv1.InitiatePaymentResponse, error) {
	if req == nil {
		return nil, errors.New("request required")
	}
	if strings.TrimSpace(req.IdempotencyKey) == "" {
		return nil, errors.New("idempotency_key required")
	}
	if strings.TrimSpace(req.FromAccountId) == "" {
		return nil, errors.New("from_account_id required")
	}
	if strings.TrimSpace(req.ToAccountId) == "" {
		return nil, errors.New("to_account_id required")
	}
	if req.Amount == nil {
		return nil, errors.New("amount required")
	}
	amt, err := money.New(req.Amount.AmountKobo, money.Currency(req.Amount.Currency))
	if err != nil {
		return nil, err
	}

	res, err := s.initiate.Handle(ctx, commands.InitiatePayment{
		FromAccountID:  ids.ID(req.FromAccountId),
		ToAccountID:    ids.ID(req.ToAccountId),
		Amount:         amt,
		Narration:      req.Narration,
		IdempotencyKey: req.IdempotencyKey,
	})
	if err != nil {
		return nil, err
	}
	return &paymentv1.InitiatePaymentResponse{PaymentId: res.PaymentID.String(), Status: toProtoStatusString(string(res.Status))}, nil
}

func (s *Server) AuthorizePayment(ctx context.Context, req *paymentv1.AuthorizePaymentRequest) (*paymentv1.AuthorizePaymentResponse, error) {
	if req == nil {
		return nil, errors.New("request required")
	}
	if strings.TrimSpace(req.PaymentId) == "" {
		return nil, errors.New("payment_id required")
	}
	if strings.TrimSpace(req.IdempotencyKey) == "" {
		return nil, errors.New("idempotency_key required")
	}
	res, err := s.authorize.Handle(ctx, commands.AuthorizePayment{PaymentID: ids.ID(req.PaymentId), IdempotencyKey: req.IdempotencyKey})
	if err != nil {
		return nil, err
	}
	return &paymentv1.AuthorizePaymentResponse{PaymentId: res.PaymentID.String(), Status: toProtoStatusString(string(res.Status))}, nil
}

func (s *Server) SettlePayment(ctx context.Context, req *paymentv1.SettlePaymentRequest) (*paymentv1.SettlePaymentResponse, error) {
	if req == nil {
		return nil, errors.New("request required")
	}
	if strings.TrimSpace(req.PaymentId) == "" {
		return nil, errors.New("payment_id required")
	}
	if strings.TrimSpace(req.IdempotencyKey) == "" {
		return nil, errors.New("idempotency_key required")
	}
	res, err := s.settle.Handle(ctx, commands.SettlePayment{PaymentID: ids.ID(req.PaymentId), IdempotencyKey: req.IdempotencyKey})
	if err != nil {
		return nil, err
	}
	return &paymentv1.SettlePaymentResponse{PaymentId: res.PaymentID.String(), Status: toProtoStatusString(string(res.Status))}, nil
}

func (s *Server) FailPayment(ctx context.Context, req *paymentv1.FailPaymentRequest) (*paymentv1.FailPaymentResponse, error) {
	if req == nil {
		return nil, errors.New("request required")
	}
	if strings.TrimSpace(req.PaymentId) == "" {
		return nil, errors.New("payment_id required")
	}
	if strings.TrimSpace(req.IdempotencyKey) == "" {
		return nil, errors.New("idempotency_key required")
	}
	res, err := s.fail.Handle(ctx, commands.FailPayment{PaymentID: ids.ID(req.PaymentId), IdempotencyKey: req.IdempotencyKey, Reason: req.Reason})
	if err != nil {
		return nil, err
	}
	return &paymentv1.FailPaymentResponse{PaymentId: res.PaymentID.String(), Status: toProtoStatusString(string(res.Status))}, nil
}

func (s *Server) GetPayment(ctx context.Context, req *paymentv1.GetPaymentRequest) (*paymentv1.GetPaymentResponse, error) {
	if req == nil {
		return nil, errors.New("request required")
	}
	pid := strings.TrimSpace(req.PaymentId)
	if pid == "" {
		return nil, errors.New("payment_id required")
	}
	p, ok, err := s.q.GetByID(ctx, pid)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, errors.New("payment not found")
	}
	return &paymentv1.GetPaymentResponse{Payment: &paymentv1.Payment{
		PaymentId:     p.PaymentID,
		FromAccountId: p.FromAccountID,
		ToAccountId:   p.ToAccountID,
		Amount:        &commonv1.Money{Currency: p.Currency, AmountKobo: p.AmountKobo},
		Narration:     p.Narration,
		Status: func() paymentv1.PaymentStatus {
			switch p.Status {
			case string(domain.StatusInitiated):
				return paymentv1.PaymentStatus_PAYMENT_STATUS_INITIATED
			case string(domain.StatusAuthorized):
				return paymentv1.PaymentStatus_PAYMENT_STATUS_AUTHORIZED
			case string(domain.StatusSettled):
				return paymentv1.PaymentStatus_PAYMENT_STATUS_SETTLED
			case string(domain.StatusFailed):
				return paymentv1.PaymentStatus_PAYMENT_STATUS_FAILED
			default:
				return paymentv1.PaymentStatus_PAYMENT_STATUS_UNSPECIFIED
			}
		}(),
		Version: p.Version,
	}}, nil
}

func toProtoStatusString(s string) paymentv1.PaymentStatus {
	s = strings.TrimSpace(strings.ToLower(s))
	switch s {
	case "initiated":
		return paymentv1.PaymentStatus_PAYMENT_STATUS_INITIATED
	case "authorized":
		return paymentv1.PaymentStatus_PAYMENT_STATUS_AUTHORIZED
	case "settled":
		return paymentv1.PaymentStatus_PAYMENT_STATUS_SETTLED
	case "failed":
		return paymentv1.PaymentStatus_PAYMENT_STATUS_FAILED
	default:
		return paymentv1.PaymentStatus_PAYMENT_STATUS_UNSPECIFIED
	}
}
