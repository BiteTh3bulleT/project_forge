import type { ApprovalRequest } from "@forge/shared";
import { GhostButton, PrimaryButton } from "@forge/ui";
import { useEffect, useRef, useState } from "react";
import { Link } from "react-router-dom";

import { HumanDataView } from "../components/HumanDataView";
import { api } from "../lib/api";
import { formatTime } from "../lib/format";
import { useUiStore } from "../stores/uiStore";

function approvalStatusClass(status: string) {
  if (status === "pending") return "forge-ops-status forge-ops-status--warn";
  if (status === "approved") return "forge-ops-status forge-ops-status--ok";
  if (status === "denied") return "forge-ops-status forge-ops-status--bad";
  return "forge-ops-status forge-ops-status--muted";
}

type ApprovalViewFilter = "pending" | "recent" | "denied";

function approvalListStatus(filter: ApprovalViewFilter) {
  return filter === "pending" ? "pending" : "resolved";
}

function approvalDecisionLabel(request: ApprovalRequest) {
  return request.decision?.decision ?? request.status;
}

function approvalVisibleInFilter(
  request: ApprovalRequest,
  filter: ApprovalViewFilter,
) {
  if (filter === "pending") return request.status === "pending";
  if (filter === "denied") return request.decision?.decision === "denied";
  return request.status !== "pending";
}

function riskClassName(risk: string) {
  const normalized = risk.trim().toLowerCase();
  if (normalized === "high" || normalized === "critical")
    return "forge-ops-status forge-ops-status--bad";
  if (normalized === "medium") return "forge-ops-status forge-ops-status--warn";
  if (normalized === "low") return "forge-ops-status forge-ops-status--ok";
  return "forge-ops-status forge-ops-status--muted";
}

function requiresNonPublicApproval(request: ApprovalRequest) {
  if (
    request.requestedAction.trim().toLowerCase() ===
    "gateway.capability.status.update"
  ) {
    return true;
  }

  const scope = request.scopeSnapshot;
  return (
    scope.publicDecisionAllowed === false ||
    scope.approvalPublicDecisionAllowed === false
  );
}

function ApprovalMetric(props: {
  label: string;
  value: string | number;
  detail: string;
  tone?: "ok" | "warn" | "bad" | "muted";
}) {
  const toneClass =
    props.tone === "ok"
      ? "text-emerald-300"
      : props.tone === "warn"
        ? "text-amber-300"
        : props.tone === "bad"
          ? "text-red-300"
          : "text-forge-ash";
  return (
    <div className="forge-ops-card p-4">
      <div className="forge-ops-label">{props.label}</div>
      <div
        className={`mt-2 text-3xl font-semibold tracking-normal ${toneClass}`}
      >
        {props.value}
      </div>
      <div className="mt-2 text-xs text-forge-mist/65">{props.detail}</div>
    </div>
  );
}

