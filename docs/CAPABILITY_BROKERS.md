# CAPABILITY_BROKERS

FORGE exposes machine reach through typed tool domains.

## Filesystem

- `fs.read`
- `fs.list`
- `fs.mkdir`
- `fs.rename`
- `fs.delete`
- `fs.copy`
- `fs.chmod`
- `fs.write`

## Shell / Process

- `proc.run`
- `proc.terminate`

## Git

- `git.status`
- `git.diff`
- `git.branch`
- `git.commit`
- `git.checkout`
- `git.stash`
- `git.apply_patch`

## System / Services

- `system.service_status`
- `system.service_control`
- `system.logs`

## Desktop / Session

- `desktop.notify`
- `desktop.open`

## Network

- `net.interfaces`
- `net.connectivity`
- `net.dns_lookup`
- `net.fetch`

## Secrets

- `secret.get`

## Broker Notes

- Each tool declares domain/action/risk/write/exec/network metadata.
- Lanes and permission profiles constrain where and how each tool can run.
- Output is normalized for audit and UI visibility.
