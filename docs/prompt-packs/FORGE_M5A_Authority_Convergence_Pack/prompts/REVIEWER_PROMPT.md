# Reviewer Prompt — M5A

You are reviewing an implementation of FORGE M5A.

Do not rewrite code unless asked. Produce a review report.

## Review scope

Check whether the implementation preserves FORGE doctrine, avoids live authority expansion, fixes modelruntime/gateway drift, correctly aligns `model.delete_file`, makes chat/generate policy explicit, adds/test-covers authority matrix, adds Control Lane approval fingerprint seam, adds HostBridge/FORGE-H snapshot cache without mutation, extends System Cockpit read-only display only, adds micro-agent acceleration design with no authority bypass, updates docs honestly, and records tests.

## Look especially for

- hidden mutation routes,
- direct shell/system calls,
- model calls from background workers,
- raw prompts/logs/secrets in status payloads,
- authority labels that say healthy when unknown,
- duplicate approval/audit systems,
- cached data treated as truth,
- missing tests for registry drift,
- stale docs.

## Output format

- Executive verdict
- Blocking issues
- High-priority issues
- Medium issues
- Low issues
- Test gaps
- Docs gaps
- Merge recommendation
