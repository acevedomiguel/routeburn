package ui

import (
	"fmt"
	"strings"
	"time"
)

// Percent returns the raw percentage and the value clamped to 0..100 for
// drawing bars, guarding against zero or negative denominators.
func Percent(part, total float64) (pct, clamped float64) {
	if total <= 0 {
		return 0, 0
	}
	pct = part / total * 100
	clamped = pct
	if clamped < 0 {
		clamped = 0
	}
	if clamped > 100 {
		clamped = 100
	}
	return pct, clamped
}

// Bar renders a fixed-width bracketed meter. It uses only ASCII so it stays
// readable everywhere and never depends on color.
func Bar(pct float64, width int) string {
	if width < 1 {
		width = 1
	}
	if pct < 0 {
		pct = 0
	}
	if pct > 100 {
		pct = 100
	}
	filled := int(float64(width)*pct/100 + 0.5)
	if filled > width {
		filled = width
	}
	return "[" + strings.Repeat("#", filled) + strings.Repeat("-", width-filled) + "]"
}

// FormatAge renders a coarse, human-readable age.
func FormatAge(d time.Duration) string {
	if d < time.Minute {
		return "just now"
	}
	minutes := int(d / time.Minute)
	if minutes < 60 {
		return fmt.Sprintf("%dm ago", minutes)
	}
	hours := minutes / 60
	if hours < 24 {
		rest := minutes % 60
		if rest == 0 {
			return fmt.Sprintf("%dh ago", hours)
		}
		return fmt.Sprintf("%dh %dm ago", hours, rest)
	}
	days := hours / 24
	rest := hours % 24
	if rest == 0 {
		return fmt.Sprintf("%dd ago", days)
	}
	return fmt.Sprintf("%dd %dh ago", days, rest)
}

// CachedLabel is the ASCII-safe freshness label for cached data.
func CachedLabel(age time.Duration, cached bool) string {
	if !cached {
		return "Live data"
	}
	return "Cached data - updated " + FormatAge(age)
}