export function ApprovalsPage() {
  const setStatus = useUiStore((s) => s.setStatusLine);
  const [statusFilter, setStatusFilter] = useState<ApprovalViewFilter>(
    "pending",
  );
  const [rows, setRows] = useState<ApprovalRequest[]>([]);
  const [err, setErr] = useState<string | null>(null);
  const [decisionNotice, setDecisionNotice] = useState<string | null>(null);
  const [decisionBusyById, setDecisionBusyById] = useState<
    Record<number, boolean>
  >({});
  const decisionBusyRef = useRef<Set<number>>(new Set());

  async function refresh() {
    try {
      const res = await api.approvals.list(approvalListStatus(statusFilter), 120);
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

  async function decideApproval(
    requestId: number,
    decision: "approve" | "deny",
  ) {
    if (decisionBusyRef.current.has(requestId)) return;

    decisionBusyRef.current.add(requestId);
    setDecisionBusyById((prev) => ({ ...prev, [requestId]: true }));
    setDecisionNotice(null);
    setErr(null);

    try {
      if (decision === "approve") {
        await api.approvals.approve(
          requestId,
          "Approved from approvals queue",
        );
        setStatus(`Approved request ${requestId}.`);
      } else {
        await api.approvals.deny(requestId, "Denied from approvals queue");
        setStatus(`Denied request ${requestId}.`);
      }
      await refresh();
    } catch (e) {
      const message = e instanceof Error ? e.message : String(e);
      if (message.toLowerCase().includes("not pending")) {
        const notice = `Request ${requestId} was already resolved.`;
        setDecisionNotice(notice);
        setStatus(notice);
        await refresh();
      } else {
        setErr(message);
      }
    } finally {
      decisionBusyRef.current.delete(requestId);
      setDecisionBusyById((prev) => {
        const next = { ...prev };
        delete next[requestId];
        return next;
      });
    }
  }

  const visibleRows = rows.filter((row) =>
    approvalVisibleInFilter(row, statusFilter),
  );
  const pendingCount = visibleRows.filter(
    (row) => row.status === "pending",
  ).length;
  const resolvedCount = visibleRows.filter(
    (row) => row.status !== "pending",
  ).length;
  const deniedCount = visibleRows.filter(
    (row) => row.decision?.decision === "denied",
  ).length;
  const writeIntentCount = visibleRows.filter((row) => row.writeIntent).length;
  const filterLabel =
    statusFilter === "pending"
      ? "pending"
      : statusFilter === "denied"
        ? "denied"
        : "recent / resolved";

  return (
    <div className="forge-ops-board space-y-5">
      <header className="flex flex-col gap-3 lg:flex-row lg:items-end lg:justify-between">
        <div>
          <div className="forge-ops-label">Operator Gate</div>
          <h1 className="mt-2 text-2xl font-semibold tracking-normal text-forge-ash sm:text-3xl">
            Approvals queue
          </h1>
          <p className="mt-1 max-w-3xl text-sm leading-6 text-forge-mist/75">
            Separate request and decision records for governed actions, write
            intent, and adapter execution boundaries.
          </p>
        </div>
        <div className="flex flex-wrap items-center gap-2">
          <span
            className={
              pendingCount > 0
                ? "forge-ops-status forge-ops-status--warn"
                : "forge-ops-status forge-ops-status--ok"
            }
          >
            {pendingCount > 0 ? `${pendingCount} pending` : "Clear"}
          </span>
          <GhostButton onClick={() => void refresh()}>Refresh</GhostButton>
        </div>
      </header>

      <section className="grid gap-3 sm:grid-cols-2 xl:grid-cols-4">
        <ApprovalMetric
          label="Visible"
          value={visibleRows.length}
          detail={`${filterLabel} filter`}
          tone="muted"
        />
        <ApprovalMetric
          label="Pending"
          value={pendingCount}
          detail="awaiting operator"
          tone={pendingCount > 0 ? "warn" : "ok"}
        />
        <ApprovalMetric
          label="Recent / Resolved"
          value={resolvedCount}
          detail="completed decisions"
          tone={resolvedCount > 0 ? "ok" : "muted"}
        />
        <ApprovalMetric
          label="Denied"
          value={deniedCount}
          detail={`${writeIntentCount} write-intent in view`}
          tone={deniedCount > 0 ? "bad" : "ok"}
        />
      </section>

      <section className="forge-ops-panel">
        <div className="forge-ops-panel__head flex-col items-stretch sm:flex-row sm:items-center">
          <div>
            <div className="forge-ops-title">Gate Filters</div>
            <div className="mt-1 text-xs text-forge-mist/65">
              Pending requests allow one-click public decisions; recent and
              denied views show completed decision evidence.
            </div>
          </div>
          <div className="flex flex-wrap gap-2">
            <button
              type="button"
              aria-pressed={statusFilter === "pending"}
              className={
                statusFilter === "pending"
                  ? "forge-btn forge-btn--primary"
                  : "forge-btn forge-btn--ghost"
              }
              onClick={() => setStatusFilter("pending")}
            >
              Pending
            </button>
            <button
              type="button"
              aria-pressed={statusFilter === "recent"}
              className={
                statusFilter === "recent"
                  ? "forge-btn forge-btn--primary"
                  : "forge-btn forge-btn--ghost"
              }
              onClick={() => setStatusFilter("recent")}
            >
              Recent / Resolved
            </button>
            <button
              type="button"
              aria-pressed={statusFilter === "denied"}
              className={
                statusFilter === "denied"
                  ? "forge-btn forge-btn--primary"
                  : "forge-btn forge-btn--ghost"
              }
              onClick={() => setStatusFilter("denied")}
            >
              Denied
            </button>
          </div>
        </div>
        {err ? (
          <div
            className="m-4 rounded border border-forge-ember/30 bg-forge-ember/10 p-3 text-sm text-forge-ash"
            role="alert"
          >
            {err}
          </div>
        ) : null}
        {decisionNotice ? (
          <div
            className="m-4 rounded border border-amber-300/30 bg-amber-300/10 p-3 text-sm text-forge-ash"
            role="status"
            aria-live="polite"
          >
            {decisionNotice}
          </div>
        ) : null}
      </section>

      <section className="forge-ops-panel">
        <div className="forge-ops-panel__head">
          <div>
            <div className="forge-ops-title">Approval Queue</div>
            <div className="mt-1 text-xs text-forge-mist/65">
              Review scope snapshot, write intent, and requested adapter before
              deciding.
            </div>
          </div>
          <span className="font-mono text-[11px] text-forge-mist/60">
            limit 120
          </span>
        </div>
        {visibleRows.length === 0 ? (
          <div className="forge-ops-panel__body text-sm text-forge-mist">
            No approval records in this filter.
          </div>
        ) : (
          <div className="divide-y divide-white/10">
            {visibleRows.map((r) => {
              const nonPublicApproval = requiresNonPublicApproval(r);
              const decisionLabel = approvalDecisionLabel(r);
              return (
                <article
                  key={r.id}
                  aria-labelledby={`approval-request-${r.id}`}
                  className="grid gap-4 p-4 lg:grid-cols-[minmax(0,1fr)_18rem]"
                >
                <div className="min-w-0">
                  <div className="flex flex-wrap items-center justify-between gap-2">
                    <div className="min-w-0">
                      <div
                        id={`approval-request-${r.id}`}
                        className="truncate text-sm font-semibold text-forge-ash"
                      >
                        Request #{r.id}
                      </div>
                      <div className="mt-1 text-xs text-forge-mist/70">
                        Job{" "}
                        <Link
                          className="font-mono text-forge-emberSoft underline"
                          to={`/jobs/${r.jobId}`}
                        >
                          {r.jobId}
                        </Link>{" "}
                        · {formatTime(r.createdAtMs)}
                      </div>
                    </div>
                    <div className="flex flex-wrap gap-2">
                      <span className={approvalStatusClass(decisionLabel)}>
                        {decisionLabel}
                      </span>
                      {r.status === "resolved" && decisionLabel !== r.status ? (
                        <span className="forge-ops-status forge-ops-status--muted">
                          resolved
                        </span>
                      ) : null}
                      <span className={riskClassName(r.riskClass)}>
                        {r.riskClass || "risk unset"}
                      </span>
                    </div>
                  </div>
                  <div className="mt-3 text-sm text-forge-ash">
                    {r.requestSummary}
                  </div>
                  <div className="mt-3 grid gap-2 text-[11px] text-forge-mist sm:grid-cols-3">
                    <div className="rounded border border-white/10 bg-black/20 px-3 py-2">
                      <div className="forge-ops-label">Action</div>
                      <div className="mt-1 truncate text-forge-ash">
                        {r.requestedAction || "none"}
                      </div>
                    </div>
                    <div className="rounded border border-white/10 bg-black/20 px-3 py-2">
                      <div className="forge-ops-label">Adapter</div>
                      <div className="mt-1 truncate text-forge-ash">
                        {r.requestedAdapter || "gateway"}
                      </div>
                    </div>
                    <div className="rounded border border-white/10 bg-black/20 px-3 py-2">
                      <div className="forge-ops-label">Write Intent</div>
                      <div
                        className={
                          r.writeIntent
                            ? "mt-1 text-amber-300"
                            : "mt-1 text-forge-ash"
                        }
                      >
                        {r.writeIntent ? "requested" : "none"}
                      </div>
                    </div>
                  </div>

                  <div className="mt-3 max-h-44 overflow-auto rounded border border-white/10 bg-black/30 p-3 text-[11px] text-forge-mist">
                    <div className="forge-ops-label mb-2">Scope Snapshot</div>
                    <HumanDataView value={r.scopeSnapshot} compact />
                  </div>
                </div>

                <aside className="rounded border border-white/10 bg-black/20 p-3">
                  {r.status === "pending" ? (
                    <div className="grid gap-2">
                      {nonPublicApproval ? (
                        <>
                          <PrimaryButton
                            className="w-full"
                            disabled
                            aria-label={`Non-public approval request ${r.id}`}
                          >
                            Non-public approval
                          </PrimaryButton>
                          <div className="rounded border border-amber-300/25 bg-amber-300/10 p-2 text-xs leading-5 text-forge-mist">
                            This request requires a non-public approval
                            authority.
                          </div>
                        </>
                      ) : (
                        <>
                          <div className="rounded border border-emerald-300/25 bg-emerald-300/10 p-2 text-xs leading-5 text-forge-mist">
                            Public one-click decision allowed for this request.
                          </div>
                          <PrimaryButton
                            className="w-full"
                            disabled={decisionBusyById[r.id]}
                            aria-label={`Approve request ${r.id}`}
                            onClick={() => void decideApproval(r.id, "approve")}
                          >
                            Approve
                          </PrimaryButton>
                        </>
                      )}
                      <GhostButton
                        className="w-full"
                        disabled={decisionBusyById[r.id]}
                        aria-label={`Deny request ${r.id}`}
                        onClick={() => void decideApproval(r.id, "deny")}
                      >
                        Deny
                      </GhostButton>
                    </div>
                  ) : r.decision ? (
                    <div className="text-xs text-forge-mist">
                      <div className="forge-ops-label">Decision</div>
                      <div className="mt-2 text-sm font-semibold text-forge-ash">
                        {r.decision.decision}
                      </div>
                      <div className="mt-1">by {r.decision.actor}</div>
                      <div className="mt-1">
                        {formatTime(r.decision.createdAtMs)}
                      </div>
                      {r.decision.note ? (
                        <div className="mt-3 rounded border border-white/10 bg-black/25 p-2">
                          {r.decision.note}
                        </div>
                      ) : null}
                    </div>
                  ) : null}
                </aside>
                </article>
              );
            })}
          </div>
        )}
      </section>
    </div>
  );
}
