import type { DashboardSummary, MemoryObservation, ReviewRecord } from "@forge/shared";
import { GhostButton, Panel, PrimaryButton } from "@forge/ui";
import { useEffect, useMemo, useState } from "react";
import { useNavigate } from "react-router-dom";

import { FoldSection } from "../components/FoldSection";
import {
  api,
  type AutonomyBudgetRecord,
  type AutonomyCharterRecord,
  type AutonomyDecisionRecord,
  type AutonomyIntentRecord,
  type AutonomyStatusSnapshot,
  type DiscordGatewayStatusResponse,
  type TelegramStatusResponse,
} from "../lib/api";
import { formatTime } from "../lib/format";
import { useUiStore } from "../stores/uiStore";

type CapabilityRecord = Awaited<ReturnType<typeof api.gateway.capabilities>>["capabilities"][number];
type InvocationRecord = Awaited<ReturnType<typeof api.gateway.invocations>>["invocations"][number];

type SimilarityNode = {
  id: number;
  title: string;
  type: string;
  confidence: number;
  stale: boolean;
  degree: number;
  x: number;
  y: number;
};

type SimilarityEdge = {
  id: string;
  from: number;
  to: number;
  weight: number;
  reasons: string[];
};

type DashboardView = "all" | "telemetry" | "queues" | "runtime";

