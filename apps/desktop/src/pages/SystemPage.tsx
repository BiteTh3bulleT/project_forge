import { GhostButton } from "@forge/ui";
import type { ReactNode } from "react";
import { useCallback, useEffect, useMemo, useState } from "react";

import { api, type ForgeSystemStatus } from "../lib/api";

const REFRESH_MS = 30_000;

function valueText(value: unknown, fallback = "not reported") {
  if (typeof value === "boolean") return value ? "yes" : "no";
  if (typeof value === "number") return Number.isFinite(value) ? String(value) : fallback;
  if (typeof value === "string" && value.trim()) return value;
  return fallback;
}

function statusClass(status?: string) {
  const normalized = (status ?? "").toLowerCase();
  if (["ok", "normal", "available", "healthy", "reachable"].includes(normalized)) {
    return "forge-ops-status forge-ops-status--ok";
  }
  if (["degraded", "elevated", "constrained", "warning", "proposed"].includes(normalized)) {
    return "forge-ops-status forge-ops-status--warn";
  }
  if (["critical", "failed", "unreachable", "error"].includes(normalized)) {
    return "forge-ops-status forge-ops-status--bad";
  }
  return "forge-ops-status forge-ops-status--muted";
}

function formatTimestamp(value?: string) {
  if (!value) return "not reported";
  const parsed = new Date(value);
  if (Number.isNaN(parsed.getTime())) return value;
  return parsed.toLocaleTimeString([], { hour: "2-digit", minute: "2-digit" });
}

function formatBytes(value?: number) {
  if (typeof value !== "number" || !Number.isFinite(value) || value <= 0) {
    return "not reported";
  }
  const units = ["B", "KB", "MB", "GB", "TB"];
  let current = value;
  let unit = 0;
  while (current >= 1024 && unit < units.length - 1) {
    current /= 1024;
    unit += 1;
  }
  return `${current.toFixed(unit === 0 ? 0 : 1)} ${units[unit]}`;
}

function Metric(props: { label: string; value: unknown; tone?: string }) {
  return (
    <div className="rounded border border-white/10 bg-black/20 p-3">
      <div className="forge-ops-label">{props.label}</div>
      <div className="mt-1 flex min-h-6 items-center gap-2 text-sm font-semibold text-forge-ash">
        {props.tone ? (
          <span className={statusClass(props.tone)}>{valueText(props.value)}</span>
        ) : (
          <span>{valueText(props.value)}</span>
        )}
      </div>
    </div>
  );
}

function Panel(props: {
  title: string;
  detail?: string;
  children: ReactNode;
}) {
  return (
    <section className="forge-ops-card p-4">
      <div className="mb-3 flex items-start justify-between gap-3">
        <div>
          <h2 className="text-sm font-semibold text-forge-ash">{props.title}</h2>
          {props.detail ? (
            <p className="mt-1 text-xs leading-5 text-forge-mist/70">
              {props.detail}
            </p>
          ) : null}
        </div>
      </div>
      {props.children}
    </section>
  );
}

function BoundaryFlag(props: { label: string; enabled?: boolean }) {
  return (
    <div className="flex items-center justify-between gap-3 border-b border-white/10 py-2 last:border-0">
      <span className="text-xs text-forge-mist">{props.label}</span>
      <span className={statusClass(props.enabled ? "ok" : "warning")}>
        {props.enabled ? "disabled" : "check"}
      </span>
    </div>
  );
}

