use serde::Serialize;
use serde_json::Value;

#[derive(Debug, Clone, Copy, PartialEq, Eq, Serialize)]
#[serde(rename_all = "snake_case")]
pub enum FixtureKind {
    SnapshotManifest,
    ContextBlock,
    ContextBundle,
    KvCacheManifest,
    RuntimeDriverManifest,
    CapabilityLike,
}

impl FixtureKind {
    pub fn as_str(self) -> &'static str {
        match self {
            FixtureKind::SnapshotManifest => "snapshot_manifest",
            FixtureKind::ContextBlock => "context_block",
            FixtureKind::ContextBundle => "context_bundle",
            FixtureKind::KvCacheManifest => "kv_cache_manifest",
            FixtureKind::RuntimeDriverManifest => "runtime_driver_manifest",
            FixtureKind::CapabilityLike => "capability_like",
        }
    }
}

pub const SNAPSHOT_TYPES: &[&str] = &[
    "SEMANTIC_SNAPSHOT",
    "CASE_SNAPSHOT",
    "CONTEXT_RESTORE_SNAPSHOT",
    "PALACE_ROUTE_SNAPSHOT",
    "WORKSPACE_SNAPSHOT",
    "DECISION_SNAPSHOT",
    "KV_SHAPE_SNAPSHOT",
    "RUNTIME_SNAPSHOT",
];

pub const SNAPSHOT_STATUSES: &[&str] = &[
    "DRAFT",
    "SEALED",
    "SUPERSEDED",
    "EXPIRED",
    "RESTORE_SEED_CREATED",
];

pub const CONTEXT_BLOCK_TYPES: &[&str] = &[
    "KERNEL_DOCTRINE",
    "POLICY_BOUNDARY",
    "TOOL_CONTRACTS",
    "WORKSPACE_IDENTITY",
    "GOVERNING_PRECEDENT",
    "CASE_SUMMARY",
    "PALACE_ROUTE_SUMMARY",
    "ADMITTED_EVIDENCE",
    "REJECTED_EVIDENCE_SUMMARY",
    "CONTRADICTION_SUMMARY",
    "SEMANTIC_OPERATION_SUMMARY",
    "SNAPSHOT_RESTORE_SEED",
    "ACTIVE_CONSTRAINTS",
    "CURRENT_TASK",
    "VOLATILE_DETAIL",
    "USER_MESSAGE",
    "FUTURE_TOKEN_PLACEHOLDER",
    "FUTURE_KV_PLACEHOLDER",
];

pub const CACHE_ELIGIBILITY: &[&str] = &[
    "CACHE_ALWAYS",
    "CACHE_IF_STABLE",
    "CACHE_EPHEMERAL",
    "DO_NOT_CACHE",
];

pub const CACHE_MODES: &[&str] = &["STRICT_PREFIX", "SNAPSHOT_PREFIX", "BACKEND_COMPOSITIONAL"];

pub const MEMORY_TIERS: &[&str] = &["GPU_HOT", "CPU_WARM", "DISK_COLD", "REMOTE_COLD", "NONE"];

pub const KV_STATUSES: &[&str] = &[
    "AVAILABLE",
    "HIT_RECORDED",
    "INVALIDATED",
    "EVICTED",
    "EXPIRED",
];

pub const DRIVER_KINDS: &[&str] = &[
    "MOCK",
    "LOCAL_DEV",
    "VLLM",
    "SGLANG",
    "TENSORRT_LLM",
    "OLLAMA",
    "LLAMA_CPP",
    "REMOTE_API",
    "FUTURE",
];

pub const RUNTIME_AUTHORITY_PROPOSAL_ONLY: &str = "PROPOSAL_ONLY";

pub fn detect_fixture_kind(value: &Value) -> Option<FixtureKind> {
    let object = value.as_object()?;
    if object.contains_key("cache_id") {
        return Some(FixtureKind::KvCacheManifest);
    }
    if object.contains_key("driver_id") && object.contains_key("driver_kind") {
        return Some(FixtureKind::RuntimeDriverManifest);
    }
    if object.contains_key("bundle_id") && object.contains_key("blocks") {
        return Some(FixtureKind::ContextBundle);
    }
    if object.contains_key("block_id") && object.contains_key("block_type") {
        return Some(FixtureKind::ContextBlock);
    }
    if object.contains_key("snapshot_id") && object.contains_key("snapshot_type") {
        return Some(FixtureKind::SnapshotManifest);
    }
    if object.contains_key("capability_id") && object.contains_key("allowed_syscalls") {
        return Some(FixtureKind::CapabilityLike);
    }
    None
}
