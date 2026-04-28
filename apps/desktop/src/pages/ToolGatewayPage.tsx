import { GhostButton, Panel, PrimaryButton } from "@forge/ui";
import { useEffect, useMemo, useState } from "react";
import { useNavigate } from "react-router-dom";
import type { ToolCapabilityStatus } from "@forge/shared";

import { FoldSection } from "../components/FoldSection";
import { api } from "../lib/api";
import { useUiStore } from "../stores/uiStore";

type ToolRow = {
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
};

type CapabilityRow = {
  id: string;
  domain: string;
  name: string;
  description: string;
  status: ToolCapabilityStatus;
  lane: string;
  effect: string[];
  risk: string;
  requiresWorkspace: boolean;
  requiresIntent: boolean;
  requiresApprovalByDefault: boolean;
  autonomyEligible: boolean;
  allowedInDryRun: boolean;
  adapterId?: string;
};

type LaneRow = {
  id: string;
  name: string;
  enabled: boolean;
  actionType: string;
  riskClass: string;
  writeIntent: boolean;
  requiresApproval: boolean;
};

type InvocationRow = {
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
  status: string;
  deniedReason: string;
  permissionProfileId: string;
  approvalRequestId?: number | null;
  result: unknown;
};

function formatTime(ms?: number | null) {
  if (!ms) return "—";
  try {
    return new Date(ms).toLocaleString();
  } catch {
    return String(ms);
  }
}

function parseInput(raw: string): Record<string, unknown> {
  const trimmed = raw.trim();
  if (!trimmed) return {};
  if (trimmed.startsWith("{")) {
    const parsed = JSON.parse(trimmed);
    if (!parsed || typeof parsed !== "object" || Array.isArray(parsed)) {
      throw new Error("Request details must be a named field list.");
    }
    return parsed as Record<string, unknown>;
  }
  const next: Record<string, unknown> = {};
  for (const line of trimmed.split(/\r?\n/)) {
    const clean = line.trim();
    if (!clean) continue;
    const splitAt = clean.indexOf(":");
    if (splitAt < 1) throw new Error(`Request detail is missing a colon: ${clean}`);
    next[clean.slice(0, splitAt).trim()] = parseReadableValue(clean.slice(splitAt + 1).trim());
  }
  return next;
}

