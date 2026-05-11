# Academy Dream Mode Integration

Status date: 2026-05-11.

FORGE Academy owns curriculum. Dream Mode owns dry-run learning simulation.

Academy curriculum may include skill books, lesson prompts, labs, exams, grading rubrics, and promotion policy. Dream Mode may study, practice, test, and persist evidence reports against that curriculum, but those reports are non-canonical evidence only.

## Authority Boundary

Dream Mode can:

- run `academy_study`, `academy_lab`, `academy_exam`, `academy_refresh`, and `academy_promotion_candidate` dry-runs
- carry `skillId`, `lessonId`, `labId`, and `examId` identifiers in evidence reports
- record study summaries, safety rules, lab results, exam answers, score/confidence fields, warnings, and promotion candidates
- persist reports in `dream_reports` as `non_canonical_evidence`

Dream Mode cannot:

- write canonical memory directly
- mark a skill promoted by itself
- bypass Courthouse/operator review
- bypass Control Lane promotion
- run model fine-tuning
- execute tools, retrieval, embeddings, or modelruntime outside existing Dream Mode policy

## Promotion Pipeline

1. Academy defines the skill curriculum, labs, exam, rubric, and promotion policy.
2. Dream Mode runs a dry-run study, lab, exam, refresh, or promotion-candidate pass.
3. Dream Mode persists the report as non-canonical evidence when requested.
4. Courthouse/operator review grades or rejects the evidence.
5. Control Lane owns any canonical skill-memory promotion.

Failed or ungraded exams produce remediation evidence. They do not create promoted skills.

## Current Fallback

If no wired Academy curriculum source exists, Dream Mode records an honest unavailable state in the report warning fields. It does not invent lesson contents, lab results, exam answers, or grades.

## Context Boundary

Academy Dream reports are operator-visible structured evidence. They are not a shortcut into LLM context. LLM-facing context still goes through the context compiler and governed retrieval paths.
