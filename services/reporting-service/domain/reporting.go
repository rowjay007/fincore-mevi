package domain

import (
	"context"
	"time"
)

type BaselIIIReport struct {
	CapitalRatio           string
	LiquidityCoverageRatio string
	GeneratedAt            time.Time
}

type AMLAlert struct {
	UserID        string
	TransactionID string
	RiskScore     float32
	Reason        string
}

type DashboardStats struct {
	TotalTransactions int64
	TotalVolumeKobo   string
	AvgFraudScore     float32
}

type ReportingPort interface {
	GetBaselIIIReport(ctx context.Context, start, end time.Time) (BaselIIIReport, error)
	GetAMLMonitoringReport(ctx context.Context, start, end time.Time, threshold float32) ([]AMLAlert, error)
	GetDashboardStats(ctx context.Context, period string) (DashboardStats, error)
}
