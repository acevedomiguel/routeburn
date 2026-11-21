package app

import (
	"time"

	"github.com/acevedomiguel/routeburn/internal/domain"
)

// DemoSnapshot is a synthetic snapshot for screenshots and manual review.
// It never contains real account data and requires no network access.
func DemoSnapshot(now time.Time) domain.Snapshot {
	now = now.UTC()
	display := domain.DisplayRange(now)

	spend := []float64{0, 0.12, 0.31, 0.44, 0.2, 0.05, 0, 0.61, 0.88, 0.42, 0.31, 0.27, 0.9, 1.2, 0.77, 0.31, 0, 0.18, 0.24, 0.55, 0.4, 0.35, 0.2, 0.6, 0.45, 0.3, 0.15, 0.25, 0.5, 0.33}
	raw := make([]domain.DailyUsage, 0, len(spend))
	for i, v := range spend {
		day := display.Start.AddDate(0, 0, i)
		if day.After(now) {
			break
		}
		raw = append(raw, domain.DailyUsage{
			Day:         day,
			CreditSpend: v,
			Tokens:      int64(v * 700000),
		})
	}

	daily := domain.AggregateDaily(display, raw)
	runwayTotal := domain.SumDaily(domain.AggregateDaily(domain.RunwayRange(now), raw))

	remaining := 42.10
	models := []domain.ModelUsage{
		{Model: "anthropic/claude-sonnet-4", CreditSpend: 7.21, Tokens: 4100000},
		{Model: "openai/gpt-5", CreditSpend: 3.82, Tokens: 2900000},
		{Model: "meta/llama-4-70b", CreditSpend: 1.31, Tokens: 1420000},
		{Model: "google/gemini-2.5-pro", CreditSpend: 0.62, Tokens: 320000},
		{Model: "mistralai/mistral-large", CreditSpend: 0.21, Tokens: 90000},
	}

	keyLimit := 50.0
	keyRemaining := 31.20
	cacheHitRate := 18.4

	total := domain.SumDaily(daily)

	return domain.Snapshot{
		SchemaVersion: domain.SchemaVersion,
		GeneratedAt:   now,
		Range:         display,
		Account: domain.AccountCredits{
			TotalPurchased: 100,
			TotalUsed:      57.90,
			Remaining:      remaining,
		},
		Key: &domain.KeyCredits{
			Label:          "production",
			Limit:          &keyLimit,
			LimitRemaining: &keyRemaining,
			LimitReset:     "monthly",
			UsageMonthly:   18.80,
		},
		Daily:        daily,
		Models:       domain.SortModels(models),
		TotalTokens:  total.Tokens,
		CacheHitRate: &cacheHitRate,
		Runway: domain.EstimateRunway(domain.EstimateInput{
			Remaining:          &remaining,
			WindowDays:         30,
			WindowSpend:        runwayTotal.CreditSpend,
			ActiveDays:         runwayTotal.ActiveDays,
			AnalyticsAvailable: true,
			Now:                now,
			Location:           time.UTC,
		}),
	}
}
