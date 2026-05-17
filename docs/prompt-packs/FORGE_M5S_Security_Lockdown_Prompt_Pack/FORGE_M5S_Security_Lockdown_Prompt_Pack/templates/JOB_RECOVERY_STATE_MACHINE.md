# Job Recovery State Machine

Queued -> requeue.  
Running -> recoverable failed or requeue only if safe/idempotent.  
Awaiting approval -> remain pending, refresh expiry.  
Terminal states unchanged.

Do not re-run destructive jobs without approval/fingerprint/idempotency proof.