export function SystemPage() {
  const [status, setStatus] = useState<ForgeSystemStatus | null>(null);
  const [pendingApprovals, setPendingApprovals] = useState<number | null>(null);
  const [approvalError, setApprovalError] = useState("");
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const [lastLoadedAt, setLastLoadedAt] = useState("");

  const load = useCallback(async () => {
    setLoading(true);
    try {
      const [next, approvals] = await Promise.all([
        api.system.status(),
        api.approvals
          .list("pending", 25)
          .then((result) => ({ count: result.approvals?.length ?? 0, error: "" }))
          .catch((err) => ({
            count: null,
            error:
              err instanceof Error ? err.message : "approval queue unavailable",
          })),
      ]);
      setStatus(next);
      setPendingApprovals(approvals.count);
      setApprovalError(approvals.error);
      setError("");
      setLastLoadedAt(new Date().toISOString());
    } catch (err) {
      setError(err instanceof Error ? err.message : "system status unavailable");
      setStatus(null);
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    void load();
    const interval = window.setInterval(() => {
      void load();
    }, REFRESH_MS);
    return () => window.clearInterval(interval);
  }, [load]);

  const proposalRows = status?.forgeh.proposals ?? [];
  const executionRows = status?.forgeh.executions?.items ?? [];
  const warnings = useMemo(() => {
    const values = [
      ...(status?.warnings ?? []),
      ...(status?.forgeh.policy?.warnings ?? []),
      ...(status?.modelruntime.warnings ?? []),
      ...(status?.modelruntime.errors ?? []),
    ];
    return Array.from(new Set(values.filter(Boolean))).slice(0, 8);
  }, [status]);

  if (error && !status) {
    return (
      <div className="forge-ops-board space-y-5">
        <div className="flex items-start justify-between gap-4">
          <div>
            <div className="forge-ops-label">System</div>
            <h1 className="text-2xl font-semibold text-forge-ash">
              System Surfaces
            </h1>
          </div>
          <GhostButton onClick={() => void load()}>Refresh</GhostButton>
        </div>
        <div className="rounded-md border border-forge-ember/30 bg-forge-ember/10 p-4">
          <div className="text-sm font-semibold text-forge-ash">
            Core unreachable
          </div>
          <p className="mt-1 text-xs text-forge-mist/75">{error}</p>
        </div>
      </div>
    );
  }

  return (
    <div className="forge-ops-board space-y-5">
      <div className="flex flex-wrap items-start justify-between gap-4">
        <div>
          <div className="forge-ops-label">System</div>
          <h1 className="text-2xl font-semibold text-forge-ash">
            System Surfaces
          </h1>
        </div>
        <div className="flex items-center gap-3">
          <span className="text-xs text-forge-mist/70">
            {loading ? "Refreshing" : `Updated ${formatTimestamp(lastLoadedAt)}`}
          </span>
          <GhostButton onClick={() => void load()}>Refresh</GhostButton>
        </div>
      </div>

      <section className="grid gap-3 sm:grid-cols-2 xl:grid-cols-4">
        <Metric
          label="Core Status"
          value={status?.core.service ?? "forge-core"}
          tone={status?.core.reachable ? "ok" : "unreachable"}
        />
        <Metric
          label="FORGE-H Posture"
          value={status?.forgeh.policy?.overall_posture}
          tone={status?.forgeh.policy?.overall_posture}
        />
        <Metric
          label="Modelruntime"
          value={status?.modelruntime.state}
          tone={status?.modelruntime.available ? "available" : "unavailable"}
        />
        <Metric
          label="Storage"
          value={status?.storage.pressure_level}
          tone={status?.storage.pressure_level}
        />
      </section>

      <div className="grid gap-4 xl:grid-cols-[minmax(0,1.15fr)_minmax(360px,0.85fr)]">
        <div className="space-y-4">
          <Panel title="Shell Session Status">
            <div className="grid gap-3 md:grid-cols-3">
              <Metric label="Shell mode" value={status?.shell_session.shell_mode} />
              <Metric
                label="Display backend"
                value={status?.shell_session.display_backend}
              />
              <Metric
                label="Compositor"
                value={status?.shell_session.compositor_session}
              />
            </div>
            <div className="mt-3 rounded border border-white/10 bg-black/20 p-3">
              <BoundaryFlag
                label="Host mutation disabled"
                enabled={status?.shell_session.host_mutation_disabled}
              />
              <BoundaryFlag
                label="Model mutation disabled"
                enabled={status?.shell_session.model_mutation_disabled}
              />
              <BoundaryFlag
                label="Semantic memory write disabled"
                enabled={status?.shell_session.semantic_memory_write_disabled}
              />
              <BoundaryFlag
                label="FORGE-K live authority disabled"
                enabled={status?.shell_session.forge_k_live_authority_disabled}
              />
            </div>
          </Panel>

          <Panel
            title="Host Diagnostics Summary"
            detail={status?.hostbridge.reason}
          >
            <div className="grid gap-3 md:grid-cols-3">
              <Metric label="Host" value={status?.hostbridge.host_identity} />
              <Metric
                label="RAM pressure"
                value={status?.hostbridge.ram_pressure}
                tone={status?.hostbridge.ram_pressure}
              />
              <Metric
                label="Disk pressure"
                value={status?.hostbridge.disk_pressure}
                tone={status?.hostbridge.disk_pressure}
              />
              <Metric
                label="GPU available"
                value={status?.hostbridge.gpu_available}
                tone={status?.hostbridge.gpu_available ? "available" : "unavailable"}
              />
              <Metric
                label="Thermal available"
                value={status?.hostbridge.thermal_available}
                tone={status?.hostbridge.thermal_available ? "available" : "unavailable"}
              />
              <Metric
                label="Source errors"
                value={status?.hostbridge.source_errors_count ?? 0}
                tone={(status?.hostbridge.source_errors_count ?? 0) > 0 ? "warning" : "ok"}
              />
            </div>
          </Panel>

          <Panel title="FORGE-H Resource Posture">
            <div className="grid gap-3 md:grid-cols-3">
              <Metric
                label="RAM"
                value={status?.forgeh.policy?.ram_pressure}
                tone={status?.forgeh.policy?.ram_pressure}
              />
              <Metric
                label="Swap"
                value={status?.forgeh.policy?.swap_pressure}
                tone={status?.forgeh.policy?.swap_pressure}
              />
              <Metric
                label="VRAM"
                value={status?.forgeh.policy?.vram_pressure}
                tone={status?.forgeh.policy?.vram_pressure}
              />
              <Metric
                label="Thermal"
                value={status?.forgeh.policy?.thermal_pressure}
                tone={status?.forgeh.policy?.thermal_pressure}
              />
              <Metric
                label="Model load"
                value={status?.forgeh.policy?.model_load_recommendation}
              />
              <Metric
                label="Background work"
                value={status?.forgeh.policy?.background_work_recommendation}
              />
            </div>
          </Panel>

          <Panel title="FORGE-H Proposals">
            {proposalRows.length === 0 ? (
              <div className="rounded border border-white/10 bg-black/20 p-3 text-sm text-forge-mist">
                No resource proposals reported.
              </div>
            ) : (
              <div className="overflow-x-auto">
                <table className="w-full min-w-[680px] text-left text-xs">
                  <thead className="text-forge-mist/70">
                    <tr>
                      <th className="py-2 pr-3">Proposal</th>
                      <th className="py-2 pr-3">Action</th>
                      <th className="py-2 pr-3">Lane</th>
                      <th className="py-2 pr-3">Risk</th>
                      <th className="py-2 pr-3">Status</th>
                      <th className="py-2 pr-3">Expires</th>
                    </tr>
                  </thead>
                  <tbody>
                    {proposalRows.map((proposal) => (
                      <tr
                        key={proposal.proposal_id ?? proposal.action_type}
                        className="border-t border-white/10"
                      >
                        <td className="py-2 pr-3 font-mono text-forge-ash">
                          {valueText(proposal.proposal_id)}
                        </td>
                        <td className="py-2 pr-3 text-forge-mist">
                          {valueText(proposal.action_type)}
                        </td>
                        <td className="py-2 pr-3 text-forge-mist">
                          {valueText(proposal.target_lane, "none")}
                        </td>
                        <td className="py-2 pr-3">
                          <span className={statusClass(proposal.risk_level)}>
                            {valueText(proposal.risk_level)}
                          </span>
                        </td>
                        <td className="py-2 pr-3">
                          <span className={statusClass(proposal.status)}>
                            {valueText(proposal.status)}
                          </span>
                        </td>
                        <td className="py-2 pr-3 text-forge-mist">
                          {formatTimestamp(proposal.expires_at)}
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            )}
          </Panel>
        </div>

        <div className="space-y-4">
          <Panel title="FORGE-H Bounded Executions">
            {!status?.forgeh.executions?.available && executionRows.length === 0 ? (
              <div className="rounded border border-dashed border-white/10 bg-black/20 p-3">
                <div className="text-sm font-semibold text-forge-ash">
                  Bounded execution ledger not wired
                </div>
                <div className="mt-1 text-xs leading-5 text-forge-mist/70">
                  {status?.forgeh.executions?.reason ??
                    "No live execution surface is exposed in this phase."}
                </div>
              </div>
            ) : (
              <div className="space-y-2">
                {executionRows.map((execution) => (
                  <div
                    key={execution.execution_id ?? execution.proposal_id}
                    className="rounded border border-white/10 bg-black/20 p-3 text-xs"
                  >
                    <div className="font-mono text-forge-ash">
                      {valueText(execution.execution_id)}
                    </div>
                    <div className="mt-2 grid gap-2 sm:grid-cols-2">
                      <span>Status: {valueText(execution.status)}</span>
                      <span>Result: {valueText(execution.result)}</span>
                      <span>Bounded: {valueText(execution.bounded)}</span>
                      <span>Host mutation: {valueText(execution.host_mutation)}</span>
                    </div>
                  </div>
                ))}
              </div>
            )}
          </Panel>

          <Panel title="Modelruntime Status">
            <div className="space-y-2 text-xs text-forge-mist">
              <div className="flex justify-between gap-3">
                <span>Available</span>
                <span>{valueText(status?.modelruntime.available)}</span>
              </div>
              <div className="flex justify-between gap-3">
                <span>State</span>
                <span>{valueText(status?.modelruntime.state)}</span>
              </div>
              <div className="flex justify-between gap-3">
                <span>Backend</span>
                <span>{valueText(status?.modelruntime.backend)}</span>
              </div>
              <div className="flex justify-between gap-3">
                <span>Mutation disabled</span>
                <span>{valueText(status?.modelruntime.mutation_disabled)}</span>
              </div>
            </div>
          </Panel>

          <Panel title="Storage Status">
            <div className="space-y-2 text-xs text-forge-mist">
              <div className="flex justify-between gap-3">
                <span>Root</span>
                <span className="font-mono">{valueText(status?.storage.root)}</span>
              </div>
              <div className="flex justify-between gap-3">
                <span>Truth authority</span>
                <span>{valueText(status?.storage.truth_authority)}</span>
              </div>
              <div className="flex justify-between gap-3">
                <span>SQLite ping</span>
                <span>{valueText(status?.storage.ping_ok)}</span>
              </div>
              <div className="flex justify-between gap-3">
                <span>Free</span>
                <span>{formatBytes(status?.storage.free_bytes)}</span>
              </div>
              <div className="flex justify-between gap-3">
                <span>Redis truth</span>
                <span>{valueText(status?.storage.redis?.truth_authority)}</span>
              </div>
              <div className="flex justify-between gap-3">
                <span>Qdrant truth</span>
                <span>{valueText(status?.storage.qdrant?.truth_authority)}</span>
              </div>
            </div>
          </Panel>

          <Panel title="Approval Queue">
            {pendingApprovals == null ? (
              <div className="rounded border border-dashed border-white/10 bg-black/20 p-3">
                <div className="text-sm font-semibold text-forge-ash">
                  Approval queue surface not wired yet
                </div>
                <div className="mt-1 text-xs leading-5 text-forge-mist/70">
                  {approvalError || status?.approval_queue.reason || "No approval summary available."}
                </div>
              </div>
            ) : (
              <div className="rounded border border-white/10 bg-black/20 p-3">
                <div className="forge-ops-label">Pending approvals</div>
                <div className="mt-1 text-2xl font-semibold text-forge-ash">
                  {pendingApprovals}
                </div>
              </div>
            )}
            {status?.approval_queue.reason && pendingApprovals != null ? (
              <div className="mt-2 text-xs leading-5 text-forge-mist/70">
                {status.approval_queue.reason}
              </div>
            ) : null}
          </Panel>

          <Panel title="Recent Warnings">
            {warnings.length === 0 ? (
              <div className="text-sm text-forge-mist">No warnings reported.</div>
            ) : (
              <ul className="space-y-2 text-xs text-forge-mist">
                {warnings.map((warning) => (
                  <li
                    key={warning}
                    className="rounded border border-forge-amber/20 bg-forge-amber/10 p-2"
                  >
                    {warning}
                  </li>
                ))}
              </ul>
            )}
          </Panel>
        </div>
      </div>
    </div>
  );
}
