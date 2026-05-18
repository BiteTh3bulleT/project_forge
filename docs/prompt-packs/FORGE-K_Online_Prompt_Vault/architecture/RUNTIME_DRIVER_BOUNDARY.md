# Runtime Driver Boundary

## Rule

Models are drivers. They produce proposal envelopes.

## Required proposal types

- answer draft
- claim proposal
- action proposal
- memory proposal
- tool proposal
- contradiction proposal
- summary proposal

## Every runtime proposal must include

- model id
- backend id
- context bundle hash
- prompt hash
- runtime config
- token counts
- correlation id
- proposal type
- provenance

## Forbidden

- direct memory writes
- direct tool execution
- direct truth commits
- bypassing Courthouse or Kernel
