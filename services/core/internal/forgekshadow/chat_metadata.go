package forgekshadow

import (
	"fmt"
	"strings"
	"time"
)

const (
	chatMetadataObservationType = "chat_metadata"
	chatMetadataRouteClass      = "chat"

	ChatOperationMessagePost     = "message_post"
	ChatOperationAssistantStream = "assistant_stream"
	ChatOperationThreadCreate    = "thread_create"
	ChatOperationThreadGet       = "thread_get"
	ChatOperationThreadList      = "thread_list"
	ChatOperationThreadPatch     = "thread_patch"
	ChatOperationThreadDelete    = "thread_delete"
	ChatOperationAttachment      = "attachment_upload"
)

const (
	maxChatMetadataWarnings      = 8
	maxChatMetadataWarningLength = 160
)

func normalizeChatMetadataInput(input ChatMetadataInput, now time.Time, observationID string) (ChatMetadataObservation, map[string]any, error) {
	operationKind := normalizeChatOperation(input.OperationKind)
	durationMS := input.Duration.Milliseconds()
	if durationMS < 0 {
		durationMS = 0
	}
	warnings, err := safeChatMetadataWarnings(input.Warnings)
	if err != nil {
		return ChatMetadataObservation{}, nil, err
	}
	threadID, err := safeChatMetadataRef(input.ThreadID, "thread_id")
	if err != nil {
		return ChatMetadataObservation{}, nil, err
	}
	messageID, err := safeChatMetadataRef(input.MessageID, "message_id")
	if err != nil {
		return ChatMetadataObservation{}, nil, err
	}
	modelID, err := safeChatMetadataRef(input.ModelID, "model_id")
	if err != nil {
		return ChatMetadataObservation{}, nil, err
	}
	providerID, err := safeChatMetadataRef(input.ProviderID, "provider_id")
	if err != nil {
		return ChatMetadataObservation{}, nil, err
	}
	roleClass := normalizeChatRole(input.RoleClass)
	streamClass := normalizeChatStreamClass(input.StreamClass)
	messageCount := input.MessageCount
	if messageCount < 0 {
		messageCount = 0
	}

	metadata := map[string]any{
		"observation_type": chatMetadataObservationType,
		"route_class":      chatMetadataRouteClass,
		"operation_kind":   operationKind,
		"duration_ms":      durationMS,
	}
	if threadID != "" {
		metadata["thread_id"] = threadID
	}
	if messageID != "" {
		metadata["message_id"] = messageID
	}
	if roleClass != "" {
		metadata["role_class"] = roleClass
	}
	if streamClass != "" {
		metadata["stream_class"] = streamClass
	}
	if modelID != "" {
		metadata["model_id"] = modelID
	}
	if providerID != "" {
		metadata["provider_id"] = providerID
	}
	if messageCount > 0 {
		metadata["message_count"] = messageCount
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
	if len(warnings) > 0 {
		metadata["warning_count"] = len(warnings)
	}
	for key, value := range input.Metadata {
		trimmedKey := strings.TrimSpace(key)
		if isReservedChatMetadataKey(trimmedKey) {
			continue
		}
		if _, exists := metadata[trimmedKey]; exists {
			continue
		}
		metadata[trimmedKey] = value
	}
	safe, err := safeMetadata(metadata)
	if err != nil {
		return ChatMetadataObservation{}, nil, err
	}
	return ChatMetadataObservation{
		ObservationID: observationID,
		ObservedAt:    now,
		OperationKind: operationKind,
		ThreadID:      threadID,
		MessageID:     messageID,
		RoleClass:     roleClass,
		StreamClass:   streamClass,
		ModelID:       modelID,
		ProviderID:    providerID,
		MessageCount:  messageCount,
		DurationMS:    durationMS,
		RequestID:     strings.TrimSpace(input.RequestID),
		WorkspaceID:   strings.TrimSpace(input.WorkspaceID),
		CorrelationID: strings.TrimSpace(input.CorrelationID),
		Warnings:      warnings,
		Metadata:      safe,
	}, safe, nil
}

func normalizeChatOperation(value string) string {
	switch strings.TrimSpace(value) {
	case ChatOperationMessagePost:
		return ChatOperationMessagePost
	case ChatOperationAssistantStream:
		return ChatOperationAssistantStream
	case ChatOperationThreadCreate:
		return ChatOperationThreadCreate
	case ChatOperationThreadGet:
		return ChatOperationThreadGet
	case ChatOperationThreadList:
		return ChatOperationThreadList
	case ChatOperationThreadPatch:
		return ChatOperationThreadPatch
	case ChatOperationThreadDelete:
		return ChatOperationThreadDelete
	case ChatOperationAttachment:
		return ChatOperationAttachment
	default:
		return ChatOperationMessagePost
	}
}

func normalizeChatRole(value string) string {
	switch strings.TrimSpace(value) {
	case "user", "assistant", "system", "tool":
		return strings.TrimSpace(value)
	default:
		return ""
	}
}

func normalizeChatStreamClass(value string) string {
	switch strings.TrimSpace(value) {
	case "none", "sync", "async", "stream":
		return strings.TrimSpace(value)
	default:
		return ""
	}
}

func safeChatMetadataRef(value, field string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", nil
	}
	if containsUnsafeTerm(value) || len(value) > maxMetadataStringLength {
		return "", fmt.Errorf("%w: %s", ErrUnsafeMetadata, field)
	}
	return value, nil
}

func safeChatMetadataWarnings(in []string) ([]string, error) {
	if len(in) == 0 {
		return nil, nil
	}
	out := make([]string, 0, len(in))
	for _, warning := range in {
		text := strings.TrimSpace(warning)
		if text == "" {
			continue
		}
		if containsUnsafeTerm(text) || len(text) > maxChatMetadataWarningLength {
			return nil, fmt.Errorf("%w: chat metadata warning", ErrUnsafeMetadata)
		}
		out = append(out, text)
		if len(out) == maxChatMetadataWarnings {
			break
		}
	}
	if len(out) == 0 {
		return nil, nil
	}
	return out, nil
}

func isReservedChatMetadataKey(key string) bool {
	switch normalizeMetadataToken(key) {
	case "observation_type", "route_class", "operation_kind", "duration_ms", "thread_id",
		"message_id", "role_class", "stream_class", "model_id", "provider_id",
		"message_count", "request_id", "workspace_id", "correlation_id", "warning_count":
		return true
	default:
		return false
	}
}
