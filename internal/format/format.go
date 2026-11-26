// Package format renders RouteBurn's display strings: credits, token counts,
// model names, and runway lines. It is presentation only. The headless
// calculations behind these values live in burnblocks.
package format

import (
	"math"
	"strings"

	"github.com/acevedomiguel/burnblocks/openrouter"
)

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

// FormatCount renders a plain count with thousands separators. It is used for
// request counts, where a K/M suffix would hide the exact figure.
func FormatCount(v int64) string {
	return groupThousands(v)
}

// ShortModelName strips the vendor prefix and a trailing release date from a
// model identifier so a dashboard row can show the name that identifies it.
// "deepseek/deepseek-v4.1-flash-20260910" becomes "deepseek-v4.1-flash".
// Names without a vendor prefix or date suffix are returned unchanged.
func ShortModelName(model string) string {
	name := model
	if i := strings.Index(name, "/"); i >= 0 {
		if i < len(name)-1 {
			name = name[i+1:]
		} else {
			name = name[:i]
		}
	}
	if len(name) > 9 && name[len(name)-9] == '-' && isAllDigits(name[len(name)-8:]) {
		name = name[:len(name)-9]
	}
	if name == "" {
		return model
	}
	return name
}

func isAllDigits(s string) bool {
	if s == "" {
		return false
	}
	for i := 0; i < len(s); i++ {
		if s[i] < '0' || s[i] > '9' {
			return false
		}
	}
	return true
}

// FormatDaysRemaining renders the headline runway value.
func FormatDaysRemaining(est openrouter.RunwayEstimate) string {
	switch est.Basis {
	case openrouter.BasisUnavailable:
		return "Unavailable"
	case openrouter.BasisNoRecentSpend:
		return "No recent spend"
	case openrouter.BasisBeyondHorizon:
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
func FormatDepletionDate(est openrouter.RunwayEstimate) string {
	if est.Basis != openrouter.BasisFinite || est.DepletionDate == nil {
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
