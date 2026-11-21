package app

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/acevedomiguel/routeburn/internal/domain"
	or "github.com/acevedomiguel/routeburn/internal/openrouter"
)

// API is the subset of the OpenRouter client the snapshot builder needs.
type API interface {
	Credits(ctx context.Context) (or.CreditSummary, error)
	Key(ctx context.Context) (or.KeyInfo, error)
	QueryAnalytics(ctx context.Context, q or.AnalyticsQuery) ([]or.AnalyticsRow, error)
	NextRefreshAllowedAt() time.Time
}

// FetchOptions configures one full snapshot fetch.
type FetchOptions struct {
	// Now is the reference instant; it is normalised to UTC.
	Now time.Time
	// Location anchors the depletion date. Defaults to UTC.
	Location *time.Location
	// Previous is the last good snapshot. Its optional sections are reused
	// when their fresh fetch fails, and the mixed freshness is warned about.
	Previous *domain.Snapshot
}

// FetchRange is the union of the display and runway windows: one extra day at
// the start lets both windows be derived from a single analytics query.
func FetchRange(now time.Time) domain.TimeRange {
	runway := domain.RunwayRange(now)
	display := domain.DisplayRange(now)
	return domain.TimeRange{Start: runway.Start, End: display.End}
}

// FetchSnapshot performs the parallel request batch and merges the results.
// Credits is required; every other section degrades to a warning.
func FetchSnapshot(ctx context.Context, api API, opts FetchOptions) (domain.Snapshot, error) {
	now := opts.Now.UTC()
	loc := opts.Location
	if loc == nil {
		loc = time.UTC
	}

	displayRange := domain.DisplayRange(now)
	runwayRange := domain.RunwayRange(now)
	fetchRange := FetchRange(now)

	var (
		wg sync.WaitGroup

		credits    or.CreditSummary
		creditsErr error

		dailyRows []or.AnalyticsRow
		dailyErr  error

		modelRows []or.AnalyticsRow
		modelErr  error

		keyInfo    or.KeyInfo
		keyErr     error
		keyMissing bool
	)

	wg.Add(3)
	go func() {
		defer wg.Done()
		credits, creditsErr = api.Credits(ctx)
	}()
	go func() {
		defer wg.Done()
		dailyRows, dailyErr = api.QueryAnalytics(ctx, or.AnalyticsQuery{
			Metrics:     []string{"total_usage", "tokens_total"},
			Granularity: "day",
			Start:       fetchRange.Start,
			End:         fetchRange.End,
		})
	}()
	go func() {
		defer wg.Done()
		modelRows, modelErr = api.QueryAnalytics(ctx, or.AnalyticsQuery{
			Metrics:     []string{"total_usage", "tokens_total"},
			Dimensions:  []string{"model"},
			Granularity: "day",
			Start:       fetchRange.Start,
			End:         fetchRange.End,
			Limit:       10,
		})
	}()
	wg.Wait()

	if creditsErr != nil {
		if errors.Is(creditsErr, or.ErrMissingManagementKey) {
			return domain.Snapshot{}, creditsErr
		}
		return domain.Snapshot{}, fmt.Errorf("app: account credits: %w", creditsErr)
	}

	// Key details are optional and need their own request. Only attempted when
	// the client has a regular API key configured.
	keyInfo, keyErr = api.Key(ctx)
	keyMissing = errors.Is(keyErr, or.ErrMissingAPIKey)
	if keyMissing {
		keyErr = nil
	}

	snap := domain.Snapshot{
		SchemaVersion: domain.SchemaVersion,
		GeneratedAt:   now,
		Range:         displayRange,
		Account: domain.AccountCredits{
			TotalPurchased: credits.TotalPurchased,
			TotalUsed:      credits.TotalUsed,
			Remaining:      credits.Remaining,
		},
	}

	var warnings []domain.Warning

	analyticsOK := true
	if dailyErr != nil {
		analyticsOK = false
		warnings = append(warnings, domain.Warning{
			Section: "Daily usage",
			Message: fmt.Sprintf("could not load 30-day analytics: %v", redactedError(dailyErr)),
		})
	}

	if opts.Previous != nil {
		snap.CacheHitRate = opts.Previous.CacheHitRate
	}

	switch {
	case analyticsOK:
		rows := toDailyRows(dailyRows)
		snap.Daily = domain.AggregateDaily(displayRange, rows)
		snap.TotalTokens = domain.SumDaily(domain.AggregateDaily(displayRange, rows)).Tokens
	case opts.Previous != nil:
		snap.Daily = opts.Previous.Daily
		snap.TotalTokens = opts.Previous.TotalTokens
		warnings = append(warnings, domain.Warning{
			Section: "Timestamps",
			Message: "daily figures come from the last successful refresh",
		})
	default:
		snap.Daily = domain.AggregateDaily(displayRange, nil)
	}

	modelsOK := modelErr == nil
	if modelErr != nil {
		warnings = append(warnings, domain.Warning{
			Section: "Cost by model",
			Message: fmt.Sprintf("could not load model breakdown: %v", redactedError(modelErr)),
		})
	}
	switch {
	case modelsOK:
		snap.Models = domain.TopModels(toModelRows(modelRows), 10)
	case opts.Previous != nil:
		snap.Models = opts.Previous.Models
	}

	// Runway always uses the 30 complete days, never the partial current day.
	windowSpend := 0.0
	activeDays := 0
	if analyticsOK {
		runwayDays := domain.AggregateDaily(runwayRange, toDailyRows(dailyRows))
		total := domain.SumDaily(runwayDays)
		windowSpend = total.CreditSpend
		activeDays = total.ActiveDays
	} else if opts.Previous != nil {
		runwayDays := domain.AggregateDaily(runwayRange, opts.Previous.Daily)
		total := domain.SumDaily(runwayDays)
		windowSpend = total.CreditSpend
		activeDays = total.ActiveDays
	}

	remaining := credits.Remaining
	snap.Runway = domain.EstimateRunway(domain.EstimateInput{
		Remaining:          &remaining,
		WindowDays:         runwayRange.Days(),
		WindowSpend:        windowSpend,
		ActiveDays:         activeDays,
		AnalyticsAvailable: analyticsOK || opts.Previous != nil,
		Now:                now,
		Location:           loc,
	})
	warnings = append(warnings, snap.Runway.Warnings...)

	switch {
	case keyErr != nil:
		warnings = append(warnings, domain.Warning{
			Section: "This API key",
			Message: fmt.Sprintf("could not load key details: %v", redactedError(keyErr)),
		})
		if opts.Previous != nil {
			snap.Key = opts.Previous.Key
		}
	case !keyMissing:
		snap.Key = &domain.KeyCredits{
			Label:          keyInfo.Label,
			Limit:          keyInfo.Limit,
			LimitRemaining: keyInfo.LimitRemaining,
			LimitReset:     keyInfo.LimitReset,
			UsageDaily:     keyInfo.UsageDaily,
			UsageWeekly:    keyInfo.UsageWeekly,
			UsageMonthly:   keyInfo.UsageMonthly,
		}
	}

	snap.Warnings = warnings
	return snap, nil
}

func toDailyRows(rows []or.AnalyticsRow) []domain.DailyUsage {
	out := make([]domain.DailyUsage, 0, len(rows))
	for _, r := range rows {
		out = append(out, domain.DailyUsage{
			Day:         r.Day.UTC(),
			CreditSpend: r.CreditSpend,
			Tokens:      r.Tokens,
		})
	}
	return out
}

func toModelRows(rows []or.AnalyticsRow) []domain.ModelUsage {
	out := make([]domain.ModelUsage, 0, len(rows))
	for _, r := range rows {
		if r.Model == "" {
			continue
		}
		out = append(out, domain.ModelUsage{Model: r.Model, CreditSpend: r.CreditSpend, Tokens: r.Tokens})
	}
	return out
}

// redactedError strips anything that looks like a credential from an error.
func redactedError(err error) string {
	if err == nil {
		return ""
	}
	return credRedact(err.Error())
}