export function DashboardPage() {
  const navigate = useNavigate();
  const setStatus = useUiStore((s) => s.setStatusLine);
  const uiMode = useUiStore((s) => s.uiMode);

  const [summary, setSummary] = useState<DashboardSummary | null>(null);
  const [autonomyStatus, setAutonomyStatus] = useState<AutonomyStatusSnapshot | null>(null);
  const [discordStatus, setDiscordStatus] = useState<DiscordGatewayStatusResponse | null>(null);
  const [telegramStatus, setTelegramStatus] = useState<TelegramStatusResponse | null>(null);
  const [autonomyIntents, setAutonomyIntents] = useState<AutonomyIntentRecord[]>([]);
  const [autonomyDecisions, setAutonomyDecisions] = useState<AutonomyDecisionRecord[]>([]);
  const [autonomyBudgets, setAutonomyBudgets] = useState<AutonomyBudgetRecord[]>([]);
  const [autonomyCharters, setAutonomyCharters] = useState<AutonomyCharterRecord[]>([]);
  const [capabilities, setCapabilities] = useState<CapabilityRecord[]>([]);
  const [invocations, setInvocations] = useState<InvocationRecord[]>([]);
  const [observations, setObservations] = useState<MemoryObservation[]>([]);
  const [similarityNotice, setSimilarityNotice] = useState<string | null>(null);
  const [pendingReviews, setPendingReviews] = useState<ReviewRecord[]>([]);
  const [dashboardView, setDashboardView] = useState<DashboardView>("all");
  const [err, setErr] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);
  const [lastUpdatedAt, setLastUpdatedAt] = useState<number>(Date.now());

  async function load() {
    setLoading(true);
    try {
      const [
        dash,
        rev,
        autoStatus,
        intentsRes,
        decisionsRes,
        budgetsRes,
        chartersRes,
        gatewayCaps,
        gatewayInvs,
      ] = await Promise.all([
        api.dashboard.summary(),
        api.reviews.list({ status: "pending", limit: 20 }),
        api.autonomy.status(),
        api.autonomy.intents({ limit: 60 }),
        api.autonomy.decisions(60),
        api.autonomy.budgets(),
        api.autonomy.charters(true),
        api.gateway.capabilities(),
        api.gateway.invocations({ limit: 120 }),
      ]);
      const memoryObs = await api.memory.listObservations({ limit: 64 }).catch((error: unknown) => {
        setSimilarityNotice(`Similarity feed unavailable: ${error instanceof Error ? error.message : String(error)}`);
        return { observations: [] as MemoryObservation[] };
      });
      const [discordGateway, telegramGateway] = await Promise.allSettled([api.discord.status(), api.telegram.status()]);
      setSummary(dash);
      setPendingReviews(rev.reviews);
      setAutonomyStatus(autoStatus);
      setDiscordStatus(discordGateway.status === "fulfilled" ? discordGateway.value : null);
      setTelegramStatus(telegramGateway.status === "fulfilled" ? telegramGateway.value : null);
      setAutonomyIntents(Array.isArray(intentsRes.intents) ? intentsRes.intents : []);
      setAutonomyDecisions(Array.isArray(decisionsRes.decisions) ? decisionsRes.decisions : []);
      setAutonomyBudgets(Array.isArray(budgetsRes.budgets) ? budgetsRes.budgets : []);
      setAutonomyCharters(Array.isArray(chartersRes.charters) ? chartersRes.charters : []);
      setCapabilities(Array.isArray(gatewayCaps.capabilities) ? gatewayCaps.capabilities : []);
      setInvocations(Array.isArray(gatewayInvs.invocations) ? gatewayInvs.invocations : []);
      setObservations(Array.isArray(memoryObs.observations) ? memoryObs.observations : []);
      if (Array.isArray(memoryObs.observations) && memoryObs.observations.length > 0) {
        setSimilarityNotice(null);
      }
      setLastUpdatedAt(Date.now());
      setErr(null);
    } catch (e) {
      setErr(e instanceof Error ? e.message : String(e));
    } finally {
      setLoading(false);
    }
  }

  useEffect(() => {
    void load();
    const id = window.setInterval(() => void load(), 5000);
    return () => window.clearInterval(id);
  }, []);

  const activeIntentCount = useMemo(
    () => autonomyIntents.filter((intent) => ["proposed", "approved", "running", "blocked"].includes(intent.status)).length,
    [autonomyIntents],
  );
  const activeCharterCount = useMemo(() => autonomyCharters.filter((charter) => charter.status === "active").length, [autonomyCharters]);
  const activeBudgetCount = useMemo(() => autonomyBudgets.filter((budget) => budget.status === "active").length, [autonomyBudgets]);

  const attentionCount = (summary?.approvalsPending ?? 0) + (summary?.reviewsPending ?? 0);
  const discordConnected =
    discordStatus?.enabled === true &&
    typeof discordStatus.status === "object" &&
    discordStatus.status !== null &&
    Boolean(discordStatus.status.connected);
  const discordSnapshot = discordStatus && typeof discordStatus.status === "object" && discordStatus.status != null ? discordStatus.status : null;
  const telegramConnected = Boolean(telegramStatus?.ready);
  const systemStatusRows = useMemo(() => flattenSystemStatus(summary?.systemStatus), [summary?.systemStatus]);

  const capabilityStatusCounts = useMemo(() => countBy(capabilities, (row) => row.status || "unknown"), [capabilities]);
  const capabilityRiskCounts = useMemo(() => countBy(capabilities, (row) => row.risk || "unknown"), [capabilities]);
  const invocationStatusCounts = useMemo(() => countBy(invocations, (row) => row.status || "unknown"), [invocations]);

  const activitySeries = useMemo(() => buildActivitySeries(invocations, autonomyDecisions), [invocations, autonomyDecisions]);
  const similarityGraph = useMemo(() => buildSimilarityGraph(observations), [observations]);

  const recentBlockedDecisions = useMemo(
    () => autonomyDecisions.filter((decision) => isBlockingDecision(decision.decision)).slice(0, 8),
    [autonomyDecisions],
  );

  const recentGatewayDenials = useMemo(
    () => invocations.filter((invocation) => invocation.status === "denied" || invocation.status === "approval_required").slice(0, 8),
    [invocations],
  );

  return (
    <div className="space-y-6">
      <Panel
        title="Main Dashboard"
        subtitle={
          uiMode === "guided"
            ? "A cleaner mission surface: focus queue, autonomy pulse, and correlation graph."
            : "Operational command deck for autonomy, capability policy, and memory correlation telemetry."
        }
        actions={
          <div className="flex items-center gap-2">
            <label className="text-[11px] text-forge-mist">
              View
              <select className="forge-input ml-2 px-2 py-1 text-[11px]" value={dashboardView} onChange={(e) => setDashboardView(e.target.value as DashboardView)}>
                <option value="all">All</option>
                <option value="telemetry">Telemetry</option>
                <option value="queues">Queues</option>
                <option value="runtime">Runtime</option>
              </select>
            </label>
            <GhostButton onClick={() => void load()} disabled={loading}>{loading ? "Refreshing..." : "Refresh"}</GhostButton>
            <GhostButton onClick={() => navigate("/autonomy")}>Open Autonomy</GhostButton>
            <GhostButton onClick={() => navigate("/gateway")}>Open Gateway</GhostButton>
          </div>
        }
      >
        {err ? <div className="rounded border border-forge-ember/30 bg-forge-ember/10 p-3 text-sm text-forge-ash">{err}</div> : null}
        {!summary ? (
          <div className="text-sm text-forge-mist">Loading dashboard telemetry...</div>
        ) : (
          <FoldSection title="Mission Snapshot" subtitle="Core operational counters and queue pressure." defaultOpen>
            <div className="grid gap-3 sm:grid-cols-2 xl:grid-cols-4">
              <Metric title="Active Jobs" value={String(summary.activeJobs.length)} hint="queued/preparing/running" onClick={() => navigate("/jobs")} />
              <Metric title="Attention Queue" value={String(attentionCount)} hint="approvals + reviews" onClick={() => navigate("/approvals")} />
              <Metric title="Active Intents" value={String(activeIntentCount)} hint="autonomy intent queue" onClick={() => navigate("/autonomy")} />
              <Metric title="Capabilities" value={String(capabilities.length)} hint="registered tool capabilities" onClick={() => navigate("/gateway")} />
              <Metric
                title="Discord Gateway"
                value={discordConnected ? "connected" : discordStatus?.enabled ? "starting" : "disabled"}
                hint="external operator I/O surface"
                onClick={() => setStatus(`Discord gateway ${discordConnected ? "connected" : "not connected"}`)}
              />
              <Metric title="Active Charters" value={String(activeCharterCount)} hint="bounded autonomy authority" onClick={() => navigate("/autonomy")} />
              <Metric title="Active Budgets" value={String(activeBudgetCount)} hint="freedom budget envelopes" onClick={() => navigate("/autonomy")} />
              <Metric title="Tool Invocations" value={String(invocations.length)} hint="recent gateway calls" onClick={() => navigate("/gateway")} />
              <Metric title="Memory Links" value={String(similarityGraph.edges.length)} hint="live similarity edges" onClick={() => navigate("/memory")} />
            </div>
          </FoldSection>
        )}
        <div className="mt-4 grid gap-2 md:grid-cols-4">
          <PrimaryButton onClick={() => navigate("/jobs")}>Open Jobs</PrimaryButton>
          <GhostButton onClick={() => navigate("/approvals")}>Open Approvals</GhostButton>
          <GhostButton onClick={() => navigate("/reviews")}>Open Reviews</GhostButton>
          <GhostButton
            onClick={() => {
              setStatus(`Dashboard refreshed at ${formatTime(lastUpdatedAt)}`);
            }}
          >
            Last update {formatTime(lastUpdatedAt)}
          </GhostButton>
        </div>
      </Panel>

      {(dashboardView === "all" || dashboardView === "telemetry") ? (
      <div className="grid gap-6 2xl:grid-cols-[1.45fr_1fr]">
        <Panel
          title="Live Similarity Connections"
          subtitle="Obsidian-style correlation map of recent memory observations based on shared tags/entities/context. This view is diagnostic evidence, not canonical truth."
          actions={<GhostButton onClick={() => navigate("/memory")}>Open Memory</GhostButton>}
        >
          {similarityNotice ? <div className="mb-2 rounded border border-forge-ember/30 bg-forge-ember/10 p-2 text-xs text-forge-ash">{similarityNotice}</div> : null}
          <SimilarityNetwork graph={similarityGraph} />
        </Panel>

        <div className="space-y-6">
          <Panel
            title="Autonomy Pulse"
            subtitle="Dream-state activity, intent pressure, and policy outcomes over recent buckets."
            actions={<GhostButton onClick={() => navigate("/autonomy")}>Inspect</GhostButton>}
          >
            <div className="grid gap-3 sm:grid-cols-2">
              <MiniMetric label="Mode" value={autonomyStatus?.mode || "off"} />
              <MiniMetric label="Dream State" value={autonomyStatus?.dream?.active ? "active" : "idle/off"} />
              <MiniMetric label="Recent Decisions" value={String(autonomyStatus?.counts?.recentDecisions ?? autonomyDecisions.length)} />
              <MiniMetric label="Workspace" value={autonomyStatus?.scope?.workspaceId || "unknown"} />
            </div>
            <div className="mt-4">
              <ActivityBars values={activitySeries} />
            </div>
            <div className="mt-4 space-y-2">
              {recentBlockedDecisions.length === 0 ? (
                <div className="text-xs text-forge-mist">No blocked or denied autonomy decisions in this window.</div>
              ) : (
                recentBlockedDecisions.map((decision) => (
                  <div key={decision.id} className="rounded border border-white/10 bg-black/20 p-2 text-[11px] text-forge-mist">
                    <div className="font-semibold text-forge-ash">{decision.decision}</div>
                    <div className="mt-1">intent {decision.intentId} | risk {decision.risk || "unknown"} | {formatTime(decision.createdAt || 0)}</div>
                  </div>
                ))
              )}
            </div>
          </Panel>

          <Panel
            title="Capability and Risk Matrix"
            subtitle="Tool capability status and risk distribution from the kernel registry."
            actions={<GhostButton onClick={() => navigate("/gateway")}>Inspect</GhostButton>}
          >
            <CountGrid
              entries={[
                ["active", capabilityStatusCounts.active ?? 0],
                ["approval_only", capabilityStatusCounts.approval_only ?? 0],
                ["stubbed", capabilityStatusCounts.stubbed ?? 0],
                ["disabled", capabilityStatusCounts.disabled ?? 0],
                ["deferred", capabilityStatusCounts.deferred ?? 0],
                ["deprecated", capabilityStatusCounts.deprecated ?? 0],
              ]}
            />
            <div className="mt-4">
              <RiskBars title="Capability Risk Tiers" counts={capabilityRiskCounts} />
            </div>
            <div className="mt-4">
              <RiskBars title="Invocation Outcomes" counts={invocationStatusCounts} />
            </div>
          </Panel>
        </div>
      </div>
      ) : null}

      {(dashboardView === "all" || dashboardView === "queues") ? (
      <div className="grid gap-6 xl:grid-cols-2">
        <Panel title="Active Jobs" subtitle="Current execution pressure and lane occupancy.">
          {!summary || summary.activeJobs.length === 0 ? (
            <div className="text-sm text-forge-mist">No active jobs.</div>
          ) : (
            <div className="space-y-2">
              {summary.activeJobs.slice(0, 12).map((job) => (
                <button
                  key={job.id}
                  type="button"
                  onClick={() => navigate(`/jobs/${job.id}`)}
                  className="w-full rounded border border-white/10 bg-black/20 p-3 text-left hover:border-forge-ember/35"
                >
                  <div className="text-sm font-semibold text-forge-ash">{job.title}</div>
                  <div className="mt-1 text-xs text-forge-mist">
                    {job.id} | {job.status} | {job.targetAdapter}
                  </div>
                  <div className="mt-1 text-[11px] text-forge-mist">{formatTime(job.createdAtMs)}</div>
                </button>
              ))}
            </div>
          )}
        </Panel>

        <Panel title="Attention Feed" subtitle="Pending reviews and denied/escalated gateway outcomes requiring operator action.">
          <div className="mb-3 flex gap-2">
            <PrimaryButton onClick={() => navigate("/reviews")}>Open Reviews</PrimaryButton>
            <GhostButton onClick={() => navigate("/approvals")}>Open Approvals</GhostButton>
          </div>
          <div className="space-y-2">
            {pendingReviews.slice(0, 6).map((review) => (
              <div key={`review-${review.id}`} className="rounded border border-white/10 bg-black/20 p-2 text-[11px] text-forge-mist">
                <div className="font-semibold text-forge-ash">Review #{review.id} | {review.status}</div>
                <div className="mt-1">{review.targetType}:{review.targetId} | {review.summary || "(no summary)"}</div>
              </div>
            ))}
            {recentGatewayDenials.map((invocation) => (
              <div key={`inv-${invocation.id}`} className="rounded border border-forge-ember/30 bg-forge-ember/10 p-2 text-[11px] text-forge-ash">
                <div className="font-semibold">{invocation.status} | {invocation.toolId}</div>
                <div className="mt-1">{invocation.deniedReason || invocation.policyOutcome || "policy blocked"}</div>
                <div className="mt-1 text-forge-mist">{formatTime(invocation.createdAtMs)}</div>
              </div>
            ))}
            {pendingReviews.length === 0 && recentGatewayDenials.length === 0 ? (
              <div className="text-sm text-forge-mist">No pending review or denial items.</div>
            ) : null}
          </div>
        </Panel>
      </div>
      ) : null}

      {(dashboardView === "all" || dashboardView === "runtime") && uiMode !== "guided" ? (
        <Panel title="Core Runtime Snapshot" subtitle="Canonical system status values rendered as operator-readable fields.">
          {!summary ? (
            <div className="text-sm text-forge-mist">No status yet.</div>
          ) : systemStatusRows.length === 0 ? (
            <div className="text-sm text-forge-mist">No status fields available.</div>
          ) : (
            <div className="grid gap-2 md:grid-cols-2">
              {systemStatusRows.map(([label, value]) => (
                <MiniMetric key={label} label={label} value={value} />
              ))}
            </div>
          )}
        </Panel>
      ) : null}

      {(dashboardView === "all" || dashboardView === "runtime") && uiMode !== "guided" ? (
        <Panel
          title="Remote Channel Status"
          subtitle="Telegram and Discord ingress health for operator control surfaces."
          actions={<GhostButton onClick={() => navigate("/settings")}>Configure Channels</GhostButton>}
        >
          <div className="grid gap-4 xl:grid-cols-2">
            <div className="rounded border border-white/10 bg-black/20 p-3">
              <div className="flex items-center justify-between">
                <div className="text-xs font-semibold uppercase tracking-[0.14em] text-forge-mist">Telegram</div>
                <StatePill ok={telegramConnected} okLabel="ready" offLabel="needs setup" />
              </div>
              <div className="mt-3 grid gap-2 md:grid-cols-2">
                <MiniMetric label="Remote Enabled" value={telegramStatus?.remoteAccessEnabled ? "yes" : "no"} />
                <MiniMetric label="Token Configured" value={telegramStatus?.tokenConfigured ? "yes" : "no"} />
                <MiniMetric label="Cross Chat Context" value={telegramStatus?.crossChatContext ? "on" : "off"} />
                <MiniMetric label="Default Chat" value={telegramStatus?.defaultChatId || "—"} />
                <MiniMetric label="Webhook" value={telegramStatus?.webhook?.url ? "configured" : "not set"} />
                <MiniMetric label="Reason" value={telegramStatus?.reason || "—"} />
              </div>
            </div>
            <div className="rounded border border-white/10 bg-black/20 p-3">
              <div className="flex items-center justify-between">
                <div className="text-xs font-semibold uppercase tracking-[0.14em] text-forge-mist">Discord</div>
                <StatePill ok={discordConnected} okLabel="connected" offLabel="offline" />
              </div>
              {!discordStatus ? (
                <div className="mt-3 text-sm text-forge-mist">No Discord gateway status yet.</div>
              ) : typeof discordStatus.status === "string" ? (
                <div className="mt-3 text-sm text-forge-mist">
                  Discord gateway is disabled.
                  {discordStatus.reason ? ` Reason: ${discordStatus.reason}` : ""}
                </div>
              ) : (
                <div className="mt-3 grid gap-2 md:grid-cols-2">
                  <MiniMetric label="Guild" value={discordSnapshot?.guildId || "any"} />
                  <MiniMetric label="Prefix" value={discordSnapshot?.commandPrefix || "!forge"} />
                  <MiniMetric label="Slash Commands" value={discordSnapshot?.enableSlash ? "on" : "off"} />
                  <MiniMetric label="Text Commands" value={discordSnapshot?.enableText ? "on" : "off"} />
                  <MiniMetric label="Passive Listening" value={discordSnapshot?.enablePassive ? "on" : "off"} />
                  <MiniMetric label="Cross Chat Context" value={discordSnapshot?.crossChatContext ? "on" : "off"} />
                  <MiniMetric label="Inbound" value={String(discordSnapshot?.inboundCount ?? 0)} />
                  <MiniMetric label="Outbound" value={String(discordSnapshot?.outboundCount ?? 0)} />
                  <MiniMetric label="Last Inbound" value={discordSnapshot?.lastInboundAtMs ? formatTime(discordSnapshot.lastInboundAtMs) : "—"} />
                  <MiniMetric label="Last Outbound" value={discordSnapshot?.lastOutboundAtMs ? formatTime(discordSnapshot.lastOutboundAtMs) : "—"} />
                </div>
              )}
            </div>
          </div>
        </Panel>
      ) : null}
    </div>
  );
}

