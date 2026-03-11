package grpc

import (
	"context"
	"errors"

	accountv1 "fincore/gen/go/account/v1"
	commonv1 "fincore/gen/go/common/v1"
	"fincore/pkg/ids"
	"fincore/pkg/money"
	"fincore/services/account-service/application/commands"
	"fincore/services/account-service/application/ports"
	"fincore/services/account-service/domain"
)

type Server struct {
	accountv1.UnimplementedAccountServiceServer
	open     *commands.OpenAccountHandler
	deposit  *commands.DepositMoneyHandler
	withdraw *commands.WithdrawMoneyHandler
	ledger   ports.LedgerClient
	uow      ports.UnitOfWork
}

func NewServer(
	open *commands.OpenAccountHandler,
	deposit *commands.DepositMoneyHandler,
	withdraw *commands.WithdrawMoneyHandler,
	ledger ports.LedgerClient,
	uow ports.UnitOfWork,
) *Server {
	return &Server{
		open:     open,
		deposit:  deposit,
		withdraw: withdraw,
		ledger:   ledger,
		uow:      uow,
	}
}

func (s *Server) OpenAccount(ctx context.Context, req *accountv1.OpenAccountRequest) (*accountv1.OpenAccountResponse, error) {
	if req.CustomerId == "" {
		return nil, errors.New("customer_id required")
	}
	res, err := s.open.Handle(ctx, commands.OpenAccount{
		CustomerID:     ids.ID(req.CustomerId),
		IdempotencyKey: req.IdempotencyKey,
	})
	if err != nil {
		return nil, err
	}
	return &accountv1.OpenAccountResponse{AccountId: res.AccountID.String()}, nil
}

func (s *Server) Deposit(ctx context.Context, req *accountv1.DepositRequest) (*accountv1.DepositResponse, error) {
	if req.Amount == nil {
		return nil, errors.New("amount required")
	}
	amt, err := money.New(req.Amount.AmountKobo, money.Currency(req.Amount.Currency))
	if err != nil {
		return nil, err
	}
	res, err := s.deposit.Handle(ctx, commands.DepositMoney{
		AccountID:      ids.ID(req.AccountId),
		Amount:         amt,
		IdempotencyKey: req.IdempotencyKey,
		Narration:      req.Narration,
	})
	if err != nil {
		return nil, err
	}
	return &accountv1.DepositResponse{EntryId: res.EntryID.String()}, nil
}

func (s *Server) Withdraw(ctx context.Context, req *accountv1.WithdrawRequest) (*accountv1.WithdrawResponse, error) {
	if req.Amount == nil {
		return nil, errors.New("amount required")
	}
	amt, err := money.New(req.Amount.AmountKobo, money.Currency(req.Amount.Currency))
	if err != nil {
		return nil, err
	}
	res, err := s.withdraw.Handle(ctx, commands.WithdrawMoney{
		AccountID:      ids.ID(req.AccountId),
		Amount:         amt,
		IdempotencyKey: req.IdempotencyKey,
		Narration:      req.Narration,
	})
	if err != nil {
		return nil, err
	}
	return &accountv1.WithdrawResponse{EntryId: res.EntryID.String()}, nil
}

func (s *Server) GetAccount(ctx context.Context, req *accountv1.GetAccountRequest) (*accountv1.GetAccountResponse, error) {
	if req.AccountId == "" {
		return nil, errors.New("account_id required")
	}

	var customerID string
	var status string
	err := s.uow.WithTx(ctx, func(ctx context.Context, es ports.AccountEventStore, ob ports.OutboxStore) error {
		events, err := es.Read(ctx, req.AccountId, 0, 1000)
		if err != nil {
			return err
		}
		if len(events) == 0 {
			return errors.New("account not found")
		}

		var domainEvents []domain.Event
		for _, e := range events {
			ev, err := commands.UnmarshalAccountEvent(e.Type, e.Data)
			if err != nil {
				return err
			}
			domainEvents = append(domainEvents, ev)
		}

		acc, err := domain.Rehydrate(domainEvents)
		if err != nil {
			return err
		}
		customerID = acc.CustomerID().String()
		status = string(acc.Status())
		return nil
	})
	if err != nil {
		return nil, err
	}

	bal, err := s.ledger.GetBalance(ctx, ids.ID(req.AccountId))
	if err != nil {
		return nil, err
	}

	return &accountv1.GetAccountResponse{
		AccountId:        req.AccountId,
		CustomerId:       customerID,
		Status:           status,
		AvailableBalance: &commonv1.Money{Currency: string(bal.Currency()), AmountKobo: bal.AmountKobo()},
	}, nil
}
