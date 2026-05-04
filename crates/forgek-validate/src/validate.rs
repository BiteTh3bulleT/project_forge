use serde::Serialize;
use serde_json::{Map, Value};

use crate::canonical::canonicalize_value;
use crate::errors::ValidateError;
use crate::hash::{
    hash_canonical_json, hash_context_block, hash_context_bundle, hash_kv_manifest_identity,
    hash_runtime_manifest, hash_snapshot_shape,
};
use crate::types::{
    detect_fixture_kind, FixtureKind, CACHE_ELIGIBILITY, CACHE_MODES, CONTEXT_BLOCK_TYPES,
    DRIVER_KINDS, KV_STATUSES, MEMORY_TIERS, RUNTIME_AUTHORITY_PROPOSAL_ONLY, SNAPSHOT_STATUSES,
    SNAPSHOT_TYPES,
};

#[derive(Debug, Clone, Serialize)]
pub struct ValidationReport {
    pub kind: String,
    pub valid: bool,
    pub errors: Vec<String>,
    pub hashes: Map<String, Value>,
}

impl ValidationReport {
    fn valid(kind: FixtureKind, hashes: Map<String, Value>) -> Self {
        Self {
            kind: kind.as_str().to_string(),
            valid: true,
            errors: Vec::new(),
            hashes,
        }
    }

    fn invalid(kind: FixtureKind, errors: Vec<String>) -> Self {
        Self {
            kind: kind.as_str().to_string(),
            valid: false,
            errors,
            hashes: Map::new(),
        }
    }
}

pub fn validate_manifest(value: &Value) -> Result<ValidationReport, ValidateError> {
    if !value.is_object() {
        return Err(ValidateError::InvalidRoot);
    }
    let kind = detect_fixture_kind(value).ok_or(ValidateError::UnknownFixtureKind)?;
    let report = match kind {
        FixtureKind::SnapshotManifest => validate_snapshot(value)?,
        FixtureKind::ContextBlock => validate_context_block(value)?,
        FixtureKind::ContextBundle => validate_context_bundle(value)?,
        FixtureKind::KvCacheManifest => validate_kv_manifest(value)?,
        FixtureKind::RuntimeDriverManifest => validate_runtime_driver_manifest(value)?,
        FixtureKind::CapabilityLike => validate_capability_like(value)?,
    };
    Ok(report)
}

pub fn validate_manifest_or_error(value: &Value) -> Result<ValidationReport, ValidateError> {
    let report = validate_manifest(value)?;
    if report.valid {
        Ok(report)
    } else {
        Err(ValidateError::InvalidFixture {
            kind: report.kind.clone(),
            errors: report.errors.clone(),
        })
    }
}

fn validate_snapshot(value: &Value) -> Result<ValidationReport, ValidateError> {
    let mut errors = Vec::new();
    require_string(value, "snapshot_id", &mut errors);
    require_enum(value, "snapshot_type", SNAPSHOT_TYPES, &mut errors);
    require_string(value, "workspace_id", &mut errors);
    require_enum(value, "status", SNAPSHOT_STATUSES, &mut errors);
    if has_large_raw_content_key(value) {
        errors.push("snapshot contains raw content-like field".to_string());
    }
    if !has_any_refs(
        value,
        &[
            "source_object_refs",
            "source_refs",
            "palace_route_refs",
            "submitted_object_refs",
            "admitted_object_refs",
            "rejected_object_refs",
            "semantic_operation_refs",
            "contradiction_refs",
            "supersession_refs",
            "derived_object_refs",
            "context_block_refs",
            "token_hash_refs",
            "kv_manifest_refs",
        ],
    ) && string_field(value, "snapshot_type") != Some("WORKSPACE_SNAPSHOT")
    {
        errors.push("snapshot requires source, ref, or operation shape".to_string());
    }
    finish_report(
        FixtureKind::SnapshotManifest,
        errors,
        &[("snapshot_shape_hash", hash_snapshot_shape(value)?)],
    )
}

