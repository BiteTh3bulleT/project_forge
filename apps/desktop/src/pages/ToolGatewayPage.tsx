import { GhostButton, Panel, PrimaryButton } from "@forge/ui";
import { useEffect, useMemo, useState } from "react";
import { useNavigate } from "react-router-dom";

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
  capabilityStatus?: string;
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
  status: string;
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
  const parsed = JSON.parse(trimmed);
  if (!parsed || typeof parsed !== "object" || Array.isArray(parsed)) {
    throw new Error("Input must be a JSON object.");
  }
  return parsed as Record<string, unknown>;
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
  const [inputRaw, setInputRaw] = useState("{}");
  const [dryRun, setDryRun] = useState(true);
  const [last, setLast] = useState<Record<string, unknown> | null>(null);
  const [err, setErr] = useState<string | null>(null);

  function newCorrelationId() {
    try {
      return crypto.randomUUID();
    } catch {
      return `gw-${Date.now()}`;
    }
  }

  const selectedTool = useMemo(() => tools.find((x) => x.id === toolId) ?? null, [toolId, tools]);

  async function refresh() {
    try {
      const [t, c, l, m, i] = await Promise.all([
        api.gateway.tools(),
        api.gateway.capabilities(),
        api.actionLanes.list(),
        api.meta(),
        api.gateway.invocations({ limit: 80, status: statusFilter || undefined }),
      ]);
      const toolRows = Array.isArray(t.tools) ? t.tools : [];
      const capabilityRows = Array.isArray(c.capabilities) ? c.capabilities : [];
      const laneRows = (Array.isArray(l.lanes) ? l.lanes : []) as LaneRow[];
      setTools(toolRows);
      setCapabilities(capabilityRows);
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
              <option value="error">error</option>
            </select>
            <GhostButton className="ml-2" onClick={() => void refresh()}>Apply</GhostButton>
          </label>
        </div>
      </Panel>

      <Panel title="Invoke Action" subtitle="Typed request contract: tool + lane + risk + execution level + scoped targets + structured input.">
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
        <div className="mt-3 grid gap-3 md:grid-cols-3">
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
        <label className="mt-3 block text-xs text-forge-mist">
          Paths (comma-separated, relative to workspace or absolute if policy allows)
          <input className="forge-input mt-1 font-mono text-xs" value={paths} onChange={(e) => setPaths(e.target.value)} placeholder="README.md, docs" />
        </label>
        <label className="mt-3 block text-xs text-forge-mist">
          Input JSON
          <textarea className="forge-input mt-1 min-h-[120px] font-mono text-xs" value={inputRaw} onChange={(e) => setInputRaw(e.target.value)} />
        </label>
        <label className="mt-3 flex items-center gap-2 text-xs text-forge-mist">
          <input type="checkbox" checked={dryRun} onChange={(e) => setDryRun(e.target.checked)} />
          Dry run
        </label>
        <div className="mt-4 flex gap-2">
          <PrimaryButton
            onClick={async () => {
              try {
                const input = parseInput(inputRaw);
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
            Execute
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
            <div key={t.id} className="rounded border border-white/10 bg-forge-slate/20 px-3 py-2 text-xs text-forge-mist">
              <div className="font-mono text-forge-ash">{t.id}</div>
              <div className="mt-1">{t.domain}.{t.action} · {t.riskClass} · {t.executionLevel} · {t.writeIntent ? "write" : "read"}</div>
              <div className="mt-1 text-[11px]">
                capability {t.capabilityId ?? "—"} · status {t.capabilityStatus ?? "—"} · risk {t.capabilityRisk ?? "—"} · adapter {t.adapterId ?? "—"}
              </div>
              <div className="mt-1">{t.description}</div>
            </div>
          ))}
        </div>
      </Panel>

      <Panel title="Capability Registry" subtitle="Kernel capability taxonomy, status, risk, and autonomy eligibility.">
        <div className="space-y-2">
          {capabilities.length === 0 ? <div className="text-sm text-forge-mist">No capability metadata available.</div> : null}
          {capabilities.map((cap) => (
            <div key={cap.id} className="rounded border border-white/10 bg-black/25 px-3 py-2 text-xs text-forge-mist">
              <div className="font-mono text-forge-ash">{cap.id}</div>
              <div className="mt-1">
                {cap.domain}.{cap.name} · {cap.status} · risk {cap.risk} · lane {cap.lane}
              </div>
              <div className="mt-1">
                effects {cap.effect.join(", ")} · intent {cap.requiresIntent ? "required" : "optional"} · dry-run{" "}
                {cap.allowedInDryRun ? "allowed" : "blocked"} · autonomy {cap.autonomyEligible ? "eligible" : "not eligible"}
              </div>
              <div className="mt-1">{cap.description}</div>
            </div>
          ))}
        </div>
      </Panel>

      <Panel title="Invocation History" subtitle="Policy outcomes, approvals, and execution results from gateway_invocations.">
        <div className="space-y-2">
          {invs.length === 0 ? <div className="text-sm text-forge-mist">No invocations in this filter.</div> : null}
          {invs.map((row) => (
            <button
              key={row.id}
              type="button"
              onClick={() => setLast(row as unknown as Record<string, unknown>)}
              className="w-full rounded border border-white/10 bg-black/25 px-3 py-2 text-left text-xs text-forge-mist hover:border-forge-accent/50"
            >
              <div className="font-mono text-[11px] text-forge-ash">#{row.id} · {row.toolId} · {row.status} · {row.policyOutcome}</div>
              <div className="mt-1">{formatTime(row.createdAtMs)} · lane {row.laneId ?? "—"} · risk {row.riskClass} · {row.executionLevel}</div>
              <div className="mt-1">{row.deniedReason || row.correlationId}</div>
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
    <div className="mt-4 rounded-md border border-white/10 bg-black/25 p-3 text-[11px] text-forge-mist">
      <div className="grid gap-1 md:grid-cols-2">
        {rows
          .filter(([, value]) => value !== "—")
          .map(([label, value]) => (
            <div key={label} className="flex items-start justify-between gap-3 border-b border-white/5 pb-1">
              <span className="text-forge-mist/75">{label}</span>
              <span className="text-right text-forge-ash">{value}</span>
            </div>
          ))}
      </div>
      {outputSummary ? (
        <div className="mt-2 rounded border border-white/10 bg-black/30 px-2 py-1 text-[11px] text-forge-ash">
          Output: {outputSummary}
        </div>
      ) : null}
      {warnings.length > 0 ? (
        <div className="mt-2 rounded border border-white/10 bg-black/30 px-2 py-1 text-[11px] text-forge-ash">
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
