package modelruntime

import (
	"context"
	"strings"
	"time"
)

func (s *Service) recordAudit(ctx context.Context, record ModelRuntimeAuditRecord) string {
	if s.audit == nil {
		return ""
	}
	auditID, err := s.audit.RecordModelRuntime(ctx, record)
	if err != nil {
		return ""
	}
	return auditID
}

func (s *Service) hasRunningForBackend(backend ModelBackendKind) bool {
	s.schedulerMu.Lock()
	defer s.schedulerMu.Unlock()
	for _, running := range s.running {
		if running.Backend == backend {
			return true
		}
	}
	return false
}

func withOptionalTimeout(ctx context.Context, timeout time.Duration) (context.Context, context.CancelFunc) {
	if timeout <= 0 {
		return ctx, func() {}
	}
	return context.WithTimeout(ctx, timeout)
}

func approxGeneratePromptTokens(req GenerateRequest) int {
	if len(req.Messages) > 0 {
		total := 0
		for _, msg := range req.Messages {
			total += len(strings.Fields(msg.Role))
			total += len(strings.Fields(msg.Content))
			total += len(strings.Fields(msg.Name))
		}
		return total
	}
	return len(strings.Fields(req.Prompt))
}

func positiveOrDefault(value, fallback int) int {
	if value > 0 {
		return value
	}
	return fallback
}

func (s *Service) loadedModelCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.loaded)
}
