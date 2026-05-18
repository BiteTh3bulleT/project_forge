package jobs

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"forge/projectforge/services/core/internal/adapters"
	"forge/projectforge/services/core/internal/packets"
)

func toAdapterScope(s ScopeInput) adapters.Scope {
	return adapters.Scope{AllowedPaths: s.AllowedPaths, ForbiddenPaths: s.ForbiddenPaths, SelectedPaths: s.SelectedPaths}
}

func adapterContent(res adapters.InvokeResult, tpl Template) (content string, title string) {
	title = fmt.Sprintf("%s output", tpl.Name)
	if handoff, ok := res.Data["handoffMarkdown"].(string); ok && strings.TrimSpace(handoff) != "" {
		return handoff, title
	}
	if response, ok := res.Data["response"].(string); ok && strings.TrimSpace(response) != "" {
		return "# Adapter Response\n\n" + response + "\n", title
	}
	b, _ := json.MarshalIndent(res.Data, "", "  ")
	return "```json\n" + string(b) + "\n```\n", title
}

func packetID(p *packets.Packet) *int64 {
	if p == nil {
		return nil
	}
	v := p.ID
	return &v
}

func scanJob(scanner interface{ Scan(dest ...any) error }) (*Job, error) {
	var j Job
	var status, risk, approval string
	var writeIntent, cancelReq int
	var queued, started, completed sql.NullInt64
	var packetID sql.NullInt64
	var resultSummary, failureInfo, lastError, lastFailureCode sql.NullString
	var metadata string
	if err := scanner.Scan(
		&j.ID,
		&j.Title,
		&j.RequestedAction,
		&j.TargetAdapter,
		&status,
		&j.CreatedAtMs,
		&j.UpdatedAtMs,
		&queued,
		&started,
		&completed,
		&j.InitiatingSource,
		&j.ExecutionBoundary,
		&risk,
		&approval,
		&writeIntent,
		&cancelReq,
		&packetID,
		&resultSummary,
		&failureInfo,
		&lastFailureCode,
		&lastError,
		&metadata,
	); err != nil {
		return nil, err
	}
	j.Status = Status(status)
	j.RiskClass = RiskClass(risk)
	j.ApprovalStatus = ApprovalStatus(approval)
	j.WriteIntent = writeIntent == 1
	j.CancelRequested = cancelReq == 1
	if queued.Valid {
		v := queued.Int64
		j.QueuedAtMs = &v
	}
	if started.Valid {
		v := started.Int64
		j.StartedAtMs = &v
	}
	if completed.Valid {
		v := completed.Int64
		j.CompletedAtMs = &v
	}
	if packetID.Valid {
		v := packetID.Int64
		j.TaskPacketID = &v
	}
	if resultSummary.Valid {
		v := resultSummary.String
		j.ResultSummary = &v
	}
	if failureInfo.Valid {
		v := failureInfo.String
		j.FailureInfo = &v
	}
	if lastError.Valid {
		v := lastError.String
		j.LastError = &v
	}
	if lastFailureCode.Valid {
		fc := FailureCode(lastFailureCode.String)
		j.LastFailureCode = &fc
	}
	j.Metadata = json.RawMessage(metadata)
	return &j, nil
}

func isTerminal(s Status) bool {
	return s == StatusSucceeded || s == StatusFailed || s == StatusCancelled
}

func boolToInt(v bool) int {
	if v {
		return 1
	}
	return 0
}

func newJobID() string {
	var b [6]byte
	_, _ = rand.Read(b[:])
	return fmt.Sprintf("job_%d_%s", time.Now().UnixMilli(), hex.EncodeToString(b[:]))
}

func nonEmpty(v, def string) string {
	if strings.TrimSpace(v) == "" {
		return def
	}
	return strings.TrimSpace(v)
}

func nonNilMap(v map[string]any) map[string]any {
	if v == nil {
		return map[string]any{}
	}
	return v
}

func readString(m map[string]any, key string) string {
	if m == nil {
		return ""
	}
	v, ok := m[key]
	if !ok {
		return ""
	}
	s, ok := v.(string)
	if !ok {
		return ""
	}
	return s
}

func readBool(m map[string]any, key string, def bool) bool {
	if m == nil {
		return def
	}
	v, ok := m[key]
	if !ok {
		return def
	}
	b, ok := v.(bool)
	if !ok {
		return def
	}
	return b
}

func readMap(m map[string]any, key string) map[string]any {
	if m == nil {
		return map[string]any{}
	}
	v, ok := m[key]
	if !ok || v == nil {
		return map[string]any{}
	}
	switch t := v.(type) {
	case map[string]any:
		return t
	case map[string]string:
		out := make(map[string]any, len(t))
		for k, v := range t {
			out[k] = v
		}
		return out
	default:
		return map[string]any{}
	}
}

func readStringSlice(m map[string]any, key string) []string {
	if m == nil {
		return nil
	}
	v, ok := m[key]
	if !ok || v == nil {
		return nil
	}
	switch t := v.(type) {
	case []string:
		out := make([]string, 0, len(t))
		for _, s := range t {
			s = strings.TrimSpace(s)
			if s != "" {
				out = append(out, s)
			}
		}
		return out
	case []any:
		out := make([]string, 0, len(t))
		for _, item := range t {
			if s, ok := item.(string); ok {
				s = strings.TrimSpace(s)
				if s != "" {
					out = append(out, s)
				}
			}
		}
		return out
	default:
		return nil
	}
}

func readInt(m map[string]any, key string, def int64) int64 {
	if m == nil {
		return def
	}
	v, ok := m[key]
	if !ok {
		return def
	}
	switch t := v.(type) {
	case float64:
		return int64(t)
	case int64:
		return t
	case int:
		return int64(t)
	case json.Number:
		n, err := t.Int64()
		if err == nil {
			return n
		}
	}
	return def
}

func readFloat(m map[string]any, key string, def float64) float64 {
	if m == nil {
		return def
	}
	v, ok := m[key]
	if !ok {
		return def
	}
	switch t := v.(type) {
	case float64:
		return t
	case float32:
		return float64(t)
	case int64:
		return float64(t)
	case int:
		return float64(t)
	case json.Number:
		f, err := t.Float64()
		if err == nil {
			return f
		}
	}
	return def
}

func readOptionalInt(m map[string]any, key string) *int64 {
	if m == nil {
		return nil
	}
	v, ok := m[key]
	if !ok {
		return nil
	}
	var out int64
	switch t := v.(type) {
	case float64:
		out = int64(t)
	case float32:
		out = int64(t)
	case int:
		out = int64(t)
	case int64:
		out = t
	case json.Number:
		i, err := t.Int64()
		if err != nil {
			return nil
		}
		out = i
	default:
		return nil
	}
	if out <= 0 {
		return nil
	}
	return &out
}

func derefInt64(v *int64) int64 {
	if v == nil {
		return 0
	}
	return *v
}
