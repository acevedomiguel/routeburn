package domain

import (
	"math"
	"time"
)

// beyondHorizonDays caps a forecast that is not meaningful to a human. A
// balance that lasts more than a century is reported qualitatively instead of
// as a number that invites false precision.
const beyondHorizonDays = 36500

// EstimateInput carries everything the runway formula needs.
// AnalyticsAvailable distinguishes "the API returned zero spend" from "we could
// not read analytics at all"; those must not render the same.
// Location is the display timezone the depletion date is anchored to; nil
// means UTC.
type EstimateInput struct {
	Remaining          *float64
	WindowDays         int
	WindowSpend        float64
	ActiveDays         int
	AnalyticsAvailable bool
	Now                time.Time
	Location           *time.Location
}

// EstimateRunway applies the documented runway formula to a window of spend.
func EstimateRunway(in EstimateInput) RunwayEstimate {
	if in.WindowDays <= 0 {
		in.WindowDays = 30
	}

	est := RunwayEstimate{WindowDays: in.WindowDays, WindowSpend: in.WindowSpend}

	if in.Remaining == nil || !in.AnalyticsAvailable {
		est.Basis = BasisUnavailable
		return est
	}

	if in.WindowSpend < 0 {
		est.Warnings = append(est.Warnings, Warning{
			Section: "runway",
			Message: "Observed spend was negative; the burn rate is treated as zero.",
		})
		in.WindowSpend = 0
	}

	if *in.Remaining <= 0 {
		est.Basis = BasisDepleted
		zero := 0.0
		est.DaysRemaining = &zero
		if *in.Remaining < 0 {
			est.Warnings = append(est.Warnings, Warning{
				Section: "runway",
				Message: "Balance is negative; top up before making further requests.",
			})
		}
		return est
	}

	est.DailyBurn = in.WindowSpend / float64(in.WindowDays)
	if est.DailyBurn <= 0 {
		est.Basis = BasisNoRecentSpend
		return est
	}

	// Multiply before dividing: 18.40 * 30 / 12 is exactly 46, while
	// 18.40 / (12/30) carries float error just below 46.
	days := *in.Remaining * float64(in.WindowDays) / in.WindowSpend
	if math.IsNaN(days) || math.IsInf(days, 0) {
		est.Basis = BasisUnavailable
		return est
	}

	est.LowConfidence = in.ActiveDays < 7 || days > beyondHorizonDays
	est.Basis = BasisFinite
	est.DaysRemaining = &days

	if days <= beyondHorizonDays {
		loc := in.Location
		if loc == nil {
			loc = time.UTC
		}
		depletion := startOfDay(in.Now.In(loc)).AddDate(0, 0, int(math.Ceil(days)))
		est.DepletionDate = &depletion
	}

	return est
}

func startOfDay(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
}
