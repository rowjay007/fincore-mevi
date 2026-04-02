package temporal

import (
	"context"
	"log"

	"fincore/services/payment-service/application/workflows"

	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"
)

type Worker struct {
	c         client.Client
	taskQueue string
	acts      *workflows.TransferActivities
}

func NewWorker(c client.Client, taskQueue string, acts *workflows.TransferActivities) *Worker {
	return &Worker{c: c, taskQueue: taskQueue, acts: acts}
}

func (w *Worker) Start(ctx context.Context) error {
	wk := worker.New(w.c, w.taskQueue, worker.Options{})
	workflows.RegisterTransferWorker(wk, w.acts)

	log.Printf("[Temporal] starting worker on task_queue=%s", w.taskQueue)
	return wk.Run(worker.InterruptCh())
}
