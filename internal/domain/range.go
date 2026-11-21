package domain

import "time"

// TimeRange is a half-open interval [Start, End) expressed in UTC.
// Half-open semantics mean the End instant belongs to the next bucket, which
// is what daily aggregation requires.
type TimeRange struct {
	Start time.Time
	End   time.Time
}

// DisplayRange returns the 30 daily buckets that include the current UTC day.
func DisplayRange(now time.Time) TimeRange {
	today := startOfUTCDay(now)
	return TimeRange{Start: today.AddDate(0, 0, -29), End: today.AddDate(0, 0, 1)}
}

// RunwayRange returns the 30 complete UTC days before today. The current
// partial day is excluded so a 2 AM launch does not count two hours as a day.
func RunwayRange(now time.Time) TimeRange {
	today := startOfUTCDay(now)
	return TimeRange{Start: today.AddDate(0, 0, -30), End: today}
}

// Days returns the number of days the range spans.
func (r TimeRange) Days() int {
	return int(r.End.Sub(r.Start) / (24 * time.Hour))
}

// Contains reports whether t falls inside [Start, End).
func (r TimeRange) Contains(t time.Time) bool {
	u := t.UTC()
	return !u.Before(r.Start) && u.Before(r.End)
}

// DailyStarts returns one UTC midnight per day bucket in the range.
func (r TimeRange) DailyStarts() []time.Time {
	n := r.Days()
	if n < 0 {
		n = 0
	}
	out := make([]time.Time, 0, n)
	for i := 0; i < n; i++ {
		out = append(out, r.Start.AddDate(0, 0, i))
	}
	return out
}

func startOfUTCDay(t time.Time) time.Time {
	u := t.UTC()
	return time.Date(u.Year(), u.Month(), u.Day(), 0, 0, 0, 0, time.UTC)
}