function ActivityBars(props: { values: number[] }) {
  const max = Math.max(1, ...props.values);
  return (
    <div>
      <div className="text-[11px] uppercase tracking-[0.14em] text-forge-mist/70">Recent Activity</div>
      <div className="mt-2 flex h-20 items-end gap-1.5 rounded border border-white/10 bg-black/20 p-2">
        {props.values.map((value, idx) => (
          <div
            key={`bar-${idx}`}
            className="flex-1 rounded-sm bg-forge-electric/70"
            style={{ height: `${Math.max(8, Math.round((value / max) * 100))}%` }}
            title={`Bucket ${idx + 1}: ${value}`}
          />
        ))}
      </div>
    </div>
  );
}

function CountGrid(props: { entries: Array<[string, number]> }) {
  return (
    <div className="grid gap-2 sm:grid-cols-2">
      {props.entries.map(([label, value]) => (
        <div key={label} className="rounded border border-white/10 bg-black/20 px-3 py-2 text-[11px] text-forge-mist">
          <div className="uppercase tracking-[0.14em] text-forge-mist/75">{label}</div>
          <div className="mt-1 text-lg font-semibold text-forge-ash">{value}</div>
        </div>
      ))}
    </div>
  );
}

function RiskBars(props: { title: string; counts: Record<string, number> }) {
  const ordered = Object.entries(props.counts).sort((a, b) => b[1] - a[1]);
  const total = ordered.reduce((acc, item) => acc + item[1], 0);
  return (
    <div>
      <div className="text-[11px] uppercase tracking-[0.14em] text-forge-mist/70">{props.title}</div>
      <div className="mt-2 space-y-1.5">
        {ordered.length === 0 ? <div className="text-xs text-forge-mist">No data.</div> : null}
        {ordered.map(([label, value]) => {
          const ratio = total <= 0 ? 0 : value / total;
          return (
            <div key={label} className="rounded border border-white/10 bg-black/20 px-2 py-1.5 text-[11px] text-forge-mist">
              <div className="flex items-center justify-between">
                <span>{label}</span>
                <span className="font-semibold text-forge-ash">{value}</span>
              </div>
              <div className="mt-1 h-1.5 rounded bg-white/10">
                <div className="h-full rounded bg-forge-electric/70" style={{ width: `${Math.max(4, Math.round(ratio * 100))}%` }} />
              </div>
            </div>
          );
        })}
      </div>
    </div>
  );
}

