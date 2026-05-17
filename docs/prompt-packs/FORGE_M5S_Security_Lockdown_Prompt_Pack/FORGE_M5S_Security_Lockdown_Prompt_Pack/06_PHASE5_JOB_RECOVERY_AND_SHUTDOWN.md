# Phase 5 — Job Recovery and Shutdown

On startup reconcile non-terminal jobs.
Queued jobs requeue. Running jobs become recoverable/failed unless safe to retry.
Terminal jobs untouched. Recovery idempotent.

Shutdown cancels workers cleanly.

Tests cover restart states, idempotency, and queue overflow goroutine behavior.
