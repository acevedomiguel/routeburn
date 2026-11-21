package app

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/acevedomiguel/routeburn/internal/domain"
	or "github.com/acevedomiguel/routeburn/internal/openrouter"
)

type fakeAPI struct {
	credits    or.CreditSummary
	creditsErr error

	daily    []or.AnalyticsRow
	dailyErr error

	models    []or.AnalyticsRow
	modelsErr error

	key    or.KeyInfo
	keyErr error

	calls   int
	queries []or.AnalyticsQuery

	// blockCh and release delay every call until the test releases them.
	blockCh chan struct{}
}

func (f *fakeAPI) Credits(ctx context.Context) (or.CreditSummary, error) {
	f.calls++
	if f.blockCh != nil {
		select {
		case <-f.blockCh:
		case <-ctx.Done():
			return or.CreditSummary{}, ctx.Err()
		}
	}
	return f.credits, f.creditsErr
}

func (f *fakeAPI) Key(ctx context.Context) (or.KeyInfo, error) {
	return f.key, f.keyErr
}

func (f *fakeAPI) QueryAnalytics(ctx context.Context, q or.AnalyticsQuery) ([]or.AnalyticsRow, error) {
	f.queries = append(f.queries, q)
	if len(q.Dimensions) > 0 {
		return f.models, f.modelsErr
	}
	return f.daily, f.dailyErr
}

func (f *fakeAPI) NextRefreshAllowedAt() time.Time { return time.Time{} }

func fixedNow() time.Time {
	return time.Date(2026, 11, 21, 15, 42, 0, 0, time.UTC)
}

func dayRow(t time.Time, spend float64, tokens int64) or.AnalyticsRow {
	return or.AnalyticsRow{Day: t, CreditSpend: spend, Tokens: tokens}
}

func TestFetchSnapshotHappyPath(t *testing.T) {
	now := fixedNow()
	api := &fakeAPI{
		credits: or.CreditSummary{TotalPurchased: 100, TotalUsed: 57.9, Remaining: 42.1},
		daily: []or.AnalyticsRow{
			dayRow(time.Date(2026, 11, 20, 0, 0, 0, 0, time.UTC), 0.4, 1000),
			dayRow(time.Date(2026, 11, 19, 0, 0, 0, 0, time.UTC), 0.2, 500),
		},
		models: []or.AnalyticsRow{
			{Model: "openai/gpt-5", CreditSpend: 3.5, Tokens: 900},
			{Model: "anthropic/claude", CreditSpend: 1.5, Tokens: 400},
		},
		key: or.KeyInfo{Label: "production"},
	}

	snap, err := FetchSnapshot(context.Background(), api, FetchOptions{Now: now})
	if err != nil {
		t.Fatalf("FetchSnapshot: %v", err)
	}
	if snap.SchemaVersion != domain.SchemaVersion {
		t.Errorf("schema = %d", snap.SchemaVersion)
	}
	if snap.Account.Remaining != 42.1 {
		t.Errorf("remaining = %v", snap.Account.Remaining)
	}
	if len(snap.Daily) != 30 {
		t.Errorf("daily buckets = %d, want 30", len(snap.Daily))
	}
	if snap.TotalTokens != 1500 {
		t.Errorf("tokens = %d", snap.TotalTokens)
	}
	if len(snap.Models) != 2 {
		t.Errorf("models = %d", len(snap.Models))
	}
	if snap.Key == nil || snap.Key.Label != "production" {
		t.Errorf("key = %+v", snap.Key)
	}
	if len(snap.Warnings) != 0 {
		t.Errorf("unexpected warnings: %+v", snap.Warnings)
	}
}

func TestFetchSnapshotRequiresCredits(t *testing.T) {
	api := &fakeAPI{creditsErr: fmt.Errorf("boom")}
	_, err := FetchSnapshot(context.Background(), api, FetchOptions{Now: fixedNow()})
	if err == nil {
		t.Fatal("expected an error when credits fail")
	}
}

func TestFetchSnapshotMissingManagementKeyIsSentinel(t *testing.T) {
	api := &fakeAPI{creditsErr: or.ErrMissingManagementKey}
	_, err := FetchSnapshot(context.Background(), api, FetchOptions{Now: fixedNow()})
	if !errors.Is(err, or.ErrMissingManagementKey) {
		t.Fatalf("err = %v, want ErrMissingManagementKey", err)
	}
}

func TestFetchSnapshotAnalyticsFailureStillShowsBalance(t *testing.T) {
	api := &fakeAPI{
		credits:   or.CreditSummary{TotalPurchased: 100, TotalUsed: 57.9, Remaining: 42.1},
		dailyErr:  errors.New("analytics down"),
		modelsErr: errors.New("analytics down"),
		keyErr:    or.ErrMissingAPIKey,
	}
	snap, err := FetchSnapshot(context.Background(), api, FetchOptions{Now: fixedNow()})
	if err != nil {
		t.Fatalf("FetchSnapshot: %v", err)
	}
	if snap.Account.Remaining != 42.1 {
		t.Error("balance must survive an analytics failure")
	}
	if snap.Runway.Basis != domain.BasisUnavailable {
		t.Errorf("runway basis = %q, want unavailable", snap.Runway.Basis)
	}
	if len(snap.Warnings) < 2 {
		t.Errorf("expected section warnings, got %+v", snap.Warnings)
	}
	if snap.Key != nil {
		t.Error("a missing API key must not fabricate key details")
	}
}