function MiniMetric(props: { label: string; value: string }) {
  return (
    <div className="rounded border border-white/10 bg-black/20 px-3 py-2 text-[11px] text-forge-mist">
      <div className="uppercase tracking-[0.14em] text-forge-mist/70">{props.label}</div>
      <div className="mt-1 text-sm font-semibold text-forge-ash">{props.value}</div>
    </div>
  );
}

function StatePill(props: { ok: boolean; okLabel: string; offLabel: string }) {
  return (
    <div className="inline-flex items-center gap-2 text-[11px]">
      <span className={props.ok ? "h-2 w-2 rounded-full bg-forge-electric" : "h-2 w-2 rounded-full bg-forge-emberSoft"} />
      <span className={props.ok ? "text-forge-ash" : "text-forge-emberSoft"}>{props.ok ? props.okLabel : props.offLabel}</span>
    </div>
  );
}

function flattenSystemStatus(raw: Record<string, unknown> | null | undefined): Array<[string, string]> {
  if (!raw || typeof raw !== "object") return [];
  const rows: Array<[string, string]> = [];
  for (const [key, value] of Object.entries(raw)) {
    if (value == null) {
      rows.push([humanizeKey(key), "—"]);
      continue;
    }
    if (typeof value === "string" || typeof value === "number" || typeof value === "boolean") {
      rows.push([humanizeKey(key), String(value)]);
      continue;
    }
    if (Array.isArray(value)) {
      rows.push([humanizeKey(key), `${value.length} items`]);
      continue;
    }
    rows.push([humanizeKey(key), "available"]);
  }
  return rows;
}

