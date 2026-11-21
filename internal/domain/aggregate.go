package domain

import (
	"math"
	"sort"
	"time"
)

// AggregateDaily maps raw rows onto exactly one bucket per UTC day in r.
// Missing days are zero-filled; duplicate rows for a day are summed.
func AggregateDaily(r TimeRange, rows []DailyUsage) []DailyUsage {
	buckets := r.DailyStarts()
	index := make(map[time.Time]int, len(buckets))
	for i, b := range buckets {
		index[b] = i
	}

	agg := make([]DailyUsage, len(buckets))
	for i, b := range buckets {
		agg[i] = DailyUsage{Day: b}
	}

	for _, row := range rows {
		day := startOfUTCDay(row.Day)
		i, ok := index[day]
		if !ok {
			continue
		}
		agg[i].CreditSpend += row.CreditSpend
		agg[i].Tokens += row.Tokens
	}

	return agg
}

// SumDaily totals spend, tokens, and active days across buckets.
func SumDaily(buckets []DailyUsage) DayTotal {
	var total DayTotal
	for _, b := range buckets {
		total.CreditSpend += b.CreditSpend
		total.Tokens += b.Tokens
		if b.CreditSpend != 0 || b.Tokens != 0 {
			total.ActiveDays++
		}
	}
	return total
}

// SortModels returns rows ordered by spend descending, ties broken by model
// name so rendering is deterministic. The input slice is not mutated.
func SortModels(models []ModelUsage) []ModelUsage {
	out := make([]ModelUsage, len(models))
	copy(out, models)
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].CreditSpend != out[j].CreditSpend {
			return out[i].CreditSpend > out[j].CreditSpend
		}
		return out[i].Model < out[j].Model
	})
	return out
}

// TopModels limits rows to the top n by spend. The remainder is folded into a
// row named "Other (remaining)" so it can never be confused with a model that
// is literally called "Other".
func TopModels(models []ModelUsage, n int) []ModelUsage {
	if n <= 0 {
		return nil
	}
	sorted := SortModels(models)
	if len(sorted) <= n {
		return sorted
	}

	top := make([]ModelUsage, 0, n)
	top = append(top, sorted[:n-1]...)

	rest := ModelUsage{Model: "Other (remaining)"}
	for _, m := range sorted[n-1:] {
		rest.CreditSpend += m.CreditSpend
		rest.Tokens += m.Tokens
	}
	return append(top, rest)
}

// FormatCredits renders a USD amount with two decimals, and distinguishes a
// tiny positive value from true zero.
func FormatCredits(v float64) string {
	if v > 0 && v < 0.005 {
		return "<$0.01"
	}
	neg := v < 0
	if neg {
		v = -v
	}
	whole := math.Floor(v)
	frac := int(math.Round((v - whole) * 100))
	if frac == 100 {
		whole++
		frac = 0
	}
	out := "$" + groupThousands(int64(whole)) + "." + twoDigits(frac)
	if neg {
		out = "-" + out
	}
	return out
}

// FormatTokens renders token counts with K/M/B suffixes for display.
func FormatTokens(v int64) string {
	neg := v < 0
	if neg {
		v = -v
	}
	var out string
	switch {
	case v >= 1_000_000_000:
		out = trimZero(float64(v)/1e9) + "B"
	case v >= 1_000_000:
		out = trimZero(float64(v)/1e6) + "M"
	case v >= 1_000:
		out = trimZero(float64(v)/1e3) + "K"
	default:
		out = groupThousands(v)
	}
	if neg {
		return "-" + out
	}
	return out
}

// FormatDaysRemaining renders the headline runway value.
func FormatDaysRemaining(est RunwayEstimate) string {
	switch est.Basis {
	case BasisUnavailable:
		return "Unavailable"
	case BasisNoRecentSpend:
		return "No recent spend"
	case BasisBeyondHorizon:
		return "Beyond horizon"
	}
	if est.DaysRemaining == nil {
		return "Unavailable"
	}
	return groupThousands(int64(math.Floor(*est.DaysRemaining))) + " days"
}

// FormatDepletionDate renders the approximate depletion date as a short
// calendar label. The date is already anchored in the display timezone when
// the estimate was built, so rendering never reinterprets it.
func FormatDepletionDate(est RunwayEstimate) string {
	if est.Basis != BasisFinite || est.DepletionDate == nil {
		return ""
	}
	return "~" + est.DepletionDate.Format("Jan 02")
}

func groupThousands(v int64) string {
	if v < 0 {
		return "-" + groupThousands(-v)
	}
	s := ""
	digits := intToString(v)
	for i := len(digits) - 1; i >= 0; i-- {
		s = string(digits[i]) + s
		if (len(digits)-i)%3 == 0 && i != 0 {
			s = "," + s
		}
	}
	return s
}

func intToString(v int64) string {
	if v == 0 {
		return "0"
	}
	var buf [20]byte
	i := len(buf)
	for v > 0 {
		i--
		buf[i] = byte('0' + v%10)
		v /= 10
	}
	return string(buf[i:])
}

func twoDigits(v int) string {
	if v < 10 {
		return "0" + intToString(int64(v))
	}
	return intToString(int64(v))
}

func trimZero(v float64) string {
	s := formatOneDecimal(v)
	if len(s) > 2 && s[len(s)-2:] == ".0" {
		return s[:len(s)-2]
	}
	return s
}

func formatOneDecimal(v float64) string {
	whole := int64(v)
	frac := int(math.Round((v - float64(whole)) * 10))
	if frac == 10 {
		whole++
		frac = 0
	}
	return groupThousands(whole) + "." + intToString(int64(frac))
}
