package api

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestGatewayInvokeDoesNotTrustBodyAuthorityFields(t *testing.T) {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime caller unavailable")
	}
	body, err := os.ReadFile(filepath.Join(filepath.Dir(file), "phase5.go"))
	if err != nil {
		t.Fatalf("read phase5 source: %v", err)
	}
	source := string(body)
	for _, forbidden := range []string{
		"Initiator:           body.Initiator",
		"ProvenanceActor:     body.ProvenanceActor",
		"ProvenanceActorType: body.ProvenanceActorType",
	} {
		if strings.Contains(source, forbidden) {
			t.Fatalf("gateway invoke must not trust body authority field %q", forbidden)
		}
	}
}
