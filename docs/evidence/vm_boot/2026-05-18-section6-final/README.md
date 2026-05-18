# Section 6 Native Desktop VM Evidence - 2026-05-18

## Scope

This run verifies the canonical operator VM reaches the native FORGE desktop
path:

1. QEMU/NixOS boot path.
2. Graphical FORGE greeter with password login.
3. `FORGE Operator` session selection.
4. Successful `operator` login using password `forge`.
5. FORGE native desktop shell with taskbar apps and modelruntime surface.

## Command

```bash
USE_TMPDIR=1 TMPDIR="$PWD/docs/evidence/vm_boot/2026-05-18-section6-final/tmp" \
NIX_DISK_IMAGE="$PWD/docs/evidence/vm_boot/2026-05-18-section6-final/forge-operator-vm.qcow2" \
QEMU_NET_OPTS='hostfwd=tcp:127.0.0.1:2224-:22' \
QEMU_OPTS="-display vnc=127.0.0.1:12 -monitor unix:$PWD/docs/evidence/vm_boot/2026-05-18-section6-final/qemu-monitor.sock,server,nowait" \
./result/bin/run-forge-operator-vm-vm
```

The generated qcow2, monitor socket, temporary directory, and raw PPM captures
were removed after evidence capture.

## Evidence

- `boot-splash.png`: NixOS boot/activation console evidence from the canonical
  VM reset path.
- `login.png`: graphical FORGE greeter for `FORGE local VM operator`.
- `session-dropdown-key.png`: selectable `FORGE Operator` session entry.
- `session-selected.png`: `FORGE Operator` selected before login.
- `after-vnc-click.png`: password prompt after pressing Login.
- `post-password.png`: successful login into the FORGE native desktop session.
- `reboot-greeter.png`: post-login desktop modelruntime command board showing
  shell/taskbar app routing.

## Result

Passed. The VM reached the native FORGE desktop session after graphical password
login, with taskbar apps visible and the Models surface reporting the expected
degraded local modelruntime state when the VM-local Ollama endpoint is not
running.
