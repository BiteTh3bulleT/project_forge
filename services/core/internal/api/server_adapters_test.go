package api

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"forge/projectforge/services/core/internal/adapters"
	"forge/projectforge/services/core/internal/config"
	"forge/projectforge/services/core/internal/gateway"
	"forge/projectforge/services/core/internal/permissions"
	"forge/projectforge/services/core/internal/store"
)

type testLegacyAdapter struct {
	id      string
	result  adapters.InvokeResult
	err     error
	lastReq adapters.InvokeRequest
	calls   int
}

func (a *testLegacyAdapter) Info(context.Context) adapters.AdapterInfo {
	return adapters.AdapterInfo{
		ID:          a.id,
		DisplayName: "Test Legacy Adapter",
		Status:      adapters.StatusReady,
		Capabilities: []string{
			"invoke",
		},
	}
}

func (a *testLegacyAdapter) Invoke(_ context.Context, req adapters.InvokeRequest) (adapters.InvokeResult, error) {
	a.calls++
	a.lastReq = req
	if a.result.Message == "" {
		a.result.Message = "legacy adapter ok"
	}
	return a.result, a.err
}

func TestLegacyAdapterInvokeRouteRemoved(t *testing.T) {
	t.Parallel()

	srv, st := newLegacyAdapterInvokeHarness(t)
	adapter := &testLegacyAdapter{
		id: "legacy-fake",
		result: adapters.InvokeResult{
			OK:      true,
			Message: "legacy adapter ok",
		},
	}
	srv.adapters.Register(adapter)

	const correlation = "corr-legacy-route-removed"
	req := httptest.NewRequest(
		http.MethodPost,
		"/api/adapters/legacy-fake/invoke?correlationId="+correlation+"&traceId=trace-retired&workspaceId=workspace-retired",
		strings.NewReader(`{"adapterId":"legacy-fake","capability":"test.invoke"}`),
	)
	rr := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected not found, got %d body=%s", rr.Code, strings.TrimSpace(rr.Body.String()))
	}
	if adapter.calls != 0 {
		t.Fatalf("expected removed route to never execute adapter, got calls=%d", adapter.calls)
	}

	mustAuditCountForCorrelation(t, st, correlation, 0)

	var invocationCount int
	if err := st.DB.QueryRowContext(context.Background(), `
SELECT COUNT(1) FROM gateway_invocations WHERE correlation_id = ?`,
		correlation,
	).Scan(&invocationCount); err != nil {
		t.Fatalf("query gateway invocations: %v", err)
	}
	if invocationCount != 0 {
		t.Fatalf("expected no gateway invocation for removed route, got %d", invocationCount)
	}
}

func TestGatewayLegacyAdapterInvokeAllowed(t *testing.T) {
	t.Parallel()

	srv, st := newLegacyAdapterInvokeHarness(t)
	adapter := &testLegacyAdapter{
		id: "legacy-fake",
		result: adapters.InvokeResult{
			OK:      true,
			Message: "legacy adapter ok",
			Data:    map[string]any{"mode": "dry-run"},
		},
	}
	srv.adapters.Register(adapter)
	setLegacyInvokePermissionProfile(t, srv, permissions.Profile{
		ID:           "legacy-gw-allowed",
		Name:         "Legacy gateway allowed",
		AllowedTools: []string{"legacy.adapter.invoke"},
		Editable:     true,
		Active:       true,
	})

	const correlation = "corr-legacy-gateway-ok"
	result := invokeLegacyAdapterViaGateway(t, srv, map[string]any{
		"toolId":        "legacy.adapter.invoke",
		"laneId":        "legacy.adapter.invoke",
		"action":        "invoke",
		"source":        "api",
		"initiator":     "api",
		"correlationId": correlation,
		"paths":         []string{},
		"input": map[string]any{
			"adapterId":     "legacy-fake",
			"capability":    "test.invoke",
			"scope":         map[string]any{"allowedPaths": []string{}, "forbiddenPaths": []string{}, "selectedPaths": []string{}},
			"writeIntent":   false,
			"timeoutMs":     5000,
			"dryRun":        false,
			"correlationId": correlation,
			"input":         map[string]any{},
		},
	})

	if result.Status != gateway.StatusOK {
		t.Fatalf("gateway status = %q want %q", result.Status, gateway.StatusOK)
	}
	if adapter.calls != 1 {
		t.Fatalf("expected adapter invoke call count 1, got %d", adapter.calls)
	}
	if adapter.lastReq.CorrelationID != correlation {
		t.Fatalf("adapter correlation = %q want %q", adapter.lastReq.CorrelationID, correlation)
	}

	rawResult, ok := result.Data["result"]
	if !ok {
		t.Fatalf("expected legacy result payload in gateway data")
	}
	typedResult, ok := rawResult.(map[string]any)
	if !ok {
		t.Fatalf("expected gateway result.data.result map, got %T", rawResult)
	}
	if value, ok := typedResult["ok"].(bool); !ok || !value {
		t.Fatalf("expected result ok=true, got %#v", typedResult["ok"])
	}

	mustGatewayInvocationMatch(t, st, correlation, "legacy.adapter.invoke", "legacy.adapter.invoke", "ok")
	mustAuditActionCount(t, st, "tool.executed", correlation)
}

