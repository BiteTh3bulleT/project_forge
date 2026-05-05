package forgekshadow

import (
	"context"
	"sync"
)

type Sink interface {
	Store(context.Context, DiagnosticReport) error
	List() []DiagnosticReport
}

type MemorySink struct {
	mu         sync.Mutex
	maxReports int
	reports    []DiagnosticReport
}

func NewMemorySink(maxReports int) *MemorySink {
	if maxReports <= 0 {
		maxReports = DefaultMaxReports
	}
	return &MemorySink{maxReports: maxReports}
}

func (s *MemorySink) Store(_ context.Context, report DiagnosticReport) error {
	if s == nil {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.reports = append(s.reports, report)
	if len(s.reports) > s.maxReports {
		drop := len(s.reports) - s.maxReports
		copy(s.reports, s.reports[drop:])
		s.reports = s.reports[:s.maxReports]
	}
	return nil
}

func (s *MemorySink) List() []DiagnosticReport {
	if s == nil {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]DiagnosticReport, len(s.reports))
	copy(out, s.reports)
	return out
}
