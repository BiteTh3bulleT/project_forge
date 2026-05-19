# Phase G8 Desktop Shell Verification Status

Status date: 2026-05-18

Phase label: `PHASE G8 - DESKTOP_SHELL_VERIFICATION_AND_OPERATOR_UX_POLISH`

Status marker: `DESKTOP_SHELL_POLISH / OPERATOR_UI_AUTHORITY / LABWC_SUBSTRATE_PRESERVED / NO_HOST_MUTATION / NO_FORGE_K_AUTHORITY_EXPANSION`

## Summary

Phase G8 is partially closed for code-level desktop shell polish. Native launch placeholders now honor refused launches, block duplicate pending requests, expire stale placeholders, and surface bounded compositor-action failures. The VM has current evidence for boot/login/empty-desktop/core/modelruntime reachability, but the full manual native-app smoke matrix still needs an operator pass.

## Verification Matrix

| Behavior | Status | Evidence | Open work |
|---|---|---|---|
| FORGE loading/login path reaches desktop | verified | `docs/evidence/vm_boot/2026-05-18-live-start/README.md`; host health checks | Repeat after next VM rebuild. |
| Empty desktop after login | verified | Prior shell login reset tests and VM observation | Capture fresh screenshot after G8 code lands. |
| Failed/refused native launch avoids phantom taskbar entry | fixed | `AppShell.test.tsx` focused test | None for frontend. |
| Duplicate pending native launch prevention | fixed | `AppShell.test.tsx` focused test | None for frontend. |
| Native pending launch expiration | fixed | `AppShell.test.tsx` focused test | Tune TTL if real hardware evidence suggests a different value. |
| Native left-click focus/restore/minimize semantics | fixed | `AppShell.test.tsx` focused test | Manual compositor smoke in VM. |
| Native action failure diagnostics | fixed | `AppShell.test.tsx` focused test | Add backend capability metadata later to hide unsupported actions. |
| Multiple native windows remain separate | fixed | `AppShell.test.tsx` covers two same-app compositor windows as separate taskbar entries | Manual two-terminal/two-file-window smoke. |
| Repeated launch of an already-open native app keeps pending launch distinct | fixed | `AppShell.test.tsx` excludes launch-time matching window ids from pending resolution | Manual repeat-launch smoke in VM. |
| Closed native apps disappear from taskbar | fixed | `AppShell.test.tsx` covers closed lifecycle snapshots being ignored by the taskbar | Manual native close smoke in VM. |
| Preferred terminal/files/browser resolution | partial | Existing backend registry entries | Operator-configurable preference resolution. |

## Validation Commands

Completed:

```bash
npm -w @forge/desktop run test -- AppShell.test.tsx
npm run validate:desktop
npm test
npm run lint
git diff --check
```

Result: focused AppShell tests passed 33 tests after the G8 native lifecycle/stale-snapshot/repeated-launch fixes; desktop validation passed 105 frontend tests plus build/typecheck; root core tests passed; core vet passed; `git diff --check` passed. The red runs before implementation failed the expected new native lifecycle/action, closed-snapshot, and repeated-launch cases.

## Manual Smoke Status

- VM/session: VirtualBox `FORGE-OS`.
- Baseline commit observed: `0867e26`.
- Core health: reachable on `http://127.0.0.1:18492/health`.
- Modelruntime health: reachable on `http://127.0.0.1:18492/forge/model-runtime/health`.
- GPU acceleration inside VirtualBox: disabled, expected for the VM path.

## Authority Non-Changes

- G8 does not replace labwc or create a compositor.
- G8 does not enable autologin or remove fallback desktop/TTY paths.
- G8 does not run `systemctl`, `nixos-rebuild`, package install, reboot, shutdown, kernel module, or model load/unload operations from the shell.
- G8 does not write semantic memory directly, bypass gateway/approval/Control Lane authority, add public mutation routes, or make FORGE-K simulator services live authority.

## Links

- Report: `docs/reports/phase_g8_desktop_shell_verification.md`
- Smoke test: `docs/runbooks/desktop_shell_operator_smoke_test.md`
- Desktop shell: `docs/DESKTOP_SHELL.md`
- Operator VM: `docs/runbooks/forge_operator_desktop_vm.md`
