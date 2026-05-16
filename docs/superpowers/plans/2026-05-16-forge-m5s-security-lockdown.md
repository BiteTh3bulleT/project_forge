# FORGE M5S Security Lockdown Plan

## Goal

Close the highest-risk local/VM security gaps before M5A authority convergence.

## Scope

1. Require backend bearer auth for non-health core routes.
2. Fail closed on wildcard binds unless explicitly opted in and token-backed.
3. Tighten CORS to exact Tauri/configured origins, with localhost browser origins behind a dev flag.
4. Move approval/cancellation actor authority to authenticated request context.
5. Restrict project-context imports to allowed roots and reject symlink/path escapes.
6. Reconcile interrupted jobs on startup and cancel running workers on shutdown.
7. Remove unconditional Unix shell/process assumptions from gateway process tools.
8. Document secret storage posture and M5S review evidence.

## Verification

Run focused Go tests first, then JS validation, build, smoke, and aggregate validation where time allows.
