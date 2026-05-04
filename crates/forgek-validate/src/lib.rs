pub mod canonical;
pub mod errors;
pub mod hash;
pub mod types;
pub mod validate;

pub use canonical::{canonical_json, canonicalize_value, parse_json};
pub use errors::ValidateError;
pub use hash::{
    hash_canonical_json, hash_context_block, hash_context_bundle, hash_kv_manifest_identity,
    hash_runtime_manifest, hash_snapshot_shape,
};
pub use validate::{validate_manifest, validate_manifest_or_error, ValidationReport};

#[cfg(test)]
mod tests {
    use serde_json::Value;

    use super::*;

    fn fixture(path: &str) -> Value {
        let full = format!("../../fixtures/forgek/{path}");
        parse_json(std::fs::read_to_string(full).unwrap().as_str()).unwrap()
    }

    #[test]
    fn valid_snapshot_fixture_passes() {
        let report = validate_manifest(&fixture("valid/snapshot_manifest.valid.json")).unwrap();
        assert!(report.valid, "{:?}", report.errors);
        assert_eq!(report.kind, "snapshot_manifest");
    }

    #[test]
    fn invalid_snapshot_missing_workspace_fails() {
        let report =
            validate_manifest(&fixture("invalid/snapshot_missing_workspace.invalid.json")).unwrap();
        assert!(!report.valid);
        assert!(report
            .errors
            .iter()
            .any(|error| error.contains("workspace_id")));
    }

    #[test]
    fn context_block_fixture_passes_and_missing_refs_fails() {
        let report = validate_manifest(&fixture("valid/context_block.valid.json")).unwrap();
        assert!(report.valid, "{:?}", report.errors);

        let invalid = validate_manifest(&fixture(
            "invalid/context_block_missing_source_refs.invalid.json",
        ))
        .unwrap();
        assert!(!invalid.valid);
        assert!(invalid
            .errors
            .iter()
            .any(|error| error.contains("provenance refs")));
    }

    #[test]
    fn context_bundle_fixture_passes() {
        let report = validate_manifest(&fixture("valid/context_bundle.valid.json")).unwrap();
        assert!(report.valid, "{:?}", report.errors);
    }

    #[test]
    fn kv_manifest_fixtures_validate() {
        let report = validate_manifest(&fixture("valid/kv_cache_manifest.valid.json")).unwrap();
        assert!(report.valid, "{:?}", report.errors);

        let missing = validate_manifest(&fixture(
            "invalid/kv_manifest_missing_token_hash.invalid.json",
        ))
        .unwrap();
        assert!(!missing.valid);
        assert!(missing
            .errors
            .iter()
            .any(|error| error.contains("token_input_hash")));

        let bad_mode =
            validate_manifest(&fixture("invalid/kv_manifest_bad_cache_mode.invalid.json")).unwrap();
        assert!(!bad_mode.valid);
        assert!(bad_mode
            .errors
            .iter()
            .any(|error| error.contains("cache_mode")));
    }

    #[test]
    fn runtime_manifest_secret_field_fails() {
        let report =
            validate_manifest(&fixture("valid/runtime_driver_manifest.valid.json")).unwrap();
        assert!(report.valid, "{:?}", report.errors);

        let invalid = validate_manifest(&fixture(
            "invalid/runtime_driver_manifest_with_secret.invalid.json",
        ))
        .unwrap();
        assert!(!invalid.valid);
        assert!(invalid
            .errors
            .iter()
            .any(|error| error.contains("secret-looking")));
    }

    #[test]
    fn canonicalization_is_stable_for_key_and_ref_order() {
        let left = parse_json(
            r#"{"source_refs":["b","a"],"workspace_id":"ws","snapshot_type":"CASE_SNAPSHOT"}"#,
        )
        .unwrap();
        let right = parse_json(
            r#"{"snapshot_type":"CASE_SNAPSHOT","workspace_id":"ws","source_refs":["a","b"]}"#,
        )
        .unwrap();
        assert_eq!(
            canonical_json(&left).unwrap(),
            canonical_json(&right).unwrap()
        );
    }

    #[test]
    fn created_at_does_not_affect_snapshot_shape_hash() {
        let mut left = fixture("valid/snapshot_manifest.valid.json");
        let mut right = left.clone();
        left["created_at"] = Value::String("2026-05-04T00:00:00Z".to_string());
        right["created_at"] = Value::String("2027-01-01T00:00:00Z".to_string());
        assert_eq!(
            hash_snapshot_shape(&left).unwrap(),
            hash_snapshot_shape(&right).unwrap()
        );
    }

    #[test]
    fn hashes_match_golden_file() {
        let golden = fixture("golden/hashes.json");
        assert_eq!(
            hash_snapshot_shape(&fixture("valid/snapshot_manifest.valid.json")).unwrap(),
            golden["snapshot_shape_hash"].as_str().unwrap()
        );
        assert_eq!(
            hash_context_block(&fixture("valid/context_block.valid.json")).unwrap(),
            golden["context_block_hash"].as_str().unwrap()
        );
        assert_eq!(
            hash_kv_manifest_identity(&fixture("valid/kv_cache_manifest.valid.json")).unwrap(),
            golden["kv_manifest_identity_hash"].as_str().unwrap()
        );
    }

