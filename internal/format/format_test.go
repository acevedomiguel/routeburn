package format

import (
	"testing"
	"time"

	"github.com/acevedomiguel/burnblocks/openrouter"
)

func TestFormatCreditsAndTokens(t *testing.T) {
	creditTests := map[float64]string{
		0:      "$0.00",
		12:     "$12.00",
		0.004:  "<$0.01",
		0.01:   "$0.01",
		1234.5: "$1,234.50",
	}
	for in, want := range creditTests {
		if got := FormatCredits(in); got != want {
			t.Errorf("FormatCredits(%v) = %q, want %q", in, got, want)
		}
	}

	tokenTests := map[int64]string{
		0:             "0",
		999:           "999",
		1500:          "1.5K",
		8_420_000:     "8.4M",
		1_250_000_000: "1.3B",
	}
	for in, want := range tokenTests {
		if got := FormatTokens(in); got != want {
			t.Errorf("FormatTokens(%d) = %q, want %q", in, got, want)
		}
	}
}

// The runway formula itself is tested in burnblocks; here the estimates are
// constructed directly so the assertions cover rendering only.
func TestFormatRunwayLines(t *testing.T) {
	days46 := 46.0
	zero := 0.0
	cases := []struct {
		name string
		est  openrouter.RunwayEstimate
		want string
	}{
		{"finite", openrouter.RunwayEstimate{Basis: openrouter.BasisFinite, DaysRemaining: &days46}, "46 days"},
		{"depleted", openrouter.RunwayEstimate{Basis: openrouter.BasisDepleted, DaysRemaining: &zero}, "0 days"},
		{"unavailable", openrouter.RunwayEstimate{Basis: openrouter.BasisUnavailable}, "Unavailable"},
		{"no recent spend", openrouter.RunwayEstimate{Basis: openrouter.BasisNoRecentSpend}, "No recent spend"},
		{"beyond horizon", openrouter.RunwayEstimate{Basis: openrouter.BasisBeyondHorizon}, "Beyond horizon"},
		{"finite without days", openrouter.RunwayEstimate{Basis: openrouter.BasisFinite}, "Unavailable"},
	}
	for _, tc := range cases {
		if got := FormatDaysRemaining(tc.est); got != tc.want {
			t.Errorf("%s: days = %q, want %q", tc.name, got, tc.want)
		}
	}
}

func TestFormatDepletionDate(t *testing.T) {
	// The estimator anchors the date in the display zone before it reaches
	// here, so rendering must never reinterpret it.
	utcDate := time.Date(2026, time.January, 6, 0, 0, 0, 0, time.UTC)
	westAnchored := time.Date(2026, time.January, 5, 0, 0, 0, 0, time.UTC)
	cases := []struct {
		name string
		est  openrouter.RunwayEstimate
		want string
	}{
		{"finite", openrouter.RunwayEstimate{Basis: openrouter.BasisFinite, DepletionDate: &utcDate}, "~Jan 06"},
		{"west-anchored date is not shifted", openrouter.RunwayEstimate{Basis: openrouter.BasisFinite, DepletionDate: &westAnchored}, "~Jan 05"},
		{"unavailable", openrouter.RunwayEstimate{Basis: openrouter.BasisUnavailable, DepletionDate: &utcDate}, ""},
		{"finite without a date", openrouter.RunwayEstimate{Basis: openrouter.BasisFinite}, ""},
	}
	for _, tc := range cases {
		if got := FormatDepletionDate(tc.est); got != tc.want {
			t.Errorf("%s: date = %q, want %q", tc.name, got, tc.want)
		}
	}
}
