package ui

import (
	"strings"
	"testing"
)

func TestChartSummary(t *testing.T) {
	snap := SampleSnapshotAt(fixedNow())
	s := ChartSummary(snap.Daily)
	if !strings.Contains(s, "30-day spend") {
		t.Fatalf("summary = %q", s)
	}
}

func TestDateRangeLabel(t *testing.T) {
	snap := SampleSnapshotAt(fixedNow())
	label := DateRangeLabel(snap.Range)
	if label != "Oct 23 - Nov 21" {
		t.Fatalf("label = %q", label)
	}
}
