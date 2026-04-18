package adapters

import "context"

type Status string

const (
	StatusDisabled  Status = "disabled"
	StatusReady     Status = "ready"
	StatusMisconfig Status = "misconfigured"
	StatusDegraded  Status = "degraded"
)

type Scope struct {
	AllowedPaths   []string `json:"allowedPaths"`
	ForbiddenPaths []string `json:"forbiddenPaths"`
	SelectedPaths  []string `json:"selectedPaths"`
}

type InvokeRequest struct {
	AdapterID     string         `json:"adapterId"`
	Capability    string         `json:"capability"`
	Scope         Scope          `json:"scope"`
	WriteIntent   bool           `json:"writeIntent"`
	TaskPacketRef *int64         `json:"taskPacketRef,omitempty"`
	TimeoutMs     int            `json:"timeoutMs"`
	DryRun        bool           `json:"dryRun"`
	CorrelationID string         `json:"correlationId"`
	Input         map[string]any `json:"input"`
}

type InvokeResult struct {
	OK          bool           `json:"ok"`
	Message     string         `json:"message"`
	FailureCode string         `json:"failureCode,omitempty"`
	Data        map[string]any `json:"data"`
}

type AdapterInfo struct {
	ID           string         `json:"id"`
	DisplayName  string         `json:"displayName"`
	Status       Status         `json:"status"`
	Detail       string         `json:"detail"`
	Capabilities []string       `json:"capabilities"`
	Config       map[string]any `json:"config"`
}

type Adapter interface {
	Info(ctx context.Context) AdapterInfo
	Invoke(ctx context.Context, req InvokeRequest) (InvokeResult, error)
}
