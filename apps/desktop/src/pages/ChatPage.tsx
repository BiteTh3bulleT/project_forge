import type { JobTemplate } from "@forge/shared";
import { useCallback, useEffect, useMemo, useRef, useState, type ReactNode } from "react";
import { Link, useNavigate, useSearchParams } from "react-router-dom";

import {
  api,
  type ChatAttachment,
  type ChatMessage,
  type ChatThreadDetail,
  type ChatThreadSummary,
  type ChatToolGatewayActivity,
  type ModelRuntimeModel,
} from "../lib/api";
import { formatTime } from "../lib/format";
import { useUiStore } from "../stores/uiStore";

function asRecord(v: unknown): Record<string, unknown> | null {
  if (v && typeof v === "object" && !Array.isArray(v)) return v as Record<string, unknown>;
  return null;
}

function normalizeMessage(m: ChatMessage): ChatMessage {
  return { ...m, metadata: asRecord(m.metadata) ?? {} };
}

function normalizeThread(d: ChatThreadDetail): ChatThreadDetail {
  return {
    ...d,
    messages: Array.isArray(d.messages) ? d.messages.map((m) => normalizeMessage(m)) : [],
  };
}

function applyPostMessages(
  prev: ChatThreadDetail | null,
  threadId: number,
  res: { userMessage: ChatMessage; assistantMessage: ChatMessage | null },
): ChatThreadDetail | null {
  if (!prev || prev.id !== threadId) return prev;
  const next = [...prev.messages, normalizeMessage(res.userMessage)];
  if (res.assistantMessage) next.push(normalizeMessage(res.assistantMessage));
  const updatedAt = Math.max(prev.updatedAtMs, res.userMessage.createdAtMs, res.assistantMessage?.createdAtMs ?? 0);
  return { ...prev, messages: next, updatedAtMs: updatedAt };
}

function applyUserMessageOnly(prev: ChatThreadDetail | null, threadId: number, um: ChatMessage): ChatThreadDetail | null {
  if (!prev || prev.id !== threadId) return prev;
  return { ...prev, messages: [...prev.messages, normalizeMessage(um)], updatedAtMs: Math.max(prev.updatedAtMs, um.createdAtMs) };
}

function appendAssistantMessage(prev: ChatThreadDetail | null, threadId: number, am: ChatMessage): ChatThreadDetail | null {
  if (!prev || prev.id !== threadId) return prev;
  return {
    ...prev,
    messages: [...prev.messages, normalizeMessage(am)],
    updatedAtMs: Math.max(prev.updatedAtMs, am.createdAtMs),
  };
}

function findAssistantReply(messages: ChatMessage[], userMessageId: number): ChatMessage | undefined {
  return messages.find((message) => {
    if (message.role !== "assistant") return false;
    const raw = message.metadata?.replyToUserMessageId;
    return raw === userMessageId || raw === String(userMessageId) || Number(raw) === userMessageId;
  });
}

function sleep(ms: number) {
  return new Promise<void>((resolve) => setTimeout(resolve, ms));
}

function trimLine(value: string, fallback: string) {
  const trimmed = value.trim();
  return trimmed ? trimmed : fallback;
}

function readJobId(meta: Record<string, unknown> | undefined): string | null {
  if (!meta) return null;
  const raw = meta.jobId;
  if (typeof raw === "string" && raw.trim()) return raw.trim();
  return null;
}

function readCorrelationId(meta: Record<string, unknown> | undefined): string | null {
  if (!meta) return null;
  const raw = meta.correlationId;
  if (typeof raw === "string" && raw.trim()) return raw.trim();
  return null;
}

function readTraceId(meta: Record<string, unknown> | undefined): string | null {
  if (!meta) return null;
  const raw = meta.traceId;
  if (typeof raw === "string" && raw.trim()) return raw.trim();
  return null;
}

const CHAT_MODEL_SELECTION_CACHE_KEY = "forge.chat.requestedModelId.v1";

function readCachedChatModelSelection(): string {
  if (typeof window === "undefined") return "";
  try {
    const raw = window.localStorage.getItem(CHAT_MODEL_SELECTION_CACHE_KEY);
    return typeof raw === "string" ? raw.trim() : "";
  } catch {
    return "";
  }
}

function writeCachedChatModelSelection(value: string) {
  if (typeof window === "undefined") return;
  try {
    const trimmed = value.trim();
    if (!trimmed) {
      window.localStorage.removeItem(CHAT_MODEL_SELECTION_CACHE_KEY);
      return;
    }
    window.localStorage.setItem(CHAT_MODEL_SELECTION_CACHE_KEY, trimmed);
  } catch {
    return;
  }
}

function extractApiErrorMessage(err: unknown): string {
  const message = err instanceof Error ? err.message : String(err);
  const trimmed = message.trim();
  if (!trimmed.startsWith("{")) return trimmed;
  try {
    const parsed = JSON.parse(trimmed) as { error?: { message?: string } };
    return parsed?.error?.message?.trim() || trimmed;
  } catch {
    return trimmed;
  }
}

function supportsChatCapability(model: ModelRuntimeModel): boolean {
  const caps = Array.isArray(model.capabilities) ? model.capabilities : [];
  if (caps.length === 0) return true;
  return caps.some((cap) => {
    const normalized = String(cap).trim().toLowerCase();
    return normalized === "chat" || normalized === "completion";
  });
}

function usableChatStatus(model: ModelRuntimeModel): boolean {
  const status = String(model.status ?? "").trim().toLowerCase();
  return status !== "disabled" && status !== "archived" && status !== "unavailable" && status !== "error";
}

function readAttachments(meta: Record<string, unknown> | undefined): ChatAttachment[] {
  if (!meta) return [];
  const raw = meta.attachments;
  if (!Array.isArray(raw)) return [];
  const out: ChatAttachment[] = [];
  for (const item of raw) {
    const rec = asRecord(item);
    if (!rec) continue;
    const idRaw = rec.artifactId;
    const titleRaw = rec.title;
    const mimeRaw = rec.mimeType;
    const fileNameRaw = rec.fileName;
    const id = Number(idRaw);
    if (!Number.isFinite(id) || id <= 0) continue;
    const title = typeof titleRaw === "string" && titleRaw.trim() ? titleRaw.trim() : `Attachment #${id}`;
    const mimeType = typeof mimeRaw === "string" && mimeRaw.trim() ? mimeRaw.trim() : "application/octet-stream";
    const fileName = typeof fileNameRaw === "string" && fileNameRaw.trim() ? fileNameRaw.trim() : title;
    const textPreview = typeof rec.textPreview === "string" ? rec.textPreview : undefined;
    out.push({ artifactId: id, title, mimeType, fileName, textPreview });
  }
  return out;
}

type MessageCodeSnippet = {
  key: string;
  messageId: number;
  createdAtMs: number;
  lang: string;
  code: string;
};

type MessagePart =
  | { type: "text"; text: string }
  | { type: "code"; text: string; lang: string };

function parseMessageParts(content: string): MessagePart[] {
  const out: MessagePart[] = [];
  const re = /```([\w-]+)?\n([\s\S]*?)```/g;
  let last = 0;
  let m: RegExpExecArray | null;
  while ((m = re.exec(content)) !== null) {
    if (m.index > last) {
      out.push({ type: "text", text: content.slice(last, m.index) });
    }
    out.push({ type: "code", text: m[2] ?? "", lang: (m[1] ?? "").trim() });
    last = re.lastIndex;
  }
  if (last < content.length) out.push({ type: "text", text: content.slice(last) });
  if (out.length === 0) return [{ type: "text", text: content }];
  return out;
}

