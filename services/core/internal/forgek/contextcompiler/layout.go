package contextcompiler

import (
	"sort"
	"time"
)

type PromptLayout struct {
	LayoutID               string             `json:"layout_id"`
	LayoutVersion          string             `json:"layout_version"`
	WorkspaceID            string             `json:"workspace_id"`
	BlockOrder             []ContextBlockType `json:"block_order"`
	StablePrefixBoundary   int                `json:"stable_prefix_boundary"`
	VolatileSuffixBoundary int                `json:"volatile_suffix_boundary"`
	PolicyVersion          string             `json:"policy_version"`
	SyscallSchemaVersion   string             `json:"syscall_schema_version"`
	CreatedAt              time.Time          `json:"created_at"`
	Metadata               map[string]any     `json:"metadata,omitempty"`
}

func DefaultBlockOrder() []ContextBlockType {
	return []ContextBlockType{
		BlockKernelDoctrine,
		BlockPolicyBoundary,
		BlockToolContracts,
		BlockWorkspaceIdentity,
		BlockGoverningPrecedent,
		BlockCaseSummary,
		BlockPalaceRouteSummary,
		BlockAdmittedEvidence,
		BlockRejectedEvidenceSummary,
		BlockContradictionSummary,
		BlockSemanticOperationSummary,
		BlockSnapshotRestoreSeed,
		BlockActiveConstraints,
		BlockCurrentTask,
		BlockVolatileDetail,
		BlockFutureTokenPlaceholder,
		BlockFutureKVPlaceholder,
		BlockUserMessage,
	}
}

func DefaultPromptLayout(workspaceID, layoutVersion, policyVersion, syscallSchemaVersion string, createdAt time.Time, metadata map[string]any) PromptLayout {
	if layoutVersion == "" {
		layoutVersion = DefaultLayoutVersion
	}
	if policyVersion == "" {
		policyVersion = DefaultPolicyVersion
	}
	if syscallSchemaVersion == "" {
		syscallSchemaVersion = DefaultSyscallSchemaVersion
	}
	order := DefaultBlockOrder()
	return PromptLayout{
		LayoutID:               "prompt-layout-" + layoutVersion,
		LayoutVersion:          layoutVersion,
		WorkspaceID:            workspaceID,
		BlockOrder:             order,
		StablePrefixBoundary:   indexOfBlock(order, BlockActiveConstraints),
		VolatileSuffixBoundary: indexOfBlock(order, BlockActiveConstraints),
		PolicyVersion:          policyVersion,
		SyscallSchemaVersion:   syscallSchemaVersion,
		CreatedAt:              createdAt,
		Metadata:               CloneMap(metadata),
	}
}

func ValidateLayout(layout PromptLayout) error {
	if layout.LayoutVersion == "" || layout.WorkspaceID == "" || len(layout.BlockOrder) == 0 {
		return ErrInvalidPromptLayout
	}
	seen := map[ContextBlockType]struct{}{}
	for _, blockType := range layout.BlockOrder {
		if !ValidBlockType(blockType) {
			return ErrInvalidPromptLayout
		}
		if _, ok := seen[blockType]; ok {
			return ErrInvalidPromptLayout
		}
		seen[blockType] = struct{}{}
	}
	if indexOfBlock(layout.BlockOrder, BlockUserMessage) != len(layout.BlockOrder)-1 {
		return ErrInvalidPromptLayout
	}
	firstVolatile := len(layout.BlockOrder)
	for i, blockType := range layout.BlockOrder {
		if !IsStableCacheEligibility(DefaultCacheEligibility(blockType)) {
			firstVolatile = i
			break
		}
	}
	for i := firstVolatile + 1; i < len(layout.BlockOrder); i++ {
		if IsStableCacheEligibility(DefaultCacheEligibility(layout.BlockOrder[i])) {
			return ErrInvalidPromptLayout
		}
	}
	return nil
}

func SortBlocksForLayout(blocks []ContextBlock, layout PromptLayout) []ContextBlock {
	out := make([]ContextBlock, len(blocks))
	for i, block := range blocks {
		out[i] = block.Clone()
	}
	order := make(map[ContextBlockType]int, len(layout.BlockOrder))
	for i, blockType := range layout.BlockOrder {
		order[blockType] = i
	}
	sort.SliceStable(out, func(i, j int) bool {
		leftOrder, leftKnown := order[out[i].BlockType]
		rightOrder, rightKnown := order[out[j].BlockType]
		if !leftKnown {
			leftOrder = len(order) - 1
		}
		if !rightKnown {
			rightOrder = len(order) - 1
		}
		if leftOrder != rightOrder {
			return leftOrder < rightOrder
		}
		if out[i].ContentHash != out[j].ContentHash {
			return out[i].ContentHash < out[j].ContentHash
		}
		return out[i].BlockID < out[j].BlockID
	})
	for i := range out {
		out[i].LayoutPosition = i + 1
		out[i] = FinalizeBlock(out[i])
	}
	return out
}

func indexOfBlock(order []ContextBlockType, want ContextBlockType) int {
	for i, blockType := range order {
		if blockType == want {
			return i
		}
	}
	return len(order)
}
