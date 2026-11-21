package domain

import "time"

// SchemaVersion is the version of the snapshot and cache schema. It is
// independent of the application version.
const SchemaVersion = 1

// Snapshot is the provider-independent view of one RouteBurn reading.
// It contains no transport types and no credentials.
type Snapshot struct {
	SchemaVersion int
	GeneratedAt   time.Time
	Range         TimeRange
	Account       AccountCredits
	Key           *KeyCredits
	Daily         []DailyUsage
	Models        []ModelUsage
	TotalTokens   int64
	CacheHitRate  *float64
	Runway        RunwayEstimate
	Warnings      []Warning
}

// AccountCredits is account-wide purchased and consumed credit, in USD.
type AccountCredits struct {
	TotalPurchased float64
	TotalUsed      float64
	Remaining      float64
}

// KeyCredits is the current API key's own limit and usage. Pointers mean
// "not configured / unlimited" and are rendered differently from zero.
type KeyCredits struct {
	Label          string
	Limit          *float64
	LimitRemaining *float64
	LimitReset     string
	UsageDaily     float64
	UsageWeekly    float64
	UsageMonthly   float64
}

// DailyUsage is credit spend and tokens for one UTC day.
type DailyUsage struct {
	Day         time.Time
	CreditSpend float64
	Tokens      int64
}

// DayTotal is an aggregated DailyUsage bucket; ActiveDays counts days with
// observed activity so the runway estimate can flag low confidence.
type DayTotal struct {
	DailyUsage
	ActiveDays int
}

// ModelUsage is credit spend and tokens attributed to one model.
type ModelUsage struct {
	Model       string
	CreditSpend float64
	Tokens      int64
}

// Warning is a user-facing, non-fatal note attached to a snapshot section.
type Warning struct {
	Section string
	Message string
}

// Basis explains how a runway estimate was derived. Every value is renderable
// text; none of them require the UI to interpret numbers.
type Basis string

const (
	BasisFinite        Basis = "finite"
	BasisDepleted      Basis = "depleted"
	BasisNoRecentSpend Basis = "no_recent_spend"
	BasisUnavailable   Basis = "unavailable"
	BasisBeyondHorizon Basis = "beyond_horizon"
)

// RunwayEstimate is the derived forecast. DaysRemaining and DepletionDate are
// nil whenever the basis makes a number meaningless.
type RunwayEstimate struct {
	WindowDays    int
	WindowSpend   float64
	DailyBurn     float64
	DaysRemaining *float64
	DepletionDate *time.Time
	Basis         Basis
	LowConfidence bool
	Warnings      []Warning
}
