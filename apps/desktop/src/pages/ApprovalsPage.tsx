import type { ApprovalRequest } from "@forge/shared";
import { GhostButton, Panel, PrimaryButton } from "@forge/ui";
import { useEffect, useState } from "react";
import { Link } from "react-router-dom";

import { api } from "../lib/api";
import { formatTime } from "../lib/format";
import { useUiStore } from "../stores/uiStore";

export function ApprovalsPage() {
  const setStatus = useUiStore((s) => s.setStatusLine);
  const [statusFilter, setStatusFilter] = useState<"pending" | "resolved">("pending");
  const [rows, setRows] = useState<ApprovalRequest[]>([]);
  const [err, setErr] = useState<string | null>(null);

  async function refresh() {
    try {
      const res = await api.approvals.list(statusFilter, 120);
      setRows(res.approvals);
      setErr(null);
    } catch (e) {
      setErr(e instanceof Error ? e.message : String(e));
    }
  }

  useEffect(() => {
    void refresh();
    const id = window.setInterval(() => void refresh(), 2000);
    return () => window.clearInterval(id);
  }, [statusFilter]);

  return (
    <div className="space-y-6">
      <Panel
        title="Approvals"
        subtitle="Operator gate for medium/high-risk actions. Requests and decisions are separate records."
        actions={<GhostButton onClick={() => void refresh()}>Refresh</GhostButton>}
      >
        <div className="flex gap-2">
          <button
            type="button"
            className={statusFilter === "pending" ? "forge-btn forge-btn--primary" : "forge-btn forge-btn--ghost"}
            onClick={() => setStatusFilter("pending")}
          >
            Pending
          </button>
          <button
            type="button"
            className={statusFilter === "resolved" ? "forge-btn forge-btn--primary" : "forge-btn forge-btn--ghost"}
            onClick={() => setStatusFilter("resolved")}
          >
            Resolved
          </button>
        </div>
        {err ? <div className="mt-4 rounded border border-forge-ember/30 bg-forge-ember/10 p-3 text-sm text-forge-ash">{err}</div> : null}
      </Panel>

      <Panel title="Approval Queue" subtitle="Review scope snapshot, write intent, and requested adapter before deciding.">
        {rows.length === 0 ? (
          <div className="text-sm text-forge-mist">No approval records in this filter.</div>
        ) : (
          <div className="space-y-3">
            {rows.map((r) => (
              <div key={r.id} className="rounded border border-forge-platinum/10 bg-black/20 p-4">
                <div className="flex flex-wrap items-center justify-between gap-2">
                  <div className="text-sm font-semibold text-forge-ash">Request #{r.id}</div>
                  <div className="text-[11px] text-forge-mist">{r.status} · {formatTime(r.createdAtMs)}</div>
                </div>
                <div className="mt-2 text-xs text-forge-mist">
                  Job: <Link className="text-forge-ash underline" to={`/jobs/${r.jobId}`}>{r.jobId}</Link> · {r.requestedAction} · {r.requestedAdapter}
                </div>
                <div className="mt-1 text-xs text-forge-mist">Risk: {r.riskClass} · Write intent: {String(r.writeIntent)}</div>
                <div className="mt-1 text-xs text-forge-mist">Summary: {r.requestSummary}</div>

                <pre className="mt-3 max-h-40 overflow-auto rounded border border-forge-platinum/10 bg-black/30 p-2 text-[11px] text-forge-mist">
                  {JSON.stringify(r.scopeSnapshot, null, 2)}
                </pre>

                {r.status === "pending" ? (
                  <div className="mt-3 flex gap-2">
                    <PrimaryButton
                      onClick={async () => {
                        await api.approvals.approve(r.id, "Approved from approvals queue");
                        setStatus(`Approved request ${r.id}.`);
                        await refresh();
                      }}
                    >
                      Approve
                    </PrimaryButton>
                    <GhostButton
                      onClick={async () => {
                        await api.approvals.deny(r.id, "Denied from approvals queue");
                        setStatus(`Denied request ${r.id}.`);
                        await refresh();
                      }}
                    >
                      Deny
                    </GhostButton>
                  </div>
                ) : r.decision ? (
                  <div className="mt-3 text-xs text-forge-mist">
                    Decision: {r.decision.decision} by {r.decision.actor} at {formatTime(r.decision.createdAtMs)}
                    {r.decision.note ? ` · ${r.decision.note}` : ""}
                  </div>
                ) : null}
              </div>
            ))}
          </div>
        )}
      </Panel>
    </div>
  );
}
