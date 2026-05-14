import type {
  AdapterMetric,
  ApprovalPreset,
  AutomationHistory,
  AutomationRule,
  DashboardSummary,
  DossierProfile,
  Dossier,
  DossierDetail,
  EmbeddingConfig,
  EvaluationRecord,
  ExecutionStrategy,
  FailurePattern,
  ImportReconciliation,
  ImportedExecution,
  JobLineage,
  PacketGuidance,
  PolicyRecommendation,
  ProjectContextRecord,
  ReembedResult,
  ReviewRecord,
  RetrievalRun,
  RoutingInsight,
  SourceEmbeddingStatus,
  TaskPacket,
  RetrievalResultVSASignal,
} from "@forge/shared";

export type {
  ChatThreadSummary,
  ChatMessage,
  ChatAttachment,
  RemoteTelegramPayload,
  RemoteDiscordPayload,
  ChatToolGatewayActivity,
  ChatThreadDetail,
  CanvasBoard,
  CanvasNote,
  CanvasBoardDetail,
  ForgeArtifact,
  ContextSnapshotInspectorCounts,
  ContextSnapshotInspectorSummary,
  ContextSnapshotInspectorDetail,
  RestoreCandidateScoreView,
  RestorePackageView,
  ResumeHintsView,
  RestoreInspectorScoreResponse,
  RestoreInspectorCandidatesResponse,
  RestoreInspectorResumeHintsResponse,
  DreamReplayCandidateView,
  DreamSalienceScoreView,
  DreamMemoryTierProposalView,
  OperatorReviewItem,
  DreamReportSummary,
  DreamReportDetail,
  DreamReportCandidatesResponse,
  DreamReportProposalsResponse,
  DreamReportWarningsResponse,
  AuditTraceLookupReport,
  AuditTraceLookupResponse,
  ProcessHealthInvocation,
  ProcessHealthCorrelationReport,
  ProcessHealthRuntime,
  ProcessHealthTraceResponse,
  AutonomyStatusSnapshot,
  AutonomyIntentRecord,
  AutonomyDecisionRecord,
  AutonomyBudgetRecord,
  AutonomyCharterRecord,
  TelegramStatusResponse,
  DiscordGatewayStatusResponse,
  ModelRuntimeModel,
  ModelRuntimeImportResult,
  ModelRuntimeLoadResult,
  ModelRuntimeManagementRequest,
  ModelRuntimeCompatibility,
  ModelRuntimeHealth,
  ModelRuntimeQueueStatus,
  ModelRuntimeLoadedModel,
  ModelRuntimeLoadedStatus,
  ModelRuntimeBackendStatus,
  ModelRuntimeUsageSummary,
  ForgeSystemStatus,
} from "./api/types";

import { approvalsApi } from "./api/approvals";
import { artifactsApi } from "./api/artifacts";
import { autonomyApi } from "./api/autonomy";
import { backupApi } from "./api/backup";
import { j } from "./api/client";
import { canvasApi } from "./api/canvas";
import { chatApi } from "./api/chat";
import {
  actionLanesApi,
  adaptersApi,
  auditApi,
  commandsApi,
  executionPermissionsApi,
  gatewayApi,
  processHealthApi,
} from "./api/gateway";
import { contextInspectorApi, dreamReportsApi } from "./api/inspectors";
import { jobsApi } from "./api/jobs";
import { memoryApi } from "./api/memory";
import { modelRuntimeApi } from "./api/modelRuntime";
import { releaseApi } from "./api/release";
import { discordApi, remoteApi, telegramApi } from "./api/remote";
import { settingsApi } from "./api/settings";
import { searchApi, sourcesApi } from "./api/sources";
import { healthApi, systemApi } from "./api/system";

