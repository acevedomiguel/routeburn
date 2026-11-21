package domain

import (
	"math"
	"testing"
	"time"
)

func sp(v float64) *float64 { return &v }

func TestRunwayHypotheticalCaseFromPlan(t *testing.T) {
	remaining := 18.40

	est := EstimateRunway(EstimateInput{
		Remaining:          &remaining,
		WindowDays:         30,
		WindowSpend:        12.00,
		ActiveDays:         30,
		AnalyticsAvailable: true,
		Now:                utc(2026, time.November, 21, 2),
	})

	if est.Basis != BasisFinite {
		t.Fatalf("basis = %q, want %q", est.Basis, BasisFinite)
	}
	if math.Abs(est.DailyBurn-0.40) > 1e-9 {
		t.Errorf("daily burn = %v, want 0.40", est.DailyBurn)
	}
	if est.DaysRemaining == nil {
		t.Fatal("days remaining must be set")
	} else if math.Abs(*est.DaysRemaining-46) > 1e-6 {
		t.Errorf("days remaining = %v, want 46", *est.DaysRemaining)
	}
	if est.DepletionDate == nil {
		t.Fatal("depletion date must be set")
	} else if want := utc(2027, time.January, 6, 0); !est.DepletionDate.Equal(want) {
		t.Errorf("depletion date = %s, want %s", est.DepletionDate, want)
	}
	if est.LowConfidence {
		t.Error("30 active days must not be low confidence")
	}
}

type runwayCase struct {
	name        string
	remaining   *float64
	spend       float64
	windowDays  int
	activeDays  int
	analytics   bool
	wantBasis   Basis
	wantDays    *float64
	wantDate    bool
	wantLowConf bool
	wantWarning bool
}

func TestRunwayEdgeCases(t *testing.T) {
	tests := []runwayCase{
		{
			name:      "zero balance is depleted",
			remaining: sp(0), spend: 12, windowDays: 30, activeDays: 30, analytics: true,
			wantBasis: BasisDepleted, wantDays: sp(0),
		},
		{
			name:      "negative balance is depleted and warns",
			remaining: sp(-2.5), spend: 12, windowDays: 30, activeDays: 30, analytics: true,
			wantBasis: BasisDepleted, wantDays: sp(0), wantWarning: true,
		},
		{
			name:      "zero spend means no recent spend",
			remaining: sp(10), spend: 0, windowDays: 30, activeDays: 0, analytics: true,
			wantBasis: BasisNoRecentSpend,
		},
		{
			name:      "missing analytics is unavailable",
			remaining: sp(10), windowDays: 30, analytics: false,
			wantBasis: BasisUnavailable,
		},
		{
			name:      "missing balance is unavailable",
			remaining: nil, spend: 12, windowDays: 30, analytics: true,
			wantBasis: BasisUnavailable,
		},
		{
			name:      "fewer than seven active days is low confidence",
			remaining: sp(10), spend: 1.4, windowDays: 30, activeDays: 3, analytics: true,
			wantBasis: BasisFinite, wantLowConf: true, wantDate: true,
		},
		{
			name:      "tiny positive spend keeps an honest day count",
			remaining: sp(0.01), spend: 0.000003, windowDays: 30, activeDays: 1, analytics: true,
			wantBasis: BasisFinite, wantLowConf: true,
		},
		{
			name:      "negative spend is preserved and warns",
			remaining: sp(10), spend: -1, windowDays: 30, activeDays: 30, analytics: true,
			wantBasis: BasisNoRecentSpend, wantWarning: true,
		},
		{
			name:      "beyond-horizon date is omitted but days stay finite",
			remaining: sp(1e9), spend: 1e-9, windowDays: 30, activeDays: 30, analytics: true,
			wantBasis: BasisFinite, wantLowConf: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			est := EstimateRunway(EstimateInput{
				Remaining:          tt.remaining,
				WindowDays:         tt.windowDays,
				WindowSpend:        tt.spend,
				ActiveDays:         tt.activeDays,
				AnalyticsAvailable: tt.analytics,
				Now:                utc(2026, time.November, 21, 12),
			})

			if est.Basis != tt.wantBasis {
				t.Errorf("basis = %q, want %q", est.Basis, tt.wantBasis)
			}
			switch {
			case tt.wantDays != nil:
				if est.DaysRemaining == nil {
					t.Errorf("days remaining = nil, want %v", *tt.wantDays)
				} else if math.Abs(*est.DaysRemaining-*tt.wantDays) > 1e-9 {
					t.Errorf("days remaining = %v, want %v", *est.DaysRemaining, *tt.wantDays)
				}
			case tt.wantBasis != BasisFinite:
				if est.DaysRemaining != nil {
					t.Errorf("days remaining = %v, want nil for basis %q", *est.DaysRemaining, tt.wantBasis)
				}
			default:
				if est.DaysRemaining == nil {
					t.Error("days remaining = nil, want a finite number")
				}
			}
			if gotDate := est.DepletionDate != nil; gotDate != tt.wantDate {
				t.Errorf("depletion date set = %v, want %v", gotDate, tt.wantDate)
			}
			if est.LowConfidence != tt.wantLowConf {
				t.Errorf("low confidence = %v, want %v", est.LowConfidence, tt.wantLowConf)
			}
			if gotWarning := len(est.Warnings) > 0; gotWarning != tt.wantWarning {
				t.Errorf("warnings = %v, want warning present = %v", est.Warnings, tt.wantWarning)
			}
			if est.DaysRemaining != nil && (math.IsNaN(*est.DaysRemaining) || math.IsInf(*est.DaysRemaining, 0)) {
				t.Errorf("days remaining must never be NaN or Inf, got %v", *est.DaysRemaining)
			}
			if math.IsNaN(est.DailyBurn) || math.IsInf(est.DailyBurn, 0) {
				t.Errorf("daily burn must never be NaN or Inf, got %v", est.DailyBurn)
			}
		})
	}
}

func TestRunwayDepletionDateRoundsUp(t *testing.T) {
	remaining := 1.0
	est := EstimateRunway(EstimateInput{
		Remaining:          &remaining,
		WindowDays:         30,
		WindowSpend:        11.99,
		ActiveDays:         30,
		AnalyticsAvailable: true,
		Now:                utc(2026, time.November, 21, 0),
	})

	if est.DaysRemaining == nil || est.DepletionDate == nil {
		t.Fatal("expected a finite estimate")
	}
	// 1/(11.99/30) = 2.5021 days -> ceil 3.
	if want := utc(2026, time.November, 24, 0); !est.DepletionDate.Equal(want) {
		t.Errorf("depletion date = %s, want %s", est.DepletionDate, want)
	}
}

func TestRunwayDepletionDateUsesStartOfUTCDayNotNow(t *testing.T) {
	remaining := 3.0
	est := EstimateRunway(EstimateInput{
		Remaining:          &remaining,
		WindowDays:         30,
		WindowSpend:        3.0,
		ActiveDays:         30,
		AnalyticsAvailable: true,
		Now:                utc(2026, time.November, 21, 23),
	})

	if est.DepletionDate == nil {
		t.Fatal("expected a depletion date")
	}
	if want := utc(2026, time.December, 21, 0); !est.DepletionDate.Equal(want) {
		t.Errorf("depletion date = %s, want %s", est.DepletionDate, want)
	}
}
