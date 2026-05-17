package watch

import (
	"testing"
	"time"
)

func TestCloseCancelsPendingDebounce(t *testing.T) {
	m, err := New(nil, nil)
	if err != nil {
		t.Fatalf("new manager: %v", err)
	}

	m.debounce = time.Hour
	m.scheduleReindex()

	if err := m.Close(); err != nil {
		t.Fatalf("close manager: %v", err)
	}

	m.mu.Lock()
	timer := m.debounceAt
	m.mu.Unlock()

	if timer != nil && timer.Stop() {
		t.Fatal("Close left a pending debounce timer active")
	}
	if timer != nil {
		t.Fatal("Close should clear the debounce timer reference")
	}
}