export function ToolGatewayPage() {
  const navigate = useNavigate();
  const setStatus = useUiStore((s) => s.setStatusLine);
  const [tools, setTools] = useState<ToolRow[]>([]);
  const [capabilities, setCapabilities] = useState<CapabilityRow[]>([]);
  const [lanes, setLanes] = useState<LaneRow[]>([]);
  const [invs, setInvs] = useState<InvocationRow[]>([]);
  const [workspace, setWorkspace] = useState("");
  const [statusFilter, setStatusFilter] = useState("");
  const [toolId, setToolId] = useState("");
  const [laneId, setLaneId] = useState("");
  const [paths, setPaths] = useState("");
  const [action, setAction] = useState("invoke");
  const [riskClass, setRiskClass] = useState("");
  const [executionLevel, setExecutionLevel] = useState("");
  const [inputRaw, setInputRaw] = useState("");
  const [dryRun, setDryRun] = useState(true);
  const [last, setLast] = useState<Record<string, unknown> | null>(null);
  const [err, setErr] = useState<string | null>(null);
  const [statusDraftById, setStatusDraftById] = useState<Record<string, ToolCapabilityStatus>>({});
  const [statusReasonById, setStatusReasonById] = useState<Record<string, string>>({});
  const [statusUpdateBusy, setStatusUpdateBusy] = useState<string | null>(null);

  function newCorrelationId() {
    try {
      return crypto.randomUUID();
    } catch {
      return `gw-${Date.now()}`;
    }
  }

  const selectedTool = useMemo(() => tools.find((x) => x.id === toolId) ?? null, [toolId, tools]);
  const selectedLane = useMemo(() => lanes.find((row) => row.id === laneId) ?? null, [laneId, lanes]);
  const selectedCapability = useMemo(() => {
    if (!selectedTool?.capabilityId) return null;
    return capabilities.find((row) => row.id === selectedTool.capabilityId) ?? null;
  }, [capabilities, selectedTool?.capabilityId]);
  const parsedInput = useMemo(() => {
    try {
      return { ok: true as const, value: parseInput(inputRaw), error: "" };
    } catch (e) {
      return { ok: false as const, value: {}, error: e instanceof Error ? e.message : String(e) };
    }
  }, [inputRaw]);
  const canExecuteLocally = useMemo(() => {
    const capabilityStatus = selectedCapability?.status ?? selectedTool?.capabilityStatus ?? "unknown";
    const capabilityAllowed = capabilityStatus === "active" || capabilityStatus === "approval_only";
    return Boolean(selectedTool && selectedLane && selectedLane.enabled && capabilityAllowed && parsedInput.ok);
  }, [parsedInput.ok, selectedCapability?.status, selectedLane, selectedTool]);
  const preflightGates = useMemo(
    () => [
      { label: "if tool is selected", pass: Boolean(selectedTool) },
      { label: "and lane is selected", pass: Boolean(selectedLane) },
      { label: "and lane is enabled", pass: Boolean(selectedLane?.enabled) },
      {
        label: "and capability status is active/approval_only",
        pass: (() => {
          const status = selectedCapability?.status ?? selectedTool?.capabilityStatus ?? "";
          return status === "active" || status === "approval_only";
        })(),
      },
      { label: "and request details are valid", pass: parsedInput.ok },
    ],
    [parsedInput.ok, selectedCapability?.status, selectedLane, selectedTool],
  );

  async function refresh() {
    try {
      const [t, c, l, m, i] = await Promise.all([
        api.gateway.tools(),
        api.gateway.capabilities(),
        api.actionLanes.list(),
        api.meta(),
        api.gateway.invocations({ limit: 80, status: statusFilter || undefined }),
      ]);
      const toolRows = (Array.isArray(t.tools) ? t.tools : []) as ToolRow[];
      const capabilityRows = (Array.isArray(c.capabilities) ? c.capabilities : []) as CapabilityRow[];
      const laneRows = (Array.isArray(l.lanes) ? l.lanes : []) as LaneRow[];
      setTools(toolRows);
      setCapabilities(capabilityRows);
      setStatusDraftById((prev) => {
        const next = { ...prev };
        for (const row of capabilityRows) {
          if (!next[row.id]) next[row.id] = row.status;
        }
        return next;
      });
      setLanes(laneRows);
      setInvs(Array.isArray(i.invocations) ? i.invocations : []);
      setWorkspace(m.workspaceDir);

      const nextToolId = toolRows.some((row) => row.id === toolId) ? toolId : toolRows[0]?.id ?? "";
      setToolId(nextToolId);
      const preferredLane = laneRows.find((row) => row.id === nextToolId && row.enabled) ?? laneRows.find((row) => row.enabled) ?? laneRows[0] ?? null;
      if (!laneRows.some((row) => row.id === laneId)) {
        setLaneId(preferredLane?.id ?? "");
      }
      const tool = toolRows.find((row) => row.id === nextToolId);
      if (tool) {
        setRiskClass((v) => v || tool.riskClass);
        setExecutionLevel((v) => v || tool.executionLevel);
      }
      setErr(null);
    } catch (e) {
      setErr(e instanceof Error ? e.message : String(e));
    }
  }

  useEffect(() => {
    void refresh();
  }, []);

  useEffect(() => {
    if (!selectedTool) return;
    setRiskClass(selectedTool.riskClass);
    setExecutionLevel(selectedTool.executionLevel);
  }, [selectedTool?.id]);

  return (
    <div className="space-y-6">
      <Panel
        title="Tool Gateway"
        subtitle="Single governed action gateway for local machine operations. Every invocation is policy-checked and audited."
        actions={<GhostButton onClick={() => void refresh()}>Refresh</GhostButton>}
      >
        {err ? <div className="rounded-md border border-forge-ember/30 bg-forge-ember/10 p-3 text-sm text-forge-ash">{err}</div> : null}
        <div className="mt-3 grid gap-2 text-xs text-forge-mist md:grid-cols-2">
          <div>
            Workspace root: <span className="font-mono text-forge-ash">{workspace || "—"}</span>
          </div>
          <label className="justify-self-start md:justify-self-end">
            Filter status
            <select className="forge-input ml-2" value={statusFilter} onChange={(e) => setStatusFilter(e.target.value)}>
              <option value="">all</option>
              <option value="ok">ok</option>
              <option value="dry_run">dry_run</option>
              <option value="needs_approval">needs_approval</option>
              <option value="denied">denied</option>
              <option value="unsupported">unsupported</option>
              <option value="disabled">disabled</option>
              <option value="error">error</option>
            </select>
            <GhostButton className="ml-2" onClick={() => void refresh()}>Apply</GhostButton>
          </label>
        </div>
      </Panel>

      <Panel title="Invoke Action" subtitle="Typed request contract: tool + lane + risk + execution level + scoped targets + request details.">
        <div className="space-y-3">
          <FoldSection title="1) Target and lane" subtitle="Choose exactly what you want to invoke." defaultOpen>
            <div className="grid gap-3 md:grid-cols-2">
              <label className="text-xs text-forge-mist">
                Tool
                <select className="forge-input mt-1" value={toolId} onChange={(e) => setToolId(e.target.value)}>
                  {tools.map((t) => (
                    <option key={t.id} value={t.id}>
                      {t.id} · {t.domain} · {t.executionLevel}
                    </option>
                  ))}
                </select>
              </label>
              <label className="text-xs text-forge-mist">
                Lane
                <select className="forge-input mt-1" value={laneId} onChange={(e) => setLaneId(e.target.value)}>
                  {lanes.map((ln) => (
                    <option key={ln.id} value={ln.id}>
                      {ln.id} · {ln.riskClass}
                      {!ln.enabled ? " (disabled)" : ""}
                    </option>
                  ))}
                </select>
              </label>
            </div>
          </FoldSection>

          <FoldSection title="2) Risk and execution" subtitle="Explicitly set intent level before running." defaultOpen>
            <div className="grid gap-3 md:grid-cols-3">
              <label className="text-xs text-forge-mist">
                Action label
                <input className="forge-input mt-1" value={action} onChange={(e) => setAction(e.target.value)} />
              </label>
              <label className="text-xs text-forge-mist">
                Risk class
                <select className="forge-input mt-1" value={riskClass} onChange={(e) => setRiskClass(e.target.value)}>
                  <option value="read_only">read_only</option>
                  <option value="safe_write">safe_write</option>
                  <option value="scoped_execute">scoped_execute</option>
                  <option value="privileged">privileged</option>
                  <option value="dangerous">dangerous</option>
                </select>
              </label>
              <label className="text-xs text-forge-mist">
                Execution level
                <select className="forge-input mt-1" value={executionLevel} onChange={(e) => setExecutionLevel(e.target.value)}>
                  <option value="L0">L0</option>
                  <option value="L1">L1</option>
                  <option value="L2">L2</option>
                  <option value="L3">L3</option>
                  <option value="L4">L4</option>
                </select>
              </label>
            </div>
          </FoldSection>

          <FoldSection title="3) Scope and details" subtitle="Constrain paths and provide named request details.">
            <label className="block text-xs text-forge-mist">
              Paths (comma-separated, relative to workspace or absolute if policy allows)
              <input className="forge-input mt-1 font-mono text-xs" value={paths} onChange={(e) => setPaths(e.target.value)} placeholder="README.md, docs" />
            </label>
            <label className="mt-3 block text-xs text-forge-mist">
              Request details
              <textarea
                className="forge-input mt-1 min-h-[120px] font-mono text-xs"
                value={inputRaw}
                onChange={(e) => setInputRaw(e.target.value)}
                placeholder={"application: minecraft\nwindowTitle: Minecraft"}
              />
            </label>
            {!parsedInput.ok ? <div className="mt-2 text-xs text-forge-emberSoft">Input error: {parsedInput.error}</div> : null}
            <label className="mt-3 flex items-center gap-2 text-xs text-forge-mist">
              <input type="checkbox" checked={dryRun} onChange={(e) => setDryRun(e.target.checked)} />
              Dry run
            </label>
          </FoldSection>

          <FoldSection
            title="4) Preflight gates"
            subtitle="UI gate preview: if/and checks before execute. Kernel policy remains authoritative."
            defaultOpen
          >
            <div className="space-y-1 rounded border border-forge-platinum/10 bg-black/25 p-3 text-xs">
              {preflightGates.map((gate, idx) => (
                <GateLine key={gate.label} prefix={idx === 0 ? "IF" : "AND"} label={gate.label} pass={gate.pass} />
              ))}
            </div>
            <div className="mt-2 text-[11px] text-forge-mist">
              Selected capability:{" "}
              <span className="text-forge-ash">{selectedCapability?.id ?? selectedTool?.capabilityId ?? "—"}</span> · status{" "}
              <span className="text-forge-ash">{selectedCapability?.status ?? selectedTool?.capabilityStatus ?? "unknown"}</span>
            </div>
          </FoldSection>
        </div>
        <div className="mt-4 flex gap-2">
          <PrimaryButton
            disabled={!canExecuteLocally}
            onClick={async () => {
              try {
                if (!canExecuteLocally) {
                  setErr("Preflight gates are not satisfied. Resolve IF/AND checks before execute.");
                  return;
                }
                const input = parsedInput.value;
                const pathList = paths
                  .split(",")
                  .map((s) => s.trim())
                  .filter(Boolean);
                const res = await api.gateway.invoke({
                  toolId,
                  laneId,
                  domain: selectedTool?.domain,
                  action,
                  riskClass,
                  executionLevel,
                  paths: pathList,
                  dryRun,
                  initiator: "operator",
                  correlationId: newCorrelationId(),
                  input,
                });
                const out = res.result as Record<string, unknown>;
                setLast(out);
                setStatus(`Gateway ${String(out.status ?? "ok")} · ${String(out.policyOutcome ?? "")}`.trim());
                await refresh();
              } catch (e) {
                setErr(e instanceof Error ? e.message : String(e));
              }
            }}
          >
            {canExecuteLocally ? "Execute" : "Blocked by preflight"}
          </PrimaryButton>
          {last && typeof last.approvalRequestId === "number" ? (
            <GhostButton onClick={() => navigate("/approvals")}>Open approvals</GhostButton>
          ) : null}
        </div>
        {last ? (
          <GatewayResultSummary result={last} />
        ) : null}
      </Panel>

      <Panel title="Registered Tools" subtitle="Bounded typed capabilities exposed by the gateway.">
        <div className="space-y-2">
          {tools.map((t) => (
            <div key={t.id} className="rounded border border-forge-platinum/10 bg-forge-slate/20 px-3 py-2 text-xs text-forge-mist">
              <div className="font-mono text-forge-ash">{t.id}</div>
              <div className="mt-1">{t.domain}.{t.action} · {t.riskClass} · {t.executionLevel} · {t.writeIntent ? "write" : "read"}</div>
              <div className="mt-1 text-[11px]">
                capability {t.capabilityId ?? "—"} · status {t.capabilityStatus ?? "—"} · risk {t.capabilityRisk ?? "—"} · adapter {t.adapterId ?? "—"}
              </div>
              <div className="mt-1 text-[11px] text-forge-mist/90">status reason: {capabilityStatusReason(t.capabilityStatus, t.adapterId)}</div>
              <div className="mt-1">{t.description}</div>
            </div>
          ))}
        </div>
      </Panel>

      <Panel title="Capability Registry" subtitle="Kernel capability taxonomy, status, risk, and autonomy eligibility.">
        <div className="space-y-2">
          {capabilities.length === 0 ? <div className="text-sm text-forge-mist">No capability details available.</div> : null}
          {capabilities.map((cap) => (
            <div key={cap.id} className="rounded border border-forge-platinum/10 bg-black/25 px-3 py-2 text-xs text-forge-mist">
              <div className="font-mono text-forge-ash">{cap.id}</div>
              <div className="mt-1">
                {cap.domain}.{cap.name} · {cap.status} · risk {cap.risk} · lane {cap.lane}
              </div>
              <div className="mt-1">
                effects {cap.effect.join(", ")} · intent {cap.requiresIntent ? "required" : "optional"} · dry-run{" "}
                {cap.allowedInDryRun ? "allowed" : "blocked"} · autonomy {cap.autonomyEligible ? "eligible" : "not eligible"}
              </div>
              <div className="mt-1 text-[11px] text-forge-mist/90">status reason: {capabilityStatusReason(cap.status, cap.adapterId)}</div>
              <div className="mt-1">{cap.description}</div>
              <div className="mt-2 grid gap-2 md:grid-cols-[180px,1fr,auto]">
                <select
                  className="forge-input"
                  value={statusDraftById[cap.id] ?? cap.status}
                  onChange={(e) =>
                    setStatusDraftById((prev) => ({
                      ...prev,
                      [cap.id]: e.target.value as ToolCapabilityStatus,
                    }))
                  }
                >
                  {CAPABILITY_STATUS_OPTIONS.map((status) => (
                    <option key={status} value={status}>
                      {status}
                    </option>
                  ))}
                </select>
                <input
                  className="forge-input"
                  value={statusReasonById[cap.id] ?? ""}
                  onChange={(e) =>
                    setStatusReasonById((prev) => ({
                      ...prev,
                      [cap.id]: e.target.value,
                    }))
                  }
                  placeholder="Transition reason (required for status changes)"
                />
                <GhostButton
                  onClick={async () => {
                    const nextStatus = statusDraftById[cap.id] ?? cap.status;
                    const reason = (statusReasonById[cap.id] ?? "").trim();
                    if (nextStatus !== cap.status && (requiresStatusReason(nextStatus) || reason === "")) {
                      setErr(`Status ${nextStatus} for ${cap.id} requires an explicit reason.`);
                      return;
                    }
                    setStatusUpdateBusy(cap.id);
                    try {
                      const res = await api.gateway.updateCapabilityStatus(cap.id, {
                        status: nextStatus,
                        reason: reason || undefined,
                        actor: "operator",
                        actorKind: "desktop",
                        source: "desktop",
                      });
                      if (res.approvalRequired && res.approvalRequestId) {
                        setStatus(`Capability ${cap.id} status change needs approval request #${res.approvalRequestId}`);
                      } else if (res.capability) {
                        setStatus(`Capability ${res.capability.id} status -> ${res.capability.status}`);
                      } else {
                        setStatus(`Capability ${cap.id} status update processed`);
                      }
                      await refresh();
                    } catch (error) {
                      setErr(error instanceof Error ? error.message : String(error));
                    } finally {
                      setStatusUpdateBusy(null);
                    }
                  }}
                  disabled={statusUpdateBusy === cap.id}
                >
                  {statusUpdateBusy === cap.id ? "Applying..." : "Apply status"}
                </GhostButton>
              </div>
            </div>
          ))}
        </div>
      </Panel>

      <Panel title="Invocation History" subtitle="Policy outcomes, approvals, and execution results from gateway history.">
        <div className="space-y-2">
          {invs.length === 0 ? <div className="text-sm text-forge-mist">No invocations in this filter.</div> : null}
          {invs.map((row) => (
            <button
              key={row.id}
              type="button"
              onClick={() => setLast(row as unknown as Record<string, unknown>)}
              className="w-full rounded border border-forge-platinum/10 bg-black/25 px-3 py-2 text-left text-xs text-forge-mist hover:border-forge-accent/50"
            >
              <div className="font-mono text-[11px] text-forge-ash">#{row.id} · {row.toolId} · {row.status} · {row.policyOutcome}</div>
              <div className="mt-1">{formatTime(row.createdAtMs)} · lane {row.laneId ?? "—"} · risk {row.riskClass} · {row.executionLevel}</div>
              <div className="mt-1">{row.deniedReason || row.correlationId}</div>
              {row.status === "unsupported" ? (
                <div className="mt-1 text-[11px] text-forge-mist/90">Unsupported reason: {row.deniedReason || "Capability is deferred or not implemented."}</div>
              ) : null}
              {row.jobId ? <div className="mt-1 text-[11px]">Job: {row.jobId}</div> : null}
              {typeof row.approvalRequestId === "number" ? <div className="mt-1 text-[11px]">Approval request: {row.approvalRequestId}</div> : null}
            </button>
          ))}
        </div>
      </Panel>
    </div>
  );
}

