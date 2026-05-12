import type {
  DossierMemoryView,
  DossierVSASummary,
  MemoryObservation,
  MemoryObservationDetail,
  MemoryRepairRun,
  MemoryRepairRunDetail,
  ObservationVSADetail,
  PacketAlignmentNote,
  RetrievalSelectionReason,
  VSAReindexRun,
  VSAReindexRunDetail,
} from "@forge/shared";

import { j } from "./client";

export const memoryApi = {
  listObservations: (params?: {
    limit?: number;
    dossierId?: number;
    type?: string;
    originKind?: string;
    staleOnly?: boolean;
  }) => {
    const qs = new URLSearchParams();
    if (params?.limit != null) qs.set("limit", String(params.limit));
    if (params?.dossierId != null) qs.set("dossierId", String(params.dossierId));
    if (params?.type) qs.set("type", params.type);
    if (params?.originKind) qs.set("originKind", params.originKind);
    if (params?.staleOnly) qs.set("staleOnly", "true");
    const q = qs.toString();
    return j<{ observations: MemoryObservation[] }>(
      `/api/memory/observations${q ? `?${q}` : ""}`,
    );
  },
  createObservation: (body: Record<string, unknown>) =>
    j<{ observation: MemoryObservation }>("/api/memory/observations", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(body),
    }),
  getObservation: (id: number) =>
    j<{ observation: MemoryObservationDetail }>(
      `/api/memory/observations/${encodeURIComponent(String(id))}`,
    ),
  patchObservation: (id: number, body: Record<string, unknown>) =>
    j<{ observation: MemoryObservationDetail }>(
      `/api/memory/observations/${encodeURIComponent(String(id))}`,
      {
        method: "PATCH",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(body),
      },
    ),
  getObservationVSA: (id: number) =>
    j<{ detail: ObservationVSADetail }>(
      `/api/memory/observations/${encodeURIComponent(String(id))}/vsa`,
    ),
  markObservationUsefulness: (id: number, body: Record<string, unknown>) =>
    j<{ ok: boolean; observationId: number }>(
      `/api/memory/observations/${encodeURIComponent(String(id))}/usefulness`,
      {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(body),
      },
    ),
  retrievalSelection: (runId: number) =>
    j<{ selection: RetrievalSelectionReason[] }>(
      `/api/memory/retrieval-runs/${encodeURIComponent(String(runId))}/selection`,
    ),
  packetAlignment: (packetId: number, limit = 80) =>
    j<{ notes: PacketAlignmentNote[] }>(
      `/api/memory/packets/${encodeURIComponent(String(packetId))}/alignment?limit=${encodeURIComponent(String(limit))}`,
    ),
  dossierView: (dossierId: number, limit = 40) =>
    j<{ view: DossierMemoryView }>(
      `/api/memory/dossiers/${encodeURIComponent(String(dossierId))}?limit=${encodeURIComponent(String(limit))}`,
    ),
  listRepairRuns: (params?: { limit?: number; dossierId?: number }) => {
    const qs = new URLSearchParams();
    if (params?.limit != null) qs.set("limit", String(params.limit));
    if (params?.dossierId != null) qs.set("dossierId", String(params.dossierId));
    const q = qs.toString();
    return j<{ runs: MemoryRepairRun[] }>(
      `/api/memory/repair-runs${q ? `?${q}` : ""}`,
    );
  },
  getRepairRun: (id: number) =>
    j<{ detail: MemoryRepairRunDetail }>(
      `/api/memory/repair-runs/${encodeURIComponent(String(id))}`,
    ),
  runRepair: (body: {
    dossierId?: number;
    maxAgeDays?: number;
    limit?: number;
    note?: string;
  }) =>
    j<{ detail: MemoryRepairRunDetail }>("/api/memory/repair/run", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(body),
    }),
  listVSAReindexRuns: (params?: { limit?: number; dossierId?: number }) => {
    const qs = new URLSearchParams();
    if (params?.limit != null) qs.set("limit", String(params.limit));
    if (params?.dossierId != null) qs.set("dossierId", String(params.dossierId));
    const q = qs.toString();
    return j<{ runs: VSAReindexRun[] }>(
      `/api/memory/vsa/reindex-runs${q ? `?${q}` : ""}`,
    );
  },
  getVSAReindexRun: (id: number) =>
    j<{ detail: VSAReindexRunDetail }>(
      `/api/memory/vsa/reindex-runs/${encodeURIComponent(String(id))}`,
    ),
  runVSAReindex: (body: {
    dossierId?: number;
    mode?: string;
    triggeredBy?: string;
    reason?: string;
    note?: string;
    limit?: number;
    staleOnly?: boolean;
    force?: boolean;
  }) =>
    j<{ detail: VSAReindexRunDetail }>("/api/memory/vsa/reindex/run", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(body),
    }),
  dossierVSASummary: (dossierId: number) =>
    j<{ summary: DossierVSASummary }>(
      `/api/memory/dossiers/${encodeURIComponent(String(dossierId))}/vsa-summary`,
    ),
};
