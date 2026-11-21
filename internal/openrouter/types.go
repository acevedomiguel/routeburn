package openrouter

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// decodeAnalyticsRows accepts the documented envelope plus the row shapes seen
// in practice. Unrecognized shapes are an explicit compatibility error rather
// than a silently empty chart.
func decodeAnalyticsRows(raw json.RawMessage) ([]AnalyticsRow, error) {
	trimmed := strings.TrimSpace(string(raw))
	if trimmed == "" || trimmed == "null" {
		return nil, nil
	}

	var rows []map[string]any
	switch trimmed[0] {
	case '[':
		if err := json.Unmarshal(raw, &rows); err != nil {
			return nil, malformedAnalytics(err)
		}
	case '{':
		var envelope map[string]any
		if err := json.Unmarshal(raw, &envelope); err != nil {
			return nil, malformedAnalytics(err)
		}
		found := false
		for _, key := range []string{"data", "rows", "results", "items"} {
			value, ok := envelope[key]
			if !ok {
				continue
			}
			switch typed := value.(type) {
			case []any:
				rows = make([]map[string]any, 0, len(typed))
				for _, item := range typed {
					m, ok := item.(map[string]any)
					if !ok {
						return nil, malformedAnalytics(fmt.Errorf("row is not an object"))
					}
					rows = append(rows, m)
				}
				found = true
			case map[string]any:
				var nested any
				for _, nestedKey := range []string{"data", "rows", "results", "items"} {
					if candidate, ok := typed[nestedKey]; ok {
						nested = candidate
						break
					}
				}
				if nested == nil {
					continue
				}
				list, ok := nested.([]any)
				if !ok {
					return nil, malformedAnalytics(fmt.Errorf("nested rows is not an array"))
				}
				rows = make([]map[string]any, 0, len(list))
				for _, item := range list {
					m, ok := item.(map[string]any)
					if !ok {
						return nil, malformedAnalytics(fmt.Errorf("row is not an object"))
					}
					rows = append(rows, m)
				}
				found = true
			}
			if found {
				break
			}
		}
		if !found {
			return nil, malformedAnalytics(fmt.Errorf("no rows found in analytics response"))
		}
	default:
		return nil, malformedAnalytics(fmt.Errorf("unexpected analytics payload"))
	}

	out := make([]AnalyticsRow, 0, len(rows))
	for _, row := range rows {
		out = append(out, AnalyticsRow{
			Day:         rowTime(row),
			Model:       rowString(row, "model", "model_name", "model_permaslug"),
			CreditSpend: rowFloat(row, "total_usage", "usage", "credit_spend", "credits"),
			Tokens:      int64(rowFloat(row, "tokens_total", "tokens", "total_tokens")),
		})
	}
	return out, nil
}

func malformedAnalytics(err error) error {
	return &APIError{
		Endpoint: "/analytics/query",
		Message:  "analytics response shape is not supported: " + err.Error(),
		kind:     ErrMalformedResponse,
	}
}

func rowTime(row map[string]any) time.Time {
	for _, key := range []string{"date__day", "day", "date", "timestamp", "bucket", "time"} {
		value, ok := row[key]
		if !ok {
			continue
		}
		s, ok := value.(string)
		if !ok {
			continue
		}
		for _, layout := range []string{"2006-01-02", time.RFC3339, time.RFC3339Nano, "2006-01-02 15:04:05"} {
			if t, err := time.Parse(layout, s); err == nil {
				return t.UTC()
			}
		}
	}
	return time.Time{}
}

func rowString(row map[string]any, keys ...string) string {
	for _, key := range keys {
		if v, ok := row[key].(string); ok {
			return v
		}
	}
	return ""
}

func rowFloat(row map[string]any, keys ...string) float64 {
	for _, key := range keys {
		value, ok := row[key]
		if !ok {
			continue
		}
		switch typed := value.(type) {
		case float64:
			return typed
		case int64:
			return float64(typed)
		case json.Number:
			if f, err := typed.Float64(); err == nil {
				return f
			}
		case string:
			if f, err := strconv.ParseFloat(typed, 64); err == nil {
				return f
			}
		}
	}
	return 0
}