fn validate_context_block(value: &Value) -> Result<ValidationReport, ValidateError> {
    let mut errors = Vec::new();
    require_string(value, "block_id", &mut errors);
    require_enum(value, "block_type", CONTEXT_BLOCK_TYPES, &mut errors);
    require_string(value, "workspace_id", &mut errors);
    require_enum(value, "cache_eligibility", CACHE_ELIGIBILITY, &mut errors);
    if !has_any_refs(
        value,
        &[
            "source_object_refs",
            "source_refs",
            "admitted_exhibit_refs",
            "rejected_exhibit_refs",
            "ruling_refs",
            "contradiction_refs",
            "supersession_refs",
            "palace_route_refs",
            "semantic_operation_refs",
            "derived_object_refs",
        ],
    ) {
        errors.push("context block requires provenance refs".to_string());
    }
    if !has_string(value, "canonical_text") && !has_string(value, "content_hash") {
        errors.push("context block requires canonical_text or content_hash".to_string());
    }
    if !has_string(value, "token_input_hash") && !has_string(value, "canonical_text") {
        errors.push(
            "context block requires token_input_hash or computable canonical_text".to_string(),
        );
    }
    finish_report(
        FixtureKind::ContextBlock,
        errors,
        &[("context_block_hash", hash_context_block(value)?)],
    )
}

fn validate_context_bundle(value: &Value) -> Result<ValidationReport, ValidateError> {
    let mut errors = Vec::new();
    require_string(value, "bundle_id", &mut errors);
    require_string(value, "workspace_id", &mut errors);
    require_string(value, "layout_version", &mut errors);
    match value.get("blocks").and_then(Value::as_array) {
        Some(blocks) if !blocks.is_empty() => {
            for (index, block) in blocks.iter().enumerate() {
                if let Ok(report) = validate_context_block(block) {
                    if !report.valid {
                        errors.push(format!(
                            "block {index} invalid: {}",
                            report.errors.join("; ")
                        ));
                    }
                } else {
                    errors.push(format!("block {index} is not a valid context block"));
                }
            }
        }
        _ => errors.push("context bundle requires non-empty blocks".to_string()),
    }
    if !has_string(value, "bundle_hash") && !has_string(value, "canonical_prompt_text") {
        errors.push(
            "context bundle requires bundle_hash or computable canonical_prompt_text".to_string(),
        );
    }
    if !has_string(value, "stable_prefix_hash")
        && !has_string(value, "volatile_suffix_hash")
        && !has_string(value, "canonical_prompt_text")
    {
        errors.push(
            "context bundle requires prefix/suffix hashes or computable canonical_prompt_text"
                .to_string(),
        );
    }
    finish_report(
        FixtureKind::ContextBundle,
        errors,
        &[("context_bundle_hash", hash_context_bundle(value)?)],
    )
}

fn validate_kv_manifest(value: &Value) -> Result<ValidationReport, ValidateError> {
    let mut errors = Vec::new();
    for field in [
        "cache_id",
        "workspace_id",
        "model_id",
        "model_revision",
        "tokenizer_id",
        "tokenizer_revision",
        "chat_template_hash",
        "prompt_layout_hash",
        "policy_schema_hash",
        "syscall_schema_hash",
        "token_input_hash",
        "runtime_backend",
        "runtime_version",
        "attention_backend",
        "rope_config_hash",
        "kv_precision",
        "cache_salt",
    ] {
        require_string(value, field, &mut errors);
    }
    if !has_string(value, "bundle_id") && !has_string(value, "block_id") {
        errors.push("kv manifest requires bundle_id or block_id".to_string());
    }
    require_enum(value, "cache_mode", CACHE_MODES, &mut errors);
    require_enum(value, "memory_tier", MEMORY_TIERS, &mut errors);
    require_enum(value, "status", KV_STATUSES, &mut errors);
    finish_report(
        FixtureKind::KvCacheManifest,
        errors,
        &[(
            "kv_manifest_identity_hash",
            hash_kv_manifest_identity(value)?,
        )],
    )
}

fn validate_runtime_driver_manifest(value: &Value) -> Result<ValidationReport, ValidateError> {
    let mut errors = Vec::new();
    require_string(value, "driver_id", &mut errors);
    require_enum(value, "driver_kind", DRIVER_KINDS, &mut errors);
    require_string(value, "runtime_backend", &mut errors);
    require_string(value, "runtime_version", &mut errors);
    if string_field(value, "authority_level").unwrap_or(RUNTIME_AUTHORITY_PROPOSAL_ONLY)
        != RUNTIME_AUTHORITY_PROPOSAL_ONLY
    {
        errors.push("runtime manifest authority_level must be PROPOSAL_ONLY".to_string());
    }
    let secret_errors = secret_like_findings(value);
    errors.extend(secret_errors);
    finish_report(
        FixtureKind::RuntimeDriverManifest,
        errors,
        &[("runtime_manifest_hash", hash_runtime_manifest(value)?)],
    )
}