function GatewayResultSummary(props: { result: Record<string, unknown> }) {
  const rows: Array<[string, string]> = [
    ["Status", toText(props.result.status)],
    ["Policy outcome", toText(props.result.policyOutcome)],
    ["Tool", toText(props.result.toolId)],
    ["Lane", toText(props.result.laneId)],
    ["Action", toText(props.result.action)],
    ["Risk class", toText(props.result.riskClass)],
    ["Execution level", toText(props.result.executionLevel)],
    ["Approval request", toText(props.result.approvalRequestId)],
    ["Job", toText(props.result.jobId)],
    ["Correlation", toText(props.result.correlationId)],
    ["Audit", toText(props.result.auditId)],
  ];
  const warnings = normalizeStringList(props.result.warnings);
  const deniedReason = toText(props.result.deniedReason);
  const errorMessage = toText(props.result.error);
  const outputSummary = summarizeOutput(props.result.output);

  return (
    <div className="mt-4 rounded-md border border-forge-platinum/10 bg-black/25 p-3 text-[11px] text-forge-mist">
      <div className="grid gap-1 md:grid-cols-2">
        {rows
          .filter(([, value]) => value !== "—")
          .map(([label, value]) => (
            <div key={label} className="flex items-start justify-between gap-3 border-b border-forge-platinum/5 pb-1">
              <span className="text-forge-mist/75">{label}</span>
              <span className="text-right text-forge-ash">{value}</span>
            </div>
          ))}
      </div>
      {outputSummary ? (
        <div className="mt-2 rounded border border-forge-platinum/10 bg-black/30 px-2 py-1 text-[11px] text-forge-ash">
          Output: {outputSummary}
        </div>
      ) : null}
      {warnings.length > 0 ? (
        <div className="mt-2 rounded border border-forge-platinum/10 bg-black/30 px-2 py-1 text-[11px] text-forge-ash">
          Warnings: {warnings.join(" | ")}
        </div>
      ) : null}
      {deniedReason !== "—" ? (
        <div className="mt-2 rounded border border-forge-ember/30 bg-forge-ember/10 px-2 py-1 text-[11px] text-forge-ash">
          Denied: {deniedReason}
        </div>
      ) : null}
      {errorMessage !== "—" ? (
        <div className="mt-2 rounded border border-forge-ember/30 bg-forge-ember/10 px-2 py-1 text-[11px] text-forge-ash">
          Error: {errorMessage}
        </div>
      ) : null}
    </div>
  );
}

