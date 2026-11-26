package app

import (
	"context"
	"sync"
	"time"

	"github.com/acevedomiguel/burnblocks/openrouter"
)

// fakeAPI is the synthetic read-only API shared by the app tests. The snapshot
// fetch tests that also used it moved to burnblocks with the code they cover.
type fakeAPI struct {
	mu sync.Mutex

	credits    openrouter.CreditSummary
	creditsErr error

	daily    []openrouter.AnalyticsRow
	dailyErr error

	models    []openrouter.AnalyticsRow
	modelsErr error

	key    openrouter.KeyInfo
	keyErr error

	calls   int
	queries []openrouter.AnalyticsQuery

	// blockCh and release delay every call until the test releases them.
	blockCh chan struct{}
}

func (f *fakeAPI) Credits(ctx context.Context) (openrouter.CreditSummary, error) {
	f.mu.Lock()
	f.calls++
	f.mu.Unlock()
	if f.blockCh != nil {
		select {
		case <-f.blockCh:
		case <-ctx.Done():
			return openrouter.CreditSummary{}, ctx.Err()
		}
	}
	return f.credits, f.creditsErr
}

func (f *fakeAPI) Key(ctx context.Context) (openrouter.KeyInfo, error) {
	return f.key, f.keyErr
}

func (f *fakeAPI) QueryAnalytics(ctx context.Context, q openrouter.AnalyticsQuery) ([]openrouter.AnalyticsRow, error) {
	f.mu.Lock()
	f.queries = append(f.queries, q)
	f.mu.Unlock()
	if len(q.Dimensions) > 0 {
		return f.models, f.modelsErr
	}
	return f.daily, f.dailyErr
}

func (f *fakeAPI) NextRefreshAllowedAt() time.Time { return time.Time{} }

func fixedNow() time.Time {
	return time.Date(2026, 11, 21, 15, 42, 0, 0, time.UTC)
}
