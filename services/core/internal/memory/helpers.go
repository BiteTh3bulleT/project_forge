package memory

import (
	"database/sql"
	"encoding/json"
	"math"
	"strings"
)

func boolToInt(v bool) int {
	if v {
		return 1
	}
	return 0
}

func nullInt64(v *int64) any {
	if v == nil {
		return nil
	}
	return *v
}

func asRawJSONArray(raw string) json.RawMessage {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return json.RawMessage("[]")
	}
	return json.RawMessage(trimmed)
}

func asRawJSONObject(raw string) json.RawMessage {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return json.RawMessage("{}")
	}
	return json.RawMessage(trimmed)
}

func nonNilStrings(in []string) []string {
	if in == nil {
		return []string{}
	}
	out := make([]string, 0, len(in))
	for _, item := range in {
		v := strings.TrimSpace(item)
		if v == "" {
			continue
		}
		out = append(out, v)
	}
	return out
}

func scanNullableInt64(v sql.NullInt64) *int64 {
	if !v.Valid {
		return nil
	}
	out := v.Int64
	return &out
}

func scanNullableString(v sql.NullString) *string {
	if !v.Valid {
		return nil
	}
	out := v.String
	return &out
}

func summarizeRawContent(raw string) string {
	v := strings.TrimSpace(raw)
	if v == "" {
		return ""
	}
	if len(v) <= 220 {
		return v
	}
	return v[:220] + "..."
}

func parseRawStringSlice(raw json.RawMessage) []string {
	trimmed := strings.TrimSpace(string(raw))
	if trimmed == "" {
		return nil
	}
	var out []string
	if err := json.Unmarshal([]byte(trimmed), &out); err == nil {
		return nonNilStrings(out)
	}
	return nonNilStrings(strings.FieldsFunc(trimmed, func(r rune) bool {
		return r == ',' || r == '\n' || r == '\t' || r == ';'
	}))
}

func clamp(v, low, high float64) float64 {
	if v < low {
		return low
	}
	if v > high {
		return high
	}
	return v
}

func round(v float64) float64 {
	return math.Round(v*10000) / 10000
}
