import type { DashboardSummary, FailurePattern, ReviewRecord } from "@forge/shared";
import { GhostButton, Panel, PrimaryButton } from "@forge/ui";
import { useEffect, useState } from "react";
import { useNavigate } from "react-router-dom";

import { api } from "../lib/api";
import { formatTime } from "../lib/format";
import { useUiStore } from "../stores/uiStore";

export function DashboardPage() {
  const navigate = useNavigate();
  const setStatus = useUiStore((s) => s.setStatusLine);
  const uiMode = useUiStore((s) => s.uiMode);
  const [summary, setSummary] = useState<DashboardSummary | null>(null);
  const [patterns, setPatterns] = useState<FailurePattern[]>([]);
  const [pendingReviews, setPendingReviews] = useState<ReviewRecord[]>([]);
  const [err, setErr] = useState<string | null>(null);

  async function load() {
    try {
      const [dash, fail, rev] = await Promise.all([
        api.dashboard.summary(),
        api.failurePatterns.list({ limit: 20 }),
        api.reviews.list({ status: "pending", limit: 20 }),
      ]);
      setSummary(dash);
      setPatterns(fail.patterns);
      setPendingReviews(rev.reviews);
      setErr(null);
    } catch (e) {
      setErr(e instanceof Error ? e.message : String(e));
    }
  }

  useEffect(() => {
    void load();
    const id = window.setInterval(() => void load(), 4000);
    return () => window.clearInterval(id);
  }, []);

  return (
    <div className="space-y-6">
      <Panel
        title="Dashboard"
        subtitle={
          uiMode === "guided"
            ? "Guided summary: what needs attention right now."
            : "Command deck for active work, policy queues, routing advisories, and system health."
        }
        actions={<GhostButton onClick={() => void load()}>Refresh</GhostButton>}
      >
        {err ? <div className="rounded border border-forge-ember/30 bg-forge-ember/10 p-3 text-sm text-forge-ash">{err}</div> : null}
        {!summary ? (
          <div className="text-sm text-forge-mist">Loading dashboard telemetry…</div>
        ) : (
          <div className="grid gap-3 md:grid-cols-4">
            <Metric title="Active Jobs" value={String(summary.activeJobs.length)} hint="queued/preparing/running" onClick={() => navigate("/jobs")} />
            <Metric title="Approvals Pending" value={String(summary.approvalsPending)} hint="risk-gated requests" onClick={() => navigate("/approvals")} />
            <Metric title="Reviews Pending" value={String(summary.reviewsPending)} hint="review queue" onClick={() => navigate("/reviews")} />
            <Metric title="Recent Failures" value={String(summary.recentFailures.length)} hint="latest failed jobs" onClick={() => navigate("/jobs")} />
          </div>
        )}
      </Panel>

      {uiMode === "guided" ? (
        <Panel title="Next Moves" subtitle="Use Start for setup guidance or jump straight into queue management.">
          <div className="grid gap-2 md:grid-cols-2 xl:grid-cols-4">
            <PrimaryButton onClick={() => navigate("/start")}>Open Start</PrimaryButton>
            <GhostButton onClick={() => navigate("/jobs")}>Open Jobs</GhostButton>
            <GhostButton onClick={() => navigate("/approvals")}>Open Approvals</GhostButton>
            <GhostButton onClick={() => navigate("/reviews")}>Open Reviews</GhostButton>
          </div>
        </Panel>
      ) : null}

      <div className="grid gap-6 xl:grid-cols-2">
        <Panel title="Active Jobs" subtitle="Current execution pressure and lane occupancy.">
          {!summary || summary.activeJobs.length === 0 ? (
            <div className="text-sm text-forge-mist">No active jobs.</div>
          ) : (
            <div className="space-y-2">
              {summary.activeJobs.slice(0, 12).map((j) => (
                <button
                  key={j.id}
                  type="button"
                  onClick={() => navigate(`/jobs/${j.id}`)}
                  className="w-full rounded border border-white/10 bg-black/20 p-3 text-left hover:border-forge-ember/35"
                >
                  <div className="text-sm font-semibold text-forge-ash">{j.title}</div>
                  <div className="mt-1 text-xs text-forge-mist">
                    {j.id} · {j.status} · {j.targetAdapter}
                  </div>
                  <div className="mt-1 text-[11px] text-forge-mist">{formatTime(j.createdAtMs)}</div>
                </button>
              ))}
            </div>
          )}
        </Panel>

        <Panel title="Review Queue" subtitle="Pending reviews tied to imports or generated outputs.">
          <div className="mb-3 flex gap-2">
            <PrimaryButton onClick={() => navigate("/reviews")}>Open Review Queue</PrimaryButton>
          </div>
          {pendingReviews.length === 0 ? (
            <div className="text-sm text-forge-mist">No pending reviews.</div>
          ) : (
            <div className="space-y-2">
              {pendingReviews.slice(0, 10).map((r) => (
                <div key={r.id} className="rounded border border-white/10 bg-black/20 p-3 text-xs text-forge-mist">
                  <div className="font-semibold text-forge-ash">
                    #{r.id} · {r.targetType}:{r.targetId}
                  </div>
                  <div className="mt-1">{r.summary || "(no summary)"}</div>
                  <div className="mt-1">{formatTime(r.updatedAtMs)}</div>
                </div>
              ))}
            </div>
          )}
        </Panel>
      </div>

      <div className="grid gap-6 xl:grid-cols-2">
        <Panel title="Recent Imports" subtitle="External execution memory entering FORGE.">
          {!summary || summary.recentImports.length === 0 ? (
            <div className="text-sm text-forge-mist">No imported runs yet.</div>
          ) : (
            <div className="space-y-2">
              {summary.recentImports.map((i) => (
                <div key={i.id} className="rounded border border-white/10 bg-black/20 p-3 text-xs text-forge-mist">
                  <div className="font-semibold text-forge-ash">
                    import #{i.id} · {i.adapterId}
                  </div>
                  <div className="mt-1">{i.summary}</div>
                  <div className="mt-1">{formatTime(i.createdAtMs)}</div>
                </div>
              ))}
            </div>
          )}
        </Panel>

        <Panel title="Dossier Health" subtitle="Project-level stability and pending review pressure.">
          {!summary || summary.dossierHealth.length === 0 ? (
            <div className="text-sm text-forge-mist">No dossier health records yet.</div>
          ) : (
            <div className="space-y-2">
              {summary.dossierHealth.map((d, idx) => (
                <div key={`${d.dossierId ?? idx}`} className="rounded border border-white/10 bg-black/20 p-3 text-xs text-forge-mist">
                  <div className="font-semibold text-forge-ash">{String(d.name ?? `Dossier ${d.dossierId ?? "n/a"}`)}</div>
                  <div className="mt-1">
                    jobs {String(d.jobCount ?? 0)} · failures {String(d.failureCount ?? 0)} · reviews {String(d.reviewPending ?? 0)}
                  </div>
                  <div className="mt-1">health: {String(d.health ?? "unknown")}</div>
                </div>
              ))}
            </div>
          )}
        </Panel>
      </div>

      {uiMode === "guided" ? null : (
        <>
          <div className="grid gap-6 xl:grid-cols-2">
            <Panel
              title="Routing Recommendations"
              subtitle="Policy outputs with explicit reasons and confidence."
              actions={<GhostButton onClick={() => navigate("/policy")}>Open Policy</GhostButton>}
            >
              {!summary || summary.routingRecommendations.length === 0 ? (
                <div className="text-sm text-forge-mist">No recommendation records yet.</div>
              ) : (
                <div className="space-y-2">
                  {summary.routingRecommendations.map((r) => (
                    <div key={r.id} className="rounded border border-white/10 bg-black/20 p-3 text-xs text-forge-mist">
                      <div className="font-semibold text-forge-ash">
                        {r.taskType} → {r.adapter}
                      </div>
                      <div className="mt-1">confidence {(r.confidence * 100).toFixed(1)}%</div>
                      <div className="mt-1">reasons: {formatRoutingReasons(r.reasons)}</div>
                      <div className="mt-1">{formatTime(r.createdAtMs)}</div>
                    </div>
                  ))}
                </div>
              )}
            </Panel>

            <Panel
              title="Failure Patterns"
              subtitle="Repeated failure signatures by adapter/strategy/retrieval/packet style."
              actions={
                <PrimaryButton
                  onClick={async () => {
                    const res = await api.failurePatterns.analyze({ lookback: 200 });
                    setStatus(`Failure pattern analysis updated (${res.patterns.length} row(s)).`);
                    await load();
                  }}
                >
                  Recompute
                </PrimaryButton>
              }
            >
              {patterns.length === 0 ? (
                <div className="text-sm text-forge-mist">No failure snapshots recorded.</div>
              ) : (
                <div className="space-y-2">
                  {patterns.slice(0, 12).map((p) => (
                    <div key={p.id} className="rounded border border-white/10 bg-black/20 p-3 text-xs text-forge-mist">
                      <div className="font-semibold text-forge-ash">
                        {p.targetAdapter} · {p.retrievalMode} · {p.packetStyle}
                      </div>
                      <div className="mt-1">failure {p.failureCode} · count {p.failureCount}</div>
                      <div className="mt-1">{p.recommendation}</div>
                    </div>
                  ))}
                </div>
              )}
            </Panel>
          </div>

          <Panel title="System Status" subtitle="Current policy/automation/system counters.">
            {!summary ? (
              <div className="text-sm text-forge-mist">No status yet.</div>
            ) : (
              <pre className="max-h-56 overflow-auto rounded border border-white/10 bg-black/30 p-3 text-[11px] text-forge-mist">
                {JSON.stringify(summary.systemStatus, null, 2)}
              </pre>
            )}
          </Panel>
        </>
      )}
    </div>
  );
}

function formatRoutingReasons(reasons: unknown): string {
  if (Array.isArray(reasons)) return reasons.map((x) => String(x)).join(" | ");
  if (typeof reasons === "string") return reasons;
  if (reasons == null) return "";
  return JSON.stringify(reasons);
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