function renderInline(text: string): ReactNode[] {
  const chunks = text.split(/(`[^`]+`)/g);
  const nodes: ReactNode[] = [];
  chunks.forEach((chunk, i) => {
    if (!chunk) return;
    if (chunk.startsWith("`") && chunk.endsWith("`") && chunk.length >= 2) {
      nodes.push(
        <code key={`inline-${i}`} className="rounded border border-white/15 bg-black/35 px-1.5 py-0.5 font-mono text-[12px] text-forge-ash">
          {chunk.slice(1, -1)}
        </code>,
      );
      return;
    }
    const links = chunk.split(/(https?:\/\/[^\s]+)/g);
    links.forEach((part, j) => {
      if (!part) return;
      if (/^https?:\/\//.test(part)) {
        nodes.push(
          <a
            key={`link-${i}-${j}`}
            href={part}
            target="_blank"
            rel="noreferrer"
            className="text-forge-emberSoft underline underline-offset-2 hover:text-forge-ash"
          >
            {part}
          </a>,
        );
      } else {
        nodes.push(<span key={`txt-${i}-${j}`}>{part}</span>);
      }
    });
  });
  return nodes;
}

function readToolGatewayActivity(meta: Record<string, unknown> | undefined): ChatToolGatewayActivity | null {
  if (!meta) return null;
  const raw = meta.toolGatewayActivity;
  if (!raw || typeof raw !== "object" || Array.isArray(raw)) return null;
  const activity = raw as ChatToolGatewayActivity;
  const callsExecuted = typeof activity.toolCallsExecuted === "number" && Number.isFinite(activity.toolCallsExecuted) ? activity.toolCallsExecuted : 0;
  const toolCallEmitted = activity.toolCallEmitted === true;
  const state = typeof activity.executionState === "string" ? activity.executionState.trim().toLowerCase() : "";
  const approvalRequired = state === "needs_approval";

  const shouldSurface =
    callsExecuted > 0 ||
    toolCallEmitted ||
    (approvalRequired && (activity.toolSelected != null || activity.executionResult != null));

  if (!shouldSurface) return null;
  return activity;
}

function readApprovalRequestIdFromGatewayResult(executionResult: unknown): number | null {
  function walk(v: unknown): number | null {
    const rec = asRecord(v);
    if (rec) {
      const raw = rec.approvalRequestId;
      if (typeof raw === "number" && Number.isFinite(raw)) return raw;
      if (typeof raw === "string" && /^\d+$/.test(raw.trim())) return Number(raw.trim());
      for (const child of Object.values(rec)) {
        const nested = walk(child);
        if (nested != null) return nested;
      }
    }
    if (Array.isArray(v)) {
      for (const item of v) {
        const nested = walk(item);
        if (nested != null) return nested;
      }
    }
    return null;
  }
  return walk(executionResult);
}

function readGatewayJobId(executionResult: unknown): string | null {
  function walk(v: unknown): string | null {
    const rec = asRecord(v);
    if (rec) {
      const raw = rec.jobId;
      if (typeof raw === "string" && raw.trim()) return raw.trim();
      for (const child of Object.values(rec)) {
        const nested = walk(child);
        if (nested) return nested;
      }
    }
    if (Array.isArray(v)) {
      for (const item of v) {
        const nested = walk(item);
        if (nested) return nested;
      }
    }
    return null;
  }
  return walk(executionResult);
}

type ApprovalUiResolution = {
  decision: string;
  updatedAtMs: number;
};

const APPROVAL_STATE_CACHE_KEY = "forge.chat.approvalResolution.v1";

function readApprovalCache(): Record<number, ApprovalUiResolution> {
  if (typeof window === "undefined") return {};
  try {
    const raw = window.localStorage.getItem(APPROVAL_STATE_CACHE_KEY);
    if (!raw) return {};
    const parsed = JSON.parse(raw) as Record<string, ApprovalUiResolution>;
    const out: Record<number, ApprovalUiResolution> = {};
    for (const [rawKey, value] of Object.entries(parsed)) {
      const id = Number(rawKey);
      if (!Number.isFinite(id) || id <= 0) continue;
      if (!value || typeof value !== "object") continue;
      const decision = typeof value.decision === "string" && value.decision.trim() ? value.decision.trim() : "resolved";
      const updatedAtMs = Number(value.updatedAtMs);
      out[id] = {
        decision,
        updatedAtMs: Number.isFinite(updatedAtMs) ? updatedAtMs : Date.now(),
      };
    }
    return out;
  } catch {
    return {};
  }
}

function writeApprovalCache(next: Record<number, ApprovalUiResolution>) {
  if (typeof window === "undefined") return;
  try {
    const serializable: Record<string, ApprovalUiResolution> = {};
    for (const [id, value] of Object.entries(next)) {
      serializable[id] = value;
    }
    window.localStorage.setItem(APPROVAL_STATE_CACHE_KEY, JSON.stringify(serializable));
  } catch {
    return;
  }
}

function readApprovalResolutionFromCache(approvalId: number | null): ApprovalUiResolution | null {
  if (!approvalId) return null;
  const cache = readApprovalCache();
  return cache[approvalId] ?? null;
}

function clearApprovalResolvedFromCache(approvalId: number | null) {
  if (!approvalId) return;
  const next = readApprovalCache();
  delete next[approvalId];
  writeApprovalCache(next);
}

function markApprovalResolvedInCache(approvalId: number | null, decision: string) {
  if (!approvalId) return;
  const next = readApprovalCache();
  next[approvalId] = {
    decision: decision || "resolved",
    updatedAtMs: Date.now(),
  };
  writeApprovalCache(next);
}

function ToolGatewayActivityPanel(props: { activity: ChatToolGatewayActivity }) {
  const a = props.activity;
  const [approvalBusy, setApprovalBusy] = useState(false);
  const [approvalStatus, setApprovalStatus] = useState<string | null>(null);
  const [approvalState, setApprovalState] = useState<string | null>(null);
  const [gatewayJobStatus, setGatewayJobStatus] = useState<string | null>(null);
  const [gatewayJobSummary, setGatewayJobSummary] = useState<string | null>(null);
  const [gatewayJobFailure, setGatewayJobFailure] = useState<string | null>(null);
  const approvalId = useMemo(() => readApprovalRequestIdFromGatewayResult(a.executionResult), [a.executionResult]);
  const gatewayJobId = useMemo(() => readGatewayJobId(a.executionResult), [a.executionResult]);
  const approvalResolved = approvalState != null && approvalState !== "pending";
  const showApprovalActions = approvalId != null && !approvalResolved;
  const gatewayJobTerminal = gatewayJobStatus === "succeeded" || gatewayJobStatus === "failed" || gatewayJobStatus === "cancelled";
  const stages = Array.isArray(a.stages) ? (a.stages as Record<string, unknown>[]) : [];
  const args = a.toolArgs && typeof a.toolArgs === "object" && !Array.isArray(a.toolArgs) ? (a.toolArgs as Record<string, unknown>) : null;

  const isAlreadyResolvedError = useCallback((message: string) => {
    const value = message.toLowerCase();
    return value.includes("not pending") || value.includes("already") || value.includes("resolved");
  }, []);

  const refreshApproval = useCallback(async () => {
    if (approvalId == null) {
      setApprovalState(null);
      return;
    }
    try {
      const res = await api.approvals.get(approvalId);
      const status = String(res.approval?.status ?? "").trim().toLowerCase();
      if (status === "pending") {
        setApprovalState("pending");
        clearApprovalResolvedFromCache(approvalId);
        setApprovalStatus(null);
        return;
      }
      const decision = String(res.approval?.decision?.decision ?? "").trim().toLowerCase();
      const resolvedState = decision || status || "resolved";
      setApprovalState(resolvedState);
      markApprovalResolvedInCache(approvalId, resolvedState);
    } catch {
      const cached = readApprovalResolutionFromCache(approvalId);
      if (cached) {
        setApprovalState(cached.decision || "resolved");
      }
    }
  }, [approvalId]);

  useEffect(() => {
    if (approvalId == null) {
      setApprovalState(null);
      setApprovalBusy(false);
      setApprovalStatus(null);
      return;
    }
    const cached = readApprovalResolutionFromCache(approvalId);
    setApprovalState(cached?.decision ?? null);
    setApprovalBusy(false);
    setApprovalStatus(null);
    void refreshApproval();
  }, [approvalId, refreshApproval]);

  useEffect(() => {
    if (approvalId == null || approvalResolved) {
      return;
    }
    const timer = window.setInterval(() => {
      void refreshApproval();
    }, 3500);
    return () => {
      window.clearInterval(timer);
    };
  }, [approvalId, approvalResolved, refreshApproval]);

  useEffect(() => {
    if (gatewayJobId == null) {
      setGatewayJobStatus(null);
      setGatewayJobSummary(null);
      setGatewayJobFailure(null);
      return;
    }
    setGatewayJobStatus(null);
    setGatewayJobSummary(null);
    setGatewayJobFailure(null);
  }, [gatewayJobId]);

  useEffect(() => {
    if (gatewayJobId == null || !approvalResolved || approvalState === "denied" || gatewayJobTerminal) {
      return;
    }
    const resolvedGatewayJobId = gatewayJobId;
    let cancelled = false;
    async function refreshGatewayJob() {
      try {
        const detail = await api.jobs.detail(resolvedGatewayJobId, 0);
        if (cancelled) return;
        const status = String(detail.job?.status ?? "").trim().toLowerCase();
        const summary = String(detail.job?.resultSummary ?? "").trim();
        const failure = String(detail.job?.failureInfo ?? detail.job?.lastError ?? "").trim();
        setGatewayJobStatus(status || "unknown");
        setGatewayJobSummary(summary || null);
        setGatewayJobFailure(failure || null);
      } catch (e) {
        if (cancelled) return;
        setGatewayJobStatus((prev) => prev ?? "unknown");
        setGatewayJobFailure(e instanceof Error ? e.message : String(e));
      }
    }
    void refreshGatewayJob();
    const timer = window.setInterval(() => {
      void refreshGatewayJob();
    }, 3000);
    return () => {
      cancelled = true;
      window.clearInterval(timer);
    };
  }, [gatewayJobId, approvalResolved, approvalState, gatewayJobTerminal]);

  return (
    <div className="forge-status-glow mt-3 rounded-lg border border-forge-ember/25 bg-forge-iron/50 px-3 py-2 text-[11px] leading-relaxed text-forge-mist">
      <div className="font-semibold uppercase tracking-[0.14em] text-forge-emberSoft/90">Tool gateway</div>
      {a.userRequestSummary ? (
        <div className="mt-2">
          <span className="text-forge-mist/70">Request · </span>
          <span className="whitespace-pre-wrap text-forge-ash">{a.userRequestSummary}</span>
        </div>
      ) : null}
      <div className="mt-2 grid gap-1 font-mono text-[10px] text-forge-mist/90">
        <div>
          <span className="text-forge-mist/60">Tool · </span>
          {a.toolSelected && String(a.toolSelected).trim() ? String(a.toolSelected) : a.toolCallEmitted ? "(model emitted tool call)" : "—"}
        </div>
        <div>
          <span className="text-forge-mist/60">Args · </span>
          {args ? summarizeInlineFields(args) : "—"}
        </div>
        <div>
          <span className="text-forge-mist/60">State · </span>
          {a.executionState ?? "—"}
        </div>
        {a.failureReason ? (
          <div className="text-forge-emberSoft">
            <span className="text-forge-mist/60">Failure · </span>
            {a.failureReason}
          </div>
        ) : null}
        {showApprovalActions ? (
          <div className="mt-2 flex flex-wrap items-center gap-2">
            <button
              type="button"
              disabled={approvalBusy}
              onClick={() => {
                void (async () => {
                  setApprovalBusy(true);
                  setApprovalStatus(null);
                  try {
                    await api.approvals.approve(approvalId!, "Approved from chat tool gateway");
                    setApprovalState("approved");
                    markApprovalResolvedInCache(approvalId, "approved");
                    setApprovalStatus("Approved. Execution continues as a gateway job.");
                    await refreshApproval();
                  } catch (e) {
                    const message = e instanceof Error ? e.message : String(e);
                    if (isAlreadyResolvedError(message)) {
                      setApprovalState("resolved");
                      markApprovalResolvedInCache(approvalId, "resolved");
                      setApprovalStatus("Approval already resolved.");
                    } else {
                      setApprovalStatus(message);
                    }
                    await refreshApproval();
                  } finally {
                    setApprovalBusy(false);
                  }
                })();
              }}
              className="forge-btn forge-btn--primary px-2 py-1 text-[10px] disabled:opacity-40"
            >
              Approve
            </button>
            <button
              type="button"
              disabled={approvalBusy}
              onClick={() => {
                void (async () => {
                  setApprovalBusy(true);
                  setApprovalStatus(null);
                  try {
                    await api.approvals.deny(approvalId!, "Denied from chat tool gateway");
                    setApprovalState("denied");
                    markApprovalResolvedInCache(approvalId, "denied");
                    setApprovalStatus("Denied.");
                    await refreshApproval();
                  } catch (e) {
                    const message = e instanceof Error ? e.message : String(e);
                    if (isAlreadyResolvedError(message)) {
                      setApprovalState("resolved");
                      markApprovalResolvedInCache(approvalId, "resolved");
                      setApprovalStatus("Approval already resolved.");
                    } else {
                      setApprovalStatus(message);
                    }
                    await refreshApproval();
                  } finally {
                    setApprovalBusy(false);
                  }
                })();
              }}
              className="forge-btn forge-btn--ghost px-2 py-1 text-[10px] disabled:opacity-40"
            >
              Deny
            </button>
            {gatewayJobId ? (
              <Link
                to={`/jobs/${encodeURIComponent(gatewayJobId)}`}
                className="text-[10px] text-forge-emberSoft underline underline-offset-2 hover:text-forge-ash"
              >
                View job
              </Link>
            ) : null}
          </div>
        ) : null}
        {approvalResolved && approvalId != null ? (
          <div className="mt-1 text-[10px] text-forge-mist/85">
            Approval resolved{approvalState && approvalState !== "resolved" ? ` (${approvalState}).` : "."}
          </div>
        ) : null}
        {gatewayJobId && approvalResolved && approvalState !== "denied" ? (
          <div className="mt-1 text-[10px] text-forge-mist/90">
            Gateway job {gatewayJobStatus ? <span className="text-forge-ash">{gatewayJobStatus}</span> : "status pending"}.
            {gatewayJobSummary ? <span> {gatewayJobSummary}</span> : null}
            {gatewayJobFailure ? <span className="text-forge-emberSoft"> {gatewayJobFailure}</span> : null}
          </div>
        ) : null}
        {approvalStatus ? <div className="mt-1 text-[10px] text-forge-mist/90">{approvalStatus}</div> : null}
        {a.executionState === "needs_approval" && approvalId == null ? (
          <div className="mt-1 text-[10px] text-forge-emberSoft/90">
            Approval is required, but no approval request id was recorded. Check that the approvals service and database are available.
          </div>
        ) : null}
        {a.executionResult !== undefined ? <InlineStructuredSummary value={a.executionResult} /> : null}
      </div>
      {stages.length > 0 ? (
        <details className="mt-2">
          <summary className="cursor-pointer text-[10px] text-forge-mist/80">Pipeline ({stages.length} stages)</summary>
          <ol className="mt-1 list-decimal space-y-1 pl-4 text-[10px] text-forge-mist/85">
            {stages.map((row, i) => (
              <li key={`st-${i}`}>
                <span className="text-forge-ash">{String(row.stage ?? "?")}</span>
                {row.atMs != null ? <span className="text-forge-mist/50"> · {String(row.atMs)}</span> : null}
              </li>
            ))}
          </ol>
        </details>
      ) : null}
    </div>
  );
}

function summarizeInlineFields(value: Record<string, unknown>) {
  const entries = Object.entries(value);
  if (entries.length === 0) return "none";
  return entries
    .slice(0, 6)
    .map(([key, raw]) => `${key}=${inlineValue(raw)}`)
    .join(" · ");
}

function inlineValue(raw: unknown) {
  if (raw == null) return "—";
  if (typeof raw === "string") return raw.trim() || "—";
  if (typeof raw === "number" || typeof raw === "boolean") return String(raw);
  if (Array.isArray(raw)) return `${raw.length} item(s)`;
  if (typeof raw === "object") return `${Object.keys(raw as Record<string, unknown>).length} field(s)`;
  return "value";
}

function InlineStructuredSummary(props: { value: unknown }) {
  const rows = summarizeResultRows(props.value);
  if (rows.length === 0) {
    return (
      <div className="mt-1 rounded border border-white/10 bg-black/30 px-2 py-1 text-[10px] text-forge-mist">
        Execution result recorded.
      </div>
    );
  }
  return (
    <div className="mt-1 rounded border border-white/10 bg-black/30 p-2 text-[10px] text-forge-mist">
      <div className="font-semibold uppercase tracking-[0.14em] text-forge-mist/70">Execution Result</div>
      <div className="mt-1 grid gap-1">
        {rows.map(([label, value]) => (
          <div key={label} className="flex items-start justify-between gap-3">
            <span className="text-forge-mist/70">{label}</span>
            <span className="text-right text-forge-ash">{value}</span>
          </div>
        ))}
      </div>
    </div>
  );
}

function summarizeResultRows(value: unknown): Array<[string, string]> {
  if (value == null) return [];
  if (typeof value === "string" || typeof value === "number" || typeof value === "boolean") {
    return [["Result", String(value)]];
  }
  if (Array.isArray(value)) {
    return [["Result items", String(value.length)]];
  }
  if (typeof value !== "object") return [];
  const row = value as Record<string, unknown>;
  const keys = ["status", "policyOutcome", "approvalRequestId", "jobId", "gatewayStatus", "reason", "error", "message"];
  const out: Array<[string, string]> = [];
  for (const key of keys) {
    const raw = row[key];
    if (raw == null) continue;
    if (typeof raw === "string" || typeof raw === "number" || typeof raw === "boolean") {
      out.push([key, String(raw)]);
    } else if (Array.isArray(raw)) {
      out.push([key, `${raw.length} item(s)`]);
    } else {
      out.push([key, `${Object.keys(raw as Record<string, unknown>).length} field(s)`]);
    }
  }
  if (out.length > 0) return out;
  return [["Fields", String(Object.keys(row).length)]];
}

function RichMessage(props: { content: string }) {
  const parts = useMemo(() => parseMessageParts(props.content), [props.content]);
  return (
    <div className="space-y-3">
      {parts.map((part, idx) => {
        if (part.type === "code") {
          return <CodeBlock key={`code-${idx}`} code={part.text} lang={part.lang} />;
        }
        const paragraphs = part.text
          .split(/\n{2,}/)
          .map((v) => v.trimEnd())
          .filter((v) => v.length > 0);
        if (paragraphs.length === 0) return null;
        return (
          <div key={`text-${idx}`} className="space-y-2 text-[15px] leading-7 text-forge-ash">
            {paragraphs.map((p, pIdx) => (
              <p key={`p-${idx}-${pIdx}`} className="whitespace-pre-wrap break-words">
                {renderInline(p)}
              </p>
            ))}
          </div>
        );
      })}
    </div>
  );
}

function CodeBlock(props: { code: string; lang: string }) {
  const [copied, setCopied] = useState(false);

  async function copy() {
    try {
      await navigator.clipboard.writeText(props.code);
      setCopied(true);
      window.setTimeout(() => setCopied(false), 1200);
    } catch {
      setCopied(false);
    }
  }

  return (
    <div className="overflow-hidden rounded-xl border border-white/12 bg-[#0a0d12]">
      <div className="flex items-center justify-between border-b border-white/10 px-3 py-1.5 text-[11px] text-forge-mist">
        <span className="uppercase tracking-[0.12em]">{props.lang || "code"}</span>
        <button
          type="button"
          onClick={() => void copy()}
          className="rounded border border-white/10 bg-white/5 px-2 py-1 text-[10px] text-forge-mist transition hover:bg-white/10"
        >
          {copied ? "Copied" : "Copy"}
        </button>
      </div>
      <pre className="forge-chat-scroll overflow-x-auto px-3 py-3 text-[12px] leading-6 text-forge-ash">
        <code>{props.code}</code>
      </pre>
    </div>
  );
}

export function ChatPage() {
  const navigate = useNavigate();
  const [params, setParams] = useSearchParams();
  const setStatus = useUiStore((s) => s.setStatusLine);

  const [threads, setThreads] = useState<ChatThreadSummary[]>([]);
  const [active, setActive] = useState<ChatThreadDetail | null>(null);
  const [draft, setDraft] = useState("");
  const [threadFilter, setThreadFilter] = useState("");
  const [requestAssistant, setRequestAssistant] = useState(true);
  const [assistantDryRun, setAssistantDryRun] = useState(false);
  const [streamAssistant, setStreamAssistant] = useState(true);
  const [blockingAssistant, setBlockingAssistant] = useState(false);
  const [showAdvanced, setShowAdvanced] = useState(false);
  const [showForgeActions, setShowForgeActions] = useState(false);
  const [streamingText, setStreamingText] = useState<string | null>(null);
  const [streamingEvents, setStreamingEvents] = useState<Array<{ at: number; kind: string; text: string }>>([]);
  const [templates, setTemplates] = useState<JobTemplate[]>([]);
  const [isEditingTitle, setIsEditingTitle] = useState(false);
  const [titleDraft, setTitleDraft] = useState("");
  const [busy, setBusy] = useState(false);
  const [uploading, setUploading] = useState(false);
  const [err, setErr] = useState<string | null>(null);
  const [pendingAttachments, setPendingAttachments] = useState<ChatAttachment[]>([]);
  const [inspectorMode, setInspectorMode] = useState<"code" | "files">("code");
  const [selectedSnippetKey, setSelectedSnippetKey] = useState<string>("");
  const [selectedAttachmentId, setSelectedAttachmentId] = useState<number>(0);
  const [chatModels, setChatModels] = useState<ModelRuntimeModel[]>([]);
  const [chatModelLoadState, setChatModelLoadState] = useState<"idle" | "loading" | "ready" | "unavailable" | "error">("idle");
  const [chatModelMessage, setChatModelMessage] = useState("");
  const [selectedChatModelId, setSelectedChatModelId] = useState<string>(() => readCachedChatModelSelection());

  const [jobForm, setJobForm] = useState<{
    templateId: string;
    title: string;
    userRequest: string;
    objective: string;
  }>({
    templateId: "search_packet",
    title: "Queued from chat",
    userRequest: "",
    objective: "",
  });

  const messagesEndRef = useRef<HTMLDivElement>(null);
  const textareaRef = useRef<HTMLTextAreaElement>(null);
  const streamEsRef = useRef<EventSource | null>(null);
  const fileInputRef = useRef<HTMLInputElement>(null);

  useEffect(() => {
    return () => {
      streamEsRef.current?.close();
      streamEsRef.current = null;
    };
  }, []);

  useEffect(() => {
    if (!textareaRef.current) return;
    textareaRef.current.style.height = "0px";
    const next = Math.max(80, Math.min(300, textareaRef.current.scrollHeight));
    textareaRef.current.style.height = `${next}px`;
  }, [draft]);

  const threadIdParam = params.get("threadId");

  const refreshThreads = useCallback(async () => {
    const res = await api.chat.threads.list(120);
    setThreads(Array.isArray(res?.threads) ? res.threads : []);
  }, []);

  const refreshChatModels = useCallback(async () => {
    setChatModelLoadState("loading");
    setChatModelMessage("");
    try {
      const res = await api.modelRuntime.list();
      const next = (Array.isArray(res.models) ? res.models : [])
        .filter((model) => supportsChatCapability(model) && usableChatStatus(model))
        .sort((a, b) => a.id.localeCompare(b.id));
      setChatModels(next);
      setSelectedChatModelId((prev) => (prev && next.some((item) => item.id === prev) ? prev : ""));
      setChatModelLoadState("ready");
      if (next.length === 0) {
        setChatModelMessage("No chat-capable runtime models are currently available.");
      }
    } catch (e) {
      const message = extractApiErrorMessage(e);
      setChatModels([]);
      if ((e instanceof Error ? e.message : String(e)).includes("MODEL_RUNTIME_UNAVAILABLE")) {
        setChatModelLoadState("unavailable");
        setChatModelMessage("Model runtime is unavailable. Chat will use configured adapter fallback.");
      } else {
        setChatModelLoadState("error");
        setChatModelMessage(message || "Model runtime model list could not be loaded.");
      }
    }
  }, []);

  const loadThread = useCallback(
    async (id: number) => {
      const d = await api.chat.threads.get(id);
      const normalized = normalizeThread(d);
      setActive(normalized);
      setTitleDraft(trimLine(normalized.title, `Thread #${normalized.id}`));
      setIsEditingTitle(false);
      setParams({ threadId: String(normalized.id) });
    },
    [setParams],
  );

  useEffect(() => {
    let cancelled = false;
    async function boot() {
      try {
        const [tRes, tpl] = await Promise.all([api.chat.threads.list(120), api.jobs.templates()]);
        if (cancelled) return;
        const threadList = Array.isArray(tRes?.threads) ? tRes.threads : [];
        const templateList = Array.isArray(tpl?.templates) ? tpl.templates : [];
        setThreads(threadList);
        setTemplates(templateList);
        setErr(null);

        const fromUrl = threadIdParam ? Number(threadIdParam) : NaN;
        if (Number.isFinite(fromUrl) && fromUrl > 0) {
          const d = await api.chat.threads.get(fromUrl);
          if (!cancelled) {
            const normalized = normalizeThread(d);
            setActive(normalized);
            setTitleDraft(trimLine(normalized.title, `Thread #${normalized.id}`));
          }
          return;
        }
        if (threadList.length > 0 && !cancelled) {
          const d = await api.chat.threads.get(threadList[0].id);
          if (!cancelled) {
            const normalized = normalizeThread(d);
            setActive(normalized);
            setTitleDraft(trimLine(normalized.title, `Thread #${normalized.id}`));
            setParams({ threadId: String(d.id) });
          }
          return;
        }
        if (!cancelled) {
          setActive(null);
          setTitleDraft("");
        }
      } catch (e) {
        if (!cancelled) {
          setErr(e instanceof Error ? e.message : String(e));
          setActive(null);
          setTitleDraft("");
        }
      }
    }
    void boot();
    return () => {
      cancelled = true;
    };
  }, [threadIdParam, setParams]);

  useEffect(() => {
    writeCachedChatModelSelection(selectedChatModelId);
  }, [selectedChatModelId]);

  useEffect(() => {
    void refreshChatModels();
  }, [refreshChatModels]);

  useEffect(() => {
    messagesEndRef.current?.scrollIntoView({ behavior: "smooth" });
  }, [active?.id, active?.messages.length, streamingText]);

  async function newThread() {
    setBusy(true);
    try {
      const res = await api.chat.threads.create({ title: "New chat" });
      await refreshThreads();
      await loadThread(res.thread.id);
      setStatus(`Thread ${res.thread.id} created.`);
      setErr(null);
      setShowForgeActions(false);
    } catch (e) {
      setErr(e instanceof Error ? e.message : String(e));
    } finally {
      setBusy(false);
    }
  }

  async function saveThreadTitle() {
    if (!active) return;
    const title = titleDraft.trim();
    if (!title) {
      setTitleDraft(trimLine(active.title, `Thread #${active.id}`));
      setIsEditingTitle(false);
      return;
    }
    setBusy(true);
    try {
      const res = await api.chat.threads.update(active.id, { title });
      setActive((prev) => (prev ? { ...prev, title: res.thread.title, updatedAtMs: res.thread.updatedAtMs } : prev));
      setThreads((prev) => prev.map((t) => (t.id === res.thread.id ? { ...t, title: res.thread.title, updatedAtMs: res.thread.updatedAtMs } : t)));
      setIsEditingTitle(false);
      setStatus("Thread renamed.");
      setErr(null);
    } catch (e) {
      setErr(e instanceof Error ? e.message : String(e));
    } finally {
      setBusy(false);
    }
  }

  async function deleteActiveThread() {
    if (!active) return;
    if (!window.confirm("Delete this chat and all messages?")) return;
    setBusy(true);
    try {
      await api.chat.threads.delete(active.id);
      setActive(null);
      setTitleDraft("");
      setParams({});
      await refreshThreads();
      setStatus("Thread deleted.");
      setErr(null);
    } catch (e) {
      setErr(e instanceof Error ? e.message : String(e));
    } finally {
      setBusy(false);
    }
  }

  function stopStreaming() {
    streamEsRef.current?.close();
    streamEsRef.current = null;
    setStreamingText(null);
    setStreamingEvents([]);
    setStatus("Assistant stream stopped.");
  }

  async function send() {
    if (!active) return;
    const text = draft.trim();
    const hasAttachments = pendingAttachments.length > 0;
    if (!text && !hasAttachments) return;

    setBusy(true);
    setErr(null);
    setStreamingText(null);
    setStreamingEvents([]);
    streamEsRef.current?.close();
    streamEsRef.current = null;

    const useStream = requestAssistant && !assistantDryRun && streamAssistant && !blockingAssistant;
    const useSyncBlock = requestAssistant && !assistantDryRun && blockingAssistant;
    const useAsyncPoll = requestAssistant && !assistantDryRun && !useStream && !useSyncBlock;

    const body: Parameters<typeof api.chat.threads.postMessage>[1] = {
      content: text || "Attached files for context.",
      attachmentArtifactIds: pendingAttachments.map((item) => item.artifactId),
      requestAssistant,
      assistantDryRun,
    };
    const requestedModel = selectedChatModelId.trim();
    if (requestedModel) {
      body.modelId = requestedModel;
    }
    if (useStream) body.stream = true;
    else if (useSyncBlock) body.syncAssistant = true;
    else if (useAsyncPoll) body.asyncAssistant = true;

    try {
      const res = await api.chat.threads.postMessage(active.id, body);
      const tid = active.id;
      setDraft("");
      setPendingAttachments([]);

      if (res.assistantMessage) {
        setActive((prev) => applyPostMessages(prev, tid, res));
        void refreshThreads();
        setStatus(requestAssistant ? "Assistant reply saved." : "Message saved.");
        textareaRef.current?.focus();
        return;
      }

      setActive((prev) => applyUserMessageOnly(prev, tid, res.userMessage));
      void refreshThreads();

      if (res.assistantPending && res.stream === true && res.userMessageId != null) {
        setStatus("Streaming assistant reply…");
        await openAssistantStream(tid, res.userMessageId);
        setStatus("Assistant reply saved.");
        textareaRef.current?.focus();
        return;
      }

      if (res.assistantPending && res.asyncAssistant && res.userMessageId != null) {
        setStatus("Waiting for assistant reply…");
        await pollForAssistantReply(tid, res.userMessageId);
        setStatus("Assistant reply saved.");
        textareaRef.current?.focus();
        return;
      }

      setStatus(requestAssistant ? "Message saved (assistant not returned)." : "Message saved.");
      textareaRef.current?.focus();
    } catch (e) {
      setErr(e instanceof Error ? e.message : String(e));
    } finally {
      setBusy(false);
      setStreamingText(null);
    }
  }

  function openAssistantStream(threadId: number, userMessageId: number): Promise<void> {
    return new Promise((resolve, reject) => {
      const url = api.chat.threads.assistantStreamUrl(threadId, userMessageId);
      setStreamingText("");
      const es = new EventSource(url);
      streamEsRef.current = es;
      setStreamingEvents([]);

      es.addEventListener("token", (ev) => {
        try {
          const raw = JSON.parse((ev as MessageEvent).data as string) as { text?: string };
          const token = typeof raw.text === "string" ? raw.text : "";
          setStreamingText((prev) => (prev == null ? token : prev + token));
        } catch {
          /* ignore malformed token payload */
        }
      });

      es.addEventListener("done", (ev) => {
        try {
          const raw = JSON.parse((ev as MessageEvent).data as string) as { assistantMessage?: ChatMessage };
          es.close();
          streamEsRef.current = null;
          setStreamingText(null);
          setStreamingEvents([]);
          const am = raw.assistantMessage;
          if (am) {
            setActive((prev) => appendAssistantMessage(prev, threadId, am));
          }
          void refreshThreads();
          resolve();
        } catch (e) {
          es.close();
          streamEsRef.current = null;
          setStreamingText(null);
          reject(e);
        }
      });

      es.onerror = () => {
        es.close();
        streamEsRef.current = null;
        setStreamingText(null);
        setStreamingEvents([]);
        reject(new Error("Assistant stream disconnected."));
      };

      es.addEventListener("agent_stage", (ev) => {
        try {
          const raw = JSON.parse((ev as MessageEvent).data as string) as Record<string, unknown>;
          const stage = typeof raw.stage === "string" ? raw.stage : "stage";
          const at = typeof raw.atMs === "number" ? raw.atMs : Date.now();
          setStreamingEvents((prev) => [...prev.slice(-39), { at, kind: "stage", text: stage }]);
        } catch {
          /* ignore malformed stage payload */
        }
      });

      es.addEventListener("tool_call", (ev) => {
        try {
          const raw = JSON.parse((ev as MessageEvent).data as string) as Record<string, unknown>;
          const modelName = typeof raw.modelName === "string" ? raw.modelName : "tool";
          const at = Date.now();
          setStreamingEvents((prev) => [...prev.slice(-39), { at, kind: "call", text: `call ${modelName}` }]);
        } catch {
          /* ignore malformed tool_call payload */
        }
      });

      es.addEventListener("tool_result", (ev) => {
        try {
          const raw = JSON.parse((ev as MessageEvent).data as string) as Record<string, unknown>;
          const modelName = typeof raw.modelName === "string" ? raw.modelName : "tool";
          const state = typeof raw.state === "string" ? raw.state : "unknown";
          const at = Date.now();
          setStreamingEvents((prev) => [...prev.slice(-39), { at, kind: "result", text: `${modelName} -> ${state}` }]);
        } catch {
          /* ignore malformed tool_result payload */
        }
      });
    });
  }

  async function pollForAssistantReply(threadId: number, userMessageId: number) {
    const deadline = Date.now() + 120_000;
    while (Date.now() < deadline) {
      const d = await api.chat.threads.get(threadId);
      const messages = Array.isArray(d.messages) ? d.messages.map((m) => normalizeMessage(m)) : [];
      if (findAssistantReply(messages, userMessageId)) {
        setActive(normalizeThread(d));
        void refreshThreads();
        return;
      }
      await sleep(400);
    }
    throw new Error("Timed out waiting for assistant reply.");
  }

  function onComposerKeyDown(e: React.KeyboardEvent<HTMLTextAreaElement>) {
    if (e.key === "Enter" && !e.shiftKey) {
      e.preventDefault();
      void send();
    }
  }

  async function uploadSelectedFiles(list: FileList | null) {
    if (!active || !list || list.length === 0) return;
    const files = Array.from(list);
    setUploading(true);
    setErr(null);
    try {
      const uploaded: ChatAttachment[] = [];
      for (const file of files) {
        const res = await api.chat.threads.uploadAttachment(active.id, file, file.name);
        uploaded.push({
          artifactId: res.artifact.id,
          title: res.artifact.title,
          mimeType: res.artifact.mimeType,
          fileName: file.name,
          textPreview: res.previewText || undefined,
        });
      }
      setPendingAttachments((prev) => {
        const next = [...prev];
        for (const item of uploaded) {
          if (!next.some((existing) => existing.artifactId === item.artifactId)) next.push(item);
        }
        return next;
      });
      setStatus(`${uploaded.length} attachment(s) uploaded.`);
      setInspectorMode("files");
    } catch (e) {
      setErr(e instanceof Error ? e.message : String(e));
    } finally {
      setUploading(false);
      if (fileInputRef.current) fileInputRef.current.value = "";
    }
  }

  async function queueJob() {
    if (!active) return;
    const templateId = templates.some((t) => t.id === jobForm.templateId) ? jobForm.templateId : templates[0]?.id;
    if (!templateId) {
      setErr("No job templates available from core.");
      return;
    }
    const lastUserMessage = [...active.messages].reverse().find((m) => m.role === "user");
    const fallbackRequest = lastUserMessage?.content || "Job requested from chat";
    setBusy(true);
    try {
      const res = await api.chat.threads.queueJob(active.id, {
        templateId,
        title: jobForm.title.trim() || "Job from chat",
        userRequest: jobForm.userRequest.trim() || fallbackRequest,
        objective: jobForm.objective.trim() || "Operator-requested job from chat thread",
        query: jobForm.userRequest.trim() || fallbackRequest,
        initiatingSource: "chat",
      });
      setStatus(`Job ${res.job.id} queued from chat.`);
      setShowForgeActions(false);
      navigate(`/jobs/${res.job.id}`);
      await loadThread(active.id);
    } catch (e) {
      setErr(e instanceof Error ? e.message : String(e));
    } finally {
      setBusy(false);
    }
  }

  const filteredThreads = useMemo(() => {
    const query = threadFilter.trim().toLowerCase();
    const ordered = [...threads].sort((a, b) => b.updatedAtMs - a.updatedAtMs);
    if (!query) return ordered;
    return ordered.filter((thread) => `${thread.title} ${thread.id}`.toLowerCase().includes(query));
  }, [threadFilter, threads]);

  const messageAttachments = useMemo(() => {
    const out: Array<{ messageId: number; createdAtMs: number; role: string; attachment: ChatAttachment }> = [];
    for (const message of active?.messages ?? []) {
      for (const attachment of readAttachments(message.metadata)) {
        out.push({
          messageId: message.id,
          createdAtMs: message.createdAtMs,
          role: message.role,
          attachment,
        });
      }
    }
    const seen = new Set<number>();
    return out
      .sort((a, b) => b.createdAtMs - a.createdAtMs)
      .filter((item) => {
        if (seen.has(item.attachment.artifactId)) return false;
        seen.add(item.attachment.artifactId);
        return true;
      });
  }, [active?.messages]);

  const assistantCodeSnippets = useMemo<MessageCodeSnippet[]>(() => {
    const out: MessageCodeSnippet[] = [];
    for (const message of active?.messages ?? []) {
      if (message.role !== "assistant") continue;
      const parts = parseMessageParts(message.content);
      let index = 0;
      for (const part of parts) {
        if (part.type !== "code") continue;
        out.push({
          key: `${message.id}:${index}`,
          messageId: message.id,
          createdAtMs: message.createdAtMs,
          lang: part.lang || "code",
          code: part.text,
        });
        index += 1;
      }
    }
    return out.sort((a, b) => b.createdAtMs - a.createdAtMs);
  }, [active?.messages]);

  useEffect(() => {
    if (!selectedSnippetKey && assistantCodeSnippets.length > 0) {
      setSelectedSnippetKey(assistantCodeSnippets[0].key);
    }
    if (selectedSnippetKey && !assistantCodeSnippets.some((item) => item.key === selectedSnippetKey)) {
      setSelectedSnippetKey(assistantCodeSnippets[0]?.key ?? "");
    }
  }, [assistantCodeSnippets, selectedSnippetKey]);

  useEffect(() => {
    if (!selectedAttachmentId && messageAttachments.length > 0) {
      setSelectedAttachmentId(messageAttachments[0].attachment.artifactId);
    }
    if (selectedAttachmentId && !messageAttachments.some((item) => item.attachment.artifactId === selectedAttachmentId)) {
      setSelectedAttachmentId(messageAttachments[0]?.attachment.artifactId ?? 0);
    }
  }, [messageAttachments, selectedAttachmentId]);

  return (
    <div className="grid h-full min-h-0 overflow-hidden rounded-2xl border border-white/10 bg-[#090b0f] md:grid-cols-[280px_minmax(0,1fr)] xl:grid-cols-[280px_minmax(0,1fr)_380px]">
      <aside className="flex min-h-0 flex-col overflow-hidden border-b border-white/10 bg-[#0a0d12] md:border-b-0 md:border-r">
        <div className="border-b border-white/10 p-3">
          <button
            type="button"
            onClick={() => void newThread()}
            disabled={busy}
            className="w-full rounded-xl border border-white/15 bg-white/5 px-3 py-2 text-sm font-semibold text-forge-ash transition hover:bg-white/10 disabled:opacity-40"
          >
            + New chat
          </button>
          <input
            aria-label="Filter chats"
            value={threadFilter}
            onChange={(e) => setThreadFilter(e.target.value)}
            placeholder="Search chats"
            className="forge-input mt-3 text-xs"
          />
          {err ? <div className="mt-2 rounded border border-forge-ember/30 bg-forge-ember/10 p-2 text-xs text-forge-ash">{err}</div> : null}
        </div>

        <div className="forge-chat-scroll min-h-0 flex-1 overflow-y-auto p-2">
          {filteredThreads.length === 0 ? (
            <div className="rounded-lg border border-dashed border-white/10 px-3 py-4 text-xs text-forge-mist">No chats yet.</div>
          ) : (
            <div className="space-y-1">
              {filteredThreads.map((thread) => {
                const isActive = active?.id === thread.id;
                return (
                  <button
                    key={thread.id}
                    type="button"
                    onClick={() => void loadThread(thread.id)}
                    className={[
                      "w-full rounded-lg border px-3 py-2 text-left transition",
                      isActive
                        ? "border-white/25 bg-white/10 text-forge-ash"
                        : "border-transparent bg-transparent text-forge-mist hover:border-white/10 hover:bg-white/5",
                    ].join(" ")}
                  >
                    <div className="truncate text-sm font-semibold">{trimLine(thread.title, `Thread #${thread.id}`)}</div>
                    <div className="mt-1 text-[10px] text-forge-mist/70">{formatTime(thread.updatedAtMs)}</div>
                  </button>
                );
              })}
            </div>
          )}
        </div>
      </aside>

      <section className="flex min-h-0 min-w-0 flex-col overflow-hidden">
        {!active ? (
          <div className="flex flex-1 items-center justify-center p-8">
            <div className="max-w-md rounded-xl border border-dashed border-white/10 bg-black/25 px-6 py-8 text-center">
              <h2 className="text-lg font-semibold text-forge-ash">Start a new chat</h2>
              <p className="mt-2 text-sm text-forge-mist">Pick a chat on the left or create a new one.</p>
              <button
                type="button"
                onClick={() => void newThread()}
                disabled={busy}
                className="mt-5 rounded-lg border border-white/15 bg-white/5 px-4 py-2 text-sm font-semibold text-forge-ash transition hover:bg-white/10 disabled:opacity-45"
              >
                New chat
              </button>
            </div>
          </div>
        ) : (
          <>
            <header className="forge-chat-header">
              <div className="mx-auto flex w-full max-w-4xl items-center justify-between gap-3">
                <div className="min-w-0 flex-1">
                  {isEditingTitle ? (
                    <div className="flex items-center gap-2">
                      <label htmlFor="thread-title" className="sr-only">
                        Thread title
                      </label>
                      <input
                        id="thread-title"
                        className="forge-input h-9 text-sm"
                        value={titleDraft}
                        onChange={(e) => setTitleDraft(e.target.value)}
                        onKeyDown={(e) => {
                          if (e.key === "Enter") {
                            e.preventDefault();
                            void saveThreadTitle();
                          }
                          if (e.key === "Escape") {
                            e.preventDefault();
                            setTitleDraft(trimLine(active.title, `Thread #${active.id}`));
                            setIsEditingTitle(false);
                          }
                        }}
                        autoFocus
                      />
                      <button
                        type="button"
                        onClick={() => void saveThreadTitle()}
                        className="forge-chat-action-btn"
                      >
                        Save
                      </button>
                    </div>
                  ) : (
                    <div className="flex items-center gap-2">
                      <h2 className="truncate text-base font-semibold text-forge-ash">{trimLine(active.title, `Thread #${active.id}`)}</h2>
                      <button
                        type="button"
                        onClick={() => setIsEditingTitle(true)}
                        className="rounded border border-transparent px-2 py-1 text-[11px] text-forge-mist transition hover:border-white/10 hover:text-forge-ash"
                      >
                        Rename
                      </button>
                    </div>
                  )}
                  <p className="text-xs text-forge-mist">
                    {active.messages.length} message(s) · updated {formatTime(active.updatedAtMs)}
                  </p>
                </div>

                <div className="flex items-center gap-2">
                  {streamingText !== null ? (
                    <button
                      type="button"
                      onClick={stopStreaming}
                      className="rounded-lg border border-forge-ember/35 bg-forge-ember/10 px-3 py-1.5 text-xs font-semibold text-forge-emberSoft"
                    >
                      Stop
                    </button>
                  ) : null}
                  <button
                    type="button"
                    onClick={() => setShowForgeActions((v) => !v)}
                    className="forge-chat-action-btn bg-transparent"
                  >
                    {showForgeActions ? "Hide Forge" : "Forge Actions"}
                  </button>
                  <button
                    type="button"
                    onClick={() => void deleteActiveThread()}
                    disabled={busy}
                    className="forge-chat-action-btn bg-transparent border-white/10 px-2.5"
                  >
                    Delete
                  </button>
                </div>
              </div>
            </header>

            {showForgeActions ? (
              <div className="forge-chat-toolbar">
                <div className="mx-auto grid w-full max-w-4xl gap-2 md:grid-cols-2">
                  <label className="block">
                    <span className="text-[11px] font-medium uppercase tracking-wide text-forge-mist">Template</span>
                    <select
                      aria-label="Job template"
                      className="forge-input mt-1"
                      value={templates.some((t) => t.id === jobForm.templateId) ? jobForm.templateId : templates[0]?.id ?? ""}
                      onChange={(e) => setJobForm((f) => ({ ...f, templateId: e.target.value }))}
                    >
                      {templates.length === 0 ? (
                        <option value="">No templates available</option>
                      ) : (
                        templates.map((tpl) => (
                          <option key={tpl.id} value={tpl.id}>
                            {tpl.name} ({tpl.id})
                          </option>
                        ))
                      )}
                    </select>
                  </label>
                  <label className="block">
                    <span className="text-[11px] font-medium uppercase tracking-wide text-forge-mist">Title</span>
                    <input
                      aria-label="Job title"
                      className="forge-input mt-1"
                      value={jobForm.title}
                      onChange={(e) => setJobForm((f) => ({ ...f, title: e.target.value }))}
                    />
                  </label>
                  <label className="block md:col-span-2">
                    <span className="text-[11px] font-medium uppercase tracking-wide text-forge-mist">Request override</span>
                    <input
                      aria-label="Job request"
                      className="forge-input mt-1"
                      value={jobForm.userRequest}
                      onChange={(e) => setJobForm((f) => ({ ...f, userRequest: e.target.value }))}
                      placeholder="Optional override for packet request"
                    />
                  </label>
                  <label className="block md:col-span-2">
                    <span className="text-[11px] font-medium uppercase tracking-wide text-forge-mist">Objective</span>
                    <input
                      aria-label="Job objective"
                      className="forge-input mt-1"
                      value={jobForm.objective}
                      onChange={(e) => setJobForm((f) => ({ ...f, objective: e.target.value }))}
                    />
                  </label>
                </div>
                <div className="mx-auto mt-3 flex w-full max-w-4xl gap-2">
                  <button
                    type="button"
                    onClick={() => void queueJob()}
                    disabled={busy || templates.length === 0}
                    className="forge-chat-action-btn"
                  >
                    Queue Job
                  </button>
                </div>
              </div>
            ) : null}

            <div className="forge-chat-scroll min-h-0 flex-1 overflow-y-auto px-4 py-6">
              <div className="mx-auto flex w-full max-w-4xl flex-col gap-6">
                {active.messages.length === 0 ? (
                  <div className="rounded-xl border border-dashed border-white/10 bg-black/25 px-5 py-6 text-center">
                    <div className="text-sm font-semibold text-forge-ash">No messages yet</div>
                    <div className="mt-1 text-xs text-forge-mist">Send the first message to begin.</div>
                  </div>
                ) : (
                  active.messages.map((message) => (
                    <MessageRow
                      key={message.id}
                      message={normalizeMessage(message)}
                      onInspectAttachment={(artifactId) => {
                        setInspectorMode("files");
                        setSelectedAttachmentId(artifactId);
                      }}
                    />
                  ))
                )}

                {streamingText !== null ? (
                  <div className="rounded-xl border border-white/12 bg-[#0c1118] p-4">
                    <div className="mb-2 text-[10px] uppercase tracking-wide text-forge-mist/70">Assistant is typing…</div>
                    <div className="text-[15px] leading-7 text-forge-ash whitespace-pre-wrap break-words">{streamingText || "…"}</div>
                    {streamingEvents.length > 0 ? (
                      <div className="mt-3 rounded-lg border border-white/10 bg-black/25 p-2">
                        <div className="mb-1 text-[10px] uppercase tracking-wide text-forge-mist/70">Agent timeline</div>
                        <div className="forge-chat-scroll max-h-36 overflow-y-auto space-y-1">
                          {streamingEvents.map((evt, i) => (
                            <div key={`se-${i}-${evt.at}`} className="font-mono text-[10px] text-forge-mist/90">
                              {formatTime(evt.at)} · {evt.text}
                            </div>
                          ))}
                        </div>
                      </div>
                    ) : null}
                  </div>
                ) : null}
                <div ref={messagesEndRef} className="h-px w-full" aria-hidden />
              </div>
            </div>

            <footer className="forge-chat-footer">
              <div className="mx-auto w-full max-w-4xl">
                <div className="mb-2 flex flex-wrap items-center gap-3 text-xs text-forge-mist">
                  <label className="inline-flex items-center gap-2">
                    <input
                      aria-label="Use assistant"
                      type="checkbox"
                      checked={requestAssistant}
                      onChange={(e) => setRequestAssistant(e.target.checked)}
                    />
                    Use assistant
                  </label>
                  <label className="inline-flex items-center gap-2">
                    <span className="text-[11px] text-forge-mist/80">Model</span>
                    <select
                      aria-label="Chat runtime model"
                      className="forge-input h-8 min-w-[240px] py-1 text-xs"
                      value={selectedChatModelId}
                      onChange={(e) => setSelectedChatModelId(e.target.value)}
                      disabled={chatModelLoadState === "loading"}
                    >
                      <option value="">Auto (runtime default / adapter fallback)</option>
                      {selectedChatModelId && !chatModels.some((model) => model.id === selectedChatModelId) ? (
                        <option value={selectedChatModelId}>Saved: {selectedChatModelId} (not in current runtime list)</option>
                      ) : null}
                      {chatModels.map((model) => (
                        <option key={model.id} value={model.id}>
                          {model.displayName?.trim() || model.id}
                        </option>
                      ))}
                    </select>
                  </label>
                  <button
                    type="button"
                    onClick={() => setShowAdvanced((v) => !v)}
                    className="forge-chat-action-btn border-white/10 bg-transparent py-1 px-2.5 text-[11px]"
                  >
                    {showAdvanced ? "Hide advanced" : "Advanced"}
                  </button>
                </div>
                {chatModelMessage ? (
                  <div className="mb-2 text-[10px] text-forge-mist/75">
                    {chatModelMessage}{" "}
                    {(chatModelLoadState === "unavailable" || chatModelLoadState === "error") && (
                      <Link to="/models" className="text-forge-emberSoft underline underline-offset-2 hover:text-forge-ash">
                        Open Models
                      </Link>
                    )}
                  </div>
                ) : null}

                {showAdvanced ? (
                  <div className="mb-3 grid gap-2 md:grid-cols-3">
                    <label className="inline-flex items-center gap-2 rounded border border-white/10 bg-black/25 px-2.5 py-2 text-xs text-forge-mist">
                      <input
                        aria-label="Assistant dry run"
                        type="checkbox"
                        checked={assistantDryRun}
                        disabled={!requestAssistant}
                        onChange={(e) => setAssistantDryRun(e.target.checked)}
                      />
                      Dry-run
                    </label>
                    <label className="inline-flex items-center gap-2 rounded border border-white/10 bg-black/25 px-2.5 py-2 text-xs text-forge-mist">
                      <input
                        aria-label="Stream assistant response"
                        type="checkbox"
                        checked={streamAssistant}
                        disabled={!requestAssistant || assistantDryRun || blockingAssistant}
                        onChange={(e) => {
                          setStreamAssistant(e.target.checked);
                          if (e.target.checked) setBlockingAssistant(false);
                        }}
                      />
                      Stream response
                    </label>
                    <label className="inline-flex items-center gap-2 rounded border border-white/10 bg-black/25 px-2.5 py-2 text-xs text-forge-mist">
                      <input
                        aria-label="Block on assistant response"
                        type="checkbox"
                        checked={blockingAssistant}
                        disabled={!requestAssistant || assistantDryRun}
                        onChange={(e) => {
                          setBlockingAssistant(e.target.checked);
                          if (e.target.checked) setStreamAssistant(false);
                        }}
                      />
                      Block request
                    </label>
                  </div>
                ) : null}

                <div className="flex items-end gap-2">
                  <label htmlFor="chat-composer" className="sr-only">
                    Message
                  </label>
                  <textarea
                    id="chat-composer"
                    aria-label="Chat message"
                    ref={textareaRef}
                    rows={1}
                    className="forge-chat-composer"
                    placeholder="Message FORGE"
                    value={draft}
                    onChange={(e) => setDraft(e.target.value)}
                    onKeyDown={onComposerKeyDown}
                    disabled={busy}
                  />
                  <input
                    ref={fileInputRef}
                    type="file"
                    className="hidden"
                    multiple
                    onChange={(e) => void uploadSelectedFiles(e.target.files)}
                    disabled={busy || uploading}
                  />
                  <button
                    type="button"
                    onClick={() => fileInputRef.current?.click()}
                    disabled={busy || uploading || !active}
                    className="h-[80px] min-w-[88px] forge-chat-action-btn"
                  >
                    {uploading ? "Uploading…" : "Attach"}
                  </button>
                  <button
                    type="button"
                    onClick={() => void send()}
                    disabled={busy || (!draft.trim() && pendingAttachments.length === 0)}
                    className="h-[80px] min-w-[96px] forge-chat-action-btn text-sm"
                  >
                    Send
                  </button>
                </div>
                {pendingAttachments.length > 0 ? (
                  <div className="mt-2 flex flex-wrap gap-2">
                    {pendingAttachments.map((item) => (
                      <span key={item.artifactId} className="inline-flex items-center gap-2 rounded-full border border-white/15 bg-white/5 px-2.5 py-1 text-[11px] text-forge-mist">
                        <button
                          type="button"
                          onClick={() => {
                            setInspectorMode("files");
                            setSelectedAttachmentId(item.artifactId);
                          }}
                          className="truncate hover:text-forge-ash"
                        >
                          {item.fileName}
                        </button>
                        <button
                          type="button"
                          onClick={() => setPendingAttachments((prev) => prev.filter((entry) => entry.artifactId !== item.artifactId))}
                          className="text-forge-mist/70 hover:text-forge-ash"
                          aria-label={`Remove ${item.fileName}`}
                        >
                          ×
                        </button>
                      </span>
                    ))}
                  </div>
                ) : null}
                <div className="mt-2 text-[11px] text-forge-mist/75">Enter to send · Shift+Enter for newline</div>
              </div>
            </footer>
          </>
        )}
      </section>

      <aside className="hidden min-h-0 min-w-0 flex-col overflow-hidden border-l border-white/10 bg-[#0a0d12] xl:flex">
        <div className="flex items-center justify-between border-b border-white/10 px-4 py-3">
          <div className="text-sm font-semibold text-forge-ash">Inspector</div>
          <div className="flex items-center gap-2 text-xs">
            <button
              type="button"
              onClick={() => setInspectorMode("code")}
              className={[
                "rounded border px-2 py-1",
                inspectorMode === "code" ? "border-white/20 bg-white/10 text-forge-ash" : "border-white/10 text-forge-mist hover:border-white/20",
              ].join(" ")}
            >
              Code
            </button>
            <button
              type="button"
              onClick={() => setInspectorMode("files")}
              className={[
                "rounded border px-2 py-1",
                inspectorMode === "files" ? "border-white/20 bg-white/10 text-forge-ash" : "border-white/10 text-forge-mist hover:border-white/20",
              ].join(" ")}
            >
              Files
            </button>
          </div>
        </div>

        {inspectorMode === "code" ? (
          <div className="grid min-h-0 flex-1 grid-rows-[220px_minmax(0,1fr)]">
            <div className="forge-chat-scroll overflow-y-auto border-b border-white/10 p-2">
              {assistantCodeSnippets.length === 0 ? (
                <div className="rounded border border-dashed border-white/10 px-3 py-4 text-xs text-forge-mist">No assistant code blocks in this thread yet.</div>
              ) : (
                assistantCodeSnippets.map((snippet) => (
                  <button
                    key={snippet.key}
                    type="button"
                    onClick={() => setSelectedSnippetKey(snippet.key)}
                    className={[
                      "mb-2 w-full rounded-lg border px-3 py-2 text-left",
                      selectedSnippetKey === snippet.key ? "border-white/20 bg-white/10" : "border-white/10 bg-black/20 hover:border-white/20",
                    ].join(" ")}
                  >
                    <div className="text-[11px] font-semibold uppercase tracking-[0.12em] text-forge-ash">{snippet.lang}</div>
                    <div className="mt-1 line-clamp-2 font-mono text-[11px] text-forge-mist">{snippet.code.trim() || "(empty code block)"}</div>
                    <div className="mt-1 text-[10px] text-forge-mist/70">{formatTime(snippet.createdAtMs)}</div>
                  </button>
                ))
              )}
            </div>
            <div className="forge-chat-scroll overflow-y-auto p-3">
              {assistantCodeSnippets.find((item) => item.key === selectedSnippetKey) ? (
                <CodeBlock
                  code={assistantCodeSnippets.find((item) => item.key === selectedSnippetKey)?.code ?? ""}
                  lang={assistantCodeSnippets.find((item) => item.key === selectedSnippetKey)?.lang ?? "code"}
                />
              ) : (
                <div className="rounded border border-dashed border-white/10 px-3 py-4 text-xs text-forge-mist">Select a code block.</div>
              )}
            </div>
          </div>
        ) : (
          <div className="grid min-h-0 flex-1 grid-rows-[220px_minmax(0,1fr)]">
            <div className="forge-chat-scroll overflow-y-auto border-b border-white/10 p-2">
              {messageAttachments.length === 0 ? (
                <div className="rounded border border-dashed border-white/10 px-3 py-4 text-xs text-forge-mist">No files attached in this thread.</div>
              ) : (
                messageAttachments.map((item) => (
                  <button
                    key={item.attachment.artifactId}
                    type="button"
                    onClick={() => setSelectedAttachmentId(item.attachment.artifactId)}
                    className={[
                      "mb-2 w-full rounded-lg border px-3 py-2 text-left",
                      selectedAttachmentId === item.attachment.artifactId ? "border-white/20 bg-white/10" : "border-white/10 bg-black/20 hover:border-white/20",
                    ].join(" ")}
                  >
                    <div className="truncate text-xs font-semibold text-forge-ash">{item.attachment.title}</div>
                    <div className="mt-1 truncate text-[11px] text-forge-mist">{item.attachment.fileName}</div>
                    <div className="mt-1 text-[10px] text-forge-mist/70">{item.attachment.mimeType}</div>
                  </button>
                ))
              )}
            </div>
            <div className="forge-chat-scroll overflow-y-auto p-3">
              {messageAttachments.find((item) => item.attachment.artifactId === selectedAttachmentId) ? (
                <AttachmentInspectorCard
                  attachment={messageAttachments.find((item) => item.attachment.artifactId === selectedAttachmentId)!.attachment}
                />
              ) : (
                <div className="rounded border border-dashed border-white/10 px-3 py-4 text-xs text-forge-mist">Select a file.</div>
              )}
            </div>
          </div>
        )}
      </aside>
    </div>
  );
}