function toText(value: unknown) {
  if (value == null) return "—";
  if (typeof value === "string") return value.trim() || "—";
  if (typeof value === "number" || typeof value === "boolean") return String(value);
  return "structured";
}

function normalizeStringList(value: unknown) {
  if (!Array.isArray(value)) return [];
  return value.map((item) => String(item).trim()).filter(Boolean);
}

function summarizeOutput(value: unknown) {
  if (value == null) return "";
  if (typeof value === "string") return value;
  if (typeof value === "number" || typeof value === "boolean") return String(value);
  if (Array.isArray(value)) return `${value.length} item(s)`;
  if (typeof value === "object") return `${Object.keys(value as Record<string, unknown>).length} field(s)`;
  return "";
}

function parseReadableValue(value: string): unknown {
  const normalized = value.trim().toLowerCase();
  if (normalized === "yes" || normalized === "true") return true;
  if (normalized === "no" || normalized === "false") return false;
  if (normalized === "none" || normalized === "null") return null;
  const numberValue = Number(value);
  if (value !== "" && Number.isFinite(numberValue)) return numberValue;
  if (value.includes(",")) {
    return value
      .split(",")
      .map((part) => part.trim())
      .filter(Boolean);
  }
  return value;
}

const CAPABILITY_STATUS_OPTIONS: ToolCapabilityStatus[] = ["active", "approval_only", "disabled", "stubbed", "deferred", "deprecated"];

