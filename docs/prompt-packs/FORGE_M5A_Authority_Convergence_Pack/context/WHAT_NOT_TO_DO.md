# WHAT NOT TO DO

This applies to every prompt in this pack.

## Do not output code in chat

Make changes in files. Final response should summarize files changed, tests run, pass/fail, and remaining gaps. Do not paste large code blocks in chat.

## Do not expand live authority

Do not make FORGE-K live authority. Do not route live memory mutation through FORGE-K simulator services. Do not make FORGE-H mutate the host. Do not make HostBridge execute host changes. Do not make the System Cockpit mutate anything. Do not make micro-agents write canonical memory. Do not let models approve actions. Do not bypass Gateway for tool execution. Do not bypass Control Lane for canonical semantic mutation. Do not bypass modelruntime governance for inference/model lifecycle. Do not treat Qdrant as truth. Do not treat Redis as canonical memory. Do not treat KV cache as memory.

## Do not add dangerous buttons

Do not add UI buttons for restart, shutdown, reboot, `systemctl`, `nixos-rebuild`, package-manager actions, kernel/module manipulation, destructive cleanup, raw model load/unload outside governed modelruntime paths, host mutation, or shell command execution.

## Do not hide missing data

If data is missing, return/report `unavailable`, `not_wired`, `unknown`, `disabled_by_default`, `deferred`, or `design_only`. Do not display missing data as healthy.

## Do not create duplicate authority paths

Do not create a second modelruntime execution path, approval system, audit system, memory write path, tool execution path, host mutation path, or route registry. Prefer extending existing authority surfaces with narrow contracts.

## Do not perform broad unrelated refactors

This sprint is about authority convergence and latency foundation. Do not rewrite the desktop design, whole gateway, whole modelruntime, whole Control Lane, Nix architecture, or memory model. Make narrow, testable changes.
