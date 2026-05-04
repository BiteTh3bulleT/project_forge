use serde_json::Value;
use sha2::{Digest, Sha256};

use crate::canonical::{canonical_json, remove_stable_excluded_fields};
use crate::errors::ValidateError;

pub fn sha256_hex(input: &str) -> String {
    let mut hasher = Sha256::new();
    hasher.update(input.as_bytes());
    let digest = hasher.finalize();
    format!("{digest:x}")
}

pub fn hash_canonical_json(value: &Value) -> Result<String, ValidateError> {
    Ok(sha256_hex(&canonical_json(value)?))
}

pub fn hash_stable_projection(value: &Value) -> Result<String, ValidateError> {
    let projected = remove_stable_excluded_fields(value);
    hash_canonical_json(&projected)
}

pub fn hash_snapshot_shape(value: &Value) -> Result<String, ValidateError> {
    hash_stable_projection(value)
}

pub fn hash_context_block(value: &Value) -> Result<String, ValidateError> {
    hash_stable_projection(value)
}

pub fn hash_context_bundle(value: &Value) -> Result<String, ValidateError> {
    hash_stable_projection(value)
}

pub fn hash_kv_manifest_identity(value: &Value) -> Result<String, ValidateError> {
    hash_stable_projection(value)
}

pub fn hash_runtime_manifest(value: &Value) -> Result<String, ValidateError> {
    hash_stable_projection(value)
}