    #[test]
    fn cli_validate_success_and_failure_codes_are_deterministic() {
        assert_eq!(
            crate::main_support::run_cli([
                "forgek-validate",
                "validate",
                "../../fixtures/forgek/valid/snapshot_manifest.valid.json",
            ]),
            0
        );
        assert_eq!(
            crate::main_support::run_cli([
                "forgek-validate",
                "validate",
                "../../fixtures/forgek/invalid/snapshot_missing_workspace.invalid.json",
            ]),
            1
        );
    }
}

pub mod main_support {
    use std::fs;
    use std::path::{Path, PathBuf};

    use serde::Serialize;
    use serde_json::{json, Value};

    use crate::canonical::{canonical_json, parse_json};
    use crate::errors::ValidateError;
    use crate::hash::hash_canonical_json;
    use crate::validate::{validate_manifest, validate_manifest_or_error};

    pub fn run_cli<I, S>(args: I) -> i32
    where
        I: IntoIterator<Item = S>,
        S: AsRef<str>,
    {
        match run_cli_result(args) {
            Ok(output) => {
                if !output.is_empty() {
                    println!("{output}");
                }
                0
            }
            Err(error) => {
                eprintln!("{error}");
                1
            }
        }
    }

    pub fn run_cli_result<I, S>(args: I) -> Result<String, ValidateError>
    where
        I: IntoIterator<Item = S>,
        S: AsRef<str>,
    {
        let args: Vec<String> = args
            .into_iter()
            .map(|arg| arg.as_ref().to_string())
            .collect();
        let command = args
            .get(1)
            .map(String::as_str)
            .ok_or_else(|| usage_error())?;
        match command {
            "validate" => {
                let value = read_json_arg(&args, 2)?;
                let report = validate_manifest_or_error(&value)?;
                json_pretty(&report)
            }
            "canonicalize" => {
                let value = read_json_arg(&args, 2)?;
                canonical_json(&value)
            }
            "hash" => {
                let value = read_json_arg(&args, 2)?;
                json_pretty(&json!({ "sha256": hash_canonical_json(&value)? }))
            }
            "validate-fixtures" => {
                let dir = args.get(2).ok_or_else(|| {
                    ValidateError::Cli("validate-fixtures requires a directory".to_string())
                })?;
                validate_fixtures(Path::new(dir))
            }
            _ => Err(usage_error()),
        }
    }

    fn read_json_arg(args: &[String], index: usize) -> Result<Value, ValidateError> {
        let path = args
            .get(index)
            .ok_or_else(|| ValidateError::Cli("command requires a file path".to_string()))?;
        parse_json(&fs::read_to_string(path)?)
    }

    fn validate_fixtures(dir: &Path) -> Result<String, ValidateError> {
        let mut files = Vec::new();
        collect_json_files(dir, &mut files)?;
        files.sort();
        let mut valid_expected = 0usize;
        let mut invalid_expected = 0usize;
        let mut failures = Vec::new();
        for file in files {
            let is_golden = file.components().any(|part| part.as_os_str() == "golden");
            if is_golden {
                continue;
            }
            let text = fs::read_to_string(&file)?;
            let value = parse_json(&text)?;
            let report = validate_manifest(&value)?;
            let expects_invalid = file.components().any(|part| part.as_os_str() == "invalid");
            if expects_invalid {
                invalid_expected += 1;
                if report.valid {
                    failures.push(format!("{} unexpectedly valid", file.display()));
                }
            } else {
                valid_expected += 1;
                if !report.valid {
                    failures.push(format!(
                        "{} invalid: {}",
                        file.display(),
                        report.errors.join("; ")
                    ));
                }
            }
        }
        if !failures.is_empty() {
            return Err(ValidateError::Cli(failures.join("\n")));
        }
        json_pretty(&json!({
            "valid_expected": valid_expected,
            "invalid_expected": invalid_expected,
            "status": "passed"
        }))
    }

    fn collect_json_files(dir: &Path, out: &mut Vec<PathBuf>) -> Result<(), ValidateError> {
        for entry in fs::read_dir(dir)? {
            let entry = entry?;
            let path = entry.path();
            if path.is_dir() {
                collect_json_files(&path, out)?;
            } else if path
                .extension()
                .is_some_and(|extension| extension == "json")
            {
                out.push(path);
            }
        }
        Ok(())
    }

    fn json_pretty<T: Serialize>(value: &T) -> Result<String, ValidateError> {
        Ok(serde_json::to_string_pretty(value)?)
    }

    fn usage_error() -> ValidateError {
        ValidateError::Cli(
            "usage: forgek-validate <validate|canonicalize|hash|validate-fixtures> <path>"
                .to_string(),
        )
    }
}
