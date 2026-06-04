# FORGE AXIOM Cognition Engine

Status: proposed internal architecture.
Date: 2026-06-04.

## Authority Banner

AXIOM is an internal FORGE cognition, search, and context layer. It does not execute tools, does not write canonical memory, and does not bypass Gateway, approvals, audit, Control Lane, or modelruntime. Search results are evidence candidates. FORGE-K remains simulator and shadow validation.

## Purpose

AXIOM names the big-brain search and context-selection subsystem inside FORGE. It is not a separate app, operator surface, approval service, model router, memory writer, or tool runner. Its job is to find, rank, explain, and package evidence candidates for existing FORGE paths.

The live authority split remains:

| Concern | Authority |
|---|---|
| Tool execution | `services/core/internal/gateway` |
| Tool approval | `services/core/internal/approvals` |
| Audit trail | `services/core/internal/audit` |
| Semantic mutation and canonical memory | `services/core/internal/aios/controllane` |
| Runtime provider calls | `services/core/internal/modelruntime` |
| Current retrieval and memory stores | `services/core/internal/retrieval`, `memory`, `search`, `embeddings`, `vectorstore` |
| FORGE-K target architecture | `services/core/internal/forgek` simulator and `forgekshadow` diagnostics |

## Single Authority Map

The single authority map is unchanged from `docs/architecture/forge_wiring_map.md`:

- Gateway remains the only authorized external-effect and bounded-tool execution boundary.
- Control Lane remains the semantic mutation path for notes, state, open loops, contradictions, supersession, and context packet records.
- Audit remains the shared append-only trace substrate.
- Approvals remain separated request and decision records.
- Modelruntime remains the only live model-provider orchestration path.
- FORGE-K remains simulator and shadow validation unless a later ADR explicitly promotes a tested live seam.

AXIOM plugs into this map as an evidence-producing and context-compiling layer. It may propose packets to existing authority paths. It may not own any authority path.

## Packet Contracts

Initial contracts should extend existing packages rather than create a parallel runtime:

```go
type TrustTier string

const (
	TrustTierLocalLive   TrustTier = "local_live"
	TrustTierOfficial    TrustTier = "official"
	TrustTierCurated     TrustTier = "curated"
	TrustTierWeb         TrustTier = "web"
	TrustTierVectorRecall TrustTier = "vector_recall"
	TrustTierLowTrust    TrustTier = "low_trust"
)

type RoutingMode string

const (
	RoutingModeAnswerContext RoutingMode = "answer_context"
	RoutingModeCodeContext   RoutingMode = "code_context"
	RoutingModeAuditReview   RoutingMode = "audit_review"
	RoutingModeOperatorBrief RoutingMode = "operator_brief"
)

type SearchEvidencePacket struct {
	ID                 string
	WorkspaceID        string
	Query              string
	RoutingMode        RoutingMode
	Candidates         []SearchEvidenceCandidate
	RejectedCandidates []RejectedSearchCandidate
	CreatedAt          time.Time
}

type ContextPacket struct {
	ID              string
	WorkspaceID     string
	RoutingMode     RoutingMode
	SourcePacketIDs []string
	SelectedRefs    []string
	RejectedRefs    []string
	TokenBudget      int
	CreatedAt        time.Time
}
```

These names are design anchors, not live API declarations yet: `SearchEvidencePacket`, `ContextPacket`, `TrustTier`, and `RoutingMode`.

## Non-Authority Rules

- A `SearchEvidencePacket` is not an approval and cannot authorize execution.
- A `ContextPacket` is not canonical memory and cannot commit truth.
- Low-trust candidates can be used for discovery, contradiction prompts, or operator review only.
- Stale memory can remain as historical evidence but must be excluded from current-answer context when fresher scoped evidence exists.
- Rejected candidates must be recorded so audit and review can explain why a source was not used.
- Official documentation and live local workspace evidence outrank old web/vector recall when scope and freshness are comparable.

## Live Flow

```text
operator/API request
  -> existing route and auth
  -> AXIOM search/context candidate production
  -> policy and context contraction
  -> existing modelruntime path for response generation
  -> if an action is proposed: existing Gateway/approval/audit path
  -> if memory/state changes are proposed: existing Control Lane syscall path
```

AXIOM provides context, citations, ranking metadata, rejected candidate records, freshness signals, and trust-tier annotations. FORGE decides through the existing live authorities.

## Required Later Tests

- Vector or web evidence alone cannot authorize execution.
- Low-trust candidates cannot become canonical memory.
- Stale memory is excluded when fresh scoped local evidence exists.
- Live local workspace evidence outranks old remote evidence.
- Official documentation outranks general web pages for factual claims about current APIs.
- Packets record rejected candidates and rejection reasons.
- AXIOM has no imports or calls that bypass Gateway, approvals, audit, Control Lane, or modelruntime.
