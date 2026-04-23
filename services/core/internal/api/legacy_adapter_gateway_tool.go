package api

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	"forge/projectforge/services/core/internal/adapters"
	"forge/projectforge/services/core/internal/gateway"
)

type legacyAdapterGatewayTool struct {
	registry *adapters.Registry
}

func newLegacyAdapterGatewayTool(registry *adapters.Registry) *legacyAdapterGatewayTool {
	return &legacyAdapterGatewayTool{registry: registry}
}

func (t *legacyAdapterGatewayTool) ID() string             { return "legacy.adapter.invoke" }
func (t *legacyAdapterGatewayTool) Domain() string         { return "gateway" }
func (t *legacyAdapterGatewayTool) Action() string         { return "invoke" }
func (t *legacyAdapterGatewayTool) RiskClass() string      { return "low" }
func (t *legacyAdapterGatewayTool) ExecutionLevel() string { return "L0" }
func (t *legacyAdapterGatewayTool) Executes() bool         { return false }
func (t *legacyAdapterGatewayTool) UsesNetwork() bool      { return false }
func (t *legacyAdapterGatewayTool) WriteIntent() bool      { return false }
func (t *legacyAdapterGatewayTool) Description() string {
	return "Legacy adapter invoke compatibility shim routed through gateway policy and audit controls."
}

func (t *legacyAdapterGatewayTool) Execute(ctx context.Context, req gateway.Request) (gateway.Result, error) {
	if t.registry == nil {
		return gateway.Result{}, errors.New("legacy adapter registry unavailable")
	}
	payload, err := decodeLegacyAdapterInvokeInput(req.Input)
	if err != nil {
		return gateway.Result{}, err
	}
	adapterID := strings.TrimSpace(payload.AdapterID)
	if adapterID == "" {
		return gateway.Result{}, errors.New("adapterId is required")
	}
	payload.AdapterID = adapterID
	if strings.TrimSpace(payload.CorrelationID) == "" {
		payload.CorrelationID = strings.TrimSpace(req.CorrelationID)
	}
	adapter, err := t.registry.Get(adapterID)
	if err != nil {
		return gateway.Result{}, err
	}
	res, err := adapter.Invoke(ctx, payload)
	if err != nil {
		return gateway.Result{}, err
	}
	return gateway.Result{
		Data: map[string]any{
			"legacyAdapterInvocation": true,
			"result":                  res,
		},
		Message: nonEmptyTrimmed(res.Message, "legacy adapter invoke executed"),
	}, nil
}

func decodeLegacyAdapterInvokeInput(input map[string]any) (adapters.InvokeRequest, error) {
	raw, err := json.Marshal(input)
	if err != nil {
		return adapters.InvokeRequest{}, err
	}
	var out adapters.InvokeRequest
	if err := json.Unmarshal(raw, &out); err != nil {
		return adapters.InvokeRequest{}, err
	}
	return out, nil
}

func nonEmptyTrimmed(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}
