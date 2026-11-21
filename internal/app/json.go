package app

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/acevedomiguel/routeburn/internal/domain"
)

// JSONSnapshot is the versioned machine-readable contract from plan 9.3.
type JSONSnapshot struct {
	SchemaVersion int         `json:"schema_version"`
	GeneratedAt   string      `json:"generated_at"`
	Cached        bool        `json:"cached"`
	Account       JSONAccount `json:"account"`
	Last30Days    JSONLast30  `json:"last_30_days"`
	Runway        JSONRunway  `json:"runway"`
	Key           *JSONKey    `json:"key"`
	Warnings      []string    `json:"warnings"`
}

// JSONAccount mirrors the credits endpoint in the public contract.
type JSONAccount struct {
	TotalCredits float64 `json:"total_credits"`
	TotalUsage   float64 `json:"total_usage"`
	Remaining    float64 `json:"remaining"`
}

// JSONLast30 is the 30-day display window summary.
type JSONLast30 struct {
	CreditSpend float64 `json:"credit_spend"`
	Tokens      int64   `json:"tokens"`
}

// JSONRunway is the estimated runway section.
type JSONRunway struct {
	WindowDays    int      `json:"window_days"`
	DailyBurn     float64  `json:"daily_burn"`
	DaysRemaining *float64 `json:"days_remaining"`
	DepletionDate *string  `json:"depletion_date"`
	Basis         string   `json:"basis"`
	LowConfidence bool     `json:"low_confidence"`
}

// JSONKey is the optional per-key section.
type JSONKey struct {
	Label          string   `json:"label,omitempty"`
	Limit          *float64 `json:"limit"`
	LimitRemaining *float64 `json:"limit_remaining"`
	LimitReset     string   `json:"limit_reset,omitempty"`
	UsageMonthly   float64  `json:"usage_monthly"`
}

// BuildJSON converts a snapshot into the stable public contract.
func BuildJSON(snap domain.Snapshot, cached bool) JSONSnapshot {
	total := 0.0
	for _, d := range snap.Daily {
		total += d.CreditSpend
	}

	rw := JSONRunway{
		WindowDays:    snap.Runway.WindowDays,
		DailyBurn:     snap.Runway.DailyBurn,
		DaysRemaining: snap.Runway.DaysRemaining,
		Basis:         string(snap.Runway.Basis),
		LowConfidence: snap.Runway.LowConfidence,
	}
	if rw.Basis == "" {
		rw.Basis = string(domain.BasisUnavailable)
	}
	if snap.Runway.DepletionDate != nil {
		d := snap.Runway.DepletionDate.Format("2006-01-02")
		rw.DepletionDate = &d
	}

	out := JSONSnapshot{
		SchemaVersion: domain.SchemaVersion,
		GeneratedAt:   snap.GeneratedAt.UTC().Format(time.RFC3339),
		Cached:        cached,
		Account: JSONAccount{
			TotalCredits: snap.Account.TotalPurchased,
			TotalUsage:   snap.Account.TotalUsed,
			Remaining:    snap.Account.Remaining,
		},
		Last30Days: JSONLast30{CreditSpend: total, Tokens: snap.TotalTokens},
		Runway:     rw,
		Warnings:   make([]string, 0, len(snap.Warnings)),
	}

	if snap.Key != nil {
		out.Key = &JSONKey{
			Label:          snap.Key.Label,
			Limit:          snap.Key.Limit,
			LimitRemaining: snap.Key.LimitRemaining,
			LimitReset:     snap.Key.LimitReset,
			UsageMonthly:   snap.Key.UsageMonthly,
		}
	}

	for _, w := range snap.Warnings {
		out.Warnings = append(out.Warnings, fmt.Sprintf("%s: %s", w.Section, w.Message))
	}

	return out
}

// MarshalJSON renders the contract as indented JSON with a trailing newline.
func MarshalJSON(snap domain.Snapshot, cached bool) ([]byte, error) {
	data, err := json.MarshalIndent(BuildJSON(snap, cached), "", "  ")
	if err != nil {
		return nil, err
	}
	return append(data, '\n'), nil
}
