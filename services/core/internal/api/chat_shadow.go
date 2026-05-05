package api

import (
	"context"
	"strconv"

	"github.com/go-chi/chi/v5/middleware"

	"forge/projectforge/services/core/internal/forgekshadow"
)

func (s *Server) observeChatMessagePostMetadata(ctx context.Context, threadID, messageID int64, modelID string, requestAssistant, assistantPending, streamResponse, asyncAssistant bool) {
	if s == nil || s.forgeKShadow == nil {
		return
	}
	streamClass := "none"
	switch {
	case streamResponse:
		streamClass = "stream"
	case asyncAssistant:
		streamClass = "async"
	case requestAssistant:
		streamClass = "sync"
	}
	s.forgeKShadow.ObserveChatMetadataBestEffort(ctx, forgekshadow.ChatMetadataInput{
		WorkspaceID:   s.cfg.WorkspaceDir,
		RequestID:     middleware.GetReqID(ctx),
		OperationKind: forgekshadow.ChatOperationMessagePost,
		ThreadID:      strconv.FormatInt(threadID, 10),
		MessageID:     strconv.FormatInt(messageID, 10),
		RoleClass:     "user",
		StreamClass:   streamClass,
		ModelID:       modelID,
		MessageCount:  1,
		Metadata: map[string]any{
			"touchpoint":        "chat_message_post",
			"request_assistant": requestAssistant,
			"assistant_pending": assistantPending,
			"stream_response":   streamResponse,
			"async_assistant":   asyncAssistant,
		},
	})
}

func boolMapValue(values map[string]any, key string) bool {
	value, _ := values[key].(bool)
	return value
}