function humanizeKey(key: string) {
  return key
    .replace(/([a-z0-9])([A-Z])/g, "$1 $2")
    .replace(/[_-]+/g, " ")
    .replace(/\s+/g, " ")
    .trim()
    .replace(/^\w/, (c) => c.toUpperCase());
}

function SimilarityNetwork(props: { graph: { nodes: SimilarityNode[]; edges: SimilarityEdge[] } }) {
  const [selectedNodeId, setSelectedNodeId] = useState<number | null>(null);
  const selectedNode = useMemo(() => {
    if (props.graph.nodes.length === 0) return null;
    const preferred = selectedNodeId ?? props.graph.nodes[0].id;
    return props.graph.nodes.find((node) => node.id === preferred) ?? props.graph.nodes[0];
  }, [props.graph.nodes, selectedNodeId]);

  const selectedEdges = useMemo(() => {
    if (!selectedNode) return [];
    return props.graph.edges.filter((edge) => edge.from === selectedNode.id || edge.to === selectedNode.id).slice(0, 6);
  }, [props.graph.edges, selectedNode]);

  if (props.graph.nodes.length === 0) {
    return <div className="text-sm text-forge-mist">No observations available for similarity visualization yet.</div>;
  }

  return (
    <div className="grid gap-3 xl:grid-cols-[1.35fr_1fr]">
      <div className="rounded-xl border border-white/10 bg-black/25 p-2">
        <svg viewBox="0 0 860 520" className="h-[360px] w-full rounded-lg bg-black/40">
          <rect x="0" y="0" width="860" height="520" fill="transparent" />
          {props.graph.edges.map((edge) => {
            const from = props.graph.nodes.find((node) => node.id === edge.from);
            const to = props.graph.nodes.find((node) => node.id === edge.to);
            if (!from || !to) return null;
            const highlighted = selectedNode ? edge.from === selectedNode.id || edge.to === selectedNode.id : false;
            return (
              <line
                key={edge.id}
                x1={from.x}
                y1={from.y}
                x2={to.x}
                y2={to.y}
                className={highlighted ? "forge-sim-graph__edge forge-sim-graph__edge--highlight" : "forge-sim-graph__edge"}
                strokeWidth={Math.min(3, 0.8 + edge.weight * 0.45)}
              >
                <title>{edge.reasons.join(" | ")}</title>
              </line>
            );
          })}
          {props.graph.nodes.map((node) => {
            const isActive = selectedNode?.id === node.id;
            return (
              <g key={`node-${node.id}`} transform={`translate(${node.x},${node.y})`} onClick={() => setSelectedNodeId(node.id)} className="cursor-pointer">
                <circle
                  r={isActive ? 11 : 8}
                  className={node.stale ? "forge-sim-graph__node forge-sim-graph__node--stale" : "forge-sim-graph__node"}
                  opacity={Math.max(0.45, Math.min(1, node.confidence))}
                >
                  <title>{node.title}</title>
                </circle>
                {isActive ? (
                  <>
                    <circle r={15} className="forge-sim-graph__halo" />
                    <text x="0" y="-16" textAnchor="middle" className="fill-forge-ash text-[11px] font-semibold">
                      {truncateLabel(node.title, 22)}
                    </text>
                  </>
                ) : null}
              </g>
            );
          })}
        </svg>
      </div>
      <div className="space-y-2">
        <div className="rounded border border-white/10 bg-black/20 p-3 text-[11px] text-forge-mist">
          <div className="font-semibold uppercase tracking-[0.14em] text-forge-mist/70">Selection</div>
          {!selectedNode ? (
            <div className="mt-2 text-forge-mist">Select a node.</div>
          ) : (
            <>
              <div className="mt-2 text-sm font-semibold text-forge-ash">{selectedNode.title}</div>
              <div className="mt-1">obs #{selectedNode.id} | {selectedNode.type} | degree {selectedNode.degree}</div>
              <div className="mt-1">confidence {(selectedNode.confidence * 100).toFixed(0)}% | stale {String(selectedNode.stale)}</div>
            </>
          )}
        </div>
        <div className="rounded border border-white/10 bg-black/20 p-3 text-[11px] text-forge-mist">
          <div className="font-semibold uppercase tracking-[0.14em] text-forge-mist/70">Connected Evidence</div>
          <div className="mt-2 space-y-1.5">
            {selectedEdges.length === 0 ? <div className="text-forge-mist">No strong links.</div> : null}
            {selectedEdges.map((edge) => (
                <div key={`selected-${edge.id}`} className="rounded border border-white/10 bg-black/30 p-2">
                  <div className="font-semibold text-forge-ash">obs #{edge.from} {"<->"} obs #{edge.to}</div>
                <div className="mt-1">{edge.reasons.join(" | ")}</div>
              </div>
            ))}
          </div>
        </div>
      </div>
    </div>
  );
}

