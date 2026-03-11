package main

import (
	"context"
	"net"
	"os"

	ledgerv1 "fincore/gen/go/ledger/v1"
	"fincore/pkg/postgres"
	"fincore/services/ledger-service/application/commands"
	ledgergrpc "fincore/services/ledger-service/infrastructure/grpc"
	ledgerpg "fincore/services/ledger-service/infrastructure/postgres"

	"google.golang.org/grpc"
)

func main() {
	ctx := context.Background()

	dsn := os.Getenv("LEDGER_DB_DSN")
	addr := os.Getenv("LEDGER_LISTEN_ADDR")
	if addr == "" {
		addr = ":50053"
	}

	pool, err := postgres.NewPool(ctx, postgres.Config{DSN: dsn})
	if err != nil {
		panic(err)
	}
	defer pool.Close()

	uow := ledgerpg.NewUnitOfWork(pool)
	post := commands.NewPostEntryHandler(uow)

	balQuery := ledgerpg.NewBalanceQuery(pool)

	l, err := net.Listen("tcp", addr)
	if err != nil {
		panic(err)
	}

	s := grpc.NewServer()
	ledgerv1.RegisterLedgerServiceServer(s, ledgergrpc.NewServer(post, balQuery))
	if err := s.Serve(l); err != nil {
		panic(err)
	}
}