fn validate_capability_like(value: &Value) -> Result<ValidationReport, ValidateError> {
    let mut errors = Vec::new();
    require_string(value, "capability_id", &mut errors);
    require_string(value, "subject_id", &mut errors);
    if !has_non_empty_array(value, "allowed_syscalls") {
        errors.push("capability-like fixture requires allowed_syscalls".to_string());
    }
    if !has_non_empty_array(value, "workspace_scope") {
        errors.push("capability-like fixture requires workspace_scope".to_string());
    }
    finish_report(
        FixtureKind::CapabilityLike,
        errors,
        &[(
            "capability_shape_hash",
            hash_canonical_json(&canonicalize_value(value))?,
        )],
    )
}

fn finish_report(
    kind: FixtureKind,
    errors: Vec<String>,
    hashes: &[(&str, String)],
) -> Result<ValidationReport, ValidateError> {
    if errors.is_empty() {
        let mut map = Map::new();
        for (key, value) in hashes {
            map.insert((*key).to_string(), Value::String(value.clone()));
        }
        Ok(ValidationReport::valid(kind, map))
    } else {
        Ok(ValidationReport::invalid(kind, errors))
    }
}

fn require_string(value: &Value, field: &str, errors: &mut Vec<String>) {
    if !has_string(value, field) {
        errors.push(format!("{field} is required"));
    }
}

fn require_enum(value: &Value, field: &str, allowed: &[&str], errors: &mut Vec<String>) {
    match string_field(value, field) {
        Some(value) if allowed.contains(&value) => {}
        Some(_) => errors.push(format!("{field} has invalid value")),
        None => errors.push(format!("{field} is required")),
    }
}

fn has_string(value: &Value, field: &str) -> bool {
    string_field(value, field).is_some_and(|value| !value.trim().is_empty())
}

fn string_field<'a>(value: &'a Value, field: &str) -> Option<&'a str> {
    value.get(field).and_then(Value::as_str)
}

fn has_any_refs(value: &Value, fields: &[&str]) -> bool {
    fields.iter().any(|field| has_non_empty_array(value, field))
}

fn has_non_empty_array(value: &Value, field: &str) -> bool {
    value
        .get(field)
        .and_then(Value::as_array)
        .is_some_and(|items| {
            items
                .iter()
                .any(|item| item.as_str().is_some_and(|s| !s.trim().is_empty()))
        })
}

fn has_large_raw_content_key(value: &Value) -> bool {
    match value {
        Value::Object(object) => object.iter().any(|(key, value)| {
            let lowered = key.to_ascii_lowercase();
            matches!(
                lowered.as_str(),
                "raw_content" | "canonical_content" | "large_content" | "content_blob" | "raw_text"
            ) || has_large_raw_content_key(value)
        }),
        Value::Array(values) => values.iter().any(has_large_raw_content_key),
        _ => false,
    }
}

pub fn secret_like_findings(value: &Value) -> Vec<String> {
    let mut out = Vec::new();
    collect_secret_like_findings(value, "$", &mut out);
    out
}

fn collect_secret_like_findings(value: &Value, path: &str, out: &mut Vec<String>) {
    match value {
        Value::Object(object) => {
            for (key, value) in object {
                let child_path = format!("{path}.{key}");
                if is_secret_like_key(key) {
                    out.push(format!("secret-looking field rejected at {child_path}"));
                }
                collect_secret_like_findings(value, &child_path, out);
            }
        }
        Value::Array(values) => {
            for (index, value) in values.iter().enumerate() {
                collect_secret_like_findings(value, &format!("{path}[{index}]"), out);
            }
        }
        Value::String(value) => {
            if is_secret_like_value(value) {
                out.push(format!("secret-looking value rejected at {path}"));
            }
        }
        _ => {}
    }
}

fn is_secret_like_key(key: &str) -> bool {
    let lowered = key.to_ascii_lowercase();
    lowered.contains("api_key")
        || lowered.contains("secret")
        || lowered.contains("password")
        || lowered.contains("private_key")
        || lowered.contains("bearer")
        || lowered == "token"
        || lowered.ends_with("_token")
        || lowered.contains("plaintext")
}

fn is_secret_like_value(value: &str) -> bool {
    let lowered = value.to_ascii_lowercase();
    lowered.contains("api_key")
        || lowered.contains("secret")
        || lowered.contains("password")
        || lowered.contains("private_key")
        || lowered.contains("bearer ")
        || lowered.starts_with("bearer")
        || lowered.contains("plaintext")
}
