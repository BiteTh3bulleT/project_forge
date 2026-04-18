import type { JobDetail, ReviewRecord } from "@forge/shared";
import { GhostButton, Panel, PrimaryButton } from "@forge/ui";
import { useEffect, useMemo, useState } from "react";
import { Link, useNavigate, useParams } from "react-router-dom";

import { api } from "../lib/api";
import { formatTime } from "../lib/format";
import { useUiStore } from "../stores/uiStore";

export function JobDetailPage() {
  const { id } = useParams();
  const navigate = useNavigate();
  const setStatus = useUiStore((s) => s.setStatusLine);
  const [detail, setDetail] = useState<JobDetail | null>(null);
  const [reviews, setReviews] = useState<ReviewRecord[]>([]);
  const [alignmentNotes, setAlignmentNotes] = useState<Array<{ id: number; note: string; retrievalResultId: number | null; observationId: number | null; createdAtMs: number }>>([]);
  const [err, setErr] = useState<string | null>(null);

  async function refresh() {
    if (!id) return;
    try {
      const [d, r] = await Promise.all([api.jobs.detail(id, 0), api.reviews.list({ limit: 260 })]);
      setDetail(d);
      setReviews(Array.isArray(r?.reviews) ? r.reviews : []);
      if (d.packet?.id) {
        const notes = await api.memory.packetAlignment(d.packet.id, 120);
        setAlignmentNotes(notes.notes);
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
    return reviews.filter((r) => r.targetId === id || (r.targetType === "job" && r.targetId === id));
  }, [reviews, id]);

  if (!id) {
    return <Panel title="Job" subtitle="Missing id">No job selected.</Panel>;
  }

  if (err) {
    return (
      <Panel title="Job Detail" subtitle="Could not load job.">
        <div className="text-sm text-forge-emberSoft">{err}</div>
        <div className="mt-4">
          <Link className="forge-btn forge-btn--ghost inline-flex" to="/jobs">
            Back to Jobs
          </Link>
        </div>
      </Panel>
    );
  }

  if (!detail) {
    return (
      <Panel title="Job Detail" subtitle="Loading…">
        <div className="text-sm text-forge-mist">Reading lifecycle projection and event truth stream.</div>
      </Panel>
    );
  }

  const j = detail.job;
  const approvalPending = detail.approvalRequest?.status === "pending";
  const events = detail.events ?? [];
  const artifacts = detail.artifacts ?? [];

  return (
    <div className="space-y-6">
      <Panel
        title={j.title}
        subtitle={`${j.id} · ${j.status} · ${j.targetAdapter} · ${j.riskClass}`}
        actions={
          <div className="flex gap-2">
            <GhostButton onClick={() => navigate("/jobs")}>Back</GhostButton>
            <GhostButton
              onClick={async () => {
                await api.jobs.cancel(j.id);
                setStatus(`Cancellation requested for ${j.id}.`);
                await refresh();
              }}
              disabled={j.status === "succeeded" || j.status === "failed" || j.status === "cancelled"}
            >
              Cancel Job
            </GhostButton>
            <GhostButton
              onClick={async () => {
                const res = await api.jobs.retry(j.id, { note: "Retry from job detail" });
                setStatus(`Retry created: ${res.job.id}.`);
                navigate(`/jobs/${res.job.id}`);
              }}
            >
              Retry
            </GhostButton>
            <GhostButton
              onClick={async () => {
                const res = await api.jobs.replay(j.id, { note: "Replay from job detail" });
                setStatus(`Replay created: ${res.job.id}.`);
                navigate(`/jobs/${res.job.id}`);
              }}
            >
              Replay
            </GhostButton>
            <GhostButton onClick={() => navigate(`/lineage?jobId=${encodeURIComponent(j.id)}`)}>
              Lineage
            </GhostButton>
            <GhostButton onClick={() => navigate(`/audit?jobId=${encodeURIComponent(j.id)}`)}>Audit</GhostButton>
            <GhostButton onClick={() => void refresh()}>Refresh</GhostButton>
          </div>
        }
      >
        <div className="grid gap-3 md:grid-cols-2">
          <div className="rounded border border-white/10 bg-black/20 p-3 text-xs text-forge-mist">
            <div>Action: {j.requestedAction}</div>
            <div>Boundary: {j.executionBoundary}</div>
            <div>Approval: {j.approvalStatus}</div>
            <div>Write intent: {String(j.writeIntent)}</div>
            <div>Created: {formatTime(j.createdAtMs)}</div>
            <div>Updated: {formatTime(j.updatedAtMs)}</div>
          </div>
          <div className="rounded border border-white/10 bg-black/20 p-3 text-xs text-forge-mist">
            <div>Packet: {j.taskPacketId ?? "none"}</div>
            <div>Initiator: {j.initiatingSource}</div>
            <div>Cancel requested: {String(j.cancelRequested)}</div>
            <div>Failure code: {j.lastFailureCode != null ? String(j.lastFailureCode) : "none"}</div>
            <div>Last error: {j.lastError ?? "none"}</div>
          </div>
        </div>
        {j.resultSummary ? <div className="mt-3 rounded border border-white/10 bg-black/20 p-3 text-sm text-forge-ash">{j.resultSummary}</div> : null}
      </Panel>

      {approvalPending && detail.approvalRequest ? (
        <Panel title="Approval Required" subtitle="Operator gate is open. Decide before this job can run.">
          <div className="space-y-2 text-sm text-forge-mist">
            <div>Requested action: {detail.approvalRequest.requestedAction}</div>
            <div>Adapter: {detail.approvalRequest.requestedAdapter}</div>
            <div>Risk class: {detail.approvalRequest.riskClass}</div>
            <div>Write intent: {String(detail.approvalRequest.writeIntent)}</div>
          </div>
          <pre className="mt-3 max-h-44 overflow-auto rounded border border-white/10 bg-black/30 p-3 text-[11px] text-forge-mist">
            {JSON.stringify(detail.approvalRequest.scopeSnapshot, null, 2)}
          </pre>
          <div className="mt-3 flex gap-2">
            <PrimaryButton
              onClick={async () => {
                await api.approvals.approve(detail.approvalRequest!.id, "Approved from job detail");
                setStatus(`Approval granted for request ${detail.approvalRequest!.id}.`);
                await refresh();
              }}
            >
              Approve
            </PrimaryButton>
            <GhostButton
              onClick={async () => {
                await api.approvals.deny(detail.approvalRequest!.id, "Denied from job detail");
                setStatus(`Approval denied for request ${detail.approvalRequest!.id}.`);
                await refresh();
              }}
            >
              Deny
            </GhostButton>
          </div>
        </Panel>
      ) : null}

      <Panel title="Packet Preview" subtitle="Versioned contract that this job executed against.">
        {detail.packet ? (
          <div className="space-y-3">
            <pre className="max-h-[520px] overflow-auto rounded border border-white/10 bg-black/30 p-4 text-[11px] text-forge-mist">
              {JSON.stringify(detail.packet, null, 2)}
            </pre>
            <div className="rounded border border-white/10 bg-black/20 p-3">
              <div className="text-xs font-semibold tracking-wide text-forge-mist">Packet Alignment Notes</div>
              {alignmentNotes.length === 0 ? (
                <div className="mt-2 text-xs text-forge-mist">No alignment notes stored for this packet.</div>
              ) : (
                <div className="mt-2 space-y-1 text-xs text-forge-mist">
                  {alignmentNotes.slice(0, 20).map((n) => (
                    <div key={n.id}>
                      #{n.id} · result {n.retrievalResultId ?? "n/a"} · observation {n.observationId ?? "n/a"} · {n.note}
                    </div>
                  ))}
                </div>
              )}
            </div>
          </div>
        ) : (
          <div className="text-sm text-forge-mist">No packet attached.</div>
        )}
      </Panel>

      <Panel title="Artifacts" subtitle="Evidence produced during execution.">
        {artifacts.length === 0 ? (
          <div className="text-sm text-forge-mist">No artifacts persisted for this job.</div>
        ) : (
          <div className="space-y-2">
            {artifacts.map((a) => (
              <div key={a.id} className="rounded border border-white/10 bg-black/20 p-3 text-xs text-forge-mist">
                <div className="font-semibold text-forge-ash">
                  {a.type} · {a.title}
                </div>
                <div className="mt-1 font-mono text-[11px]">{a.filePath}</div>
                <div className="mt-1">{formatTime(a.createdAtMs)}</div>
                <div className="mt-2">
                  <Link className="text-forge-emberSoft underline" to={`/workbench?artifactId=${encodeURIComponent(String(a.id))}&jobId=${encodeURIComponent(j.id)}`}>
                    Open in Workbench
                  </Link>
                </div>
              </div>
            ))}
          </div>
        )}
      </Panel>

      <Panel title="Reviews" subtitle="Review records linked to this job output.">
        {jobReviews.length === 0 ? (
          <div className="text-sm text-forge-mist">No review records linked to this job.</div>
        ) : (
          <div className="space-y-2">
            {jobReviews.map((r) => (
              <div key={r.id} className="rounded border border-white/10 bg-black/20 p-3 text-xs text-forge-mist">
                <div className="font-semibold text-forge-ash">
                  #{r.id} · {r.status}
                </div>
                <div className="mt-1">{r.summary || "(no summary)"}</div>
                <div className="mt-1">{r.notes || "(no notes)"}</div>
                <div className="mt-1">reviewer {r.reviewer} · {formatTime(r.updatedAtMs)}</div>
              </div>
            ))}
          </div>
        )}
      </Panel>

      <Panel title="Event Stream" subtitle="Append-only job events (truth stream).">
        {events.length === 0 ? (
          <div className="text-sm text-forge-mist">No events yet.</div>
        ) : (
          <div className="space-y-2">
            {events.map((ev) => (
              <div key={ev.id} className="rounded border border-white/10 bg-black/20 p-3">
                <div className="flex items-center justify-between gap-2">
                  <div className="text-sm font-semibold text-forge-ash">{ev.type}</div>
                  <div className="text-[11px] text-forge-mist">#{ev.id} · {formatTime(ev.createdAtMs)}</div>
                </div>
                <div className="mt-2 text-xs text-forge-mist">{ev.message}</div>
                <pre className="mt-2 max-h-48 overflow-auto whitespace-pre-wrap rounded border border-white/10 bg-black/30 p-2 text-[11px] text-forge-mist">
                  {JSON.stringify(ev.payload, null, 2)}
                </pre>
              </div>
            ))}
          </div>
        )}
        {latestEvent ? <div className="mt-3 text-[11px] text-forge-mist">Latest event: {latestEvent.type}</div> : null}
      </Panel>
    </div>
  );
}
