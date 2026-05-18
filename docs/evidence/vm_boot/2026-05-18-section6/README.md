# Section 6 VM Boot Evidence Attempt

Date: 2026-05-18

Command path:

```bash
./result/bin/run-forge-operator-vm-vm
```

Display/monitor setup:

```text
QEMU_OPTS="-display vnc=127.0.0.1:11 -monitor unix:docs/evidence/vm_boot/2026-05-18-section6/qemu-monitor.sock,server,nowait"
QEMU_NET_OPTS="hostfwd=tcp:127.0.0.1:2223-:22"
```

Captured evidence:

- `boot-current.png`: canonical Nix VM reached the FORGE-branded graphical login greeter.
- `login-after-enter.png`: greeter session menu opened and showed `FORGE Operator`.
- `post-session-select.png`: keyboard-only QEMU monitor interaction returned to the greeter; it did not reach the desktop session.

Conclusion:

This attempt verifies that the VM boots to the graphical FORGE-branded login surface. It does not complete the full Section 6 VM evidence requirement because it does not show successful `operator` login or the FORGE native desktop session. The punchlist item remains open until splash/login/session evidence is captured end to end.
