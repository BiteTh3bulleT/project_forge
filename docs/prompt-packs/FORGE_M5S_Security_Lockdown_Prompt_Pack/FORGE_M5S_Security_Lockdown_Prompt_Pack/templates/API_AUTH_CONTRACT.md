# API Auth Contract

Public:
- `GET /health`

Protected:
- every other route.

Header:
`Authorization: Bearer <token>`

Wildcard bind:
- allowed only with auth token configured/generated.
