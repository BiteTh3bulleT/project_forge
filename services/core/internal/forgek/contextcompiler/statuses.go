package contextcompiler

type ContextBlockType string

const (
	BlockKernelDoctrine           ContextBlockType = "KERNEL_DOCTRINE"
	BlockPolicyBoundary           ContextBlockType = "POLICY_BOUNDARY"
	BlockToolContracts            ContextBlockType = "TOOL_CONTRACTS"
	BlockWorkspaceIdentity        ContextBlockType = "WORKSPACE_IDENTITY"
	BlockGoverningPrecedent       ContextBlockType = "GOVERNING_PRECEDENT"
	BlockCaseSummary              ContextBlockType = "CASE_SUMMARY"
	BlockPalaceRouteSummary       ContextBlockType = "PALACE_ROUTE_SUMMARY"
	BlockAdmittedEvidence         ContextBlockType = "ADMITTED_EVIDENCE"
	BlockRejectedEvidenceSummary  ContextBlockType = "REJECTED_EVIDENCE_SUMMARY"
	BlockContradictionSummary     ContextBlockType = "CONTRADICTION_SUMMARY"
	BlockSemanticOperationSummary ContextBlockType = "SEMANTIC_OPERATION_SUMMARY"
	BlockSnapshotRestoreSeed      ContextBlockType = "SNAPSHOT_RESTORE_SEED"
	BlockActiveConstraints        ContextBlockType = "ACTIVE_CONSTRAINTS"
	BlockCurrentTask              ContextBlockType = "CURRENT_TASK"
	BlockVolatileDetail           ContextBlockType = "VOLATILE_DETAIL"
	BlockUserMessage              ContextBlockType = "USER_MESSAGE"
	BlockFutureTokenPlaceholder   ContextBlockType = "FUTURE_TOKEN_PLACEHOLDER"
	BlockFutureKVPlaceholder      ContextBlockType = "FUTURE_KV_PLACEHOLDER"
)

type CacheEligibility string

const (
	CacheAlways    CacheEligibility = "CACHE_ALWAYS"
	CacheIfStable  CacheEligibility = "CACHE_IF_STABLE"
	CacheEphemeral CacheEligibility = "CACHE_EPHEMERAL"
	DoNotCache     CacheEligibility = "DO_NOT_CACHE"
)

const (
	WarningMissingAdmittedEvidence        = "missing_admitted_evidence"
	WarningRejectedEvidenceSummary        = "rejected_evidence_summary_included"
	WarningContradictionsPresent          = "contradictions_present"
	WarningRestoreSeedIncluded            = "restore_seed_included"
	WarningVolatileUserMessagePresent     = "volatile_user_message_present"
	WarningTokenEstimateExceedsBudget     = "token_estimate_exceeds_budget"
	WarningNoCacheablePrefix              = "no_cacheable_prefix"
	DefaultLayoutVersion                  = "context-layout-v1"
	DefaultPolicyVersion                  = "policy-v1"
	DefaultSyscallSchemaVersion           = "syscall-schema-v1"
	DefaultContextInvalidationScope       = "workspace"
	DefaultTokenizerNeutralHashDescriptor = "tokenizer-neutral-text-hash"
)

func ValidBlockType(blockType ContextBlockType) bool {
	switch blockType {
	case BlockKernelDoctrine,
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
		BlockUserMessage,
		BlockFutureTokenPlaceholder,
		BlockFutureKVPlaceholder:
		return true
	default:
		return false
	}
}

func DefaultCacheEligibility(blockType ContextBlockType) CacheEligibility {
	switch blockType {
	case BlockKernelDoctrine, BlockPolicyBoundary, BlockToolContracts:
		return CacheAlways
	case BlockWorkspaceIdentity,
		BlockGoverningPrecedent,
		BlockCaseSummary,
		BlockPalaceRouteSummary,
		BlockAdmittedEvidence,
		BlockRejectedEvidenceSummary,
		BlockContradictionSummary,
		BlockSemanticOperationSummary,
		BlockSnapshotRestoreSeed:
		return CacheIfStable
	case BlockActiveConstraints, BlockCurrentTask, BlockVolatileDetail, BlockFutureTokenPlaceholder, BlockFutureKVPlaceholder:
		return CacheEphemeral
	case BlockUserMessage:
		return DoNotCache
	default:
		return DoNotCache
	}
}

func IsStableCacheEligibility(eligibility CacheEligibility) bool {
	return eligibility == CacheAlways || eligibility == CacheIfStable
}
