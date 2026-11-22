package app

import (
	"time"

	"github.com/acevedomiguel/routeburn/internal/domain"
)

// SampleSnapshotForJSON builds a deterministic snapshot for contract tests.
func SampleSnapshotForJSON() domain.Snapshot {
	now := time.Date(2026, 11, 21, 15, 42, 0, 0, time.UTC)
	rng := domain.DisplayRange(now)
	daily := make([]domain.DailyUsage, 0, 30)
	for i, day := range rng.DailyStarts() {
		daily = append(daily, domain.DailyUsage{
			Day:         day,
			CreditSpend: 1.25 + float64(i)/10,
			Tokens:      250000,
		})
	}

	days := 46.0
	date := time.Date(2027, 1, 6, 0, 0, 0, 0, time.UTC)
	limit := 50.0
	remaining := 31.2

	return domain.Snapshot{
		SchemaVersion: domain.SchemaVersion,
		GeneratedAt:   now,
		Range:         rng,
		Account: domain.AccountCredits{
			TotalPurchased: 100,
			TotalUsed:      57.9,
			Remaining:      42.1,
		},
		Daily:       daily,
		TotalTokens: 8420000,
		Runway: domain.RunwayEstimate{
			WindowDays:    30,
			WindowSpend:   12,
			DailyBurn:     0.4,
			DaysRemaining: &days,
			DepletionDate: &date,
			Basis:         domain.BasisFinite,
		},
		Key: &domain.KeyCredits{
			Label:          "production",
			Limit:          &limit,
			LimitRemaining: &remaining,
			LimitReset:     "monthly",
			UsageMonthly:   18.8,
		},
	}
}