func TestGatewayLegacyAdapterInvokeDeniedByPolicy(t *testing.T) {
	t.Parallel()

	srv, st := newLegacyAdapterInvokeHarness(t)
	adapter := &testLegacyAdapter{id: "legacy-fake"}
	srv.adapters.Register(adapter)
	setLegacyInvokePermissionProfile(t, srv, permissions.Profile{
		ID:           "legacy-gw-denied",
		Name:         "Legacy gateway denied",
		AllowedTools: []string{"legacy.adapter.invoke"},
		ForbiddenPaths: []string{
			"/forbidden",
		},
		Editable: true,
		Active:   true,
	})

	const correlation = "corr-legacy-gateway-denied"
	result := invokeLegacyAdapterViaGateway(t, srv, map[string]any{
		"toolId":        "legacy.adapter.invoke",
		"laneId":        "legacy.adapter.invoke",
		"action":        "invoke",
		"source":        "api",
		"initiator":     "api",
		"correlationId": correlation,
		"paths":         []string{"/forbidden/secret.txt"},
		"input": map[string]any{
			"adapterId":     "legacy-fake",
			"capability":    "test.invoke",
			"scope":         map[string]any{"allowedPaths": []string{}, "forbiddenPaths": []string{}, "selectedPaths": []string{"/forbidden/secret.txt"}},
			"writeIntent":   false,
			"timeoutMs":     5000,
			"dryRun":        false,
			"correlationId": correlation,
			"input":         map[string]any{},
		},
	})

	if result.Status != gateway.StatusDenied {
		t.Fatalf("gateway status = %q want %q", result.Status, gateway.StatusDenied)
	}
	if adapter.calls != 0 {
		t.Fatalf("expected adapter not invoked on denied request, got calls=%d", adapter.calls)
	}

	mustGatewayInvocationMatch(t, st, correlation, "legacy.adapter.invoke", "legacy.adapter.invoke", "denied")
	mustAuditActionCount(t, st, "tool.denied", correlation)
}

func TestGatewayLegacyAdapterInvokeUnknownAdapterErrors(t *testing.T) {
	t.Parallel()

	srv, st := newLegacyAdapterInvokeHarness(t)
	setLegacyInvokePermissionProfile(t, srv, permissions.Profile{
		ID:           "legacy-gw-not-found",
		Name:         "Legacy gateway not found",
		AllowedTools: []string{"legacy.adapter.invoke"},
		Editable:     true,
		Active:       true,
	})

	const correlation = "corr-legacy-gateway-not-found"
	result := invokeLegacyAdapterViaGateway(t, srv, map[string]any{
		"toolId":        "legacy.adapter.invoke",
		"laneId":        "legacy.adapter.invoke",
		"action":        "invoke",
		"source":        "api",
		"initiator":     "api",
		"correlationId": correlation,
		"paths":         []string{},
		"input": map[string]any{
			"adapterId":     "no-such",
			"capability":    "test.invoke",
			"scope":         map[string]any{"allowedPaths": []string{}, "forbiddenPaths": []string{}, "selectedPaths": []string{}},
			"writeIntent":   false,
			"timeoutMs":     5000,
			"dryRun":        false,
			"correlationId": correlation,
			"input":         map[string]any{},
		},
	})

	if result.Status != gateway.StatusError {
		t.Fatalf("gateway status = %q want %q", result.Status, gateway.StatusError)
	}
	if !strings.Contains(result.Message, `unknown adapter "no-such"`) {
		t.Fatalf("expected unknown adapter message, got %q", result.Message)
	}

	mustGatewayInvocationMatch(t, st, correlation, "legacy.adapter.invoke", "legacy.adapter.invoke", "error")
	mustAuditActionCount(t, st, "tool.error", correlation)
}

