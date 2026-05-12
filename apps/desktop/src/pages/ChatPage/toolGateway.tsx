import { useCallback, useEffect, useMemo, useState } from "react";
import { Link } from "react-router-dom";

import {
  api,
  type ChatMessage,
  type ChatToolGatewayActivity,
} from "../../lib/api";

function asRecord(v: unknown): Record<string, unknown> | null {
  if (v && typeof v === "object" && !Array.isArray(v))
    return v as Record<string, unknown>;
  return null;
}

export type ChatToolEntry = {
  key: string;
  messageId: number;
  createdAtMs: number;
  tool: string;
  state: string;
  args: Record<string, unknown>;
  result: Record<string, unknown>;
};

export function readToolGatewayActivity(
  meta: Record<string, unknown> | undefined,
): ChatToolGatewayActivity | null {
  if (!meta) return null;
  const raw = meta.toolGatewayActivity;
  if (!raw || typeof raw !== "object" || Array.isArray(raw)) return null;
  const activity = raw as ChatToolGatewayActivity;
  const callsExecuted =
    typeof activity.toolCallsExecuted === "number" &&
    Number.isFinite(activity.toolCallsExecuted)
      ? activity.toolCallsExecuted
      : 0;
  const toolCallEmitted = activity.toolCallEmitted === true;
  const state =
    typeof activity.executionState === "string"
      ? activity.executionState.trim().toLowerCase()
      : "";
  const approvalRequired = state === "needs_approval";

  const shouldSurface =
    callsExecuted > 0 ||
    toolCallEmitted ||
    (approvalRequired &&
      (activity.toolSelected != null || activity.executionResult != null));

  if (!shouldSurface) return null;
  return activity;
}

function normalizeToolName(raw: unknown): string {
  return typeof raw === "string" ? raw.trim() : "";
}

export function readToolEntries(message: ChatMessage): ChatToolEntry[] {
  const activity = readToolGatewayActivity(message.metadata);
  if (!activity) return [];
  const tool = normalizeToolName(activity.toolSelected);
  if (!tool) return [];
  const args = asRecord(activity.toolArgs) ?? {};
  const result = asRecord(activity.executionResult) ?? {};
  return [
    {
      key: `${message.id}-${tool}`,
      messageId: message.id,
      createdAtMs: message.createdAtMs,
      tool,
      state: normalizeToolName(activity.executionState) || "recorded",
      args,
      result,
    },
  ];
}

export function isTerminalTool(tool: string) {
  return tool === "proc.run";
}

export function isBrowserTool(tool: string) {
  return (
    tool === "web.search" || tool === "net.fetch" || tool === "desktop.open"
  );
}

function stringField(record: Record<string, unknown>, key: string): string {
  const raw = record[key];
  return typeof raw === "string" ? raw : "";
}

export function numberField(
  record: Record<string, unknown>,
  key: string,
): number | null {
  const raw = record[key];
  if (typeof raw === "number" && Number.isFinite(raw)) return raw;
  if (typeof raw === "string" && raw.trim() && Number.isFinite(Number(raw)))
    return Number(raw);
  return null;
}

function readSearchResults(
  record: Record<string, unknown>,
): Array<Record<string, unknown>> {
  const raw = record.results;
  if (!Array.isArray(raw)) return [];
  return raw
    .map((item) => asRecord(item))
    .filter((item): item is Record<string, unknown> => item != null);
}

function clipText(value: string, max = 6000): string {
  if (value.length <= max) return value;
  return `${value.slice(0, max)}\n... (truncated)`;
}

