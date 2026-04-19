import type {
  AdapterInfo,
  AdapterInvokeRequest,
  AdapterMetric,
  ApprovalPreset,
  ApprovalRequest,
  AutomationHistory,
  AutomationRule,
  DashboardSummary,
  DossierMemoryView,
  DossierProfile,
  Dossier,
  DossierDetail,
  EmbeddingConfig,
  EvaluationRecord,
  ExecutionStrategy,
  FailurePattern,
  ForgeEvent,
  ImportReconciliation,
  ImportedExecution,
  InvokeResult,
  JobDetail,
  JobLineage,
  JobRecord,
  JobTemplate,
  PacketGuidance,
  PacketAlignmentNote,
  PolicyRecommendation,
  ProjectContextRecord,
  ReembedResult,
  ReviewRecord,
  RetrievalRun,
  RetrievalSelectionReason,
  RoutingInsight,
  MemoryObservation,
  MemoryObservationDetail,
  MemoryRepairRun,
  MemoryRepairRunDetail,
  SearchHit,
  SourceEmbeddingStatus,
  SourceRow,
  TaskPacket,
} from "@forge/shared";

/** Persisted chat thread (SQLite). */
export type ChatThreadSummary = {
  id: number;
  title: string;
  createdAtMs: number;
  updatedAtMs: number;
  dossierId?: number;
};

export type ChatMessage = {
  id: number;
  threadId: number;
  role: string;
  content: string;
  createdAtMs: number;
  metadata: Record<string, unknown>;
};

export type ChatAttachment = {
  artifactId: number;
  title: string;
  mimeType: string;
  fileName: string;
  textPreview?: string;
};

/** Persisted assistant metadata: governed tool activity from chat (see core chat_assistant_gateway). */
export type ChatToolGatewayActivity = {
  userRequestSummary?: string;
  toolManifest?: unknown;
  stages?: unknown;
  toolCallEmitted?: boolean;
  toolSelected?: string;
  toolArgs?: Record<string, unknown>;
  executionState?: string;
  executionResult?: unknown;
  failureReason?: string;
};

export type ChatThreadDetail = ChatThreadSummary & { messages: ChatMessage[] };

export type CanvasBoard = {
  id: number;
  title: string;
  dossierId?: number;
  createdAtMs: number;
  updatedAtMs: number;
};

export type CanvasNote = {
  id: number;
  boardId: number;
  title: string;
  body: string;
  x: number;
  y: number;
  width: number;
  height: number;
  pinned: boolean;
  color: string;
  links: Array<Record<string, unknown>>;
  createdAtMs: number;
  updatedAtMs: number;
};

export type CanvasBoardDetail = CanvasBoard & { notes: CanvasNote[] };

export type ForgeArtifact = {
  id: number;
  createdAtMs: number;
  jobId?: string;
  packetId?: number;
  type: string;
  title: string;
  filePath: string;
  mimeType: string;
  metadata: unknown;
};

const base = () => import.meta.env.VITE_FORGE_API_URL ?? "http://127.0.0.1:18492";

async function j<T>(path: string, init?: RequestInit): Promise<T> {
  const res = await fetch(`${base()}${path}`, {
    ...init,
    headers: {
      Accept: "application/json",
      ...(init?.headers ?? {}),
    },
  });
  if (!res.ok) {
    const t = await res.text().catch(() => "");
    throw new Error(t || `${res.status} ${res.statusText}`);
  }
  if (res.status === 204) {
    return undefined as T;
  }
  return (await res.json()) as T;
}

export type SettingsRecord = {
  extensionsCsv: string;
  theme: string;
  ollamaBaseUrl: string;
  ollamaModel: string;
  embeddingProvider: string;
  embeddingModel: string;
  embeddingDims: string;
  retrievalWeightKeyword: string;
  retrievalWeightSemantic: string;
  chatPersonalityPrompt: string;
  chatPromptDefault: string;
  remoteAccessEnabled: boolean;
  remoteAccessToken: string;
  remoteDefaultThreadId: string;
  telegramBotToken: string;
  telegramDefaultChatId: string;
  discordBotToken: string;
  discordDefaultChannelId: string;
  discordWebhookUrl: string;
};