function MessageRow(props: { message: ChatMessage; onInspectAttachment: (artifactId: number) => void }) {
  const { message } = props;
  const role = message.role.toLowerCase();
  const isUser = role === "user";
  const isAssistant = role === "assistant";
  const jobId = readJobId(message.metadata);
  const correlationId = readCorrelationId(message.metadata);
  const traceId = readTraceId(message.metadata);
  const attachments = readAttachments(message.metadata);
  const toolActivity = isAssistant ? readToolGatewayActivity(message.metadata) : null;

  if (role === "system") {
    return (
      <div className="rounded-xl border border-white/10 bg-white/5 px-4 py-3 text-xs text-forge-mist">
        <div className="font-semibold uppercase tracking-wide text-forge-mist/80">System</div>
        <div className="mt-2 whitespace-pre-wrap break-words text-forge-ash">{message.content}</div>
        <div className="mt-2 text-[10px] text-forge-mist/60">{formatTime(message.createdAtMs)}</div>
      </div>
    );
  }

  return (
    <div className={`flex ${isUser ? "justify-end" : "justify-start"}`}>
      <div className={["w-full max-w-[46rem]", isUser ? "max-w-[42rem]" : ""].join(" ")}>
        <div
          className={[
            "rounded-2xl border px-4 py-3",
            isUser
              ? "border-white/15 bg-white/10 text-forge-ash"
              : "border-white/10 bg-[#0c1016] text-forge-ash",
          ].join(" ")}
        >
          <div className="mb-2 flex items-center justify-between gap-3">
            <div className="text-[10px] uppercase tracking-wide text-forge-mist/80">{isUser ? "You" : isAssistant ? "FORGE" : role}</div>
            <div className="text-[10px] text-forge-mist/65">{formatTime(message.createdAtMs)}</div>
          </div>
          <RichMessage content={message.content} />
          {toolActivity ? <ToolGatewayActivityPanel activity={toolActivity} /> : null}
          {attachments.length > 0 ? (
            <div className="mt-3 border-t border-white/10 pt-2">
              <div className="mb-1 text-[10px] uppercase tracking-[0.12em] text-forge-mist/70">Attachments</div>
              <div className="flex flex-wrap gap-2">
                {attachments.map((attachment) => (
                  <button
                    key={attachment.artifactId}
                    type="button"
                    onClick={() => props.onInspectAttachment(attachment.artifactId)}
                    className="rounded-full border border-white/15 bg-white/5 px-2.5 py-1 text-[11px] text-forge-mist transition hover:text-forge-ash"
                  >
                    {attachment.fileName}
                  </button>
                ))}
              </div>
            </div>
          ) : null}
          {jobId ? (
            <div className="mt-3 border-t border-white/10 pt-2 text-xs text-forge-mist">
              <Link to={`/jobs/${encodeURIComponent(jobId)}`} className="underline underline-offset-2 hover:text-forge-ash">
                Open Job {jobId}
              </Link>
            </div>
          ) : null}
          {correlationId || traceId ? (
            <div className="mt-3 border-t border-white/10 pt-2 text-xs text-forge-mist">
              <div className="flex flex-wrap gap-3">
                {correlationId ? (
                  <Link
                    to={`/inspectors?correlationId=${encodeURIComponent(correlationId)}`}
                    className="underline underline-offset-2 hover:text-forge-ash"
                  >
                    Inspect correlation {correlationId}
                  </Link>
                ) : null}
                {traceId ? (
                  <Link
                    to={`/inspectors?traceId=${encodeURIComponent(traceId)}`}
                    className="underline underline-offset-2 hover:text-forge-ash"
                  >
                    Inspect trace {traceId}
                  </Link>
                ) : null}
              </div>
            </div>
          ) : null}
        </div>
      </div>
    </div>
  );
}