function readApprovalRequestIdFromGatewayResult(
  executionResult: unknown,
): number | null {
  function walk(v: unknown): number | null {
    const rec = asRecord(v);
    if (rec) {
      const raw = rec.approvalRequestId;
      if (typeof raw === "number" && Number.isFinite(raw)) return raw;
      if (typeof raw === "string" && /^\d+$/.test(raw.trim()))
        return Number(raw.trim());
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
      const decision =
        typeof value.decision === "string" && value.decision.trim()
          ? value.decision.trim()
          : "resolved";
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
    window.localStorage.setItem(
      APPROVAL_STATE_CACHE_KEY,
      JSON.stringify(serializable),
    );
  } catch {
    return;
  }
}

function readApprovalResolutionFromCache(
  approvalId: number | null,
): ApprovalUiResolution | null {
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

function markApprovalResolvedInCache(
  approvalId: number | null,
  decision: string,
) {
  if (!approvalId) return;
  const next = readApprovalCache();
  next[approvalId] = {
    decision: decision || "resolved",
    updatedAtMs: Date.now(),
  };
  writeApprovalCache(next);
}

export function ToolGatewayActivityPanel(props: {
  activity: ChatToolGatewayActivity;
}) {
  const a = props.activity;
  const [approvalBusy, setApprovalBusy] = useState(false);
  const [approvalStatus, setApprovalStatus] = useState<string | null>(null);
  const [approvalState, setApprovalState] = useState<string | null>(null);
  const [gatewayJobStatus, setGatewayJobStatus] = useState<string | null>(null);
  const [gatewayJobSummary, setGatewayJobSummary] = useState<string | null>(
    null,
  );
  const [gatewayJobFailure, setGatewayJobFailure] = useState<string | null>(
    null,
  );
  const approvalId = useMemo(
    () => readApprovalRequestIdFromGatewayResult(a.executionResult),
    [a.executionResult],
  );
  const gatewayJobId = useMemo(
    () => readGatewayJobId(a.executionResult),
    [a.executionResult],
  );
  const approvalResolved = approvalState != null && approvalState !== "pending";
  const showApprovalActions = approvalId != null && !approvalResolved;
  const gatewayJobTerminal =
    gatewayJobStatus === "succeeded" ||
    gatewayJobStatus === "failed" ||
    gatewayJobStatus === "cancelled";
  const stages = Array.isArray(a.stages)
    ? (a.stages as Record<string, unknown>[])
    : [];
  const args =
    a.toolArgs && typeof a.toolArgs === "object" && !Array.isArray(a.toolArgs)
      ? (a.toolArgs as Record<string, unknown>)
      : null;
  const inlineToolEntry = useMemo<ChatToolEntry | null>(() => {
    const tool = normalizeToolName(a.toolSelected);
    const result = asRecord(a.executionResult);
    if (!tool || !result) return null;
    return {
      key: `inline-${tool}`,
      messageId: 0,
      createdAtMs: Date.now(),
      tool,
      state: normalizeToolName(a.executionState) || "recorded",
      args: args ?? {},
      result,
    };
  }, [a.executionResult, a.executionState, a.toolSelected, args]);

  const isAlreadyResolvedError = useCallback((message: string) => {
    const value = message.toLowerCase();
    return (
      value.includes("not pending") ||
      value.includes("already") ||
      value.includes("resolved")
    );
  }, []);

  const refreshApproval = useCallback(async () => {
    if (approvalId == null) {
      setApprovalState(null);
      return;
    }
    try {
      const res = await api.approvals.get(approvalId);
      const status = String(res.approval?.status ?? "")
        .trim()
        .toLowerCase();
      if (status === "pending") {
        setApprovalState("pending");
        clearApprovalResolvedFromCache(approvalId);
        setApprovalStatus(null);
        return;
      }
      const decision = String(res.approval?.decision?.decision ?? "")
        .trim()
        .toLowerCase();
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
    if (
      gatewayJobId == null ||
      !approvalResolved ||
      approvalState === "denied" ||
      gatewayJobTerminal
    ) {
      return;
    }
    const resolvedGatewayJobId = gatewayJobId;
    let cancelled = false;
    async function refreshGatewayJob() {
      try {
        const detail = await api.jobs.detail(resolvedGatewayJobId, 0);
        if (cancelled) return;
        const status = String(detail.job?.status ?? "")
          .trim()
          .toLowerCase();
        const summary = String(detail.job?.resultSummary ?? "").trim();
        const failure = String(
          detail.job?.failureInfo ?? detail.job?.lastError ?? "",
        ).trim();
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
      <div className="font-semibold uppercase tracking-[0.14em] text-forge-emberSoft/90">
        Tool gateway
      </div>
      {a.userRequestSummary ? (
        <div className="mt-2">
          <span className="text-forge-mist/70">Request · </span>
          <span className="whitespace-pre-wrap text-forge-ash">
            {a.userRequestSummary}
          </span>
        </div>
      ) : null}
      <div className="mt-2 grid gap-1 font-mono text-[10px] text-forge-mist/90">
        <div>
          <span className="text-forge-mist/60">Tool · </span>
          {a.toolSelected && String(a.toolSelected).trim()
            ? String(a.toolSelected)
            : a.toolCallEmitted
              ? "(model emitted tool call)"
              : "—"}
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
                    await api.approvals.approve(
                      approvalId!,
                      "Approved from chat tool gateway",
                    );
                    setApprovalState("approved");
                    markApprovalResolvedInCache(approvalId, "approved");
                    setApprovalStatus(
                      "Approved. Execution continues as a gateway job.",
                    );
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
                    await api.approvals.deny(
                      approvalId!,
                      "Denied from chat tool gateway",
                    );
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
            Approval resolved
            {approvalState && approvalState !== "resolved"
              ? ` (${approvalState}).`
              : "."}
          </div>
        ) : null}
        {gatewayJobId && approvalResolved && approvalState !== "denied" ? (
          <div className="mt-1 text-[10px] text-forge-mist/90">
            Gateway job{" "}
            {gatewayJobStatus ? (
              <span className="text-forge-ash">{gatewayJobStatus}</span>
            ) : (
              "status pending"
            )}
            .{gatewayJobSummary ? <span> {gatewayJobSummary}</span> : null}
            {gatewayJobFailure ? (
              <span className="text-forge-emberSoft"> {gatewayJobFailure}</span>
            ) : null}
          </div>
        ) : null}
        {approvalStatus ? (
          <div className="mt-1 text-[10px] text-forge-mist/90">
            {approvalStatus}
          </div>
        ) : null}
        {a.executionState === "needs_approval" && approvalId == null ? (
          <div className="mt-1 text-[10px] text-forge-emberSoft/90">
            Approval is required, but no approval request id was recorded. Check
            that the approvals service and database are available.
          </div>
        ) : null}
        {inlineToolEntry && isTerminalTool(inlineToolEntry.tool) ? (
          <TerminalTranscript entry={inlineToolEntry} compact />
        ) : null}
        {inlineToolEntry && isBrowserTool(inlineToolEntry.tool) ? (
          <BrowserResultPanel entry={inlineToolEntry} compact />
        ) : null}
        {a.executionResult !== undefined &&
        (!inlineToolEntry ||
          (!isTerminalTool(inlineToolEntry.tool) &&
            !isBrowserTool(inlineToolEntry.tool))) ? (
          <InlineStructuredSummary value={a.executionResult} />
        ) : null}
      </div>
      {stages.length > 0 ? (
        <details className="mt-2">
          <summary className="cursor-pointer text-[10px] text-forge-mist/80">
            Pipeline ({stages.length} stages)
          </summary>
          <ol className="mt-1 list-decimal space-y-1 pl-4 text-[10px] text-forge-mist/85">
            {stages.map((row, i) => (
              <li key={`st-${i}`}>
                <span className="text-forge-ash">
                  {String(row.stage ?? "?")}
                </span>
                {row.atMs != null ? (
                  <span className="text-forge-mist/50">
                    {" "}
                    · {String(row.atMs)}
                  </span>
                ) : null}
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
  if (typeof raw === "object")
    return `${Object.keys(raw as Record<string, unknown>).length} field(s)`;
  return "value";
}

function InlineStructuredSummary(props: { value: unknown }) {
  const rows = summarizeResultRows(props.value);
  if (rows.length === 0) {
    return (
      <div className="mt-1 rounded border border-forge-platinum/10 bg-black/30 px-2 py-1 text-[10px] text-forge-mist">
        Execution result recorded.
      </div>
    );
  }
  return (
    <div className="mt-1 rounded border border-forge-platinum/10 bg-black/30 p-2 text-[10px] text-forge-mist">
      <div className="font-semibold uppercase tracking-[0.14em] text-forge-mist/70">
        Execution Result
      </div>
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

export function TerminalTranscript(props: {
  entry: ChatToolEntry;
  compact?: boolean;
}) {
  const command =
    stringField(props.entry.result, "command") ||
    stringField(props.entry.args, "command");
  const cwd = stringField(props.entry.result, "cwd");
  const stdout = clipText(stringField(props.entry.result, "stdout"));
  const stderr = clipText(stringField(props.entry.result, "stderr"), 3000);
  const exitCode = numberField(props.entry.result, "exitCode");
  const timedOut = props.entry.result.timedOut === true;
  return (
    <div className="overflow-hidden rounded-xl border border-forge-platinum/15 bg-black/60 font-mono text-[11px] text-forge-ash">
      <div className="flex flex-wrap items-center justify-between gap-2 border-b border-forge-platinum/10 bg-forge-onyx/80 px-3 py-2">
        <span className="text-forge-mist">
          $ {command || "command recorded"}
        </span>
        <span
          className={
            exitCode === 0 ? "text-forge-platinum" : "text-forge-emberSoft"
          }
        >
          exit {exitCode ?? "?"}
          {timedOut ? " timed out" : ""}
        </span>
      </div>
      {cwd ? (
        <div className="border-b border-forge-platinum/10 px-3 py-1.5 text-[10px] text-forge-mist/70">
          cwd {cwd}
        </div>
      ) : null}
      <div
        className={[
          "forge-chat-scroll overflow-auto p-3",
          props.compact ? "max-h-72" : "max-h-full",
        ].join(" ")}
      >
        {stdout ? (
          <pre className="whitespace-pre-wrap break-words text-forge-ash">
            {stdout}
          </pre>
        ) : (
          <div className="text-forge-mist/65">No stdout.</div>
        )}
        {stderr ? (
          <pre className="mt-3 whitespace-pre-wrap break-words border-t border-forge-ember/20 pt-3 text-forge-emberSoft">
            {stderr}
          </pre>
        ) : null}
      </div>
    </div>
  );
}

export function BrowserResultPanel(props: {
  entry: ChatToolEntry;
  compact?: boolean;
}) {
  const tool = props.entry.tool;
  const query =
    stringField(props.entry.result, "query") ||
    stringField(props.entry.args, "query");
  const url =
    stringField(props.entry.result, "url") ||
    stringField(props.entry.args, "url") ||
    stringField(props.entry.result, "searchUrl");
  const statusCode = numberField(props.entry.result, "statusCode");
  const body = clipText(
    stringField(props.entry.result, "body"),
    props.compact ? 3000 : 9000,
  );
  const results = readSearchResults(props.entry.result);
  return (
    <div className="overflow-hidden rounded-xl border border-forge-platinum/15 bg-black/50 text-xs text-forge-mist">
      <div className="flex flex-wrap items-center justify-between gap-2 border-b border-forge-platinum/10 bg-forge-onyx/80 px-3 py-2">
        <span className="font-semibold text-forge-ash">
          {tool === "web.search"
            ? "Web search"
            : tool === "net.fetch"
              ? "Fetched page"
              : "Browser open"}
        </span>
        <span className="font-mono text-[10px] text-forge-mist/70">
          {props.entry.state}
          {statusCode != null ? ` · ${statusCode}` : ""}
        </span>
      </div>
      <div
        className={[
          "forge-chat-scroll overflow-auto p-3",
          props.compact ? "max-h-72" : "max-h-full",
        ].join(" ")}
      >
        {query ? (
          <div className="mb-3 font-mono text-[11px] text-forge-ash">
            query: {query}
          </div>
        ) : null}
        {url ? (
          <a
            href={url}
            target="_blank"
            rel="noreferrer"
            className="mb-3 block break-all font-mono text-[11px] text-forge-platinum underline underline-offset-2"
          >
            {url}
          </a>
        ) : null}
        {results.length > 0 ? (
          <div className="space-y-3">
            {results.map((item, index) => {
              const title = stringField(item, "title") || `Result ${index + 1}`;
              const resultUrl = stringField(item, "url");
              const snippet = stringField(item, "snippet");
              return (
                <div
                  key={`${props.entry.key}-result-${index}`}
                  className="rounded-lg border border-forge-platinum/10 bg-forge-carbon/80 p-3"
                >
                  <div className="font-semibold text-forge-ash">{title}</div>
                  {resultUrl ? (
                    <a
                      href={resultUrl}
                      target="_blank"
                      rel="noreferrer"
                      className="mt-1 block break-all font-mono text-[10px] text-forge-platinum underline underline-offset-2"
                    >
                      {resultUrl}
                    </a>
                  ) : null}
                  {snippet ? (
                    <div className="mt-2 text-forge-mist">{snippet}</div>
                  ) : null}
                </div>
              );
            })}
          </div>
        ) : body ? (
          <pre className="whitespace-pre-wrap break-words font-mono text-[11px] text-forge-ash">
            {body}
          </pre>
        ) : (
          <div className="text-forge-mist/70">
            No browser/search payload recorded.
          </div>
        )}
      </div>
    </div>
  );
}

function summarizeResultRows(value: unknown): Array<[string, string]> {
  if (value == null) return [];
  if (
    typeof value === "string" ||
    typeof value === "number" ||
    typeof value === "boolean"
  ) {
    return [["Result", String(value)]];
  }
  if (Array.isArray(value)) {
    return [["Result items", String(value.length)]];
  }
  if (typeof value !== "object") return [];
  const row = value as Record<string, unknown>;
  const keys = [
    "status",
    "policyOutcome",
    "approvalRequestId",
    "jobId",
    "gatewayStatus",
    "reason",
    "error",
    "message",
  ];
  const out: Array<[string, string]> = [];
  for (const key of keys) {
    const raw = row[key];
    if (raw == null) continue;
    if (
      typeof raw === "string" ||
      typeof raw === "number" ||
      typeof raw === "boolean"
    ) {
      out.push([key, String(raw)]);
    } else if (Array.isArray(raw)) {
      out.push([key, `${raw.length} item(s)`]);
    } else {
      out.push([
        key,
        `${Object.keys(raw as Record<string, unknown>).length} field(s)`,
      ]);
    }
  }
  if (out.length > 0) return out;
  return [["Fields", String(Object.keys(row).length)]];
}
