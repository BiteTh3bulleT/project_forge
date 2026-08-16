# K20J — alternate live authority retirement

Status: `LIVE BOOT PATH RETIRED / OFFLINE ROLLBACK ONLY`

Production authority selection no longer accepts a mode. The configuration
field and `FORGE_KERNEL_AUTHORITY_MODE` environment selector are removed;
daemon construction can only build the production Kernel over its durable
port. Missing port or authorization dependencies leave semantic mutation fail
closed instead of exposing the raw Control Lane processor.

The production chain is unconditionally FORGE-K Kernel -> deterministic
DurablePort preflight/commit -> Kernel receipt verification. The Control Lane
combined `Process` method remains temporarily only for isolated adapter tests
while those tests migrate to the Kernel harness; no production assembly or
feature package selects it.

Operational rollback is daemon-stopped and offline: preserve or restore a
verified prior SQLite store and Nix generation, then restart the sole FORGE-K
authority. It is not an environment switch to a second live cognitive kernel.

Tests and a production-source guard prove there is exactly one Kernel
construction callsite, no authority selector exists in configuration, and
missing Kernel dependencies report authority unavailable rather than exposing
a fallback owner.
