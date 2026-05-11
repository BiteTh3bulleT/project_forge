# FORGE Full Code Review + Public Architecture Markdown Report

You are acting as a senior principal engineer, systems architect, technical writer, and product reviewer.

Your job is to perform a complete, honest, production-grade review of the FORGE repository and produce one polished Markdown report.

FORGE stands for:

**F.O.R.G.E. — Foundry for Organic Reasoning, Growth, and Execution**

Tagline:

**Turn chaos into cognition. Turn cognition into action.**

FORGE should be treated as an AI-OS / cognitive runtime project, not merely a chatbot wrapper, script pile, or UI demo.

The final output must be a single Markdown file saved into the repository at:

```text
docs/reports/FORGE_FULL_REVIEW.md

Create the directory if it does not exist.

Primary Objective

Review the full repository and produce a clear Markdown report explaining:

What FORGE is.
What FORGE can currently do.
How FORGE works at a high level.
What systems, modules, and services exist.
What is wired, partially wired, stubbed, broken, duplicated, fragile, or missing.
What needs to be improved.
What should be refactored.
What should be prioritized next.
How FORGE can be described publicly without revealing private design logic or secret sauce.
Whether the repo is currently coherent enough to continue building on.

This should be a real engineering review, not motivational fluff.

Be direct. Be useful. Be surgical.

Important Privacy / Secret Sauce Rule

FORGE may contain unusual architecture ideas, naming conventions, cognitive models, symbolic systems, private project logic, or experimental reasoning structures.

Do not expose private mechanics, hidden formulas, unpublished internal doctrine, exact cognitive algorithms, or secret architectural details in the public-facing sections.

You may describe FORGE using safe abstraction, such as:

AI-OS
cognitive runtime
semantic operating layer
memory-aware orchestration system
local-first AI control plane
modular reasoning environment
event-driven runtime
workspace-oriented AI shell
deterministic state and context layer
agent-support infrastructure

Do not reveal exact private mechanisms unless they are already plainly documented as public-facing architecture.

When unsure, abstract.

Example:

Bad:

FORGE uses [private mechanism] to route cognition through [specific secret pattern].

Good:

FORGE uses a layered reasoning and state-management design to keep AI behavior more structured, inspectable, and resilient.

Review Scope

Inspect the full repository, including but not limited to:

Root files
README files
AGENTS.md
Documentation
Architecture docs
ADRs
Roadmaps
Source code
UI code
Server/backend code
API routes
CLI tools
Config files
Tests
Scripts
Build files
Docker/container files
CI/CD files
Package manifests
Generated or temporary files
Phase files
Any duplicated planning files
Any hidden architectural assumptions

Do not assume the documentation is accurate. Verify it against the code.

Required Final Markdown Structure

The final Markdown file must use this structure:

# FORGE Full Code Review

## 1. Executive Summary

## 2. What FORGE Is

## 3. What FORGE Can Do Today

## 4. How FORGE Works

## 5. Public-Safe Architecture Explanation

## 6. Repository Map

## 7. Major Systems and Modules

## 8. Backend Review

## 9. Frontend / UI Review

## 10. Runtime / Service Layer Review

## 11. Memory, State, and Context Review

## 12. API and Integration Review

## 13. Testing Review

## 14. Build, Packaging, and Deployment Review

## 15. Documentation Review

## 16. Security and Safety Review

## 17. Performance Review

## 18. Reliability Review

## 19. Code Quality Review

## 20. What Is Working

## 21. What Is Partially Working

## 22. What Is Stubbed or Placeholder

## 23. What Appears Broken

## 24. What Is Duplicated or Confusing

## 25. Architectural Risks

## 26. Refactor Recommendations

## 27. Server / Backend Split Plan

## 28. UI Improvement Plan

## 29. Documentation Cleanup Plan

## 30. Testing and CI Plan

## 31. Priority Punch List

## 32. Difficulty-Ranked Fix List

## 33. Suggested Next Build Phases

## 34. Public-Facing FORGE Description

## 35. One-Page Investor / Partner Safe Summary

## 36. Final Verdict

You may add subsections where useful, but do not remove any of the required sections.

Section Requirements
1. Executive Summary

Give a concise, honest summary of the repo’s current condition.

Include:

Overall project maturity
Whether it is coherent
Whether the architecture is visible in code
Whether the repo is prototype, alpha, or production-ready
The biggest strengths
The biggest problems
The next highest-leverage action
2. What FORGE Is

Explain FORGE as a product/system.

Use public-safe language.

Make it understandable to a smart technical person who has never seen the project.

FORGE should be described as an AI-OS / cognitive runtime / AI workbench / local-first AI control layer, depending on what the code actually supports.

Do not overclaim. Separate vision from current implementation.

Use this format:

FORGE is currently best understood as...
FORGE is intended to become...
FORGE is not...
3. What FORGE Can Do Today

List actual current capabilities verified from code.

Separate into:

### Confirmed Capabilities
### Partial Capabilities
### Planned But Not Yet Implemented

Only call something confirmed if it is backed by code.

4. How FORGE Works

Explain the actual runtime flow.

Cover, where applicable:

Startup path
Main entrypoints
Backend/server lifecycle
Frontend lifecycle
API request flow
State/memory flow
Config loading
Service initialization
Event/logging flow
Workspace/project flow
LLM/provider integration flow

Keep this technical but not secret-revealing.

5. Public-Safe Architecture Explanation

Write a version that could be shared publicly.

It should explain FORGE without exposing internal proprietary logic.

Include:

2-paragraph simple explanation
1 technical paragraph
1 “what makes it different” paragraph
1 “what it does not reveal” note
6. Repository Map

Create a tree-style summary of important directories and files.

For each important area, explain what it appears to be responsible for.

Flag:

Important files
Dead files
Duplicate files
Generated files
Temporary clutter
Files that should move
Files that should be documented
7. Major Systems and Modules

Identify the major components.

For each component, include:

### Component Name

**Location:**  
**Purpose:**  
**Current Status:** Working / Partial / Stub / Broken / Unknown  
**Key Files:**  
**Strengths:**  
**Problems:**  
**Recommended Action:**  
8. Backend Review

Review backend architecture.

Look for:

Monolithic files
Mixed responsibilities
Hardcoded config
Poor error handling
Missing interfaces
Weak routing structure
Unsafe assumptions
Missing lifecycle management
Inconsistent logging
Missing tests
Dead endpoints
Unused services

If there is a large server file, propose how to split it.

9. Frontend / UI Review

Review the UI.

Look for:

Component structure
State management
Routing
API usage
Desktop-shell design readiness
Theme consistency
Responsiveness
Error states
Loading states
Accessibility
Layout scalability
Overcoupling to backend assumptions

FORGE’s target UI direction is a desktop-like AI operating shell, not a generic SaaS dashboard.

Review against that standard.

10. Runtime / Service Layer Review

Review whether FORGE behaves like a runtime.

Look for:

Service registry
Daemon/process concepts
Event bus
Scheduler/job runner
Watchers
Internal services
Health checks
Lifecycle hooks
Startup/shutdown discipline
Runtime observability

Be clear about what exists versus what is only planned.

11. Memory, State, and Context Review

Review memory/state/context systems.

Look for:

Raw event storage
Semantic notes
Links
active state
open loops
derived models
context compilation
persistence
deterministic behavior
deduplication
audit trails
conflict/contradiction handling

Do not reveal private secret mechanics.

Describe safely.

12. API and Integration Review

Review:

API structure
Endpoint naming
Request/response models
Error handling
Provider integrations
LLM abstraction
External dependencies
Local model readiness
Future connector readiness

Flag brittle or unsafe integration patterns.

13. Testing Review

Inspect tests.

Report:

Existing tests
Missing tests
Broken tests
Test quality
Critical paths without coverage
Suggested unit tests
Suggested integration tests
Suggested smoke tests
Suggested UI tests

Include a proposed test matrix.

14. Build, Packaging, and Deployment Review

Review:

Build commands
Package manifests
Docker files
Compose files
Environment files
CI workflows
Local dev setup
Deployment assumptions
Cross-platform concerns
Nix/NixOS friendliness if visible
Tauri readiness if visible
15. Documentation Review

Review docs for:

Accuracy
Duplication
Staleness
Missing architecture docs
Missing setup docs
Missing user docs
Missing developer docs
Missing API docs
Missing diagrams
Misleading claims

Recommend a cleaned documentation structure.

16. Security and Safety Review

Review for:

Secrets in repo
Unsafe env handling
Command execution risks
File access risks
Prompt injection surfaces
Untrusted input handling
Authentication gaps
Authorization gaps
Localhost assumptions
Overbroad permissions
Dangerous defaults

Do not exploit anything. Just report.

17. Performance Review

Review likely performance bottlenecks.

Look for:

Blocking operations
Large synchronous work
Bad polling
Excessive file reads
Poor caching
frontend rendering inefficiencies
backend bottlenecks
startup overhead
memory growth risks
inefficient model calls
18. Reliability Review

Review:

Error handling
Recovery behavior
Restart behavior
Data durability
Corruption risks
Idempotency
Duplicate handling
logging quality
observability
graceful shutdown
degraded-mode behavior
19. Code Quality Review

Review:

Naming
File organization
Type discipline
Modularity
Coupling
Duplication
Dead code
Comments
Complexity
Consistency
Maintainability

Give concrete examples with file paths.

20. What Is Working

Create a clear list of working pieces.

Include evidence by file path.

21. What Is Partially Working

Create a clear list of partial pieces.

Explain what exists and what is missing.

22. What Is Stubbed or Placeholder

Create a clear list of stubs/placeholders/TODOs/mocked behavior.

Include file paths.

23. What Appears Broken

Create a clear list of broken or likely broken items.

For each:

### Issue

**Location:**  
**Problem:**  
**Impact:**  
**Suggested Fix:**  
**Difficulty:** Easy / Medium / Hard  
24. What Is Duplicated or Confusing

Identify:

Duplicate docs
Duplicate concepts
Duplicate config
Duplicate functions
Conflicting naming
Competing architecture descriptions
Old phase files
Root clutter
Files that should be archived
25. Architectural Risks

List the biggest risks.

Examples:

Backend monolith risk
Docs ahead of code
UI not aligned with runtime
No stable state contract
Too many concepts without enforcement
No deterministic lifecycle
Missing tests around core behavior

Use only risks supported by repo evidence.

26. Refactor Recommendations

Give a refactor plan.

Prioritize low-risk, high-impact cleanup first.

Use this format:

### Refactor Item

**Why:**  
**Files Affected:**  
**Expected Benefit:**  
**Risk:**  
**Difficulty:**  
27. Server / Backend Split Plan

If there is a large server file or backend monolith, propose a split.

Use a practical target structure like:

server/
  main.go
  config/
  routes/
  handlers/
  services/
  runtime/
  memory/
  workspace/
  providers/
  middleware/
  health/
  logging/
  models/

Adapt this to the actual language/framework in the repo.

For each proposed package/module, explain its responsibility.

Do not write implementation code unless explicitly necessary. This is a review document.

28. UI Improvement Plan

FORGE’s UI target is a desktop-like AI operating shell.

Review how close the current UI is to that target.

Recommend:

Shell layout
Window/workspace model
Navigation model
Command palette
Status bar
Service monitor
Memory/context inspector
Activity log
Settings/config surface
Theme system
Responsive behavior
Error/loading states

Keep it practical.

29. Documentation Cleanup Plan

Recommend a documentation structure like:

docs/
  README.md
  architecture/
  adr/
  build/
  dev/
  api/
  ui/
  runtime/
  memory/
  security/
  reports/
  archive/

Move obsolete or duplicate planning docs into archive recommendations.

30. Testing and CI Plan

Recommend the next testing setup.

Include:

Unit tests
Integration tests
API tests
UI tests
Smoke tests
Linting
Formatting
Type checks
CI checks
Local developer test command
31. Priority Punch List

Create a punch list ordered by priority.

Use this format:

| Priority | Task | Area | Why It Matters | Difficulty |
|---|---|---|---|---|

Priorities:

P0: Critical
P1: High
P2: Medium
P3: Nice-to-have
32. Difficulty-Ranked Fix List

Create another list ordered by difficulty:

## Easy Fixes
## Medium Fixes
## Hard Fixes
## Deep Architecture Work
33. Suggested Next Build Phases

Create a practical phased roadmap.

Use:

### Phase 0 — Stabilize and Map
### Phase 1 — Split and Structure
### Phase 2 — Runtime Hardening
### Phase 3 — Memory/State Contract
### Phase 4 — UI Shell Alignment
### Phase 5 — Testing and CI
### Phase 6 — Packaging and Deployment

Adapt if needed.

Each phase must include:

Goal
Deliverables
Definition of done
Risks
34. Public-Facing FORGE Description

Write a polished description suitable for GitHub README, LinkedIn, or a project page.

Do not reveal secret sauce.

Include:

### Short Version
### Technical Version
### Builder Version
### What Makes FORGE Different
35. One-Page Investor / Partner Safe Summary

Write a concise one-page summary that explains FORGE’s value without exposing proprietary architecture.

Include:

Problem
Solution
Why now
What FORGE does
Technical advantage, safely abstracted
Current status
Next milestones
Why it matters
36. Final Verdict

End with a blunt but fair verdict.

Use this format:

FORGE is currently...
The strongest part of the project is...
The weakest part of the project is...
The highest-leverage next move is...
The project should not proceed to [X] until...
Overall verdict:
Evidence Requirements

Use file paths heavily.

When making claims, cite the relevant file path.

Good:

The backend currently appears centered around `server.go`, which combines route setup, runtime behavior, and service logic.

Bad:

The backend is messy.

Be specific. Be useful.

Output Rules
Create or update only this file unless you need to create the parent directory:
docs/reports/FORGE_FULL_REVIEW.md
Do not modify source code.
Do not refactor files.
Do not delete files.
Do not create implementation patches.
Do not expose private architecture details.
Do not invent capabilities.
Do not overpraise.
Do not be vague.
Do not skip uncomfortable findings.
Tone

Use a tone that is:

Senior
Direct
Technically serious
Practical
Builder-friendly
Public-safe
Slightly sharp when needed

No corporate fog machine.

What Not To Do

Do not:

Reveal secret sauce.
Dump private symbolic/project logic into the report.
Describe hidden reasoning systems in exact detail.
Claim FORGE is production-ready unless the repo proves it.
Rewrite the code.
Create multiple scattered reports.
Ignore TODOs, stubs, or broken files.
Treat docs as truth without checking code.
Produce a shallow README-style summary.
Turn this into hype.
Use generic statements without file evidence.
Remove or alter project files.
Make architecture recommendations that ignore the existing repo.
Say “everything looks good” unless it actually does, which it probably does not. Be brave.
Working Method

Before writing the final report:

Inspect the repository structure.
Read the major documentation files.
Identify the main runtime/backend entrypoints.
Identify the frontend/UI structure.
Identify config/build/test files.
Search for TODO, FIXME, stub, placeholder, mock, panic, console.log, hardcoded, secret, token, password, API key, and deprecated.
Check whether tests exist and whether they map to core behavior.
Compare stated architecture against implemented code.
Build a factual punch list.
Then write the Markdown report.
Final Instruction

Produce the final Markdown report at:

docs/reports/FORGE_FULL_REVIEW.md

The report should be detailed enough that a second engineer could use it as a refactor and stabilization plan without needing this prompt.