function requiresStatusReason(status: ToolCapabilityStatus) {
  return status === "disabled" || status === "stubbed" || status === "deferred" || status === "deprecated";
}

function capabilityStatusReason(status?: ToolCapabilityStatus, adapterId?: string) {
  switch (status) {
    case "active":
      return "Executable through the gateway when policy and scope checks pass.";
    case "approval_only":
      return "Execution is blocked until an explicit approval decision is granted.";
    case "disabled":
      return "Capability is registered but disabled by default policy.";
    case "stubbed":
      if (!adapterId) return "Capability is registered but implementation is incomplete and currently unsupported.";
      return `Capability wiring is incomplete; adapter ${adapterId} is not considered executable in default policy.`;
    case "deferred":
      return "Capability is intentionally deferred and returns unsupported until promoted.";
    case "deprecated":
      return "Capability remains registered for taxonomy completeness but is intentionally non-executable.";
    default:
      return "Capability status is unknown; execution is denied until status is corrected.";
  }
}

function GateLine(props: { prefix: "IF" | "AND"; label: string; pass: boolean }) {
  return (
    <div className="flex items-center justify-between gap-3 border-b border-forge-platinum/5 pb-1 last:border-b-0 last:pb-0">
      <div className="font-mono text-[11px] text-forge-mist">
        <span className="mr-2 text-forge-mist/60">{props.prefix}</span>
        {props.label}
      </div>
      <div className={props.pass ? "text-[11px] font-semibold text-forge-electric" : "text-[11px] font-semibold text-forge-emberSoft"}>
        {props.pass ? "pass" : "fail"}
      </div>
    </div>
  );
}
