// Package routeburn exposes RouteBurn's headless snapshot fetching and
// schema-2 JSON encoding for Go applications. Callers supply credentials and
// dependencies explicitly; the package does not read environment variables,
// keyring entries, configuration, cache, or write to terminal streams.
package routeburn

import (
	"context"
	"time"

	"github.com/acevedomiguel/routeburn/internal/app"
	"github.com/acevedomiguel/routeburn/internal/domain"
)

// Domain vocabulary retained by the public snapshot contract.
type (
	Snapshot       = domain.Snapshot
	AccountCredits = domain.AccountCredits
	KeyCredits     = domain.KeyCredits
	DailyUsage     = domain.DailyUsage
	ModelUsage     = domain.ModelUsage
	RunwayEstimate = domain.RunwayEstimate
	Warning        = domain.Warning
	TimeRange      = domain.TimeRange
	Basis          = domain.Basis
)

const (
	// SchemaVersion is the version of the snapshot and JSON schema.
	SchemaVersion = domain.SchemaVersion

	// Runway bases describe how an estimate was derived.
	BasisFinite        = domain.BasisFinite
	BasisDepleted      = domain.BasisDepleted
	BasisNoRecentSpend = domain.BasisNoRecentSpend
	BasisUnavailable   = domain.BasisUnavailable
	BasisBeyondHorizon = domain.BasisBeyondHorizon
)

// API is the read-only fetch boundary required to build a snapshot.
type API interface {
	Credits(ctx context.Context) (CreditSummary, error)
	Key(ctx context.Context) (KeyInfo, error)
	QueryAnalytics(ctx context.Context, q AnalyticsQuery) ([]AnalyticsRow, error)
	NextRefreshAllowedAt() time.Time
}

// FetchOptions configures one full snapshot fetch.
type FetchOptions = app.FetchOptions

// Fetch builds a snapshot from api without reading credentials or cache.
func Fetch(ctx context.Context, api API, opts FetchOptions) (Snapshot, error) {
	return app.FetchSnapshot(ctx, api, opts)
}

// FetchRange returns the union of the display and runway windows.
func FetchRange(now time.Time) TimeRange {
	return app.FetchRange(now)
}
