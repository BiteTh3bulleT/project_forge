# Codex Execution Harness — M5A

You are Codex working inside `BiteTh3bulleT/project_forge`.

Do not output code in chat. Make all changes in files.

Your task is to implement M5A: Authority Matrix + Runtime Policy Convergence + First Latency Foundation.

Read the required authority docs and files listed in `00_MASTER_PROMPT_M5A.md`.

Implement in this order:

1. Create `docs/reviews/m5a_authority_convergence_review.md`.
2. Add machine-readable authority matrix with tests.
3. Fix `model.delete_file` registry/runtime drift.
4. Make `model.chat` and `model.generate` authority posture explicit.
5. Add Control Lane approval fingerprint seam and tests.
6. Add HostBridge/FORGE-H snapshot cache or TTL wrapper and tests.
7. Extend read-only System status/cockpit with authority summary.
8. Add `docs/architecture/micro_agent_acceleration.md`.
9. Add/update validation and latency baseline docs.

Hard boundaries:
- No FORGE-K live authority expansion.
- No host mutation.
- No UI mutation buttons.
- No new model execution side channel.
- No direct `systemctl`, `nixos-rebuild`, reboot, shutdown, package manager, kernel/module, or destructive cleanup paths.
- No canonical memory writes from micro-agents.
- No raw logs/prompts/secrets in System status.

Run validation and document results honestly.