export const api = {
  health: () => j<{ ok: boolean; service: string }>("/health"),
  meta: () => j<{ dataDir: string; dbPath: string; workspaceDir: string }>("/api/meta"),
  settings: {
    get: () => j<SettingsRecord>("/api/settings"),
    patch: (body: Record<string, unknown>) =>
      j<SettingsRecord>("/api/settings", {
        method: "PATCH",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(body),
      }),
    ollamaModels: (baseUrl?: string) => {
      const q = new URLSearchParams();
      if (baseUrl) q.set("baseUrl", baseUrl);
      const path = `/api/settings/ollama-models${q.toString() ? `?${q.toString()}` : ""}`;
      return j<{ models: string[]; baseUrl: string; status: string; error?: string }>(path);
    },
  },
  sources: {
    list: () => j<{ sources: SourceRow[] }>("/api/sources"),
    add: (path: string) =>
      j<{ id: number; path: string }>("/api/sources", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ path }),
      }),
    delete: (id: number) =>
      j<void>(`/api/sources/${id}`, {
        method: "DELETE",
      }),
  },
  reindex: (sourceId?: number) => {
    const q = sourceId != null ? `?sourceId=${encodeURIComponent(String(sourceId))}` : "";
    return j<{ ok: boolean; scope: string }>(`/api/reindex${q}`, { method: "POST" });
  },
  search: (q: string, limit = 50) =>
    j<{ hits: SearchHit[] }>(`/api/search?q=${encodeURIComponent(q)}&limit=${encodeURIComponent(String(limit))}`),
  chunk: (id: number) => j<SearchHit>(`/api/chunks/${id}`),
  events: (limit = 120) => j<{ events: ForgeEvent[] }>(`/api/events?limit=${encodeURIComponent(String(limit))}`),
  adapters: {
    list: () => j<{ adapters: AdapterInfo[] }>("/api/adapters"),
    invoke: (id: string, body: AdapterInvokeRequest) =>
      j<InvokeResult>(`/api/adapters/${encodeURIComponent(id)}/invoke`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(body),
      }),
  },
  jobs: {
    templates: () => j<{ templates: JobTemplate[] }>("/api/jobs/templates"),
    list: (status = "", limit = 120) => {
      const params = new URLSearchParams();
      if (status) params.set("status", status);
      params.set("limit", String(limit));
      return j<{ jobs: JobRecord[] }>(`/api/jobs?${params.toString()}`);
    },
    create: (body: Record<string, unknown>) =>
      j<{ job: JobRecord }>("/api/jobs", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(body),
      }),
    detail: (id: string, afterEventId = 0) =>
      j<JobDetail>(`/api/jobs/${encodeURIComponent(id)}?afterEventId=${encodeURIComponent(String(afterEventId))}`),
    cancel: (id: string, actor = "operator") =>
      j<{ ok: boolean; jobId: string }>(`/api/jobs/${encodeURIComponent(id)}/cancel`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ actor }),
      }),
    retry: (id: string, body: Record<string, unknown>) =>
      j<{ job: JobRecord }>(`/api/jobs/${encodeURIComponent(id)}/retry`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(body),
      }),
    replay: (id: string, body: Record<string, unknown>) =>
      j<{ job: JobRecord }>(`/api/jobs/${encodeURIComponent(id)}/replay`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(body),
      }),
  },
  approvals: {
    list: (status = "pending", limit = 100) =>
      j<{ approvals: ApprovalRequest[] }>(`/api/approvals?status=${encodeURIComponent(status)}&limit=${encodeURIComponent(String(limit))}`),
    get: (id: number) =>
      j<{ approval: ApprovalRequest }>(`/api/approvals/${encodeURIComponent(String(id))}`),
    approve: (id: number, note = "") =>
      j<{ decision: unknown }>(`/api/approvals/${id}/approve`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ actor: "operator", note }),
      }),
    deny: (id: number, note = "") =>
      j<{ decision: unknown }>(`/api/approvals/${id}/deny`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ actor: "operator", note }),
      }),
    cancel: (id: number, note = "") =>
      j<{ decision: unknown }>(`/api/approvals/${id}/cancel`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ actor: "operator", note }),
      }),
  },
  packets: {
    get: (id: number) => j<TaskPacket>(`/api/packets/${id}`),
  },
  projectContext: {
    get: () => j<{ record: ProjectContextRecord | null }>("/api/project-context"),
    import: (sourcePath = "", notes = "") =>
      j<{ record: ProjectContextRecord }>("/api/project-context/import", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ sourcePath, notes }),
      }),
    regenerate: () =>
      j<{ record: ProjectContextRecord }>("/api/project-context/regenerate", {
        method: "POST",
      }),
  },
  embeddings: {
    status: (provider = "", model = "") => {
      const params = new URLSearchParams();
      if (provider) params.set("provider", provider);
      if (model) params.set("model", model);
      const q = params.toString();
      return j<{ config: EmbeddingConfig; status: SourceEmbeddingStatus[] }>(`/api/embeddings/status${q ? `?${q}` : ""}`);
    },
    reembed: (body: Record<string, unknown>) =>
      j<{ result: ReembedResult }>("/api/embeddings/reembed", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(body),
      }),
  },
  retrieval: {
    createRun: (body: Record<string, unknown>) =>
      j<{ run: RetrievalRun }>("/api/retrieval/runs", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(body),
      }),
    listRuns: (params?: { limit?: number; dossierId?: number; jobId?: string }) => {
      const qs = new URLSearchParams();
      if (params?.limit != null) qs.set("limit", String(params.limit));
      if (params?.dossierId != null) qs.set("dossierId", String(params.dossierId));
      if (params?.jobId) qs.set("jobId", params.jobId);
      const q = qs.toString();
      return j<{ runs: RetrievalRun[] }>(`/api/retrieval/runs${q ? `?${q}` : ""}`);
    },
    getRun: (id: number) => j<{ run: RetrievalRun }>(`/api/retrieval/runs/${id}`),
    markUsefulness: (id: number, body: Record<string, unknown>) =>
      j<{ ok: boolean; resultId: number }>(`/api/retrieval/results/${id}/usefulness`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(body),
      }),
  },
  memory: {
    listObservations: (params?: { limit?: number; dossierId?: number; type?: string; originKind?: string; staleOnly?: boolean }) => {
      const qs = new URLSearchParams();
      if (params?.limit != null) qs.set("limit", String(params.limit));
      if (params?.dossierId != null) qs.set("dossierId", String(params.dossierId));
      if (params?.type) qs.set("type", params.type);
      if (params?.originKind) qs.set("originKind", params.originKind);
      if (params?.staleOnly) qs.set("staleOnly", "true");
      const q = qs.toString();
      return j<{ observations: MemoryObservation[] }>(`/api/memory/observations${q ? `?${q}` : ""}`);
    },
    createObservation: (body: Record<string, unknown>) =>
      j<{ observation: MemoryObservation }>("/api/memory/observations", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(body),
      }),
    getObservation: (id: number) => j<{ observation: MemoryObservationDetail }>(`/api/memory/observations/${encodeURIComponent(String(id))}`),
    patchObservation: (id: number, body: Record<string, unknown>) =>
      j<{ observation: MemoryObservationDetail }>(`/api/memory/observations/${encodeURIComponent(String(id))}`, {
        method: "PATCH",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(body),
      }),
    markObservationUsefulness: (id: number, body: Record<string, unknown>) =>
      j<{ ok: boolean; observationId: number }>(`/api/memory/observations/${encodeURIComponent(String(id))}/usefulness`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(body),
      }),
    retrievalSelection: (runId: number) =>
      j<{ selection: RetrievalSelectionReason[] }>(`/api/memory/retrieval-runs/${encodeURIComponent(String(runId))}/selection`),
    packetAlignment: (packetId: number, limit = 80) =>
      j<{ notes: PacketAlignmentNote[] }>(`/api/memory/packets/${encodeURIComponent(String(packetId))}/alignment?limit=${encodeURIComponent(String(limit))}`),
    dossierView: (dossierId: number, limit = 40) =>
      j<{ view: DossierMemoryView }>(`/api/memory/dossiers/${encodeURIComponent(String(dossierId))}?limit=${encodeURIComponent(String(limit))}`),
    listRepairRuns: (params?: { limit?: number; dossierId?: number }) => {
      const qs = new URLSearchParams();
      if (params?.limit != null) qs.set("limit", String(params.limit));
      if (params?.dossierId != null) qs.set("dossierId", String(params.dossierId));
      const q = qs.toString();
      return j<{ runs: MemoryRepairRun[] }>(`/api/memory/repair-runs${q ? `?${q}` : ""}`);
    },
    getRepairRun: (id: number) => j<{ detail: MemoryRepairRunDetail }>(`/api/memory/repair-runs/${encodeURIComponent(String(id))}`),
    runRepair: (body: { dossierId?: number; maxAgeDays?: number; limit?: number; note?: string }) =>
      j<{ detail: MemoryRepairRunDetail }>("/api/memory/repair/run", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(body),
      }),
  },
  dossiers: {
    list: (limit = 120) => j<{ dossiers: Dossier[] }>(`/api/dossiers?limit=${encodeURIComponent(String(limit))}`),
    create: (body: Record<string, unknown>) =>
      j<{ dossier: Dossier }>("/api/dossiers", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(body),
      }),
    detail: (id: number) => j<{ detail: DossierDetail }>(`/api/dossiers/${id}`),
    update: (id: number, body: Record<string, unknown>) =>
      j<{ dossier: Dossier }>(`/api/dossiers/${id}`, {
        method: "PATCH",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(body),
      }),
    generateBrief: (id: number, notes = "") =>
      j<{ brief: unknown }>(`/api/dossiers/${id}/briefs/generate`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ notes }),
      }),
  },
  evaluations: {
    create: (body: Record<string, unknown>) =>
      j<{ evaluation: EvaluationRecord }>("/api/evaluations", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(body),
      }),
    list: (limit = 120, dossierId?: number) => {
      const qs = new URLSearchParams();
      qs.set("limit", String(limit));
      if (dossierId != null) qs.set("dossierId", String(dossierId));
      return j<{ evaluations: EvaluationRecord[] }>(`/api/evaluations?${qs.toString()}`);
    },
    metrics: (dossierId?: number) => {
      const q = dossierId != null ? `?dossierId=${encodeURIComponent(String(dossierId))}` : "";
      return j<{ metrics: AdapterMetric[] }>(`/api/evaluations/metrics${q}`);
    },
  },
  lineage: {
    byJob: (jobId: string) => j<JobLineage>(`/api/lineage/jobs/${encodeURIComponent(jobId)}`),
  },
  imports: {
    list: (limit = 100, dossierId?: number) => {
      const qs = new URLSearchParams();
      qs.set("limit", String(limit));
      if (dossierId != null) qs.set("dossierId", String(dossierId));
      return j<{ imports: ImportedExecution[] }>(`/api/imports/executions?${qs.toString()}`);
    },
    create: (body: Record<string, unknown>) =>
      j<{ importedExecution: ImportedExecution }>("/api/imports/executions", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(body),
      }),
  },
  insights: {
    list: (limit = 120, dossierId?: number) => {
      const qs = new URLSearchParams();
      qs.set("limit", String(limit));
      if (dossierId != null) qs.set("dossierId", String(dossierId));
      return j<{ insights: RoutingInsight[] }>(`/api/insights?${qs.toString()}`);
    },
    generate: (dossierId?: number) =>
      j<{ insights: RoutingInsight[] }>("/api/insights/generate", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ dossierId }),
      }),
  },
  dashboard: {
    summary: () => j<DashboardSummary>("/api/dashboard"),
  },
  strategies: {
    list: (params?: { enabled?: boolean; limit?: number }) => {
      const qs = new URLSearchParams();
      if (params?.enabled != null) qs.set("enabled", String(params.enabled));
      if (params?.limit != null) qs.set("limit", String(params.limit));
      const q = qs.toString();
      return j<{ strategies: ExecutionStrategy[] }>(`/api/strategies${q ? `?${q}` : ""}`);
    },
    save: (body: Record<string, unknown>) =>
      j<{ strategy: ExecutionStrategy }>("/api/strategies", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(body),
      }),
  },
  policy: {
    listPresets: (limit = 60) => j<{ presets: ApprovalPreset[] }>(`/api/policy/presets?limit=${encodeURIComponent(String(limit))}`),
    savePreset: (body: Record<string, unknown>) =>
      j<{ preset: ApprovalPreset }>("/api/policy/presets", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(body),
      }),
    getGlobalPreset: () => j<{ presetId: string }>("/api/policy/global-preset"),
    setGlobalPreset: (presetId: string) =>
      j<{ ok: boolean; presetId: string }>("/api/policy/global-preset", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ presetId }),
      }),
    getDossierProfile: (id: number) => j<{ profile: DossierProfile | null }>(`/api/policy/dossiers/${encodeURIComponent(String(id))}`),
    saveDossierProfile: (id: number, body: Record<string, unknown>) =>
      j<{ profile: DossierProfile }>(`/api/policy/dossiers/${encodeURIComponent(String(id))}`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(body),
      }),
    recommend: (body: Record<string, unknown>) =>
      j<{ recommendation: PolicyRecommendation }>("/api/policy/recommend", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(body),
      }),
    listRecommendations: (params?: { limit?: number; dossierId?: number }) => {
      const qs = new URLSearchParams();
      if (params?.limit != null) qs.set("limit", String(params.limit));
      if (params?.dossierId != null) qs.set("dossierId", String(params.dossierId));
      const q = qs.toString();
      return j<{ recommendations: PolicyRecommendation[] }>(`/api/policy/recommendations${q ? `?${q}` : ""}`);
    },
  },
  automation: {
    listRules: (params?: { enabled?: boolean; limit?: number }) => {
      const qs = new URLSearchParams();
      if (params?.enabled != null) qs.set("enabled", String(params.enabled));
      if (params?.limit != null) qs.set("limit", String(params.limit));
      const q = qs.toString();
      return j<{ rules: AutomationRule[] }>(`/api/automation/rules${q ? `?${q}` : ""}`);
    },
    saveRule: (body: Record<string, unknown>) =>
      j<{ rule: AutomationRule }>("/api/automation/rules", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(body),
      }),
    history: (limit = 120) =>
      j<{ history: AutomationHistory[] }>(`/api/automation/history?limit=${encodeURIComponent(String(limit))}`),
    runRule: (body: Record<string, unknown>) =>
      j<{
        result: {
          rule: AutomationRule;
          matched: boolean;
          dryRun: boolean;
          preview: Record<string, unknown>;
          executed: boolean;
          execution: Record<string, unknown>;
          historyId: number;
        };
      }>("/api/automation/run", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(body),
      }),
  },
  packetGuidance: {
    list: (params?: { limit?: number; packetId?: number }) => {
      const qs = new URLSearchParams();
      if (params?.limit != null) qs.set("limit", String(params.limit));
      if (params?.packetId != null) qs.set("packetId", String(params.packetId));
      const q = qs.toString();
      return j<{ guidance: PacketGuidance[] }>(`/api/packet-guidance${q ? `?${q}` : ""}`);
    },
    analyze: (body: Record<string, unknown>) =>
      j<{ guidance: PacketGuidance }>("/api/packet-guidance/analyze", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(body),
      }),
  },
  reconciliation: {
    getByImport: (importId: number) =>
      j<{ reconciliation: ImportReconciliation }>(`/api/reconciliation/imports/${encodeURIComponent(String(importId))}`),
    saveByImport: (importId: number, body: Record<string, unknown>) =>
      j<{ reconciliation: ImportReconciliation }>(`/api/reconciliation/imports/${encodeURIComponent(String(importId))}`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(body),
      }),
    list: (params?: { limit?: number; reviewStatus?: string }) => {
      const qs = new URLSearchParams();
      if (params?.limit != null) qs.set("limit", String(params.limit));
      if (params?.reviewStatus) qs.set("reviewStatus", params.reviewStatus);
      const q = qs.toString();
      return j<{ reconciliations: ImportReconciliation[] }>(`/api/reconciliation${q ? `?${q}` : ""}`);
    },
  },
  reviews: {
    list: (params?: { limit?: number; status?: string }) => {
      const qs = new URLSearchParams();
      if (params?.limit != null) qs.set("limit", String(params.limit));
      if (params?.status) qs.set("status", params.status);
      const q = qs.toString();
      return j<{ reviews: ReviewRecord[] }>(`/api/reviews${q ? `?${q}` : ""}`);
    },
    create: (body: Record<string, unknown>) =>
      j<{ review: ReviewRecord }>("/api/reviews", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(body),
      }),
    update: (id: number, body: Record<string, unknown>) =>
      j<{ review: ReviewRecord }>(`/api/reviews/${encodeURIComponent(String(id))}`, {
        method: "PATCH",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(body),
      }),
  },
  failurePatterns: {
    list: (params?: { limit?: number; dossierId?: number }) => {
      const qs = new URLSearchParams();
      if (params?.limit != null) qs.set("limit", String(params.limit));
      if (params?.dossierId != null) qs.set("dossierId", String(params.dossierId));
      const q = qs.toString();
      return j<{ patterns: FailurePattern[] }>(`/api/failure-patterns${q ? `?${q}` : ""}`);
    },
    analyze: (body: Record<string, unknown>) =>
      j<{ patterns: FailurePattern[] }>("/api/failure-patterns/analyze", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(body),
      }),
  },
  commands: {
    execute: (name: string, args?: Record<string, unknown>) =>
      j<{ ok: boolean; note?: string; jobId?: string }>("/api/commands/execute", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ name, args: args ?? {} }),
      }),
  },
  gateway: {
    tools: () =>
      j<{
        tools: Array<{
          id: string;
          domain: string;
          action: string;
          description: string;
          riskClass: string;
          executionLevel: string;
          executes: boolean;
          usesNetwork: boolean;
          writeIntent: boolean;
        }>;
      }>("/api/gateway/tools"),
    invoke: (body: Record<string, unknown>) =>
      j<{ result: Record<string, unknown> }>("/api/gateway/invoke", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(body),
      }),
    invocations: (params?: { limit?: number; status?: string }) => {
      const qs = new URLSearchParams();
      if (params?.limit != null) qs.set("limit", String(params.limit));
      if (params?.status) qs.set("status", params.status);
      const q = qs.toString();
      return j<{
        invocations: Array<{
          id: number;
          correlationId: string;
          createdAtMs: number;
          completedAtMs?: number | null;
          toolId: string;
          laneId?: string | null;
          jobId?: string | null;
          packetId?: number | null;
          initiator: string;
          action: string;
          domain: string;
          riskClass: string;
          executionLevel: string;
          policyOutcome: string;
          writeIntent: boolean;
          scope: unknown;
          input: unknown;
          status: string;
          deniedReason: string;
          result: unknown;
          artifacts: unknown;
          permissionProfileId: string;
          approvalRequestId?: number | null;
        }>;
      }>(`/api/gateway/invocations${q ? `?${q}` : ""}`);
    },
  },
  actionLanes: {
    list: () => j<{ lanes: unknown[] }>("/api/action-lanes"),
    save: (body: Record<string, unknown>) =>
      j<{ lane: unknown }>("/api/action-lanes", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(body),
      }),
    delete: (id: string) =>
      j<void>(`/api/action-lanes/${encodeURIComponent(id)}`, {
        method: "DELETE",
      }),
  },
  executionPermissions: {
    profiles: () =>
      j<{ profiles: unknown[]; active: unknown | null; summary: Record<string, unknown> }>("/api/permissions/profiles"),
    saveProfile: (body: Record<string, unknown>) =>
      j<{ profile: unknown }>("/api/permissions/profiles", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(body),
      }),
    activateProfile: (id: string) =>
      j<{ profile: unknown }>(`/api/permissions/profiles/${encodeURIComponent(id)}/activate`, {
        method: "POST",
      }),
    deleteProfile: (id: string) =>
      j<void>(`/api/permissions/profiles/${encodeURIComponent(id)}`, {
        method: "DELETE",
      }),
  },
  audit: {
    list: (params?: { limit?: number; category?: string; correlationId?: string; jobId?: string; outcome?: string }) => {
      const qs = new URLSearchParams();
      if (params?.limit != null) qs.set("limit", String(params.limit));
      if (params?.category) qs.set("category", params.category);
      if (params?.correlationId) qs.set("correlationId", params.correlationId);
      if (params?.jobId) qs.set("jobId", params.jobId);
      if (params?.outcome) qs.set("outcome", params.outcome);
      const q = qs.toString();
      return j<{ records: unknown[] }>(`/api/audit${q ? `?${q}` : ""}`);
    },
    trace: (correlationId: string) =>
      j<{ correlationId: string; records: unknown[] }>(`/api/audit/trace/${encodeURIComponent(correlationId)}`),
  },
  backup: {
    bundles: (limit?: number) => {
      const q = limit != null ? `?limit=${encodeURIComponent(String(limit))}` : "";
      return j<{ bundles: unknown[]; backupDir: string; exportDir: string; knownKinds: string[] }>(`/api/backup/bundles${q}`);
    },
    createBundle: (body: Record<string, unknown>) =>
      j<{ bundle: unknown }>("/api/backup/bundles", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(body),
      }),
    deleteBundle: (id: number) =>
      j<void>(`/api/backup/bundles/${encodeURIComponent(String(id))}`, {
        method: "DELETE",
      }),
    restore: (body: Record<string, unknown>) =>
      j<{ result: Record<string, unknown> }>("/api/backup/restore", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(body),
      }),
  },
  chat: {
    threads: {
      list: (limit = 80) =>
        j<{ threads: ChatThreadSummary[] }>(`/api/chat/threads?limit=${encodeURIComponent(String(limit))}`),
      create: (body: { title?: string; dossierId?: number }) =>
        j<{ thread: ChatThreadSummary }>("/api/chat/threads", {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify(body),
        }),
      get: (id: number) => j<ChatThreadDetail>(`/api/chat/threads/${encodeURIComponent(String(id))}`),
      update: (id: number, body: { title: string }) =>
        j<{ thread: ChatThreadSummary }>(`/api/chat/threads/${encodeURIComponent(String(id))}`, {
          method: "PATCH",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify(body),
        }),
      delete: (id: number) =>
        j<void>(`/api/chat/threads/${encodeURIComponent(String(id))}`, { method: "DELETE" }),
      postMessage: (
        id: number,
        body: {
          content: string;
          attachmentArtifactIds?: number[];
          requestAssistant?: boolean;
          assistantDryRun?: boolean;
          /** When true, client opens SSE assistant-stream after POST returns. */
          stream?: boolean;
          /** When true, run Ollama in a background job; client polls thread. Default when not streaming/sync. */
          asyncAssistant?: boolean;
          /** When true, block until assistant completes (legacy). */
          syncAssistant?: boolean;
        },
      ) =>
        j<{
          userMessage: ChatMessage;
          assistantMessage: ChatMessage | null;
          assistantPending?: boolean;
          userMessageId?: number;
          stream?: boolean;
          asyncAssistant?: boolean;
        }>(`/api/chat/threads/${encodeURIComponent(String(id))}/messages`, {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify(body),
        }),
      uploadAttachment: async (id: number, file: File, title?: string) => {
        const fd = new FormData();
        fd.append("file", file);
        if (title && title.trim()) fd.append("title", title.trim());
        const res = await fetch(`${base()}/api/chat/threads/${encodeURIComponent(String(id))}/attachments`, {
          method: "POST",
          body: fd,
        });
        if (!res.ok) {
          const t = await res.text().catch(() => "");
          throw new Error(t || `${res.status} ${res.statusText}`);
        }
        return (await res.json()) as {
          artifact: ForgeArtifact;
          bytes: number;
          previewText: string;
        };
      },
      /** Full URL for GET SSE assistant token stream (use with EventSource). */
      assistantStreamUrl: (threadId: number, userMessageId: number) =>
        `${base()}/api/chat/threads/${encodeURIComponent(String(threadId))}/assistant-stream?userMessageId=${encodeURIComponent(String(userMessageId))}`,
      queueJob: (id: number, body: Record<string, unknown>) =>
        j<{ job: JobRecord }>(`/api/chat/threads/${encodeURIComponent(String(id))}/jobs`, {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify(body),
        }),
    },
  },
  canvas: {
    boards: {
      list: (limit = 60) =>
        j<{ boards: CanvasBoard[] }>(`/api/canvas/boards?limit=${encodeURIComponent(String(limit))}`),
      create: (body: { title?: string; dossierId?: number }) =>
        j<{ board: CanvasBoard }>("/api/canvas/boards", {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify(body),
        }),
      get: (id: number) => j<CanvasBoardDetail>(`/api/canvas/boards/${encodeURIComponent(String(id))}`),
      delete: (id: number) =>
        j<void>(`/api/canvas/boards/${encodeURIComponent(String(id))}`, { method: "DELETE" }),
      createNote: (
        boardId: number,
        body: { title?: string; body?: string; x?: number; y?: number; width?: number; height?: number },
      ) =>
        j<{ note: CanvasNote }>(`/api/canvas/boards/${encodeURIComponent(String(boardId))}/notes`, {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify(body),
        }),
      patchNote: (boardId: number, noteId: number, body: Record<string, unknown>) =>
        j<{ note: CanvasNote }>(
          `/api/canvas/boards/${encodeURIComponent(String(boardId))}/notes/${encodeURIComponent(String(noteId))}`,
          {
            method: "PATCH",
            headers: { "Content-Type": "application/json" },
            body: JSON.stringify(body),
          },
        ),
      deleteNote: (boardId: number, noteId: number) =>
        j<void>(
          `/api/canvas/boards/${encodeURIComponent(String(boardId))}/notes/${encodeURIComponent(String(noteId))}`,
          { method: "DELETE" },
        ),
    },
  },
  artifacts: {
    list: (params?: { limit?: number; jobId?: string }) => {
      const qs = new URLSearchParams();
      if (params?.limit != null) qs.set("limit", String(params.limit));
      if (params?.jobId) qs.set("jobId", params.jobId);
      const q = qs.toString();
      return j<{ artifacts: ForgeArtifact[] }>(`/api/artifacts${q ? `?${q}` : ""}`);
    },
    get: (id: number) => j<ForgeArtifact>(`/api/artifacts/${encodeURIComponent(String(id))}`),
    content: (id: number) =>
      j<{
        artifact: ForgeArtifact;
        textual: boolean;
        content: string;
        previewLimited: boolean;
      }>(`/api/artifacts/${encodeURIComponent(String(id))}/content`),
  },
  release: {
    readiness: () => j<{ checklist: Record<string, unknown> }>("/api/release/readiness"),
    artifacts: (limit?: number) => {
      const q = limit != null ? `?limit=${encodeURIComponent(String(limit))}` : "";
      return j<{ artifacts: unknown[] }>(`/api/release/artifacts${q}`);
    },
    recordArtifact: (body: Record<string, unknown>) =>
      j<{ artifact: unknown }>("/api/release/artifacts", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(body),
      }),
    firstRun: () => j<{ firstRun: Record<string, unknown> }>("/api/release/first-run"),
  },
};
