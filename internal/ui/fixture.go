package ui

import (
	"time"

	"github.com/acevedomiguel/routeburn/internal/domain"
)

// SampleSnapshotAt is a synthetic, sanitized fixture for rendering tests. It
// never contains real account data.
func SampleSnapshotAt(now time.Time) domain.Snapshot {
	rng := domain.DisplayRange(now)
	daily := make([]domain.DailyUsage, 0, 30)
	spend := []float64{0, 0.12, 0.31, 0.44, 0.2, 0.05, 0, 0.61, 0.88, 0.42, 0.31, 0.27, 0.9, 1.2, 0.77, 0.31, 0.0, 0.18, 0.24, 0.55, 0.4, 0.35, 0.2, 0.6, 0.45, 0.3, 0.15, 0.25, 0.5, 0.33}
	for i := 0; i < 30; i++ {
		v := 0.0
		if i < len(spend) {
			v = spend[i]
		}
		daily = append(daily, domain.DailyUsage{
			Day:         rng.Start.AddDate(0, 0, i),
			CreditSpend: v,
			Tokens:      int64(v * 700000),
		})
	}

	remaining := 42.10
	days := 102.0
	depletion := time.Date(2027, 3, 3, 0, 0, 0, 0, time.UTC)
	cacheHit := 18.4
	limit := 50.0
	limitRemaining := 31.20

	return domain.Snapshot{
		SchemaVersion: domain.SchemaVersion,
		GeneratedAt:   now,
		Range:         rng,
		Account: domain.AccountCredits{
			TotalPurchased: 100,
			TotalUsed:      57.90,
			Remaining:      remaining,
		},
		Key: &domain.KeyCredits{
			Label:          "production",
			Limit:          &limit,
			LimitRemaining: &limitRemaining,
			LimitReset:     "monthly",
			UsageMonthly:   18.80,
		},
		Daily: daily,
		Models: []domain.ModelUsage{
			{Model: "anthropic/claude-sonnet-4", CreditSpend: 7.21, Tokens: 4100000},
			{Model: "openai/gpt-5", CreditSpend: 3.82, Tokens: 2900000},
			{Model: "meta/llama-4-70b", CreditSpend: 1.31, Tokens: 1420000},
		},
		TotalTokens:      8420000,
		CacheHitRate:     &cacheHit,
		Requests:         1240,
		TokensPrompt:     7100000,
		TokensCompletion: 1320000,
		CachedTokens:     1512000,
		ReasoningTokens:  480000,
		Runway: domain.RunwayEstimate{
			WindowDays:    30,
			WindowSpend:   12.34,
			DailyBurn:     0.4113,
			DaysRemaining: &days,
			DepletionDate: &depletion,
			Basis:         domain.BasisFinite,
		},
	}
}

// SampleSnapshot is SampleSnapshotAt with a fixed clock.
func SampleSnapshot() domain.Snapshot {
	return SampleSnapshotAt(time.Date(2026, 11, 21, 15, 42, 0, 0, time.UTC))
}
