package lymphatic

type SweepKind string
type FindingType string
type FindingSeverity string
type ProposalType string
type ReportStatus string

const (
	SweepSnapshotHygiene  SweepKind = "SNAPSHOT_HYGIENE"
	SweepKVHygiene        SweepKind = "KV_HYGIENE"
	SweepRuntimeResult    SweepKind = "RUNTIME_RESULT_HYGIENE"
	SweepContextBundle    SweepKind = "CONTEXT_BUNDLE_HYGIENE"
	SweepContradiction    SweepKind = "CONTRADICTION_SWEEP"
	SweepSupersession     SweepKind = "SUPERSESSION_SWEEP"
	SweepOrphanRef        SweepKind = "ORPHAN_REF_SWEEP"
	SweepPalaceRoute      SweepKind = "PALACE_ROUTE_HYGIENE"
	SweepSemanticObject   SweepKind = "SEMANTIC_OBJECT_HYGIENE"
	SweepCase             SweepKind = "CASE_HYGIENE"
	SweepJournalIntegrity SweepKind = "JOURNAL_INTEGRITY_SWEEP"
)

const (
	FindingStaleSnapshot            FindingType = "STALE_SNAPSHOT"
	FindingSupersededSnapshot       FindingType = "SUPERSEDED_SNAPSHOT"
	FindingExpiredSnapshotCandidate FindingType = "EXPIRED_SNAPSHOT_CANDIDATE"
	FindingInvalidatedKVManifest    FindingType = "INVALIDATED_KV_MANIFEST"
	FindingEvictableKVManifest      FindingType = "EVICTABLE_KV_MANIFEST"
	FindingStaleRuntimeResult       FindingType = "STALE_RUNTIME_RESULT"
	FindingOpenContradiction        FindingType = "OPEN_CONTRADICTION"
	FindingOrphanedReference        FindingType = "ORPHANED_REFERENCE"
	FindingUnreferencedRoute        FindingType = "UNREFERENCED_ROUTE"
	FindingUnknownObjectRef         FindingType = "UNKNOWN_OBJECT_REF"
	FindingJournalGapWarning        FindingType = "JOURNAL_GAP_WARNING"
)

const (
	SeverityInfo     FindingSeverity = "INFO"
	SeverityLow      FindingSeverity = "LOW"
	SeverityMedium   FindingSeverity = "MEDIUM"
	SeverityHigh     FindingSeverity = "HIGH"
	SeverityCritical FindingSeverity = "CRITICAL"
)

const (
	ProposalExpireSnapshot              ProposalType = "EXPIRE_SNAPSHOT"
	ProposalSupersedeSnapshot           ProposalType = "SUPERSEDE_SNAPSHOT"
	ProposalInvalidateKV                ProposalType = "INVALIDATE_KV"
	ProposalEvictKV                     ProposalType = "EVICT_KV"
	ProposalDemoteKVTier                ProposalType = "DEMOTE_KV_TIER"
	ProposalMarkRuntimeResultStale      ProposalType = "MARK_RUNTIME_RESULT_STALE"
	ProposalRegisterContradictionReview ProposalType = "REGISTER_CONTRADICTION_REVIEW"
	ProposalRepairOrphanRef             ProposalType = "REPAIR_ORPHAN_REF"
	ProposalExpireRestoreSeed           ProposalType = "EXPIRE_RESTORE_SEED"
	ProposalNoOpReview                  ProposalType = "NO_OP_REVIEW"
)

const (
	ReportComplete ReportStatus = "COMPLETE"
	ReportRejected ReportStatus = "REJECTED"
)

func ValidSweepKind(kind SweepKind) bool {
	switch kind {
	case SweepSnapshotHygiene, SweepKVHygiene, SweepRuntimeResult, SweepContextBundle,
		SweepContradiction, SweepSupersession, SweepOrphanRef, SweepPalaceRoute,
		SweepSemanticObject, SweepCase, SweepJournalIntegrity:
		return true
	default:
		return false
	}
}

func AllSweepKinds() []SweepKind {
	return []SweepKind{
		SweepSnapshotHygiene,
		SweepKVHygiene,
		SweepRuntimeResult,
		SweepContextBundle,
		SweepContradiction,
		SweepSupersession,
		SweepOrphanRef,
		SweepPalaceRoute,
		SweepSemanticObject,
		SweepCase,
		SweepJournalIntegrity,
	}
}
