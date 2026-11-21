package domain

import (
	"testing"
	"time"
)

func utc(y int, m time.Month, d, h int) time.Time {
	return time.Date(y, m, d, h, 0, 0, 0, time.UTC)
}

func TestDisplayRangeCoversThirtyBucketsIncludingToday(t *testing.T) {
	now := utc(2026, time.November, 21, 2)
	r := DisplayRange(now)

	wantStart := utc(2026, time.October, 23, 0)
	wantEnd := utc(2026, time.November, 22, 0)
	if !r.Start.Equal(wantStart) {
		t.Errorf("start = %s, want %s", r.Start, wantStart)
	}
	if !r.End.Equal(wantEnd) {
		t.Errorf("end = %s, want %s", r.End, wantEnd)
	}
	if got := r.Days(); got != 30 {
		t.Errorf("days = %d, want 30", got)
	}
}

func TestDisplayRangeIsHalfOpenAndEndExclusive(t *testing.T) {
	r := DisplayRange(utc(2026, time.November, 21, 23))

	if !r.Contains(utc(2026, time.October, 23, 0)) {
		t.Error("start boundary must be inside the range")
	}
	if r.Contains(r.End) {
		t.Error("end boundary must be outside the range")
	}
	if !r.Contains(r.End.Add(-time.Nanosecond)) {
		t.Error("instant before end must be inside the range")
	}
}

func TestRunwayRangeExcludesCurrentPartialDay(t *testing.T) {
	now := utc(2026, time.November, 21, 2)
	r := RunwayRange(now)

	wantStart := utc(2026, time.October, 22, 0)
	wantEnd := utc(2026, time.November, 21, 0)
	if !r.Start.Equal(wantStart) {
		t.Errorf("start = %s, want %s", r.Start, wantStart)
	}
	if !r.End.Equal(wantEnd) {
		t.Errorf("end = %s, want %s", r.End, wantEnd)
	}
	if got := r.Days(); got != 30 {
		t.Errorf("days = %d, want 30", got)
	}
	if r.Contains(utc(2026, time.November, 21, 0)) {
		t.Error("current day must be excluded from the runway range")
	}
}

func TestRangesUseUTCRegardlessOfLocalZone(t *testing.T) {
	loc := time.FixedZone("UTC+13", 13*60*60)
	now := time.Date(2026, time.November, 22, 1, 0, 0, 0, loc) // still 2026-11-21 in UTC

	dr := DisplayRange(now)
	rr := RunwayRange(now)

	if !dr.End.Equal(utc(2026, time.November, 22, 0)) {
		t.Errorf("display end = %s, want 2026-11-22T00:00:00Z", dr.End)
	}
	if !rr.End.Equal(utc(2026, time.November, 21, 0)) {
		t.Errorf("runway end = %s, want 2026-11-21T00:00:00Z", rr.End)
	}
	if dr.Start.Location() != time.UTC || rr.Start.Location() != time.UTC {
		t.Error("ranges must be expressed in UTC")
	}
}

func TestRangesAcrossMonthAndLeapYearBoundaries(t *testing.T) {
	tests := []struct {
		name      string
		now       time.Time
		wantStart time.Time
	}{
		{"month boundary", utc(2026, time.March, 5, 12), utc(2026, time.February, 4, 0)},
		{"leap year", utc(2028, time.March, 1, 12), utc(2028, time.February, 1, 0)},
		{"year boundary", utc(2027, time.January, 1, 0), utc(2026, time.December, 3, 0)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := DisplayRange(tt.now).Start; !got.Equal(tt.wantStart) {
				t.Errorf("start = %s, want %s", got, tt.wantStart)
			}
		})
	}
}

func TestDailyBucketsMatchRangeDays(t *testing.T) {
	r := DisplayRange(utc(2026, time.November, 21, 2))
	buckets := r.DailyStarts()

	if len(buckets) != 30 {
		t.Fatalf("buckets = %d, want 30", len(buckets))
	}
	if !buckets[0].Equal(r.Start) {
		t.Errorf("first bucket = %s, want %s", buckets[0], r.Start)
	}
	if !buckets[len(buckets)-1].Equal(r.End.Add(-24 * time.Hour)) {
		t.Errorf("last bucket = %s, want %s", buckets[len(buckets)-1], r.End.Add(-24*time.Hour))
	}
}
