package gateway

import (
	"errors"
	"strings"
	"testing"
)

func TestNormalizeGatewayRequestMetadataIsBounded(t *testing.T) {
	t.Parallel()
	value, err := normalizeGatewayRequestMetadata("  operator  ", "initiator", maxGatewayRequestMetadataBytes)
	if err != nil {
		t.Fatalf("expected valid request metadata, got %v", err)
	}
	if value != "operator" {
		t.Fatalf("unexpected normalized request metadata %q", value)
	}
	if _, err := normalizeGatewayRequestMetadata(strings.Repeat("m", maxGatewayRequestMetadataBytes+1), "initiator", maxGatewayRequestMetadataBytes); !errors.Is(err, errGatewayRequestMetadataTooLarge) {
		t.Fatalf("expected request metadata size rejection, got %v", err)
	}
}
