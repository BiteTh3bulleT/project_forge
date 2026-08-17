# Security Policy

FORGE is an engineering-alpha local AI workstation. Security reports are taken seriously, especially when they involve tool execution, authentication, remote ingress, durable authority, model-output visibility, secrets, host integration, backup, or recovery.

## Supported versions

FORGE does not yet publish a stable supported-release matrix. Security fixes are applied to the current development line unless a tagged release states otherwise.

| Version | Supported |
| --- | --- |
| Current `main` development line | Best-effort security fixes |
| Untagged historical commits and branches | Not supported |
| Third-party forks and local modifications | Maintainer-dependent |

## Reporting a vulnerability

Do **not** open a normal public issue containing exploit details, credentials, sensitive logs, private paths, model API keys, bearer tokens, or personal data.

Preferred reporting path:

1. Use GitHub's **private security advisory** flow for this repository when available.
2. Include a concise description, affected component, reproduction conditions, likely impact, and any suggested mitigation.
3. Use redacted or synthetic evidence. Never include live secrets.

When the private advisory flow is unavailable, open a minimal public issue asking the maintainer for a secure reporting channel. Do not include vulnerability details in that issue.

## What to include

A useful report contains:

- affected commit, branch, or build;
- operating system and deployment profile;
- affected route, tool, model surface, or subsystem;
- prerequisites and attack boundary;
- exact reproduction steps using synthetic data;
- expected and actual behavior;
- impact assessment;
- whether the issue crosses a workspace, user, capability, approval, or authority boundary;
- relevant logs or traces with secrets removed;
- proposed remediation when known.

## High-priority areas

Please report issues involving:

- authentication or actor-identity bypass;
- remote Telegram, Discord, webhook, or API ingress;
- arbitrary shell, process, filesystem, network, or host execution;
- workspace escape, path traversal, symlink, or archive-extraction escape;
- approval bypass, replay, or fingerprint mismatch;
- capability scope expansion;
- model-created authority or unsupported completion claims becoming visible as fact;
- prompt/tool-output injection that changes governed execution;
- journal, audit-outbox, idempotency, or provenance tampering;
- canonical-state corruption or unauthorized memory admission;
- backup, restore, rollback, or generation-switch corruption;
- secret exposure in logs, traces, UI, fixtures, or exported bundles;
- insecure default network binding or cross-origin behavior;
- Tauri command, shell, filesystem, or window capability escalation;
- dependency vulnerabilities that are reachable in the shipped runtime.

## Disclosure expectations

Please allow reasonable time to investigate and prepare a fix before public disclosure. The maintainer will attempt to acknowledge a complete report promptly, but FORGE is currently a small engineering-alpha project and cannot guarantee enterprise response times.

A coordinated disclosure may include:

- affected versions or commits;
- severity and impact;
- mitigation or upgrade instructions;
- recovery requirements;
- credit to the reporter when requested;
- a public advisory after a fix is available.

## Security design notes

FORGE's intended security posture includes:

- models are proposal-only;
- Gateway is the only tool-execution authority;
- production semantic syscalls enter through the production Kernel boundary;
- unimplemented authority paths fail closed;
- dangerous actions require explicit capability and approval policy;
- authenticated actor and provenance identities must survive asynchronous work;
- tool completion claims require execution and audit evidence;
- live raw row-merge restore is disabled;
- secrets must not be persisted in canonical evidence, traces, or fixtures.

A violation of one of these rules should be treated as security-relevant even when the immediate effect appears limited.