func TestFetchSnapshotReusesPreviousSections(t *testing.T) {
	prev := &domain.Snapshot{
		SchemaVersion: domain.SchemaVersion,
		GeneratedAt:   fixedNow().Add(-time.Hour),
		Daily:         domain.AggregateDaily(domain.RunwayRange(fixedNow()), []domain.DailyUsage{{Day: time.Date(2026, 11, 15, 0, 0, 0, 0, time.UTC), CreditSpend: 0.6, Tokens: 10}}),
		Models:        []domain.ModelUsage{{Model: "cached/model", CreditSpend: 0.6}},
		Key:           &domain.KeyCredits{Label: "cached-key"},
	}
	api := &fakeAPI{
		credits:   or.CreditSummary{TotalPurchased: 100, TotalUsed: 57.9, Remaining: 42.1},
		dailyErr:  errors.New("down"),
		modelsErr: errors.New("down"),
		keyErr:    errors.New("down"),
	}

	snap, err := FetchSnapshot(context.Background(), api, FetchOptions{Now: fixedNow(), Previous: prev})
	if err != nil {
		t.Fatalf("FetchSnapshot: %v", err)
	}
	if len(snap.Models) != 1 || snap.Models[0].Model != "cached/model" {
		t.Errorf("models not reused: %+v", snap.Models)
	}
	if snap.Key == nil || snap.Key.Label != "cached-key" {
		t.Errorf("key not reused: %+v", snap.Key)
	}
	if snap.Runway.Basis != domain.BasisFinite {
		t.Errorf("runway from cached data should still estimate, got %q", snap.Runway.Basis)
	}
	found := false
	for _, w := range snap.Warnings {
		if w.Section == "Timestamps" {
			found = true
		}
	}
	if !found {
		t.Error("mixed freshness must be labelled")
	}
}

func TestFetchSnapshotRequestsOneRangeForBothWindows(t *testing.T) {
	api := &fakeAPI{
		credits: or.CreditSummary{TotalPurchased: 10, TotalUsed: 1, Remaining: 9},
		keyErr:  or.ErrMissingAPIKey,
	}
	now := fixedNow()
	if _, err := FetchSnapshot(context.Background(), api, FetchOptions{Now: now}); err != nil {
		t.Fatalf("FetchSnapshot: %v", err)
	}

	want := FetchRange(now)
	for i, q := range api.queries {
		if !q.Start.Equal(want.Start) || !q.End.Equal(want.End) {
			t.Errorf("query %d range = %v..%v, want %v..%v", i, q.Start, q.End, want.Start, want.End)
		}
	}
	if !want.Start.Equal(domain.RunwayRange(now).Start) {
		t.Error("fetch range must start with the runway window")
	}
}

func TestFetchSnapshotRunwayUsesCompleteDaysOnly(t *testing.T) {
	now := fixedNow()
	// Spend today (partial) must not enter the runway window.
	today := time.Date(2026, 11, 21, 0, 0, 0, 0, time.UTC)
	api := &fakeAPI{
		credits: or.CreditSummary{TotalPurchased: 100, TotalUsed: 12, Remaining: 18.40},
		daily: []or.AnalyticsRow{
			dayRow(today, 5.0, 100),
			dayRow(time.Date(2026, 11, 20, 0, 0, 0, 0, time.UTC), 0.4, 100),
			dayRow(time.Date(2026, 11, 19, 0, 0, 0, 0, time.UTC), 0.4, 100),
		},
		keyErr: or.ErrMissingAPIKey,
	}
	snap, err := FetchSnapshot(context.Background(), api, FetchOptions{Now: now})
	if err != nil {
		t.Fatalf("FetchSnapshot: %v", err)
	}
	if snap.Runway.WindowSpend != 0.8 {
		t.Errorf("window spend = %v, want 0.8 (today excluded)", snap.Runway.WindowSpend)
	}
}

func TestFetchSnapshotContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	api := &fakeAPI{blockCh: make(chan struct{})}
	done := make(chan error, 1)
	go func() {
		_, err := FetchSnapshot(ctx, api, FetchOptions{Now: fixedNow()})
		done <- err
	}()

	time.Sleep(20 * time.Millisecond)
	cancel()

	select {
	case err := <-done:
		if err == nil {
			t.Fatal("expected a cancellation error")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("FetchSnapshot ignored cancellation")
	}
}

func TestFetchSnapshotNegativeBalanceIsPreserved(t *testing.T) {
	api := &fakeAPI{
		credits: or.CreditSummary{TotalPurchased: 10, TotalUsed: 13.5, Remaining: -3.5},
		keyErr:  or.ErrMissingAPIKey,
	}
	snap, err := FetchSnapshot(context.Background(), api, FetchOptions{Now: fixedNow()})
	if err != nil {
		t.Fatalf("FetchSnapshot: %v", err)
	}
	if snap.Account.Remaining != -3.5 {
		t.Errorf("remaining = %v, negative balances must be preserved", snap.Account.Remaining)
	}
	if snap.Runway.Basis != domain.BasisDepleted {
		t.Errorf("basis = %q", snap.Runway.Basis)
	}
}

func TestFetchSnapshotNeverLeaksSecretsInWarnings(t *testing.T) {
	api := &fakeAPI{
		credits:   or.CreditSummary{TotalPurchased: 10, TotalUsed: 1, Remaining: 9},
		dailyErr:  errors.New("request failed for bearer sk-or-v1-abc123def"),
		modelsErr: or.ErrServer,
		keyErr:    or.ErrMissingAPIKey,
	}
	snap, err := FetchSnapshot(context.Background(), api, FetchOptions{Now: fixedNow()})
	if err != nil {
		t.Fatalf("FetchSnapshot: %v", err)
	}
	for _, w := range snap.Warnings {
		if w.Section == "Daily usage" && containsSecret(w.Message) {
			t.Errorf("warning leaked a secret: %q", w.Message)
		}
	}
}

func containsSecret(s string) bool { return strings.Contains(s, "sk-or-v1-abc123def") }
