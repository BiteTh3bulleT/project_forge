package gateway

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"
)

type secretGetTool struct{ db *sql.DB }

type timeNowTool struct{}

func (t *timeNowTool) ID() string             { return "time.now" }
func (t *timeNowTool) Domain() string         { return "time" }
func (t *timeNowTool) Action() string         { return "get_system_time" }
func (t *timeNowTool) RiskClass() string      { return "read_only" }
func (t *timeNowTool) ExecutionLevel() string { return "L0" }
func (t *timeNowTool) Executes() bool         { return false }
func (t *timeNowTool) UsesNetwork() bool      { return false }
func (t *timeNowTool) WriteIntent() bool      { return false }
func (t *timeNowTool) Description() string {
	return "Read current system clock in UTC and local timezone"
}
func (t *timeNowTool) Execute(_ context.Context, _ Request) (Result, error) {
	now := time.Now()
	_, offset := now.Zone()
	return Result{
		Data: map[string]any{
			"unixMs":      now.UnixMilli(),
			"iso8601":     now.Format(time.RFC3339Nano),
			"utcIso8601":  now.UTC().Format(time.RFC3339Nano),
			"zoneOffsetS": offset,
		},
		Message: "system time captured",
	}, nil
}

func (t *secretGetTool) ID() string             { return "secret.get" }
func (t *secretGetTool) Domain() string         { return "secrets" }
func (t *secretGetTool) Action() string         { return "get_secret_ref" }
func (t *secretGetTool) RiskClass() string      { return "privileged" }
func (t *secretGetTool) ExecutionLevel() string { return "L3" }
func (t *secretGetTool) Executes() bool         { return false }
func (t *secretGetTool) UsesNetwork() bool      { return false }
func (t *secretGetTool) WriteIntent() bool      { return false }
func (t *secretGetTool) Description() string {
	return "Resolve secret logical name and return masked metadata only"
}
func (t *secretGetTool) Execute(ctx context.Context, req Request) (Result, error) {
	name, _ := req.Input["name"].(string)
	name = strings.TrimSpace(name)
	if name == "" {
		return Result{}, errors.New("secret.get requires input.name")
	}
	var raw string
	err := t.db.QueryRowContext(ctx, `SELECT value FROM settings WHERE key = ?`, "secret."+name).Scan(&raw)
	if err != nil {
		return Result{}, fmt.Errorf("secret %q not found", name)
	}
	return Result{
		Data: map[string]any{
			"name":     name,
			"exists":   true,
			"length":   len(strings.TrimSpace(raw)),
			"revealed": false,
			"masked":   maskSecret(raw),
		},
		Message: "secret metadata resolved",
	}, nil
}
