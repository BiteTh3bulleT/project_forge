use serde_json::{Map, Value};

use crate::errors::ValidateError;

pub fn parse_json(input: &str) -> Result<Value, ValidateError> {
    Ok(serde_json::from_str(input)?)
}

pub fn canonical_json(value: &Value) -> Result<String, ValidateError> {
    let canonical = canonicalize_value(value);
    Ok(serde_json::to_string_pretty(&canonical)?)
}

pub fn canonicalize_value(value: &Value) -> Value {
    canonicalize_inner(value, None)
}

fn canonicalize_inner(value: &Value, parent_key: Option<&str>) -> Value {
    match value {
        Value::Object(object) => {
            let mut keys: Vec<&String> = object.keys().collect();
            keys.sort();
            let mut out = Map::new();
            for key in keys {
                if let Some(value) = object.get(key) {
                    out.insert(key.clone(), canonicalize_inner(value, Some(key)));
                }
            }
            Value::Object(out)
        }
        Value::Array(values) => {
            let mut out: Vec<Value> = values
                .iter()
                .map(|value| canonicalize_inner(value, parent_key))
                .collect();
            if parent_key.map(is_unordered_array_key).unwrap_or(false) {
                out.sort_by(|left, right| stable_sort_key(left).cmp(&stable_sort_key(right)));
            }
            Value::Array(out)
        }
        Value::String(value) => Value::String(normalize_whitespace(value)),
        _ => value.clone(),
    }
}

pub fn normalize_whitespace(value: &str) -> String {
    value.split_whitespace().collect::<Vec<_>>().join(" ")
}

pub fn stable_sort_key(value: &Value) -> String {
    serde_json::to_string(value).unwrap_or_default()
}

pub fn is_unordered_array_key(key: &str) -> bool {
    key.ends_with("_refs")
        || key == "source_refs"
        || key == "blocks_refs"
        || key == "supported_models"
        || key == "supported_capabilities"
        || key == "allowed_syscalls"
        || key == "workspace_scope"
        || key == "provenance_refs"
        || key == "context_block_refs"
}

pub fn remove_stable_excluded_fields(value: &Value) -> Value {
    let mut out = canonicalize_value(value);
    remove_excluded_recursive(&mut out);
    out
}

fn remove_excluded_recursive(value: &mut Value) {
    match value {
        Value::Object(object) => {
            for key in [
                "created_at",
                "updated_at",
                "sealed_at",
                "expired_at",
                "last_used_at",
                "invalidated_at",
                "journal_refs",
                "shape_hash",
                "source_hash",
                "content_hash",
                "token_input_hash",
                "bundle_hash",
                "stable_prefix_hash",
                "volatile_suffix_hash",
                "cache_id",
                "snapshot_id",
                "block_id",
                "bundle_id",
                "driver_id",
                "reuse_count",
            ] {
                object.remove(key);
            }
            for value in object.values_mut() {
                remove_excluded_recursive(value);
            }
        }
        Value::Array(values) => {
            for value in values {
                remove_excluded_recursive(value);
            }
        }
        _ => {}
    }
}
