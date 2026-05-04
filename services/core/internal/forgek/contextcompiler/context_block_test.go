package contextcompiler

import (
	"encoding/json"
	"errors"
	"reflect"
	"testing"
	"time"
)

func testBlockInput() ContextBlockInput {
	return ContextBlockInput{
		BlockID:              "block-a",
		BlockType:            BlockAdmittedEvidence,
		WorkspaceID:          "workspace-a",
		CaseID:               "case-a",
		SourceObjectRefs:     []string{"semantic-b", "semantic-a", "semantic-a"},
		SourceRefs:           []string{"doc-b", "doc-a"},
		AdmittedExhibitRefs:  []string{"exhibit-b", "exhibit-a", "exhibit-a"},
		ContentSummary:       "  admitted   semantic   shape\n",
		PolicyVersion:        "policy-v1",
		SyscallSchemaVersion: "schema-v1",
		CreatedBy:            "operator",
		CreatedAt:            time.Date(2026, 5, 4, 12, 0, 0, 0, time.UTC),
	}
}

func TestContextBlockValidatesAndNormalizesRefs(t *testing.T) {
	block, err := NewContextBlock(testBlockInput())
	if err != nil {
		t.Fatalf("NewContextBlock failed: %v", err)
	}
	if block.BlockType != BlockAdmittedEvidence || block.WorkspaceID != "workspace-a" {
		t.Fatalf("unexpected block: %#v", block)
	}
	if !reflect.DeepEqual(block.SourceObjectRefs, []string{"semantic-a", "semantic-b"}) {
		t.Fatalf("source refs not normalized: %#v", block.SourceObjectRefs)
	}
	if !reflect.DeepEqual(block.AdmittedExhibitRefs, []string{"exhibit-a", "exhibit-b"}) {
		t.Fatalf("admitted refs not normalized: %#v", block.AdmittedExhibitRefs)
	}
	if block.ContentSummary != "admitted semantic shape" {
		t.Fatalf("summary whitespace not normalized: %q", block.ContentSummary)
	}
	if block.ContentHash == "" || block.TokenInputHash == "" || block.TokenCountEstimate == 0 {
		t.Fatalf("hashes/token estimate missing: %#v", block)
	}
	if block.IsCanonicalTruth() || block.IsKVCache() {
		t.Fatal("context block claimed truth or KV cache authority")
	}
}

func TestContextBlockRejectsInvalidInput(t *testing.T) {
	input := testBlockInput()
	input.BlockType = "UNKNOWN"
	if _, err := NewContextBlock(input); !errors.Is(err, ErrInvalidBlockType) {
		t.Fatalf("expected invalid block type, got %v", err)
	}
	input = testBlockInput()
	input.WorkspaceID = ""
	if _, err := NewContextBlock(input); !errors.Is(err, ErrInvalidContextBlock) {
		t.Fatalf("expected missing workspace rejection, got %v", err)
	}
}

func TestContextBlockHashesAreStableAndExcludeCreatedAt(t *testing.T) {
	left, err := NewContextBlock(testBlockInput())
	if err != nil {
		t.Fatalf("left block: %v", err)
	}
	input := testBlockInput()
	input.CreatedAt = input.CreatedAt.Add(2 * time.Hour)
	input.SourceObjectRefs = []string{"semantic-a", "semantic-b"}
	right, err := NewContextBlock(input)
	if err != nil {
		t.Fatalf("right block: %v", err)
	}
	if left.ContentHash != right.ContentHash || left.TokenInputHash != right.TokenInputHash {
		t.Fatalf("created_at/ref order changed hashes: left=%#v right=%#v", left, right)
	}
	input.ContentSummary = "different shape"
	changed, err := NewContextBlock(input)
	if err != nil {
		t.Fatalf("changed block: %v", err)
	}
	if changed.ContentHash == left.ContentHash {
		t.Fatal("content hash did not change when semantic shape changed")
	}
}

func TestContextBlockCacheEligibilityDefaultsAndSerializes(t *testing.T) {
	for blockType, want := range map[ContextBlockType]CacheEligibility{
		BlockKernelDoctrine:    CacheAlways,
		BlockCaseSummary:       CacheIfStable,
		BlockActiveConstraints: CacheEphemeral,
		BlockUserMessage:       DoNotCache,
	} {
		input := testBlockInput()
		input.BlockID = string(blockType) + "-block"
		input.BlockType = blockType
		block, err := NewContextBlock(input)
		if err != nil {
			t.Fatalf("block %s: %v", blockType, err)
		}
		if block.CacheEligibility != want {
			t.Fatalf("cache eligibility for %s = %s, want %s", blockType, block.CacheEligibility, want)
		}
		if _, err := json.Marshal(block); err != nil {
			t.Fatalf("block did not serialize: %v", err)
		}
	}
}