func invokeLegacyAdapterViaGateway(t *testing.T, srv *Server, body map[string]any) gateway.Result {
	t.Helper()
	raw, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("marshal gateway invoke body: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, "/api/gateway/invoke", bytes.NewReader(raw))
	rr := httptest.NewRecorder()
	srv.handleGatewayInvoke(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("gateway invoke http status = %d body=%s", rr.Code, strings.TrimSpace(rr.Body.String()))
	}

	var payload struct {
		Result gateway.Result `json:"result"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode gateway invoke response: %v body=%s", err, rr.Body.String())
	}
	return payload.Result
}

func setLegacyInvokePermissionProfile(t *testing.T, srv *Server, profile permissions.Profile) {
	t.Helper()
	if _, err := srv.permissions.Save(context.Background(), profile); err != nil {
		t.Fatalf("save permission profile %q: %v", profile.ID, err)
	}
}

func newLegacyAdapterInvokeHarness(t *testing.T) (*Server, *store.Store) {
	t.Helper()
	dataDir := t.TempDir()
	workspaceDir := t.TempDir()
	st, err := store.Open(dataDir)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })

	srv := NewServer(st, config.Config{
		DataDir:      dataDir,
		WorkspaceDir: workspaceDir,
	})
	t.Cleanup(func() { srv.ShutdownWatch() })
	return srv, st
}

func mustAuditRecordByActionAndCorrelation(t *testing.T, st *store.Store, action, correlation string) (string, string, string) {
	t.Helper()
	var corr, outcome, payload string
	if err := st.DB.QueryRowContext(context.Background(), `
SELECT correlation_id, outcome, payload_json
FROM audit_records
WHERE action = ? AND correlation_id = ?
ORDER BY id DESC LIMIT 1`,
		action, correlation,
	).Scan(&corr, &outcome, &payload); err != nil {
		t.Fatalf("query audit record action=%s correlation=%s: %v", action, correlation, err)
	}
	return corr, outcome, payload
}

func mustAuditActionCount(t *testing.T, st *store.Store, action, correlation string) {
	t.Helper()
	var count int
	if err := st.DB.QueryRowContext(context.Background(), `
SELECT COUNT(1) FROM audit_records
WHERE action = ? AND correlation_id = ?`,
		action, correlation,
	).Scan(&count); err != nil {
		t.Fatalf("query audit action count action=%s correlation=%s: %v", action, correlation, err)
	}
	if count == 0 {
		t.Fatalf("expected at least one audit record for action=%s correlation=%s", action, correlation)
	}
}

func mustAuditCountForCorrelation(t *testing.T, st *store.Store, correlation string, want int) {
	t.Helper()
	var count int
	if err := st.DB.QueryRowContext(context.Background(), `
SELECT COUNT(1) FROM audit_records
WHERE correlation_id = ?`,
		correlation,
	).Scan(&count); err != nil {
		t.Fatalf("query audit count correlation=%s: %v", correlation, err)
	}
	if count != want {
		t.Fatalf("expected audit count=%d for correlation=%s, got %d", want, correlation, count)
	}
}

func mustGatewayInvocationMatch(t *testing.T, st *store.Store, correlation, toolID, laneID, status string) {
	t.Helper()
	var gotTool, gotStatus string
	var gotLane sql.NullString
	if err := st.DB.QueryRowContext(context.Background(), `
SELECT tool_id, lane_id, status
FROM gateway_invocations
WHERE correlation_id = ?
ORDER BY id DESC LIMIT 1`,
		correlation,
	).Scan(&gotTool, &gotLane, &gotStatus); err != nil {
		t.Fatalf("query gateway invocation correlation=%s: %v", correlation, err)
	}
	if gotTool != toolID {
		t.Fatalf("gateway tool = %q want %q", gotTool, toolID)
	}
	if !gotLane.Valid || gotLane.String != laneID {
		t.Fatalf("gateway lane = %q want %q", gotLane.String, laneID)
	}
	if gotStatus != status {
		t.Fatalf("gateway status = %q want %q", gotStatus, status)
	}
}
