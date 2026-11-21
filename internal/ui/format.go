package ui

import (
	"fmt"
	"time"
)

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
