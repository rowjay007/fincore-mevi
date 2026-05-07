package grpc

import (
	"context"
	"errors"
	"strings"

	commonv1 "fincore/gen/go/common/v1"
	ledgerv1 "fincore/gen/go/ledger/v1"
	"fincore/pkg/ids"
	"fincore/pkg/money"
	"fincore/services/ledger-service/application/commands"
	"fincore/services/ledger-service/domain"
)

type Server struct {
	ledgerv1.UnimplementedLedgerServiceServer
	post *commands.PostEntryHandler
	bal  BalanceQuery
}

type BalanceQuery interface {
	GetBalanceKobo(ctx context.Context, accountID string, version int) (int64, error)
}

func NewServer(post *commands.PostEntryHandler, bal BalanceQuery) *Server {
	return &Server{post: post, bal: bal}
}

func (s *Server) PostEntry(ctx context.Context, req *ledgerv1.PostEntryRequest) (*ledgerv1.PostEntryResponse, error) {
	if req == nil {
		return nil, errors.New("request required")
	}
	if req.Account == nil || strings.TrimSpace(req.Account.AccountId) == "" {
		return nil, errors.New("account_id required")
	}
	if req.Amount == nil {
		return nil, errors.New("amount required")
	}
	amt, err := money.New(req.Amount.AmountKobo, money.Currency(req.Amount.Currency))
	if err != nil {
		return nil, err
	}

	var typ domain.EntryType
	switch req.EntryType {
	case ledgerv1.EntryType_ENTRY_TYPE_DEPOSIT:
		typ = domain.EntryTypeDeposit
	case ledgerv1.EntryType_ENTRY_TYPE_WITHDRAWAL:
		typ = domain.EntryTypeWithdrawal
	default:
		return nil, errors.New("unsupported entry type")
	}

	res, err := s.post.Handle(ctx, commands.PostEntry{
		IdempotencyKey: req.IdempotencyKey,
		AccountID:      ids.ID(req.Account.AccountId),
		Type:           typ,
		Amount:         amt,
		Narration:      req.Narration,
	})
	if err != nil {
		return nil, err
	}
	return &ledgerv1.PostEntryResponse{EntryId: res.EntryID.String()}, nil
}

func (s *Server) GetBalance(ctx context.Context, req *ledgerv1.GetBalanceRequest) (*ledgerv1.GetBalanceResponse, error) {
	if req == nil {
		return nil, errors.New("request required")
	}
	if req.Account == nil || strings.TrimSpace(req.Account.AccountId) == "" {
		return nil, errors.New("account_id required")
	}

	bal, err := s.bal.GetBalanceKobo(ctx, req.Account.AccountId, 1)
	if err != nil {
		return nil, err
	}
	return &ledgerv1.GetBalanceResponse{AvailableBalance: &commonv1.Money{Currency: "NGN", AmountKobo: bal}}, nil
}