function buildActivitySeries(invocations: InvocationRecord[], decisions: AutonomyDecisionRecord[]) {
  const buckets = new Array<number>(14).fill(0);
  const now = Date.now();
  const windowMs = 14 * 5 * 60_000;
  const bucketMs = windowMs / buckets.length;

  for (const item of invocations) {
    if (!item.createdAtMs) continue;
    const delta = now - item.createdAtMs;
    if (delta < 0 || delta > windowMs) continue;
    const idx = buckets.length - 1 - Math.floor(delta / bucketMs);
    if (idx >= 0 && idx < buckets.length) buckets[idx] += 1;
  }

  for (const item of decisions) {
    const createdAt = item.createdAt ?? 0;
    if (!createdAt) continue;
    const delta = now - createdAt;
    if (delta < 0 || delta > windowMs) continue;
    const idx = buckets.length - 1 - Math.floor(delta / bucketMs);
    if (idx >= 0 && idx < buckets.length) buckets[idx] += 1;
  }

  return buckets;
}

function buildSimilarityGraph(observations: MemoryObservation[]) {
  const sampled = [...observations].sort((a, b) => b.observedAtMs - a.observedAtMs).slice(0, 24);
  const edges: SimilarityEdge[] = [];

  for (let i = 0; i < sampled.length; i += 1) {
    for (let j = i + 1; j < sampled.length; j += 1) {
      const left = sampled[i];
      const right = sampled[j];
      const reasons: string[] = [];
      let weight = 0;

      const sharedTags = overlapCount(left.tags, right.tags);
      if (sharedTags > 0) {
        weight += Math.min(2, sharedTags);
        reasons.push(`shared tags (${sharedTags})`);
      }

      const sharedEntities = overlapCount(left.entities, right.entities);
      if (sharedEntities > 0) {
        weight += Math.min(3, sharedEntities);
        reasons.push(`shared entities (${sharedEntities})`);
      }

      if (left.taskType && right.taskType && left.taskType === right.taskType) {
        weight += 1;
        reasons.push("same task type");
      }

      if (left.sourcePath && right.sourcePath && left.sourcePath === right.sourcePath) {
        weight += 2;
        reasons.push("same source");
      }

      if (left.projectKey && right.projectKey && left.projectKey === right.projectKey) {
        weight += 1;
        reasons.push("same project key");
      }
      if (left.dossierId != null && right.dossierId != null && left.dossierId === right.dossierId) {
        weight += 1;
        reasons.push("same dossier");
      }

      if (weight >= 2) {
        edges.push({
          id: `${left.id}-${right.id}`,
          from: left.id,
          to: right.id,
          weight,
          reasons,
        });
      }
    }
  }

  edges.sort((a, b) => b.weight - a.weight);
  let trimmedEdges = edges.slice(0, 70);
  if (trimmedEdges.length === 0 && sampled.length > 1) {
    // Fallback so the panel stays useful when semantic overlap is sparse.
    trimmedEdges = sampled.slice(1).map((observation, idx) => ({
      id: `fallback-${sampled[idx].id}-${observation.id}`,
      from: sampled[idx].id,
      to: observation.id,
      weight: 1,
      reasons: ["temporal adjacency"],
    }));
  }

  const degreeMap = new Map<number, number>();
  for (const edge of trimmedEdges) {
    degreeMap.set(edge.from, (degreeMap.get(edge.from) ?? 0) + 1);
    degreeMap.set(edge.to, (degreeMap.get(edge.to) ?? 0) + 1);
  }

  const maxDegree = Math.max(1, ...Array.from(degreeMap.values()));
  const width = 860;
  const height = 520;
  const centerX = width / 2;
  const centerY = height / 2;
  const baseRadius = Math.min(width, height) * 0.44;

  const nodes: SimilarityNode[] = sampled.map((observation, idx) => {
    const degree = degreeMap.get(observation.id) ?? 0;
    const degreeRatio = degree / maxDegree;
    const angle = (Math.PI * 2 * idx) / Math.max(1, sampled.length) + hashJitter(String(observation.id)) * 0.65;
    const radius = baseRadius * (0.5 + (1 - degreeRatio) * 0.45);

    return {
      id: observation.id,
      title: observation.summary || observation.rawContent || `Observation ${observation.id}`,
      type: observation.type || "observation",
      confidence: clamp(observation.confidence || 0.5, 0.1, 1),
      stale: Boolean(observation.stale),
      degree,
      x: centerX + Math.cos(angle) * radius,
      y: centerY + Math.sin(angle) * radius,
    };
  });

  return { nodes, edges: trimmedEdges };
}

