package shadowharness

import (
	"encoding/json"
	"reflect"
	"testing"
	"time"
)

func TestShadowObservationValidatesAndSerializesDeterministically(t *testing.T) {
	observedAt := time.Unix(100, 0).UTC()
	observation, err := NewShadowObservation(ShadowObservation{
		ObservationID:  "obs-1",
		WorkspaceID:    "workspace-a",
		RequestID:      "request-a",
		ObservedAt:     observedAt,
		LivePath:       "api routes",
		RequestSummary: "live request observed for diagnostics",
		InputRefs:      []string{"input-b", "input-a", "input-a"},
		EvidenceRefs:   []string{"evidence-b", "evidence-a"},
		RetrievalRefs:  []string{"retrieval-a"},
		ContextRefs:    []string{"context-a"},
		ConsensusRefs:  []string{"consensus-a"},
		RuntimeRefs:    []string{"runtime-a"},
		KVRefs:         []string{"kv-a"},
		RiskFlags:      []string{"medium", "medium", "low"},
		Metadata: map[string]any{
			"source": "shadow-design",
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(observation.InputRefs, []string{"input-a", "input-b"}) {
		t.Fatalf("input refs not normalized: %#v", observation.InputRefs)
	}
	if !observation.IsDiagnosticOnly() || observation.CanMutateLiveState() || observation.CanAffectUserVisibleOutput() {
		t.Fatalf("observation should be diagnostic-only: %#v", observation)
	}
	first, _ := json.Marshal(observation)
	again, err := NewShadowObservation(ShadowObservation{
		ObservationID:  "obs-1",
		WorkspaceID:    "workspace-a",
		RequestID:      "request-a",
		ObservedAt:     observedAt,
		LivePath:       "api routes",
		RequestSummary: "live request observed for diagnostics",
		InputRefs:      []string{"input-b", "input-a"},
		EvidenceRefs:   []string{"evidence-a", "evidence-b"},
		RetrievalRefs:  []string{"retrieval-a"},
		ContextRefs:    []string{"context-a"},
		ConsensusRefs:  []string{"consensus-a"},
		RuntimeRefs:    []string{"runtime-a"},
		KVRefs:         []string{"kv-a"},
		RiskFlags:      []string{"low", "medium"},
		Metadata: map[string]any{
			"source": "shadow-design",
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	second, _ := json.Marshal(again)
	if string(first) != string(second) {
		t.Fatalf("serialization should be deterministic\nfirst: %s\nsecond:%s", first, second)
	}
}

func TestShadowObservationRejectsRequiredFieldGapsAndSecretMetadata(t *testing.T) {
	if _, err := NewShadowObservation(ShadowObservation{
		WorkspaceID:    "workspace-a",
		RequestID:      "request-a",
		ObservedAt:     time.Unix(100, 0).UTC(),
		LivePath:       "api routes",
		RequestSummary: "summary",
	}); err == nil {
		t.Fatal("expected missing observation_id to be rejected")
	}
	if _, err := NewShadowObservation(ShadowObservation{
		ObservationID:  "obs-1",
		WorkspaceID:    "workspace-a",
		RequestID:      "request-a",
		ObservedAt:     time.Unix(100, 0).UTC(),
		LivePath:       "api routes",
		RequestSummary: "summary",
		Metadata: map[string]any{
			"api_token": "secret",
		},
	}); err == nil {
		t.Fatal("expected secret-looking metadata key to be rejected")
	}
}
