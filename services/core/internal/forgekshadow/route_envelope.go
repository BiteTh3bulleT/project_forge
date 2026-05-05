package forgekshadow

import (
	"fmt"
	"net/http"
	"strings"
	"time"
)

const (
	RouteClassHealth          = "health"
	RouteClassAPI             = "api"
	RouteClassForge           = "forge"
	RouteClassOpenAICompat    = "openai_compat"
	RouteClassStaticOrUnknown = "static_or_unknown"
	RouteClassOther           = "other"
)

const (
	maxRouteEnvelopeWarnings      = 8
	maxRouteEnvelopeWarningLength = 160
)

func NormalizeRouteClass(pathValue, routePattern string) string {
	pathValue = strings.TrimSpace(pathValue)
	routePattern = strings.TrimSpace(routePattern)
	candidate := routePattern
	if candidate == "" {
		candidate = pathValue
	}
	candidate = strings.ToLower(strings.TrimSpace(strings.Split(candidate, "?")[0]))
	switch {
	case candidate == "/health":
		return RouteClassHealth
	case strings.HasPrefix(candidate, "/api/"):
		return RouteClassAPI
	case candidate == "/api":
		return RouteClassAPI
	case strings.HasPrefix(candidate, "/forge/"):
		return RouteClassForge
	case candidate == "/forge":
		return RouteClassForge
	case strings.HasPrefix(candidate, "/v1/"):
		return RouteClassOpenAICompat
	case candidate == "/v1":
		return RouteClassOpenAICompat
	case candidate == "" || candidate == "/" || strings.Contains(candidate, "."):
		return RouteClassStaticOrUnknown
	default:
		return RouteClassOther
	}
}

func normalizeRouteEnvelopeInput(input RouteEnvelopeInput, now time.Time, observationID string) (RouteEnvelopeObservation, map[string]any, error) {
	method := strings.ToUpper(strings.TrimSpace(input.Method))
	if method == "" {
		method = http.MethodGet
	}
	routePattern := safeRoutePattern(input.RoutePattern)
	routeClass := strings.TrimSpace(input.RouteClass)
	if routeClass == "" {
		routeClass = NormalizeRouteClass(input.Path, routePattern)
	}
	pathValue := routePattern
	if pathValue == "" && routeClass == RouteClassHealth {
		pathValue = "/health"
	}
	durationMS := input.Duration.Milliseconds()
	if durationMS < 0 {
		durationMS = 0
	}
	warnings, err := safeRouteEnvelopeWarnings(input.Warnings)
	if err != nil {
		return RouteEnvelopeObservation{}, nil, err
	}
	metadata := map[string]any{
		"observation_type": "route_envelope",
		"method":           method,
		"route_class":      routeClass,
		"duration_ms":      durationMS,
	}
	if pathValue != "" {
		metadata["path"] = pathValue
	}
	if routePattern != "" {
		metadata["route_pattern"] = routePattern
	}
	if requestID := strings.TrimSpace(input.RequestID); requestID != "" {
		metadata["request_id"] = requestID
	}
	if workspaceID := strings.TrimSpace(input.WorkspaceID); workspaceID != "" {
		metadata["workspace_id"] = workspaceID
	}
	if correlationID := strings.TrimSpace(input.CorrelationID); correlationID != "" {
		metadata["correlation_id"] = correlationID
	}
	if input.StatusCode > 0 {
		metadata["status_code"] = input.StatusCode
	}
	if len(warnings) > 0 {
		metadata["warning_count"] = len(warnings)
	}
	for key, value := range input.Metadata {
		if _, exists := metadata[key]; exists {
			continue
		}
		metadata[key] = value
	}
	safe, err := safeMetadata(metadata)
	if err != nil {
		return RouteEnvelopeObservation{}, nil, err
	}
	return RouteEnvelopeObservation{
		ObservationID: observationID,
		ObservedAt:    now,
		Method:        method,
		Path:          pathValue,
		RoutePattern:  routePattern,
		RouteClass:    routeClass,
		StatusCode:    input.StatusCode,
		DurationMS:    durationMS,
		RequestID:     strings.TrimSpace(input.RequestID),
		WorkspaceID:   strings.TrimSpace(input.WorkspaceID),
		CorrelationID: strings.TrimSpace(input.CorrelationID),
		Warnings:      warnings,
		Metadata:      safe,
	}, safe, nil
}

func safeRoutePattern(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	value = strings.Split(value, "?")[0]
	if containsUnsafeTerm(value) {
		return ""
	}
	if len(value) > maxMetadataStringLength {
		return ""
	}
	if !strings.HasPrefix(value, "/") {
		return ""
	}
	return value
}

func safeRouteEnvelopeWarnings(in []string) ([]string, error) {
	if len(in) == 0 {
		return nil, nil
	}
	out := make([]string, 0, len(in))
	for _, warning := range in {
		text := strings.TrimSpace(warning)
		if text == "" {
			continue
		}
		if containsUnsafeTerm(text) || len(text) > maxRouteEnvelopeWarningLength {
			return nil, fmt.Errorf("%w: route envelope warning", ErrUnsafeMetadata)
		}
		out = append(out, text)
		if len(out) == maxRouteEnvelopeWarnings {
			break
		}
	}
	if len(out) == 0 {
		return nil, nil
	}
	return out, nil
}
