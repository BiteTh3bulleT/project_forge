import type { JobDetail, ReviewRecord } from "@forge/shared";
import { GhostButton, Panel, PrimaryButton } from "@forge/ui";
import type { ReactNode } from "react";
import { useEffect, useMemo, useState } from "react";
import { Link, useNavigate, useParams } from "react-router-dom";

import { HumanDataView } from "../components/HumanDataView";
import { api } from "../lib/api";
import { arrayOrEmpty } from "../lib/arrays";
import { formatTime } from "../lib/format";
import { useUiStore } from "../stores/uiStore";

type JobEvent = NonNullable<JobDetail["events"]>[number];
type JobArtifact = NonNullable<JobDetail["artifacts"]>[number];

function normalizeJobDetail(detail: JobDetail): JobDetail {
  return {
    ...detail,
    events: arrayOrEmpty<JobEvent>(detail.events),
    artifacts: arrayOrEmpty<JobArtifact>(detail.artifacts),
  };
}

const terminalStatuses = new Set(["succeeded", "failed", "cancelled"]);

export function JobDetailPage() {
  const { id } = useParams();
  const navigate = useNavigate();
  const setStatus = useUiStore((s) => s.setStatusLine);
  const [detail, setDetail] = useState<JobDetail | null>(null);
  const [reviews, setReviews] = useState<ReviewRecord[]>([]);
  const [alignmentNotes, setAlignmentNotes] = useState<
    Array<{
      id: number;
      note: string;
      retrievalResultId: number | null;
      observationId: number | null;
      createdAtMs: number;
    }>
  >([]);
  const [err, setErr] = useState<string | null>(null);

  async function refresh() {
    if (!id) return;
    try {
      const [d, r] = await Promise.all([
        api.jobs.detail(id, 0),
        api.reviews.list({ limit: 260 }),
      ]);
      setDetail(normalizeJobDetail(d));
      setReviews(Array.isArray(r?.reviews) ? r.reviews : []);
      if (d.packet?.id) {
        const notes = await api.memory.packetAlignment(d.packet.id, 120);
        setAlignmentNotes(
          arrayOrEmpty<{
            id: number;
            note: string;
            retrievalResultId: number | null;
            observationId: number | null;
            createdAtMs: number;
          }>(notes.notes),
        );
      } else {
        setAlignmentNotes([]);
      }
      setErr(null);
    } catch (e) {
      setErr(e instanceof Error ? e.message : String(e));
      setDetail(null);
      setReviews([]);
      setAlignmentNotes([]);
    }
  }

  useEffect(() => {
    void refresh();
    const timer = window.setInterval(() => void refresh(), 1400);
    return () => window.clearInterval(timer);
  }, [id]);

  const latestEvent = useMemo(() => {
    const evs = detail?.events;
    if (!evs || evs.length === 0) return null;
    return evs[evs.length - 1];
  }, [detail]);

  const jobReviews = useMemo(() => {
    if (!id) return [];
    return reviews.filter(
      (r) => r.targetId === id || (r.targetType === "job" && r.targetId === id),
    );
  }, [reviews, id]);

  if (!id) {
    return (
      <Panel title="Job" subtitle="Missing id">
        No job selected.
      </Panel>
    );
  }

  if (err) {
    return (
      <section className="forge-ops-board">
        <div className="forge-ops-panel border-forge-ember/30 bg-forge-ember/10">
          <div className="forge-ops-panel__head">
            <div>
              <div className="forge-ops-label">Run Detail</div>
              <div className="forge-ops-title">Could not load job</div>
            </div>
            <Link className="forge-btn forge-btn--ghost inline-flex" to="/jobs">
              Back to Jobs
            </Link>
          </div>
          <div className="forge-ops-panel__body text-sm text-forge-emberSoft">
            {err}
          </div>
        </div>
      </section>
    );
  }

  if (!detail) {
    return (
      <section className="forge-ops-board">
        <div className="forge-ops-panel">
          <div className="forge-ops-panel__head">
            <div>
              <div className="forge-ops-label">Run Detail</div>
              <div className="forge-ops-title">Loading job projection</div>
            </div>
          </div>
          <div className="forge-ops-panel__body text-sm text-forge-mist">
            Reading lifecycle projection and event truth stream.
          </div>
        </div>
      </section>
    );
  }

  const j = detail.job;
  const approvalPending = detail.approvalRequest?.status === "pending";
  const events = arrayOrEmpty<JobEvent>(detail.events);
  const artifacts = arrayOrEmpty<JobArtifact>(detail.artifacts);
  const stages = buildStages(events, j.status);
  const canCancel = !terminalStatuses.has(String(j.status));
  const latestSignal = latestEvent?.type ?? "no events recorded";

  return (
    <div className="forge-ops-board space-y-5">
      <header className="forge-ops-panel space-y-4 p-4">
        <div className="flex flex-col gap-3 lg:flex-row lg:items-start lg:justify-between">
          <div className="min-w-0">
            <div className="flex flex-wrap items-center gap-2 text-xs text-forge-mist/65">
              <Link className="hover:text-forge-ash" to="/jobs">
                Jobs
              </Link>
              <span>/</span>
              <span className="font-mono">{j.id}</span>
            </div>
            <div className="mt-2 flex flex-wrap items-start gap-3">
              <h1 className="min-w-0 max-w-4xl text-2xl font-semibold tracking-normal text-forge-ash sm:text-3xl">
                {j.title}
              </h1>
              <span className={statusPillClass(j.status)}>{j.status}</span>
            </div>
            <p className="mt-2 max-w-3xl text-sm leading-6 text-forge-mist/75">
              {j.requestedAction} through {j.targetAdapter} at{" "}
              {j.executionBoundary} boundary. Latest signal: {latestSignal}.
            </p>
          </div>
          <div className="flex w-full flex-col gap-2 sm:w-auto sm:flex-row sm:flex-wrap sm:items-center sm:justify-end">
            <GhostButton
              className="w-full sm:w-auto"
              onClick={() => navigate("/jobs")}
            >
              Back
            </GhostButton>
            <GhostButton
              className="w-full sm:w-auto"
              onClick={async () => {
                await api.jobs.cancel(j.id);
                setStatus(`Cancellation requested for ${j.id}.`);
                await refresh();
              }}
              disabled={!canCancel}
            >
              Cancel
            </GhostButton>
            <GhostButton
              className="w-full sm:w-auto"
              onClick={async () => {
                const res = await api.jobs.retry(j.id, {
                  note: "Retry from job detail",
                });
                setStatus(`Retry created: ${res.job.id}.`);
                navigate(`/jobs/${res.job.id}`);
              }}
            >
              Retry
            </GhostButton>
            <GhostButton
              className="w-full sm:w-auto"
              onClick={async () => {
                const res = await api.jobs.replay(j.id, {
                  note: "Replay from job detail",
                });
                setStatus(`Replay created: ${res.job.id}.`);
                navigate(`/jobs/${res.job.id}`);
              }}
            >
              Replay
            </GhostButton>
            <PrimaryButton
              className="w-full sm:w-auto"
              onClick={() => void refresh()}
            >
              Refresh
            </PrimaryButton>
          </div>
        </div>

        <div className="grid gap-2 border-t border-forge-platinum/10 pt-3 text-[11px] text-forge-mist/65 sm:grid-cols-3">
          <HeaderSignal label="Adapter" value={j.targetAdapter || "gateway"} />
          <HeaderSignal
            label="Boundary"
            value={j.executionBoundary || "bounded"}
          />
          <HeaderSignal label="Updated" value={formatTime(j.updatedAtMs)} />
        </div>

        <nav className="flex gap-1 overflow-x-auto border-t border-forge-platinum/10 pt-3 text-xs">
          {[
            ["overview", "Overview"],
            ["timeline", "Timeline"],
            ["evidence", "Evidence"],
            ["events", "Logs & Events"],
            ["packet", "Packet"],
          ].map(([target, label]) => (
            <button
              key={target}
              type="button"
              className="shrink-0 rounded border border-forge-platinum/10 bg-black/20 px-3 py-2 font-semibold text-forge-mist/70 transition hover:border-forge-ember/45 hover:bg-forge-ember/10 hover:text-forge-ash"
              onClick={() =>
                document
                  .getElementById(`job-${target}`)
                  ?.scrollIntoView({ behavior: "smooth", block: "start" })
              }
            >
              {label}
            </button>
          ))}
        </nav>
      </header>

      <section
        id="job-overview"
        className="grid gap-3 md:grid-cols-2 xl:grid-cols-4"
      >
        <MetricCard
          label="Approval"
          value={j.approvalStatus}
          detail={
            approvalPending
              ? "operator decision required"
              : "current gate state"
          }
          tone={approvalPending ? "warn" : j.approvalStatus}
        />
        <MetricCard
          label="Risk"
          value={j.riskClass}
          detail={`write intent ${String(j.writeIntent)}`}
          tone={j.riskClass}
        />
        <MetricCard
          label="Artifacts"
          value={String(artifacts.length)}
          detail={`${jobReviews.length} linked review(s)`}
          tone={artifacts.length > 0 ? "ok" : "muted"}
        />
        <MetricCard
          label="Events"
          value={String(events.length)}
          detail={
            latestEvent
              ? formatTime(latestEvent.createdAtMs)
              : "no truth stream yet"
          }
          tone={j.status}
        />
      </section>

      <section className="grid gap-4 xl:grid-cols-[minmax(0,1.35fr)_minmax(21rem,0.75fr)]">
        <div className="space-y-4">
          <div className="forge-ops-panel">
            <div className="forge-ops-panel__head">
              <div>
                <div className="forge-ops-label">Run Contract</div>
                <div className="forge-ops-title">
                  Execution boundary and route
                </div>
              </div>
              <div className="flex flex-wrap gap-2">
                <GhostButton
                  className="w-full sm:w-auto"
                  onClick={() =>
                    navigate(`/lineage?jobId=${encodeURIComponent(j.id)}`)
                  }
                >
                  Lineage
                </GhostButton>
                <GhostButton
                  className="w-full sm:w-auto"
                  onClick={() =>
                    navigate(`/audit?jobId=${encodeURIComponent(j.id)}`)
                  }
                >
                  Audit
                </GhostButton>
              </div>
            </div>
            <div className="forge-ops-panel__body">
              <SectionRows
                rows={[
                  ["Action", j.requestedAction],
                  ["Boundary", j.executionBoundary],
                  ["Adapter", j.targetAdapter],
                  ["Initiator", j.initiatingSource],
                  [
                    "Packet",
                    j.taskPacketId != null ? String(j.taskPacketId) : "none",
                  ],
                  ["Cancel requested", String(j.cancelRequested)],
                  ["Created", formatTime(j.createdAtMs)],
                  ["Updated", formatTime(j.updatedAtMs)],
                ]}
              />
              {j.resultSummary ? (
                <div className="mt-3 rounded-md border border-forge-platinum/10 bg-black/25 p-3 text-sm leading-6 text-forge-ash">
                  {j.resultSummary}
                </div>
              ) : null}
              {j.lastError || j.lastFailureCode ? (
                <div className="mt-3 rounded-md border border-forge-ember/30 bg-forge-ember/10 p-3 text-xs text-forge-ash">
                  <div className="font-semibold text-forge-emberSoft">
                    Failure evidence
                  </div>
                  <div className="mt-1">
                    Code:{" "}
                    {j.lastFailureCode != null
                      ? String(j.lastFailureCode)
                      : "none"}
                  </div>
                  <div className="mt-1">Error: {j.lastError ?? "none"}</div>
                </div>
              ) : null}
            </div>
          </div>

          <div id="job-timeline" className="forge-ops-panel">
            <div className="forge-ops-panel__head">
              <div>
                <div className="forge-ops-label">Stage Timeline</div>
                <div className="forge-ops-title">Lifecycle checkpoints</div>
              </div>
              <span className="text-xs text-forge-mist/60">
                {events.length} event(s)
              </span>
            </div>
            <div className="forge-ops-panel__body">
              <div className="grid gap-3 md:grid-cols-5">
                {stages.map((stage) => (
                  <div
                    key={stage.label}
                    className="rounded-md border border-forge-platinum/10 bg-black/20 p-3"
                  >
                    <span
                      className={[
                        "mb-3 block h-0.5 rounded-full",
                        metricAccentClass(stage.state),
                      ].join(" ")}
                    />
                    <div className="flex items-center justify-between gap-2">
                      <span className="flex items-center gap-2 text-xs font-semibold text-forge-ash">
                        <span
                          className={[
                            "h-1.5 w-1.5 rounded-full",
                            statusDotClass(stage.state),
                          ].join(" ")}
                        />
                        {stage.state}
                      </span>
                      <span className="text-[11px] text-forge-mist/50">
                        {stage.count}
                      </span>
                    </div>
                    <div className="mt-3 text-sm font-semibold text-forge-ash">
                      {stage.label}
                    </div>
                    <div className="mt-1 min-h-8 text-xs leading-5 text-forge-mist/65">
                      {stage.detail}
                    </div>
                  </div>
                ))}
              </div>
            </div>
          </div>
        </div>

        <aside className="space-y-4">
          {approvalPending && detail.approvalRequest ? (
            <div className="forge-ops-panel border-forge-amber/30 bg-forge-ember/10">
              <div className="forge-ops-panel__head">
                <div>
                  <div className="forge-ops-label">Approval Gate</div>
                  <div className="forge-ops-title">
                    Operator decision required
                  </div>
                </div>
                <span className={statusPillClass("pending")}>pending</span>
              </div>
              <div className="forge-ops-panel__body">
                <SectionRows
                  rows={[
                    ["Action", detail.approvalRequest.requestedAction],
                    ["Adapter", detail.approvalRequest.requestedAdapter],
                    ["Risk", detail.approvalRequest.riskClass],
                    [
                      "Write intent",
                      String(detail.approvalRequest.writeIntent),
                    ],
                  ]}
                />
                <div className="mt-3 max-h-44 overflow-auto rounded-md border border-forge-platinum/10 bg-black/30 p-3 text-[11px] text-forge-mist">
                  <HumanDataView
                    value={detail.approvalRequest.scopeSnapshot}
                    compact
                  />
                </div>
                <div className="mt-3 grid grid-cols-2 gap-2">
                  <PrimaryButton
                    className="py-3"
                    onClick={async () => {
                      await api.approvals.approve(
                        detail.approvalRequest!.id,
                        "Approved from job detail",
                      );
                      setStatus(
                        `Approval granted for request ${detail.approvalRequest!.id}.`,
                      );
                      await refresh();
                    }}
                  >
                    Approve
                  </PrimaryButton>
                  <GhostButton
                    className="py-3"
                    onClick={async () => {
                      await api.approvals.deny(
                        detail.approvalRequest!.id,
                        "Denied from job detail",
                      );
                      setStatus(
                        `Approval denied for request ${detail.approvalRequest!.id}.`,
                      );
                      await refresh();
                    }}
                  >
                    Deny
                  </GhostButton>
                </div>
              </div>
            </div>
          ) : null}

          <CompactPanel
            title="Reviews"
            subtitle={`${jobReviews.length} linked`}
          >
            {jobReviews.length === 0 ? (
              <EmptyState>No review records linked to this job.</EmptyState>
            ) : (
              <div className="space-y-2">
                {jobReviews.slice(0, 5).map((r) => (
                  <div
                    key={r.id}
                    className="rounded-md border border-forge-platinum/10 bg-black/20 p-3 text-xs"
                  >
                    <div className="flex items-center justify-between gap-2">
                      <span className="flex min-w-0 items-center gap-2 font-semibold text-forge-ash">
                        <span
                          className={[
                            "h-1.5 w-1.5 shrink-0 rounded-full",
                            statusDotClass(r.status),
                          ].join(" ")}
                        />
                        <span className="truncate">#{r.id}</span>
                      </span>
                      <span className={statusPillClass(r.status)}>
                        {r.status}
                      </span>
                    </div>
                    <div className="mt-2 line-clamp-2 text-forge-mist/75">
                      {r.summary || "(no summary)"}
                    </div>
                    <div className="mt-2 text-[11px] text-forge-mist/55">
                      reviewer {r.reviewer} | {formatTime(r.updatedAtMs)}
                    </div>
                  </div>
                ))}
              </div>
            )}
          </CompactPanel>
        </aside>
      </section>

      <section
        id="job-evidence"
        className="grid gap-4 xl:grid-cols-[minmax(0,1fr)_minmax(0,1fr)]"
      >
        <CompactPanel
          title="Artifacts"
          subtitle="Evidence produced during execution"
        >
          {artifacts.length === 0 ? (
            <EmptyState>No artifacts persisted for this job.</EmptyState>
          ) : (
            <div className="overflow-x-auto">
              <table className="forge-ops-table">
                <thead>
                  <tr>
                    <th>Artifact</th>
                    <th>Type</th>
                    <th>Created</th>
                    <th />
                  </tr>
                </thead>
                <tbody>
                  {artifacts.map((a) => (
                    <tr key={a.id}>
                      <td>
                        <div className="font-semibold text-forge-ash">
                          {a.title}
                        </div>
                        <div className="mt-0.5 max-w-sm truncate font-mono text-[11px] text-forge-mist/55">
                          {a.filePath}
                        </div>
                      </td>
                      <td>{a.type}</td>
                      <td>{formatTime(a.createdAtMs)}</td>
                      <td className="text-right">
                        <Link
                          className="text-forge-emberSoft hover:text-forge-ash"
                          to={`/workbench?artifactId=${encodeURIComponent(String(a.id))}&jobId=${encodeURIComponent(j.id)}`}
                        >
                          Open
                        </Link>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
        </CompactPanel>

        <CompactPanel
          title="Packet Alignment"
          subtitle={detail.packet ? `packet ${detail.packet.id}` : "no packet"}
        >
          {detail.packet ? (
            <div className="space-y-2">
              {alignmentNotes.length === 0 ? (
                <EmptyState>
                  No alignment notes stored for this packet.
                </EmptyState>
              ) : (
                alignmentNotes.slice(0, 12).map((n) => (
                  <div
                    key={n.id}
                    className="rounded-md border border-forge-platinum/10 bg-black/20 px-3 py-2.5 text-xs text-forge-mist/75"
                  >
                    <div className="flex items-center justify-between gap-2">
                      <span className="font-semibold text-forge-ash">
                        note #{n.id}
                      </span>
                      <span className="text-[11px] text-forge-mist/50">
                        {formatTime(n.createdAtMs)}
                      </span>
                    </div>
                    <div className="mt-1">{n.note}</div>
                    <div className="mt-1 text-[11px] text-forge-mist/50">
                      result {n.retrievalResultId ?? "n/a"} | observation{" "}
                      {n.observationId ?? "n/a"}
                    </div>
                  </div>
                ))
              )}
            </div>
          ) : (
            <EmptyState>No packet attached.</EmptyState>
          )}
        </CompactPanel>
      </section>

      <section id="job-events" className="forge-ops-panel">
        <div className="forge-ops-panel__head">
          <div>
            <div className="forge-ops-label">Logs & Events</div>
            <div className="forge-ops-title">
              Append-only event truth stream
            </div>
          </div>
          {latestEvent ? (
            <span className="text-xs text-forge-mist/60">
              latest {latestEvent.type}
            </span>
          ) : null}
        </div>
        <div className="forge-ops-panel__body">
          {events.length === 0 ? (
            <EmptyState>No events yet.</EmptyState>
          ) : (
            <div className="space-y-2">
              {events.map((ev) => (
                <EventRow key={ev.id} event={ev} />
              ))}
            </div>
          )}
        </div>
      </section>

      <section id="job-packet" className="forge-ops-panel">
        <div className="forge-ops-panel__head">
          <div>
            <div className="forge-ops-label">Packet Preview</div>
            <div className="forge-ops-title">Versioned execution contract</div>
          </div>
        </div>
        <div className="forge-ops-panel__body">
          {detail.packet ? (
            <div className="max-h-[520px] overflow-auto rounded-md border border-forge-platinum/10 bg-black/30 p-4 text-[11px] text-forge-mist">
              <HumanDataView value={detail.packet} />
            </div>
          ) : (
            <EmptyState>No packet attached.</EmptyState>
          )}
        </div>
      </section>
    </div>
  );
}

function MetricCard(props: {
  label: string;
  value: string;
  detail: string;
  tone: string;
}) {
  return (
    <div className="forge-ops-card min-h-[8rem] p-4">
      <span
        className={[
          "absolute inset-x-0 top-0 h-0.5",
          metricAccentClass(props.tone),
        ].join(" ")}
      />
      <div className="flex items-start justify-between gap-3">
        <div className="min-w-0">
          <div className="forge-ops-label">{props.label}</div>
          <div className="mt-2 truncate text-2xl font-semibold tracking-normal text-forge-ash">
            {props.value}
          </div>
        </div>
        <span className={statusPillClass(props.tone)}>{props.tone}</span>
      </div>
      <div className="mt-3 flex items-center gap-2 text-xs text-forge-mist/65">
        <span
          className={[
            "h-1.5 w-1.5 shrink-0 rounded-full",
            statusDotClass(props.tone),
          ].join(" ")}
        />
        <span className="min-w-0 truncate">{props.detail}</span>
      </div>
    </div>
  );
}

function HeaderSignal(props: { label: string; value: string }) {
  return (
    <div className="flex items-center justify-between gap-3 rounded border border-forge-platinum/10 bg-black/20 px-3 py-2">
      <span className="forge-ops-label">{props.label}</span>
      <span className="min-w-0 truncate font-mono text-forge-ash">
        {props.value}
      </span>
    </div>
  );
}

function CompactPanel(props: {
  title: string;
  subtitle: string;
  children: ReactNode;
}) {
  return (
    <div className="forge-ops-panel">
      <div className="forge-ops-panel__head">
        <div>
          <div className="forge-ops-title">{props.title}</div>
          <div className="mt-1 text-xs text-forge-mist/65">
            {props.subtitle}
          </div>
        </div>
      </div>
      <div className="forge-ops-panel__body">{props.children}</div>
    </div>
  );
}

function SectionRows(props: { rows: Array<[string, string]> }) {
  return (
    <div className="grid gap-2 md:grid-cols-2">
      {props.rows.map(([label, value]) => (
        <div
          key={label}
          className="flex min-h-10 items-center justify-between gap-3 rounded-md border border-forge-platinum/10 bg-black/20 px-3 py-2 text-xs"
        >
          <span className="text-forge-mist/65">{label}</span>
          <span className="min-w-0 truncate text-right font-semibold text-forge-ash">
            {value}
          </span>
        </div>
      ))}
    </div>
  );
}

function EventRow(props: { event: JobEvent }) {
  const ev = props.event;
  const tone = eventTone(ev);
  return (
    <div className="grid gap-3 rounded-md border border-forge-platinum/10 bg-black/20 p-3 md:grid-cols-[9rem_minmax(0,1fr)]">
      <div className="text-xs">
        <div className="font-mono text-forge-mist/55">#{ev.id}</div>
        <div className="mt-1 flex items-center gap-2 font-semibold text-forge-ash">
          <span
            className={["h-1.5 w-1.5 rounded-full", statusDotClass(tone)].join(
              " ",
            )}
          />
          <span className="min-w-0 truncate">{ev.type}</span>
        </div>
        <div className="mt-1 text-[11px] text-forge-mist/55">
          {formatTime(ev.createdAtMs)}
        </div>
      </div>
      <div className="min-w-0">
        <div className="text-sm text-forge-mist/85">{ev.message}</div>
        <details className="mt-2 group">
          <summary className="cursor-pointer text-[11px] font-semibold text-forge-emberSoft group-open:text-forge-ash">
            Payload
          </summary>
          <div className="mt-2 max-h-48 overflow-auto rounded-md border border-forge-platinum/10 bg-black/30 p-2 text-[11px] text-forge-mist">
            <HumanDataView value={ev.payload} compact />
          </div>
        </details>
      </div>
    </div>
  );
}

function EmptyState(props: { children: ReactNode }) {
  return (
    <div className="rounded-md border border-dashed border-forge-platinum/10 bg-black/15 p-3 text-sm text-forge-mist/65">
      {props.children}
    </div>
  );
}

function buildStages(events: JobEvent[], status: string) {
  const normalized = String(status || "").toLowerCase();
  const hasFailure =
    normalized.includes("fail") ||
    events.some(
      (ev) =>
        String(ev.type).toLowerCase().includes("fail") ||
        String(ev.message).toLowerCase().includes("fail"),
    );
  const hasApproval =
    normalized.includes("approval") ||
    events.some((ev) => String(ev.type).toLowerCase().includes("approval"));
  const hasRunning =
    normalized.includes("running") ||
    normalized.includes("succeed") ||
    normalized.includes("fail") ||
    events.some((ev) => /run|start|execute|dispatch/i.test(String(ev.type)));
  const hasComplete =
    normalized.includes("succeed") ||
    normalized.includes("cancel") ||
    hasFailure;
  return [
    {
      label: "Queued",
      state: "ok",
      count: eventCount(events, /queue|created|submit/i),
      detail: "Request accepted into the job projection.",
    },
    {
      label: "Prepared",
      state: hasRunning || hasComplete ? "ok" : "pending",
      count: eventCount(events, /prepare|packet|context/i),
      detail: "Packet, context, and execution scope assembled.",
    },
    {
      label: "Approval",
      state: hasApproval ? "warn" : "muted",
      count: eventCount(events, /approval/i),
      detail: hasApproval
        ? "Explicit operator gate is represented in the stream."
        : "No approval checkpoint recorded.",
    },
    {
      label: "Execution",
      state: hasFailure ? "bad" : hasRunning ? "warn" : "muted",
      count: eventCount(events, /run|start|execute|dispatch/i),
      detail: hasRunning
        ? "Worker execution has emitted runtime evidence."
        : "Execution has not emitted runtime evidence.",
    },
    {
      label: "Commit",
      state: hasFailure
        ? "bad"
        : normalized.includes("succeed")
          ? "ok"
          : hasComplete
            ? "warn"
            : "muted",
      count: eventCount(events, /success|complete|commit|fail|cancel/i),
      detail: hasComplete
        ? `Current terminal signal is ${status}.`
        : "Waiting for terminal event.",
    },
  ];
}

function eventCount(events: JobEvent[], pattern: RegExp) {
  return events.filter(
    (ev) => pattern.test(String(ev.type)) || pattern.test(String(ev.message)),
  ).length;
}

function statusPillClass(status: string) {
  const normalized = String(status || "").toLowerCase();
  if (
    [
      "ok",
      "success",
      "completed",
      "ready",
      "online",
      "verified",
      "enabled",
      "clear",
      "succeeded",
      "approved",
    ].some((item) => normalized.includes(item))
  )
    return "forge-ops-status forge-ops-status--ok";
  if (
    ["fail", "error", "blocked", "denied", "bad", "cancelled"].some((item) =>
      normalized.includes(item),
    )
  )
    return "forge-ops-status forge-ops-status--bad";
  if (
    [
      "warn",
      "pending",
      "running",
      "queued",
      "degraded",
      "active",
      "approval",
      "high",
    ].some((item) => normalized.includes(item))
  )
    return "forge-ops-status forge-ops-status--warn";
  return "forge-ops-status forge-ops-status--muted";
}

function statusDotClass(status: string) {
  const className = statusPillClass(status);
  if (className.includes("--ok"))
    return "bg-forge-electric shadow-[0_0_14px_rgba(85,214,255,0.45)]";
  if (className.includes("--bad"))
    return "bg-forge-emberSoft shadow-[0_0_14px_rgba(255,122,51,0.42)]";
  if (className.includes("--warn"))
    return "bg-forge-ember shadow-[0_0_14px_rgba(255,154,61,0.38)]";
  return "bg-forge-platinum/35";
}

function metricAccentClass(status: string) {
  const className = statusPillClass(status);
  if (className.includes("--ok")) return "bg-forge-electric/80";
  if (className.includes("--bad")) return "bg-forge-emberSoft/90";
  if (className.includes("--warn")) return "bg-forge-ember/90";
  return "bg-forge-platinum/20";
}

function eventTone(event: JobEvent) {
  const text = `${event.type} ${event.message}`.toLowerCase();
  if (/fail|error|denied|blocked|cancel/.test(text)) return "bad";
  if (/success|complete|commit|approved/.test(text)) return "ok";
  if (/approval|pending|queue|run|start|dispatch|warn/.test(text))
    return "warn";
  return "muted";
}
