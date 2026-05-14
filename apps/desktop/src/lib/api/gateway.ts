import type {
  AdapterInfo,
  AdapterInvokeRequest,
  InvokeResult,
  ToolCapability,
  ToolCapabilityStatus,
} from "@forge/shared";
import { j } from "./client";
import type { AuditTraceLookupResponse, ProcessHealthTraceResponse } from "./types";

export const adaptersApi = {
  list: () => j<{ adapters: AdapterInfo[] }>("/api/adapters"),
  invoke: async (id: string, body: AdapterInvokeRequest) => {
    const scope = body.scope ?? {
      allowedPaths: [],
      forbiddenPaths: [],
      selectedPaths: [],
    };
    const paths = [
      ...scope.selectedPaths,
      ...scope.allowedPaths,
      ...scope.forbiddenPaths,
    ]
      .map((value) => value.trim())
      .filter((value) => value.length > 0);
    const correlationId =
      body.correlationId?.trim() || `legacy.adapter.invoke:${id}:${Date.now()}`;
    const gatewayResponse = await j<{
      result: {
        status?: string;
        message?: string;
        deniedReason?: string;
        data?: Record<string, unknown>;
      };
    }>("/api/gateway/invoke", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({
        toolId: "legacy.adapter.invoke",
        laneId: "legacy.adapter.invoke",
        action: "invoke",
        source: "api",
        initiator: "api",
        correlationId,
        paths,
        input: {
          adapterId: id,
          capability: body.capability,
          scope,
          writeIntent: body.writeIntent ?? false,
          timeoutMs: body.timeoutMs ?? 0,
          dryRun: body.dryRun ?? false,
          correlationId,
          input: body.input ?? {},
          ...(body.taskPacketRef != null
            ? { taskPacketRef: body.taskPacketRef }
            : {}),
        },
        metadata: {
          legacyAdapterCompatibilityWrapper: true,
          legacyIngressRemoved: "/api/adapters/{id}/invoke",
        },
      }),
    });

    const gwResult = gatewayResponse.result ?? {};
    if ((gwResult.status ?? "").toLowerCase() !== "ok") {
      throw new Error(
        gwResult.message?.trim() ||
          gwResult.deniedReason?.trim() ||
          "adapter invocation denied",
      );
    }

    const rawResult = (gwResult.data as { result?: unknown } | undefined)
      ?.result;
    if (!rawResult || typeof rawResult !== "object") {
      throw new Error("gateway response missing adapter invocation result");
    }
    const typed = rawResult as Partial<InvokeResult>;
    return {
      ok: Boolean(typed.ok),
      message:
        typeof typed.message === "string"
          ? typed.message
          : "adapter invocation completed",
      ...(typeof typed.failureCode === "string"
        ? { failureCode: typed.failureCode }
        : {}),
      data:
        typeof typed.data === "object" && typed.data != null ? typed.data : {},
    };
  },
};

export const commandsApi = {
  execute: (name: string, args?: Record<string, unknown>) =>
    j<{ ok: boolean; note?: string; jobId?: string }>(
      "/api/commands/execute",
      {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ name, args: args ?? {} }),
      },
    ),
};

export const gatewayApi = {
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
        capabilityId?: string;
        capabilityStatus?: ToolCapabilityStatus;
        capabilityRisk?: string;
        adapterId?: string;
        requiresApprovalByDefault?: boolean;
        autonomyEligible?: boolean;
        allowedInDryRun?: boolean;
      }>;
    }>("/api/gateway/tools"),
  capabilities: () =>
    j<{ capabilities: ToolCapability[] }>("/api/gateway/capabilities"),
  updateCapabilityStatus: (
    id: string,
    body: {
      status: ToolCapabilityStatus;
      reason?: string;
      actor?: string;
      actorKind?: string;
      source?: string;
      approvalId?: string;
      correlationId?: string;
      traceId?: string;
      dryRun?: boolean;
    },
  ) =>
    j<{
      success: boolean;
      capability?: ToolCapability;
      capabilityId?: string;
      previousStatus: ToolCapabilityStatus;
      newStatus: ToolCapabilityStatus;
      riskClass?: string;
      approvalRequired?: boolean;
      approvalRequestId?: number;
      rejectionReason?: string;
      correlationId?: string;
      traceId?: string;
      auditCategory?: string;
    }>(`/api/gateway/capabilities/${encodeURIComponent(id)}/status`, {
      method: "PATCH",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(body),
    }),
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
};

export const actionLanesApi = {
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
};

export const executionPermissionsApi = {
  profiles: () =>
    j<{
      profiles: unknown[];
      active: unknown | null;
      summary: Record<string, unknown>;
    }>("/api/permissions/profiles"),
  saveProfile: (body: Record<string, unknown>) =>
    j<{ profile: unknown }>("/api/permissions/profiles", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(body),
    }),
  activateProfile: (id: string) =>
    j<{ profile: unknown }>(
      `/api/permissions/profiles/${encodeURIComponent(id)}/activate`,
      {
        method: "POST",
      },
    ),
  deleteProfile: (id: string) =>
    j<void>(`/api/permissions/profiles/${encodeURIComponent(id)}`, {
      method: "DELETE",
    }),
};

export const auditApi = {
  list: (params?: {
    limit?: number;
    category?: string;
    correlationId?: string;
    jobId?: string;
    outcome?: string;
  }) => {
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
    j<{
      correlationId: string;
      records: unknown[];
      report?: Record<string, unknown>;
    }>(`/api/audit/trace/${encodeURIComponent(correlationId)}`),
  lookup: (params: { correlationId?: string; traceId?: string }) => {
    const qs = new URLSearchParams();
    if (params.correlationId) qs.set("correlationId", params.correlationId);
    if (params.traceId) qs.set("traceId", params.traceId);
    const q = qs.toString();
    return j<AuditTraceLookupResponse>(`/api/audit/trace${q ? `?${q}` : ""}`);
  },
};

export const processHealthApi = (params: {
  correlationId?: string;
  traceId?: string;
}) => {
  const qs = new URLSearchParams();
  if (params.correlationId) qs.set("correlationId", params.correlationId);
  if (params.traceId) qs.set("traceId", params.traceId);
  const q = qs.toString();
  return j<ProcessHealthTraceResponse>(
    `/api/process/health${q ? `?${q}` : ""}`,
  );
};
