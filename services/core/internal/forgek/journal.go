package forgek

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"sync"
)

type Journal struct {
	mu     sync.RWMutex
	events []JournalEvent
	byID   map[string]JournalEvent
	ids    IDProvider
}

func NewJournal(ids IDProvider) *Journal {
	return &Journal{
		events: make([]JournalEvent, 0),
		byID:   make(map[string]JournalEvent),
		ids:    ids,
	}
}

func (j *Journal) Append(event JournalEvent) (JournalEvent, error) {
	j.mu.Lock()
	defer j.mu.Unlock()

	if event.EventID == "" {
		event.EventID = j.ids.NextID("event")
	}
	if _, exists := j.byID[event.EventID]; exists {
		return JournalEvent{}, ErrInvalidInput
	}
	if len(j.events) > 0 {
		prior := j.events[len(j.events)-1]
		event.PriorHash = prior.EventHash
		event.PriorEventRefs = append(event.PriorEventRefs, prior.EventID)
	}
	event.ObjectRefs = append([]string(nil), event.ObjectRefs...)
	event.CapabilityRefs = append([]string(nil), event.CapabilityRefs...)
	event.PriorEventRefs = append([]string(nil), event.PriorEventRefs...)
	event.EventHash = hashJournalEvent(event)

	stored := cloneJournalEvent(event)
	j.events = append(j.events, stored)
	j.byID[stored.EventID] = stored
	return cloneJournalEvent(stored), nil
}

func (j *Journal) ListEvents() []JournalEvent {
	j.mu.RLock()
	defer j.mu.RUnlock()

	out := make([]JournalEvent, len(j.events))
	for i, event := range j.events {
		out[i] = cloneJournalEvent(event)
	}
	return out
}

func (j *Journal) GetEvent(eventID string) (JournalEvent, bool) {
	j.mu.RLock()
	defer j.mu.RUnlock()

	event, ok := j.byID[eventID]
	if !ok {
		return JournalEvent{}, false
	}
	return cloneJournalEvent(event), true
}

func (j *Journal) len() int {
	j.mu.RLock()
	defer j.mu.RUnlock()

	return len(j.events)
}

func hashJournalEvent(event JournalEvent) string {
	copyEvent := event
	copyEvent.EventHash = ""
	encoded, _ := json.Marshal(copyEvent)
	sum := sha256.Sum256(encoded)
	return hex.EncodeToString(sum[:])
}

func hashValue(value any) string {
	encoded, _ := json.Marshal(value)
	sum := sha256.Sum256(encoded)
	return hex.EncodeToString(sum[:])
}

func cloneJournalEvent(event JournalEvent) JournalEvent {
	event.ObjectRefs = append([]string(nil), event.ObjectRefs...)
	event.CapabilityRefs = append([]string(nil), event.CapabilityRefs...)
	event.PriorEventRefs = append([]string(nil), event.PriorEventRefs...)
	return event
}
