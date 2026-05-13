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
    use std::path::{Path, PathBuf};

    use serde_json::json;
    use serde_json::Value;

    use super::*;

    fn fixture(path: &str) -> Value {
        let full = format!("../../fixtures/forgek/{path}");
        parse_json(std::fs::read_to_string(full).unwrap().as_str()).unwrap()
    }

    fn fixture_text(path: &str) -> String {
        let full = format!("../../fixtures/forgek/{path}");
        std::fs::read_to_string(full)
            .unwrap()
            .trim_end_matches(['\r', '\n'])
            .to_string()
    }

    fn fixture_files(path: &str) -> Vec<PathBuf> {
        let root = Path::new("../../fixtures/forgek").join(path);
        let mut files = Vec::new();
        for entry in std::fs::read_dir(root).unwrap() {
            let entry = entry.unwrap();
            let path = entry.path();
            if path
                .extension()
                .is_some_and(|extension| extension == "json")
            {
                files.push(path);
            }
        }
        files.sort();
        files
    }

    fn fixture_label(path: &Path) -> String {
        path.strip_prefix("../../fixtures/forgek")
            .unwrap()
            .to_string_lossy()
            .trim_start_matches('/')
            .to_string()
    }

    fn compact_canonical_json(value: &Value) -> String {
        serde_json::to_string(&canonicalize_value(value)).unwrap()
    }

    #[test]
    fn every_valid_fixture_passes() {
        let files = fixture_files("valid");
        assert!(!files.is_empty());
        for file in files {
            let value = parse_json(&std::fs::read_to_string(&file).unwrap()).unwrap();
            let report = validate_manifest(&value).unwrap();
            assert!(
                report.valid,
                "{} invalid: {:?}",
                fixture_label(&file),
                report.errors
            );
            assert!(
                !report.hashes.is_empty(),
                "{} produced no validation hash",
                fixture_label(&file)
            );
        }
    }

    #[test]
    fn every_invalid_fixture_is_rejected() {
        let files = fixture_files("invalid");
        assert!(!files.is_empty());
        for file in files {
            let value = parse_json(&std::fs::read_to_string(&file).unwrap()).unwrap();
            let report = validate_manifest(&value).unwrap();
            assert!(!report.valid, "{} unexpectedly valid", fixture_label(&file));
            assert!(
                !report.errors.is_empty(),
                "{} failed without diagnostic errors",
                fixture_label(&file)
            );
        }
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
    fn excluded_timestamps_do_not_affect_stable_hashes() {
        let mut left = fixture("valid/context_block.valid.json");
        let mut right = left.clone();
        left["created_at"] = Value::String("2026-05-04T00:00:00Z".to_string());
        left["updated_at"] = Value::String("2026-05-04T00:01:00Z".to_string());
        right["created_at"] = Value::String("2027-01-01T00:00:00Z".to_string());
        right["updated_at"] = Value::String("2027-01-01T00:01:00Z".to_string());
        assert_eq!(
            hash_context_block(&left).unwrap(),
            hash_context_block(&right).unwrap()
        );
    }

    #[test]
    fn stable_identity_fields_affect_hashes() {
        let mut changed = fixture("valid/snapshot_manifest.valid.json");
        let original = hash_snapshot_shape(&changed).unwrap();
        changed["workspace_id"] = Value::String("workspace-b".to_string());
        assert_ne!(original, hash_snapshot_shape(&changed).unwrap());
    }

    #[test]
    fn missing_refs_fail_validation() {
        let mut value = fixture("valid/context_block.valid.json");
        value.as_object_mut().unwrap().remove("source_object_refs");
        value.as_object_mut().unwrap().remove("source_refs");
        value
            .as_object_mut()
            .unwrap()
            .remove("admitted_exhibit_refs");
        let report = validate_manifest(&value).unwrap();
        assert!(!report.valid);
        assert!(report
            .errors
            .iter()
            .any(|error| error.contains("provenance refs")));
    }

    #[test]
    fn secret_looking_runtime_fields_fail_validation() {
        let mut value = fixture("valid/runtime_driver_manifest.valid.json");
        value["runtime_api_key"] = Value::String("redacted".to_string());
        let report = validate_manifest(&value).unwrap();
        assert!(!report.valid);
        assert!(report
            .errors
            .iter()
            .any(|error| error.contains("secret-looking")));
    }

    #[test]
    fn kv_runtime_assumption_changes_affect_identity_hash() {
        let mut changed = fixture("valid/kv_cache_manifest.valid.json");
        let original = hash_kv_manifest_identity(&changed).unwrap();
        changed["runtime_backend"] = Value::String("mock-v2".to_string());
        assert_ne!(original, hash_kv_manifest_identity(&changed).unwrap());
    }

    #[test]
    fn canonical_golden_files_match_valid_fixtures() {
        for (fixture_path, golden_path) in [
            (
                "valid/snapshot_manifest.valid.json",
                "golden/canonical_snapshot_manifest.json",
            ),
            (
                "valid/context_block.valid.json",
                "golden/canonical_context_block.json",
            ),
            (
                "valid/kv_cache_manifest.valid.json",
                "golden/canonical_kv_manifest.json",
            ),
        ] {
            assert_eq!(
                compact_canonical_json(&fixture(fixture_path)),
                fixture_text(golden_path),
                "{fixture_path} canonical golden drifted"
            );
        }
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
    fn validate_fixtures_compares_existing_golden_hash_manifest() {
        assert_eq!(
            crate::main_support::run_cli([
                "forgek-validate",
                "validate-fixtures",
                "../../fixtures/forgek",
            ]),
            0
        );
    }

    #[test]
    fn expanded_hashes_manifest_shape_is_supported() {
        let manifest = json!({
            "schema_version": "forgek.hashes.v1",
            "fixtures": [
                {
                    "fixture_path": "valid/snapshot_manifest.valid.json",
                    "hash_kind": "snapshot_shape_hash",
                    "expected_hash": hash_snapshot_shape(&fixture("valid/snapshot_manifest.valid.json")).unwrap(),
                    "included_fields_summary": ["workspace_id", "snapshot_type", "source refs"],
                    "excluded_fields_summary": ["timestamps", "generated ids", "stored hash fields"]
                }
            ]
        });
        crate::main_support::compare_hash_manifest(&manifest, Path::new("../../fixtures/forgek"))
            .unwrap();
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
    use std::collections::BTreeMap;
    use std::fs;
    use std::path::{Path, PathBuf};

    use serde::Serialize;
    use serde_json::{json, Map, Value};

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
        let golden_hash_entries = compare_golden_hashes_if_present(dir)?;
        json_pretty(&json!({
            "valid_expected": valid_expected,
            "invalid_expected": invalid_expected,
            "golden_hash_entries": golden_hash_entries,
            "status": "passed"
        }))
    }

    fn compare_golden_hashes_if_present(dir: &Path) -> Result<usize, ValidateError> {
        let path = dir.join("golden").join("hashes.json");
        if !path.exists() {
            return Ok(0);
        }
        let manifest = parse_json(&fs::read_to_string(path)?)?;
        compare_hash_manifest(&manifest, dir)
    }

    pub(crate) fn compare_hash_manifest(
        manifest: &Value,
        fixture_root: &Path,
    ) -> Result<usize, ValidateError> {
        if manifest.get("fixtures").is_some() {
            compare_expanded_hash_manifest(manifest, fixture_root)
        } else {
            compare_flat_hash_manifest(manifest, fixture_root)
        }
    }

    fn compare_flat_hash_manifest(
        manifest: &Value,
        fixture_root: &Path,
    ) -> Result<usize, ValidateError> {
        let object = manifest.as_object().ok_or_else(|| {
            ValidateError::Cli("golden hashes.json must be an object".to_string())
        })?;
        let current = hash_reports_by_kind(fixture_root)?;
        let mut failures = Vec::new();
        let mut compared = 0usize;
        for (hash_kind, expected) in object {
            compared += 1;
            let expected = expected.as_str().ok_or_else(|| {
                ValidateError::Cli(format!("golden hash {hash_kind} must be a string"))
            })?;
            match current.get(hash_kind) {
                Some(actual) if actual == expected => {}
                Some(actual) => failures.push(format!(
                    "golden hash drift for {hash_kind}: expected {expected}, got {actual}"
                )),
                None => failures.push(format!("golden hash {hash_kind} has no matching fixture")),
            }
        }
        if failures.is_empty() {
            Ok(compared)
        } else {
            Err(ValidateError::Cli(failures.join("\n")))
        }
    }

    fn compare_expanded_hash_manifest(
        manifest: &Value,
        fixture_root: &Path,
    ) -> Result<usize, ValidateError> {
        let entries = manifest
            .get("fixtures")
            .and_then(Value::as_array)
            .ok_or_else(|| {
                ValidateError::Cli("expanded hashes.json requires fixtures array".to_string())
            })?;
        if entries.is_empty() {
            return Err(ValidateError::Cli(
                "expanded hashes.json fixtures array must not be empty".to_string(),
            ));
        }

        let mut failures = Vec::new();
        for (index, entry) in entries.iter().enumerate() {
            match compare_expanded_hash_entry(entry, fixture_root) {
                Ok(()) => {}
                Err(error) => failures.push(format!("hashes.json fixtures[{index}]: {error}")),
            }
        }
        if failures.is_empty() {
            Ok(entries.len())
        } else {
            Err(ValidateError::Cli(failures.join("\n")))
        }
    }

    fn compare_expanded_hash_entry(
        entry: &Value,
        fixture_root: &Path,
    ) -> Result<(), ValidateError> {
        let object = entry.as_object().ok_or_else(|| {
            ValidateError::Cli("expanded hash entry must be an object".to_string())
        })?;
        for field in [
            "fixture_path",
            "hash_kind",
            "expected_hash",
            "included_fields_summary",
            "excluded_fields_summary",
        ] {
            if !object.contains_key(field) {
                return Err(ValidateError::Cli(format!("{field} is required")));
            }
        }
        require_string_entry(object, "fixture_path")?;
        let hash_kind = require_string_entry(object, "hash_kind")?;
        let expected_hash = require_string_entry(object, "expected_hash")?;
        require_non_empty_summary_entry(object, "included_fields_summary")?;
        require_non_empty_summary_entry(object, "excluded_fields_summary")?;

        let fixture_path = require_string_entry(object, "fixture_path")?;
        let value = parse_json(&fs::read_to_string(resolve_fixture_path(
            fixture_root,
            fixture_path,
        ))?)?;
        let report = validate_manifest_or_error(&value)?;
        let actual = report
            .hashes
            .get(hash_kind)
            .and_then(Value::as_str)
            .ok_or_else(|| {
                ValidateError::Cli(format!("{fixture_path} did not produce {hash_kind}"))
            })?;
        if actual != expected_hash {
            return Err(ValidateError::Cli(format!(
                "{fixture_path} {hash_kind} drift: expected {expected_hash}, got {actual}"
            )));
        }
        Ok(())
    }

    fn require_string_entry<'a>(
        object: &'a Map<String, Value>,
        field: &str,
    ) -> Result<&'a str, ValidateError> {
        object
            .get(field)
            .and_then(Value::as_str)
            .filter(|value| !value.trim().is_empty())
            .ok_or_else(|| ValidateError::Cli(format!("{field} must be a non-empty string")))
    }

    fn require_non_empty_summary_entry(
        object: &Map<String, Value>,
        field: &str,
    ) -> Result<(), ValidateError> {
        let valid = match object.get(field) {
            Some(Value::String(value)) => !value.trim().is_empty(),
            Some(Value::Array(values)) => {
                !values.is_empty()
                    && values
                        .iter()
                        .all(|value| value.as_str().is_some_and(|value| !value.trim().is_empty()))
            }
            Some(Value::Object(values)) => !values.is_empty(),
            _ => false,
        };
        if valid {
            Ok(())
        } else {
            Err(ValidateError::Cli(format!(
                "{field} must be a non-empty string, string array, or object"
            )))
        }
    }

    fn resolve_fixture_path(fixture_root: &Path, fixture_path: &str) -> PathBuf {
        if let Some(relative) = fixture_path.strip_prefix("fixtures/forgek/") {
            fixture_root.join(relative)
        } else {
            fixture_root.join(fixture_path)
        }
    }

    fn hash_reports_by_kind(
        fixture_root: &Path,
    ) -> Result<BTreeMap<String, String>, ValidateError> {
        let mut files = Vec::new();
        collect_json_files(&fixture_root.join("valid"), &mut files)?;
        files.sort();
        let mut out = BTreeMap::new();
        for file in files {
            let value = parse_json(&fs::read_to_string(file)?)?;
            let report = validate_manifest_or_error(&value)?;
            for (kind, value) in report.hashes {
                if let Some(hash) = value.as_str() {
                    out.insert(kind, hash.to_string());
                }
            }
        }
        Ok(out)
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