function AttachmentInspectorCard(props: { attachment: ChatAttachment }) {
  return (
    <div className="space-y-3 rounded-xl border border-white/12 bg-[#0c1016] p-3">
      <div>
        <div className="text-[10px] uppercase tracking-[0.12em] text-forge-mist/70">Attachment</div>
        <div className="mt-1 text-sm font-semibold text-forge-ash">{props.attachment.title}</div>
      </div>
      <div className="text-[11px] text-forge-mist">
        <div className="truncate">{props.attachment.fileName}</div>
        <div className="mt-1">{props.attachment.mimeType}</div>
      </div>
      <div className="flex flex-wrap gap-2">
        <Link
          to={`/workbench?artifactId=${encodeURIComponent(String(props.attachment.artifactId))}`}
          className="rounded border border-white/15 bg-white/5 px-2.5 py-1 text-[11px] text-forge-mist hover:text-forge-ash"
        >
          Open in Workbench
        </Link>
      </div>
      {props.attachment.textPreview ? (
        <pre className="forge-chat-scroll max-h-[420px] overflow-auto whitespace-pre-wrap rounded border border-white/10 bg-black/25 p-2 font-mono text-[11px] text-forge-mist">
          {props.attachment.textPreview}
        </pre>
      ) : (
        <div className="text-[11px] text-forge-mist/75">No inline preview for this file type.</div>
      )}
    </div>
  );
}