function isBlockingDecision(value: string) {
  const lowered = value.toLowerCase();
  return lowered.includes("blocked") || lowered.includes("deny") || lowered.includes("approval_required");
}

function overlapCount(left: string[] | null | undefined, right: string[] | null | undefined) {
  if (!Array.isArray(left) || !Array.isArray(right) || left.length === 0 || right.length === 0) return 0;
  const rightSet = new Set(right.map(normalizeToken).filter(Boolean));
  let count = 0;
  for (const token of left) {
    const normalized = normalizeToken(token);
    if (!normalized) continue;
    if (rightSet.has(normalized)) count += 1;
  }
  return count;
}

function normalizeToken(value: string) {
  return value.trim().toLowerCase();
}

function countBy<T>(items: T[], keyFn: (item: T) => string) {
  const result: Record<string, number> = {};
  for (const item of items) {
    const key = keyFn(item);
    result[key] = (result[key] ?? 0) + 1;
  }
  return result;
}

function hashJitter(value: string) {
  let hash = 0;
  for (let i = 0; i < value.length; i += 1) {
    hash = (hash << 5) - hash + value.charCodeAt(i);
    hash |= 0;
  }
  return Math.abs(hash % 1000) / 1000;
}

function clamp(value: number, min: number, max: number) {
  return Math.min(max, Math.max(min, value));
}

function truncateLabel(value: string, max: number) {
  const normalized = value.replace(/\s+/g, " ").trim();
  if (normalized.length <= max) return normalized;
  return `${normalized.slice(0, max - 1)}...`;
}

function Metric(props: { title: string; value: string; hint: string; onClick: () => void }) {
  return (
    <button
      type="button"
      onClick={props.onClick}
      className="rounded border border-white/10 bg-black/20 p-3 text-left hover:border-forge-ember/35"
    >
      <div className="text-[11px] uppercase tracking-wide text-forge-mist">{props.title}</div>
      <div className="mt-1 text-2xl font-semibold text-forge-ash">{props.value}</div>
      <div className="mt-1 text-[11px] text-forge-mist">{props.hint}</div>
    </button>
  );
}
