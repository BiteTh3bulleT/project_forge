package forgekshadow

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
	"unicode"
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
	routeClass := normalizeProvidedRouteClass(input.RouteClass, input.Path, routePattern)
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
		trimmedKey := strings.TrimSpace(key)
		if isReservedRouteEnvelopeMetadataKey(trimmedKey) {
			continue
		}
		if _, exists := metadata[trimmedKey]; exists {
			continue
		}
		metadata[trimmedKey] = value
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
	value = strings.TrimRight(value, "/")
	if value == "" {
		value = "/"
	}
	if containsUnsafeTerm(value) {
		return ""
	}
	if len(value) > maxMetadataStringLength {
		return ""
	}
	if !strings.HasPrefix(value, "/") {
		return ""
	}
	if hasRawDynamicRouteSegment(value) {
		return ""
	}
	return value
}

func normalizeProvidedRouteClass(routeClass, pathValue, routePattern string) string {
	switch strings.TrimSpace(routeClass) {
	case RouteClassHealth:
		return RouteClassHealth
	case RouteClassAPI:
		return RouteClassAPI
	case RouteClassForge:
		return RouteClassForge
	case RouteClassOpenAICompat:
		return RouteClassOpenAICompat
	case RouteClassStaticOrUnknown:
		return RouteClassStaticOrUnknown
	case RouteClassOther:
		return RouteClassOther
	default:
		return NormalizeRouteClass(pathValue, routePattern)
	}
}

func isReservedRouteEnvelopeMetadataKey(key string) bool {
	switch normalizeMetadataToken(key) {
	case "observation_type", "method", "route_class", "duration_ms", "path", "route_pattern",
		"request_id", "workspace_id", "correlation_id", "status_code", "warning_count":
		return true
	default:
		return false
	}
}

func hasRawDynamicRouteSegment(routePattern string) bool {
	for _, segment := range strings.Split(routePattern, "/") {
		segment = strings.TrimSpace(segment)
		if segment == "" || strings.HasPrefix(segment, "{") && strings.HasSuffix(segment, "}") {
			continue
		}
		if isNumericSegment(segment) || looksLikeUUID(segment) {
			return true
		}
	}
	return false
}

func isNumericSegment(value string) bool {
	if value == "" {
		return false
	}
	for _, r := range value {
		if !unicode.IsDigit(r) {
			return false
		}
	}
	return true
}

func looksLikeUUID(value string) bool {
	if len(value) != 36 {
		return false
	}
	parts := strings.Split(value, "-")
	if len(parts) != 5 {
		return false
	}
	widths := []int{8, 4, 4, 4, 12}
	for i, part := range parts {
		if len(part) != widths[i] {
			return false
		}
		if _, err := strconv.ParseUint(part, 16, 64); err != nil {
			return false
		}
	}
	return true
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
