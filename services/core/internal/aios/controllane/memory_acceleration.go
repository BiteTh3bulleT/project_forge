package controllane

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"

	"forge/projectforge/services/core/internal/aios/domain"
	"forge/projectforge/services/core/internal/forgekernel"
	"forge/projectforge/services/core/internal/memory/vsaprojection"
)

func validateRebuildMemoryAcceleration(req domain.SyscallRequest) []domain.SyscallError {
	issues := []domain.SyscallError{}
	if strings.TrimSpace(req.Scope.LaneID) == "" {
		issues = append(issues, errField(domain.ErrInvalidScope, "scope.laneId", "exact lane scope is required"))
	}
	if readString(req.Payload, "algorithmName") != vsaprojection.AlgorithmName {
		issues = append(issues, errField(domain.ErrInvalidPayload, "payload.algorithmName", "unsupported VSA projection algorithm"))
	}
	if readString(req.Payload, "algorithmVersion") != vsaprojection.AlgorithmVersion {
		issues = append(issues, errField(domain.ErrInvalidPayload, "payload.algorithmVersion", "unsupported VSA projection algorithm version"))
	}
	dimensions := readInt(req.Payload, "dimensions", 0)
	if dimensions < 8 || dimensions > 4096 {
		issues = append(issues, errField(domain.ErrInvalidPayload, "payload.dimensions", "dimensions must be between 8 and 4096"))
	}
	seed := readInt(req.Payload, "seed", 0)
	if seed <= 0 {
		issues = append(issues, errField(domain.ErrInvalidPayload, "payload.seed", "seed must be a positive integer"))
	}
	if !validProjectionHash(readString(req.Payload, "expectedManifestHash")) {
		issues = append(issues, errField(domain.ErrInvalidPayload, "payload.expectedManifestHash", "expectedManifestHash must be a sha256 identity"))
	}
	prior := readString(req.Payload, "expectedPriorManifestHash")
	if prior != "" && !validProjectionHash(prior) {
		issues = append(issues, errField(domain.ErrInvalidPayload, "payload.expectedPriorManifestHash", "expectedPriorManifestHash must be empty or a sha256 identity"))
	}
	requestedAt := readInt64(req.Payload, "requestedAtMs")
	if requestedAt <= 0 || requestedAt != req.RequestedAt || !exactInteger(req.Payload["requestedAtMs"]) {
		issues = append(issues, errField(domain.ErrInvalidPayload, "payload.requestedAtMs", "requestedAtMs must exactly match the sealed request timestamp"))
	}
	return issues
}

func exactInteger(value any) bool {
	switch typed := value.(type) {
	case int, int64, int32, uint, uint32, uint64:
		return true
	case float64:
		return typed == float64(int64(typed))
	case float32:
		return typed == float32(int64(typed))
	default:
		return false
	}
}

func validProjectionHash(value string) bool {
	if !strings.HasPrefix(value, "sha256:") || len(value) != len("sha256:")+64 {
		return false
	}
	_, err := hex.DecodeString(strings.TrimPrefix(value, "sha256:"))
	return err == nil
}

func applyRebuildMemoryAcceleration(ctx context.Context, store SemanticStore, req domain.SyscallRequest) ([]string, map[string]any, []string, []domain.SyscallError) {
	owner, _ := req.Metadata["kernelAuthorityOwner"].(string)
	ingress, _ := req.Metadata["forgeKIngressAuthority"].(bool)
	if !ingress || owner != forgekernel.AuthorityOwnerForgeK {
		return nil, nil, nil, []domain.SyscallError{{Code: domain.ErrUnauthorized, Field: "metadata.kernelAuthorityOwner", Message: "memory acceleration rebuild requires production FORGE-K ingress"}}
	}
	commit, err := store.RebuildMemoryAcceleration(ctx, MemoryAccelerationRebuildRequest{
		Scope: vsaprojection.Scope{WorkspaceID: req.Scope.WorkspaceID, LaneID: req.Scope.LaneID},
		Algorithm: vsaprojection.Algorithm{
			Name: readString(req.Payload, "algorithmName"), Version: readString(req.Payload, "algorithmVersion"),
			Dimensions: readInt(req.Payload, "dimensions", 0), Seed: uint64(readInt(req.Payload, "seed", 0)),
		},
		ExpectedManifestHash:      readString(req.Payload, "expectedManifestHash"),
		ExpectedPriorManifestHash: readString(req.Payload, "expectedPriorManifestHash"),
		RequestedAtMs:             req.RequestedAt,
	})
	if err != nil {
		code, field := domain.ErrConflict, "memoryAcceleration"
		if errors.Is(err, ErrMemoryAccelerationNoGovernedSources) {
			code, field = domain.ErrInvalidStateTransition, "memoryAcceleration.sources"
		} else if errors.Is(err, vsaprojection.ErrInvalidScope) || errors.Is(err, vsaprojection.ErrInvalidAlgorithm) || errors.Is(err, vsaprojection.ErrInvalidSource) || errors.Is(err, vsaprojection.ErrInvalidLink) {
			code, field = domain.ErrInvalidPayload, "memoryAcceleration.manifest"
		}
		return nil, nil, nil, []domain.SyscallError{{Code: code, Field: field, Message: err.Error()}}
	}
	return []string{commit.Manifest.ManifestHash}, map[string]any{
		"memoryAcceleration": map[string]any{
			"projectionOnly": true, "atomicSwap": true,
			"workspaceId": commit.Manifest.Scope.WorkspaceID, "laneId": commit.Manifest.Scope.LaneID,
			"manifestHash": commit.Manifest.ManifestHash, "priorManifestHash": commit.PriorManifestHash,
			"sourceSetHash": commit.Manifest.SourceSetHash, "linkSetHash": commit.Manifest.LinkSetHash,
			"pointerCount": commit.PointerCount, "bindingCount": commit.BindingCount, "associationCount": commit.AssociationCount,
		},
	}, nil, nil
}

func (s *InMemorySemanticStore) RebuildMemoryAcceleration(_ context.Context, req MemoryAccelerationRebuildRequest) (MemoryAccelerationCommit, error) {
	return MemoryAccelerationCommit{}, fmt.Errorf("%w: in-memory store has no scoped admitted memory sources for %s/%s", ErrMemoryAccelerationNoGovernedSources, req.Scope.WorkspaceID, req.Scope.LaneID)
}

func (s *TransactionalSemanticStore) RebuildMemoryAcceleration(_ context.Context, req MemoryAccelerationRebuildRequest) (MemoryAccelerationCommit, error) {
	return MemoryAccelerationCommit{}, fmt.Errorf("%w: transactional memory store has no scoped admitted memory sources for %s/%s", ErrMemoryAccelerationNoGovernedSources, req.Scope.WorkspaceID, req.Scope.LaneID)
}
