# Review Prompt: FORGE-HMK Authority Audit

Review the implementation for authority violations.

Check:

- Can FORGE-HMK write canonical memory?
- Can a worker write canonical memory?
- Can Crucible commit truth directly?
- Can cache/VSA/vector output become truth?
- Are public mutation routes exposed?
- Are claim envelopes required before promotion?
- Is Control Lane still the commit path?
- Are no-effect tests present?

Output: PASS/FAIL, critical findings, risky files, required fixes, and what not to do next.
