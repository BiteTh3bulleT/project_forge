package controllane

import (
	"strings"
	"testing"

	"forge/projectforge/services/core/internal/aios/domain"
	"forge/projectforge/services/core/internal/forgekernel/contextcompile"
)

func TestGovernedContextBundleRejectsAdapterDecisionContentTamper(t *testing.T) {
	_, store, _ := newTestKernel()
	req := validBaseRequest(domain.ActionCompileContext)
	req.ID = "context-adapter-tamper"
	req.IdempotencyKey = "idem-context-adapter-tamper"
	req.Payload = map[string]any{
		"query": "bounded context", "budget": map[string]any{"maxTokens": 256, "maxEvents": 8, "maxNotes": 8},
		"persistSnapshot": true, "snapshotKind": "chat",
	}
	req.Metadata = map[string]any{"forgeKAuthorizationProof": "sha256:" + strings.Repeat("0", 64)}
	input, err := prepareContextCompileAuthorityInput(req, store)
	if err != nil {
		t.Fatal(err)
	}
	decision, err := contextcompile.Compile(input)
	if err != nil {
		t.Fatal(err)
	}
	bundle, err := buildGovernedContextBundle(req, input, decision, store)
	if err != nil {
		t.Fatal(err)
	}
	bundle.Candidate.PacketHash = "sha256:" + strings.Repeat("f", 64)
	if err := store.CreateGovernedContextBundle(bundle); err == nil {
		t.Fatal("adapter-substituted context candidate was accepted")
	}
}
