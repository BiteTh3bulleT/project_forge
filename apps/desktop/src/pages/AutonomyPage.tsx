import type { ForgeEvent } from "@forge/shared";
import { GhostButton, Panel } from "@forge/ui";
import { useEffect, useState } from "react";

import {
  api,
  type AutonomyBudgetRecord,
  type AutonomyCharterRecord,
  type AutonomyDecisionRecord,
  type AutonomyIntentRecord,
  type AutonomyStatusSnapshot,
} from "../lib/api";
import { formatTime } from "../lib/format";

function fmtTime(ms?: number) {
  if (!ms || ms <= 0) return "—";
  return formatTime(ms);
}

function shortText(value: unknown, max = 220) {
  const raw = typeof value === "string" ? value : JSON.stringify(value);
  if (!raw) return "";
  return raw.length > max ? `${raw.slice(0, max)}…` : raw;
}

export function AutonomyPage() {
  const [status, setStatus] = useState<AutonomyStatusSnapshot | null>(null);
  const [intents, setIntents] = useState<AutonomyIntentRecord[]>([]);
  const [decisions, setDecisions] = useState<AutonomyDecisionRecord[]>([]);
  const [budgets, setBudgets] = useState<AutonomyBudgetRecord[]>([]);
  const [charters, setCharters] = useState<AutonomyCharterRecord[]>([]);
  const [events, setEvents] = useState<ForgeEvent[]>([]);
  const [selectedIntentId, setSelectedIntentId] = useState<string>("");
  const [intentExplain, setIntentExplain] = useState<Record<string, unknown> | null>(null);
  const [loading, setLoading] = useState(false);
  const [err, setErr] = useState<string | null>(null);

  async function refresh() {
    setLoading(true);
    try {
      const [statusRes, intentsRes, decisionsRes, budgetsRes, chartersRes, eventsRes] = await Promise.all([
        api.autonomy.status(),
        api.autonomy.intents({ limit: 40 }),
        api.autonomy.decisions(40),
        api.autonomy.budgets(),
        api.autonomy.charters(false),
        api.autonomy.events(60),
      ]);
      setStatus(statusRes);
      setIntents(Array.isArray(intentsRes.intents) ? intentsRes.intents : []);
      setDecisions(Array.isArray(decisionsRes.decisions) ? decisionsRes.decisions : []);
      setBudgets(Array.isArray(budgetsRes.budgets) ? budgetsRes.budgets : []);
      setCharters(Array.isArray(chartersRes.charters) ? chartersRes.charters : []);
      setEvents(Array.isArray(eventsRes.events) ? eventsRes.events : []);
      setErr(null);
    } catch (e) {
      setErr(e instanceof Error ? e.message : String(e));
    } finally {
      setLoading(false);
    }
  }

  async function loadIntentExplain(intentId: string) {
    setSelectedIntentId(intentId);
    try {
      const payload = await api.autonomy.explainIntent(intentId);
      setIntentExplain(payload);
      setErr(null);
    } catch (e) {
      setIntentExplain(null);
      setErr(e instanceof Error ? e.message : String(e));
    }
  }

  useEffect(() => {
    void refresh();
    const id = window.setInterval(() => void refresh(), 5000);
    return () => window.clearInterval(id);
  }, []);

  return (
    <div className="space-y-6">
      <Panel
        title="Autonomy"
        subtitle="Track dream-state, maintenance/improvement activity, intents, decisions, budgets, and charters."
        actions={
          <GhostButton onClick={() => void refresh()} disabled={loading}>
            {loading ? "Refreshing…" : "Refresh"}
          </GhostButton>
        }
      >
        {err ? <div className="rounded-md border border-forge-ember/30 bg-forge-ember/10 p-3 text-sm text-forge-ash">{err}</div> : null}
        {!status?.available ? (
          <div className="rounded-md border border-yellow-500/30 bg-yellow-500/10 p-3 text-sm text-forge-ash">
            Autonomy loop is not active{status?.reason ? ` (${status.reason})` : ""}.
          </div>
        ) : (
          <div className="grid gap-3 md:grid-cols-2 xl:grid-cols-4">
            <MetricCard label="Mode" value={status.mode || "—"} />
            <MetricCard label="Workspace" value={status.scope?.workspaceId || "—"} />
            <MetricCard label="Dream State" value={status.dream?.active ? "active" : "idle/off"} />
            <MetricCard label="Last Tick" value={fmtTime(status.dream?.lastTickAt)} />
            <MetricCard label="Last Maintenance" value={fmtTime(status.dream?.lastMaintenanceAt)} />
            <MetricCard label="Last Improvement" value={fmtTime(status.dream?.lastImprovementAt)} />
            <MetricCard label="Active Intents" value={String(status.counts?.activeIntents ?? 0)} />
            <MetricCard label="Recent Decisions" value={String(status.counts?.recentDecisions ?? 0)} />
          </div>
        )}
      </Panel>

      <section className="grid gap-4 xl:grid-cols-2">
        <Panel title="Intents" subtitle="Self-initiated and agent-fed intent queue.">
          <div className="space-y-2">
            {intents.length === 0 ? (
              <div className="text-sm text-forge-mist">No intents recorded.</div>
            ) : (
              intents.map((intent) => (
                <button
                  key={intent.id}
                  type="button"
                  onClick={() => void loadIntentExplain(intent.id)}
                  className={[
                    "w-full rounded-lg border bg-forge-iron/40 p-3 text-left",
                    selectedIntentId === intent.id ? "border-forge-electric/70" : "border-white/10",
                  ].join(" ")}
                >
                  <div className="flex flex-wrap items-center justify-between gap-2">
                    <div className="text-sm font-semibold text-forge-ash">{intent.title || intent.id}</div>
                    <div className="text-[11px] text-forge-mist">{intent.status}</div>
                  </div>
                  <div className="mt-1 text-[11px] text-forge-mist">
                    {intent.type} · risk {intent.risk || "unknown"} · {fmtTime(intent.updatedAt)}
                  </div>
                  {intent.blockedReasons?.length ? (
                    <div className="mt-1 text-[11px] text-forge-emberSoft">{shortText(intent.blockedReasons[0])}</div>
                  ) : null}
                </button>
              ))
            )}
          </div>
        </Panel>

        <Panel title="Intent Explanation" subtitle="Structured explain output for selected intent.">
          {intentExplain ? (
            <pre className="max-h-96 overflow-auto whitespace-pre-wrap break-words rounded-lg border border-white/10 bg-black/30 p-3 font-mono text-[11px] leading-relaxed text-forge-mist">
              {JSON.stringify(intentExplain, null, 2)}
            </pre>
          ) : (
            <div className="text-sm text-forge-mist">Select an intent to inspect explain details.</div>
          )}
        </Panel>
      </section>

      <section className="grid gap-4 xl:grid-cols-2">
        <Panel title="Decisions" subtitle="Autonomy policy decisions with risk and outcome.">
          <div className="space-y-2">
            {decisions.length === 0 ? (
              <div className="text-sm text-forge-mist">No decisions recorded.</div>
            ) : (
              decisions.map((decision) => (
                <div key={decision.id} className="rounded-lg border border-white/10 bg-forge-iron/40 p-3">
                  <div className="flex flex-wrap items-center justify-between gap-2">
                    <div className="text-sm font-semibold text-forge-ash">{decision.decision}</div>
                    <div className="text-[11px] text-forge-mist">{fmtTime(decision.createdAt)}</div>
                  </div>
                  <div className="mt-1 text-[11px] text-forge-mist">
                    intent {decision.intentId} · risk {decision.risk || "unknown"} · level {decision.autonomyLevel || "—"}
                  </div>
                  {decision.deniedReasons?.length ? (
                    <div className="mt-1 text-[11px] text-forge-emberSoft">{shortText(decision.deniedReasons[0])}</div>
                  ) : null}
                </div>
              ))
            )}
          </div>
        </Panel>

        <Panel title="Autonomy Events" subtitle="Recent autonomy.* event stream.">
          <div className="space-y-2">
            {events.length === 0 ? (
              <div className="text-sm text-forge-mist">No autonomy events yet.</div>
            ) : (
              events.map((ev) => (
                <div key={ev.id} className="rounded-lg border border-white/10 bg-forge-iron/40 p-3">
                  <div className="flex flex-wrap items-center justify-between gap-2">
                    <div className="text-sm font-semibold text-forge-ash">{ev.type}</div>
                    <div className="text-[11px] text-forge-mist">{fmtTime(ev.createdAtMs)}</div>
                  </div>
                  <div className="mt-1 text-[11px] text-forge-mist">{shortText(ev.payload, 260)}</div>
                </div>
              ))
            )}
          </div>
        </Panel>
      </section>

      <section className="grid gap-4 xl:grid-cols-2">
        <Panel title="Budgets" subtitle="Freedom budget usage and reset windows.">
          <div className="space-y-2">
            {budgets.length === 0 ? (
              <div className="text-sm text-forge-mist">No budgets loaded.</div>
            ) : (
              budgets.map((budget) => (
                <div key={budget.id} className="rounded-lg border border-white/10 bg-forge-iron/40 p-3">
                  <div className="flex flex-wrap items-center justify-between gap-2">
                    <div className="text-sm font-semibold text-forge-ash">{budget.name}</div>
                    <div className="text-[11px] text-forge-mist">{budget.status}</div>
                  </div>
                  <div className="mt-1 text-[11px] text-forge-mist">{budget.id} · {budget.period} · resets {fmtTime(budget.resetsAt)}</div>
                </div>
              ))
            )}
          </div>
        </Panel>

        <Panel title="Charters" subtitle="Scope-bounded action authority definitions.">
          <div className="space-y-2">
            {charters.length === 0 ? (
              <div className="text-sm text-forge-mist">No charters loaded.</div>
            ) : (
              charters.map((charter) => (
                <div key={charter.id} className="rounded-lg border border-white/10 bg-forge-iron/40 p-3">
                  <div className="flex flex-wrap items-center justify-between gap-2">
                    <div className="text-sm font-semibold text-forge-ash">{charter.name}</div>
                    <div className="text-[11px] text-forge-mist">{charter.status}</div>
                  </div>
                  <div className="mt-1 text-[11px] text-forge-mist">{charter.id}</div>
                  {charter.purpose ? <div className="mt-1 text-[11px] text-forge-mist">{shortText(charter.purpose, 180)}</div> : null}
                </div>
              ))
            )}
          </div>
        </Panel>
      </section>
    </div>
  );
}

function MetricCard(props: { label: string; value: string }) {
  return (
    <div className="rounded-lg border border-white/10 bg-forge-iron/40 p-3">
      <div className="text-[10px] uppercase tracking-[0.14em] text-forge-mist">{props.label}</div>
      <div className="mt-1 text-sm font-semibold text-forge-ash">{props.value}</div>
    </div>
  );
}
