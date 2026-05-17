# Reviewer Prompt — M5S

Block merge if:
- any non-health route is unauthenticated,
- wildcard bind works without auth,
- Docker ships unauth wildcard bind,
- approval body actor grants authority,
- project-context can import out-of-root files,
- generated tokens leak in logs/status,
- high-risk approval can be anonymously/self approved.

Review tests, docs, and runtime behavior.