export const api = {
  health: healthApi.health,
  meta: healthApi.meta,
  system: systemApi,
  settings: settingsApi,
  modelRuntime: modelRuntimeApi,
  remote: remoteApi,
  telegram: telegramApi,
  sources: sourcesApi,
  reindex: searchApi.reindex,
  search: searchApi.search,
  chunk: searchApi.chunk,
  events: searchApi.events,
  autonomy: autonomyApi,
  discord: discordApi,
  adapters: adaptersApi,
  jobs: jobsApi,
  approvals: approvalsApi,
  packets: {
    get: (id: number) => j<TaskPacket>(`/api/packets/${id}`),
  },
  projectContext: {
    get: () =>
      j<{ record: ProjectContextRecord | null }>("/api/project-context"),
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
      return j<{ config: EmbeddingConfig; status: SourceEmbeddingStatus[] }>(
        `/api/embeddings/status${q ? `?${q}` : ""}`,
      );
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
    listRuns: (params?: {
      limit?: number;
      dossierId?: number;
      jobId?: string;
    }) => {
      const qs = new URLSearchParams();
      if (params?.limit != null) qs.set("limit", String(params.limit));
      if (params?.dossierId != null)
        qs.set("dossierId", String(params.dossierId));
      if (params?.jobId) qs.set("jobId", params.jobId);
      const q = qs.toString();
      return j<{ runs: RetrievalRun[] }>(
        `/api/retrieval/runs${q ? `?${q}` : ""}`,
      );
    },
    getRun: (id: number) =>
      j<{ run: RetrievalRun }>(`/api/retrieval/runs/${id}`),
    getRunVSASignals: (runId: number) =>
      j<{ signals: RetrievalResultVSASignal[] }>(
        `/api/retrieval/runs/${encodeURIComponent(String(runId))}/vsa-signals`,
      ),
    markUsefulness: (id: number, body: Record<string, unknown>) =>
      j<{ ok: boolean; resultId: number }>(
        `/api/retrieval/results/${id}/usefulness`,
        {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify(body),
        },
      ),
  },
  memory: memoryApi,
  dossiers: {
    list: (limit = 120) =>
      j<{ dossiers: Dossier[] }>(
        `/api/dossiers?limit=${encodeURIComponent(String(limit))}`,
      ),
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
      return j<{ evaluations: EvaluationRecord[] }>(
        `/api/evaluations?${qs.toString()}`,
      );
    },
    metrics: (dossierId?: number) => {
      const q =
        dossierId != null
          ? `?dossierId=${encodeURIComponent(String(dossierId))}`
          : "";
      return j<{ metrics: AdapterMetric[] }>(`/api/evaluations/metrics${q}`);
    },
  },
  lineage: {
    byJob: (jobId: string) =>
      j<JobLineage>(`/api/lineage/jobs/${encodeURIComponent(jobId)}`),
  },
  imports: {
    list: (limit = 100, dossierId?: number) => {
      const qs = new URLSearchParams();
      qs.set("limit", String(limit));
      if (dossierId != null) qs.set("dossierId", String(dossierId));
      return j<{ imports: ImportedExecution[] }>(
        `/api/imports/executions?${qs.toString()}`,
      );
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
      return j<{ insights: RoutingInsight[] }>(
        `/api/insights?${qs.toString()}`,
      );
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
      return j<{ strategies: ExecutionStrategy[] }>(
        `/api/strategies${q ? `?${q}` : ""}`,
      );
    },
    save: (body: Record<string, unknown>) =>
      j<{ strategy: ExecutionStrategy }>("/api/strategies", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(body),
      }),
  },
  policy: {
    listPresets: (limit = 60) =>
      j<{ presets: ApprovalPreset[] }>(
        `/api/policy/presets?limit=${encodeURIComponent(String(limit))}`,
      ),
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
    getDossierProfile: (id: number) =>
      j<{ profile: DossierProfile | null }>(
        `/api/policy/dossiers/${encodeURIComponent(String(id))}`,
      ),
    saveDossierProfile: (id: number, body: Record<string, unknown>) =>
      j<{ profile: DossierProfile }>(
        `/api/policy/dossiers/${encodeURIComponent(String(id))}`,
        {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify(body),
        },
      ),
    recommend: (body: Record<string, unknown>) =>
      j<{ recommendation: PolicyRecommendation }>("/api/policy/recommend", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(body),
      }),
    listRecommendations: (params?: { limit?: number; dossierId?: number }) => {
      const qs = new URLSearchParams();
      if (params?.limit != null) qs.set("limit", String(params.limit));
      if (params?.dossierId != null)
        qs.set("dossierId", String(params.dossierId));
      const q = qs.toString();
      return j<{ recommendations: PolicyRecommendation[] }>(
        `/api/policy/recommendations${q ? `?${q}` : ""}`,
      );
    },
  },
  automation: {
    listRules: (params?: { enabled?: boolean; limit?: number }) => {
      const qs = new URLSearchParams();
      if (params?.enabled != null) qs.set("enabled", String(params.enabled));
      if (params?.limit != null) qs.set("limit", String(params.limit));
      const q = qs.toString();
      return j<{ rules: AutomationRule[] }>(
        `/api/automation/rules${q ? `?${q}` : ""}`,
      );
    },
    saveRule: (body: Record<string, unknown>) =>
      j<{ rule: AutomationRule }>("/api/automation/rules", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(body),
      }),
    history: (limit = 120) =>
      j<{ history: AutomationHistory[] }>(
        `/api/automation/history?limit=${encodeURIComponent(String(limit))}`,
      ),
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
      return j<{ guidance: PacketGuidance[] }>(
        `/api/packet-guidance${q ? `?${q}` : ""}`,
      );
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
      j<{ reconciliation: ImportReconciliation }>(
        `/api/reconciliation/imports/${encodeURIComponent(String(importId))}`,
      ),
    saveByImport: (importId: number, body: Record<string, unknown>) =>
      j<{ reconciliation: ImportReconciliation }>(
        `/api/reconciliation/imports/${encodeURIComponent(String(importId))}`,
        {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify(body),
        },
      ),
    list: (params?: { limit?: number; reviewStatus?: string }) => {
      const qs = new URLSearchParams();
      if (params?.limit != null) qs.set("limit", String(params.limit));
      if (params?.reviewStatus) qs.set("reviewStatus", params.reviewStatus);
      const q = qs.toString();
      return j<{ reconciliations: ImportReconciliation[] }>(
        `/api/reconciliation${q ? `?${q}` : ""}`,
      );
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
      j<{ review: ReviewRecord }>(
        `/api/reviews/${encodeURIComponent(String(id))}`,
        {
          method: "PATCH",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify(body),
        },
      ),
  },
  failurePatterns: {
    list: (params?: { limit?: number; dossierId?: number }) => {
      const qs = new URLSearchParams();
      if (params?.limit != null) qs.set("limit", String(params.limit));
      if (params?.dossierId != null)
        qs.set("dossierId", String(params.dossierId));
      const q = qs.toString();
      return j<{ patterns: FailurePattern[] }>(
        `/api/failure-patterns${q ? `?${q}` : ""}`,
      );
    },
    analyze: (body: Record<string, unknown>) =>
      j<{ patterns: FailurePattern[] }>("/api/failure-patterns/analyze", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(body),
      }),
  },
  commands: commandsApi,
  gateway: gatewayApi,
  actionLanes: actionLanesApi,
  executionPermissions: executionPermissionsApi,
  audit: auditApi,
  processHealth: processHealthApi,
  contextInspector: contextInspectorApi,
  dreamReports: dreamReportsApi,
  backup: backupApi,
  chat: chatApi,
  canvas: canvasApi,
  artifacts: artifactsApi,
  release: releaseApi,
};
