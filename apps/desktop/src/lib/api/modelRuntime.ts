import { j } from "./client";
import type {
  ModelRuntimeBackendStatus,
  ModelRuntimeCompatibility,
  ModelRuntimeHealth,
  ModelRuntimeImportResult,
  ModelRuntimeLoadResult,
  ModelRuntimeLoadedStatus,
  ModelRuntimeManagementRequest,
  ModelRuntimeModel,
  ModelRuntimeQueueStatus,
  ModelRuntimeUsageSummary,
} from "./types";

type RuntimeEnvelope<T> = T & {
  correlationId?: string;
  traceId?: string;
  workspaceId?: string;
};

export const modelRuntimeApi = {
  list: () =>
    j<RuntimeEnvelope<{ models: ModelRuntimeModel[] }>>("/forge/models"),
  get: (id: string) =>
    j<RuntimeEnvelope<{ model: ModelRuntimeModel }>>(
      `/forge/models/${encodeURIComponent(id)}`,
    ),
  import: (body: {
    path: string;
    id?: string;
    displayName?: string;
    family?: string;
    backend?: string;
    capabilities?: string[];
    license?: string;
    quantization?: string;
    contextLength?: number;
    preferred?: boolean;
    actor?: string;
    source?: string;
    workspaceId?: string;
    laneId?: string;
    capabilityId?: string;
    approvalId?: string;
    dryRun?: boolean;
    metadata?: Record<string, unknown>;
  }) =>
    j<RuntimeEnvelope<{ result: ModelRuntimeImportResult }>>(
      "/forge/models/import",
      {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(body),
      },
    ),
  scan: (body?: ModelRuntimeManagementRequest) =>
    j<RuntimeEnvelope<{ models: ModelRuntimeModel[]; count: number }>>(
      "/forge/models/scan",
      {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(body ?? {}),
      },
    ),
  verify: (id: string, body?: ModelRuntimeManagementRequest) =>
    j<RuntimeEnvelope<{ model: ModelRuntimeModel }>>(
      `/forge/models/${encodeURIComponent(id)}/verify`,
      {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(body ?? {}),
      },
    ),
  enable: (id: string, body?: ModelRuntimeManagementRequest) =>
    j<RuntimeEnvelope<{ model: ModelRuntimeModel }>>(
      `/forge/models/${encodeURIComponent(id)}/enable`,
      {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(body ?? {}),
      },
    ),
  disable: (id: string, body?: ModelRuntimeManagementRequest) =>
    j<RuntimeEnvelope<{ model: ModelRuntimeModel }>>(
      `/forge/models/${encodeURIComponent(id)}/disable`,
      {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(body ?? {}),
      },
    ),
  archive: (id: string, body?: ModelRuntimeManagementRequest) =>
    j<RuntimeEnvelope<{ model: ModelRuntimeModel }>>(
      `/forge/models/${encodeURIComponent(id)}/archive`,
      {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(body ?? {}),
      },
    ),
  remove: (id: string, body?: ModelRuntimeManagementRequest) =>
    j<RuntimeEnvelope<{ result: { modelId: string; removedPath?: string } }>>(
      `/forge/models/${encodeURIComponent(id)}/remove`,
      {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(body ?? {}),
      },
    ),
  load: (id: string, body?: ModelRuntimeManagementRequest) =>
    j<RuntimeEnvelope<{ result: ModelRuntimeLoadResult }>>(
      `/forge/models/${encodeURIComponent(id)}/load`,
      {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(body ?? {}),
      },
    ),
  unload: (id: string, body?: ModelRuntimeManagementRequest) =>
    j<RuntimeEnvelope<{ result: ModelRuntimeLoadResult }>>(
      `/forge/models/${encodeURIComponent(id)}/unload`,
      {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(body ?? {}),
      },
    ),
  compatibility: (id: string) =>
    j<RuntimeEnvelope<{ compatibility: ModelRuntimeCompatibility }>>(
      `/forge/models/${encodeURIComponent(id)}/compatibility`,
    ),
  health: () =>
    j<RuntimeEnvelope<{ health: ModelRuntimeHealth }>>(
      "/forge/model-runtime/health",
    ),
  backends: () =>
    j<RuntimeEnvelope<{ backends: ModelRuntimeBackendStatus[] }>>(
      "/forge/model-runtime/backends",
    ),
  usage: () =>
    j<RuntimeEnvelope<{ usage: ModelRuntimeUsageSummary }>>(
      "/forge/model-runtime/usage",
    ),
  queue: () =>
    j<RuntimeEnvelope<{ queue: ModelRuntimeQueueStatus }>>(
      "/forge/model-runtime/queue",
    ),
  loaded: () =>
    j<RuntimeEnvelope<{ loaded: ModelRuntimeLoadedStatus }>>(
      "/forge/model-runtime/loaded",
    ),
};
