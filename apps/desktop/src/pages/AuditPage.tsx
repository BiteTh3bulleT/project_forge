import { GhostButton, PrimaryButton } from "@forge/ui";
import { useEffect, useMemo, useState } from "react";
import { Link, useSearchParams } from "react-router-dom";

import { HumanDataView } from "../components/HumanDataView";
import { api, type AuditTraceLookupResponse } from "../lib/api";
import { arrayOrEmpty } from "../lib/arrays";
import { formatTime } from "../lib/format";

type AuditRecord = {
  id: number;
  createdAtMs: number;
  correlationId: string;
  category: string;
  action: string;
  actor: string;
  subjectType: string;
  subjectId: string;
  jobId?: string;
  gatewayInvocationId?: number;
  approvalRequestId?: number;
  riskClass: string;
  outcome: string;
  summary: string;
  payload: unknown;
};

function cx(...parts: Array<string | false | null | undefined>) {
  return parts.filter(Boolean).join(" ");
}

function JsonBlock(props: {
  value: unknown;
  empty?: string;
  maxHeightClass?: string;
}) {
  if (
    props.value == null ||
    (Array.isArray(props.value) && props.value.length === 0) ||
    (typeof props.value === "object" &&
      props.value !== null &&
      Object.keys(props.value as Record<string, unknown>).length === 0)
  ) {
    return (
      <div className="text-xs text-forge-mist/75">
        {props.empty ?? "No recorded details."}
      </div>
    );
  }

  return (
    <div
      className={cx(
        "overflow-auto rounded border border-white/10 bg-black/25 p-3 text-[11px] text-forge-mist",
        props.maxHeightClass ?? "max-h-[220px]",
      )}
    >
      <HumanDataView value={props.value} compact />
    </div>
  );
}

function CountPill(props: { label: string; value: string | number }) {
  return (
    <div className="rounded-full border border-white/10 bg-white/5 px-2.5 py-1 text-[11px] text-forge-mist">
      <span className="text-forge-mist/65">{props.label}</span> {props.value}
    </div>
  );
}

function AuditMetric(props: {
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

function auditOutcomeClass(outcome: string) {
  const normalized = outcome.trim().toLowerCase();
  if (
    normalized === "success" ||
    normalized === "approved" ||
    normalized === "allowed"
  )
    return "forge-ops-status forge-ops-status--ok";
  if (
    normalized === "failed" ||
    normalized === "error" ||
    normalized === "denied" ||
    normalized === "blocked"
  )
    return "forge-ops-status forge-ops-status--bad";
  if (normalized === "pending" || normalized === "requested")
    return "forge-ops-status forge-ops-status--warn";
  return "forge-ops-status forge-ops-status--muted";
}

function recordFromUnknown(value: unknown): Record<string, unknown> {
  if (value && typeof value === "object" && !Array.isArray(value)) {
    return value as Record<string, unknown>;
  }
  return {};
}

function countReportArray(report: Record<string, unknown>, key: string) {
  const value = report[key];
  return Array.isArray(value) ? value.length : 0;
}

function countTraceLinks(report: Record<string, unknown>) {
  const links = recordFromUnknown(report.links);
  return Object.values(links).reduce<number>(
    (sum, value) => sum + (Array.isArray(value) ? value.length : 0),
    0,
  );
}

function pickTraceReport(response: AuditTraceLookupResponse) {
  const directReport = recordFromUnknown(response.report);
  if (Object.keys(directReport).length > 0) return directReport;
  const firstReport = response.reports?.[0]?.report;
  return recordFromUnknown(firstReport);
}

function recordsFromTraceLookup(response: AuditTraceLookupResponse) {
  if (Array.isArray(response.records)) {
    return arrayOrEmpty<AuditRecord>(response.records);
  }
  return arrayOrEmpty<AuditRecord>(
    response.reports?.flatMap((report) => report.records ?? []),
  );
}

function AuthorityChainPanel({ report }: { report: Record<string, unknown> }) {
  const hasReport = Object.keys(report).length > 0;
  if (!hasReport) return null;

  const linkCount = countTraceLinks(report);
  const links = recordFromUnknown(report.links);

  return (
    <section className="mt-4 rounded border border-white/10 bg-white/[0.03] p-3">
      <div className="flex flex-wrap items-start justify-between gap-3">
        <div>
          <div className="forge-ops-title">Authority Chain</div>
          <div className="mt-1 text-xs text-forge-mist/65">
            Read-only correlation graph across gateway, audit, artifact,
            provenance, and journal evidence.
          </div>
        </div>
        <span className="forge-ops-status forge-ops-status--muted">
          {linkCount} linked edges
        </span>
      </div>
      <div className="mt-3 grid gap-2 sm:grid-cols-2 xl:grid-cols-3">
        <CountPill
          label="Gateway Invocations"
          value={countReportArray(report, "gatewayInvocations")}
        />
        <CountPill
          label="Audit Records"
          value={countReportArray(report, "auditRecords")}
        />
        <CountPill
          label="Artifacts"
          value={countReportArray(report, "artifactRecords")}
        />
        <CountPill
          label="Provenance Records"
          value={countReportArray(report, "provenanceRecords")}
        />
        <CountPill
          label="Journal Events"
          value={countReportArray(report, "journalEvents")}
        />
        <CountPill
          label="Artifact Refs"
          value={countReportArray(report, "artifactRefs")}
        />
      </div>
      <div className="mt-3">
        <JsonBlock
          value={links}
          empty="No link edges recorded for this trace."
          maxHeightClass="max-h-[180px]"
        />
      </div>
    </section>
  );
}

export function AuditPage() {
  const [params, setParams] = useSearchParams();
  const jobId = useMemo(() => params.get("jobId") ?? "", [params]);
  const [category, setCategory] = useState(() => params.get("category") ?? "");
  const [outcome, setOutcome] = useState(() => params.get("outcome") ?? "");
  const [correlation, setCorrelation] = useState(
    () => params.get("correlationId") ?? "",
  );
  const [traceId, setTraceId] = useState(
    () => params.get("traceId") ?? params.get("correlationId") ?? "",
  );
  const [records, setRecords] = useState<AuditRecord[]>([]);
  const [trace, setTrace] = useState<AuditRecord[]>([]);
  const [traceReport, setTraceReport] = useState<Record<string, unknown>>({});
  const [err, setErr] = useState<string | null>(null);
  const [traceErr, setTraceErr] = useState<string | null>(null);

  function syncParams(next: Record<string, string>) {
    const merged = new URLSearchParams(params);
    for (const [key, value] of Object.entries(next)) {
      const trimmed = value.trim();
      if (!trimmed) merged.delete(key);
      else merged.set(key, trimmed);
    }
    setParams(merged, { replace: true });
  }

  async function loadList() {
    try {
      const r = await api.audit.list({
        limit: 120,
        category: category || undefined,
        outcome: outcome || undefined,
        correlationId: correlation || undefined,
        jobId: jobId || undefined,
      });
      setRecords(arrayOrEmpty<AuditRecord>(r.records));
      setErr(null);
      syncParams({
        category,
        outcome,
        correlationId: correlation,
      });
    } catch (e) {
      setErr(e instanceof Error ? e.message : String(e));
    }
  }

  async function loadTrace(
    nextTraceId = traceId,
    mode: "correlation" | "trace" = "correlation",
  ) {
    const id = nextTraceId.trim() || correlation.trim();
    if (!id) {
      setTraceErr("Provide a correlation id to load a trace.");
      setTrace([]);
      setTraceReport({});
      return;
    }
    try {
      const r = await api.audit.lookup(
        mode === "trace" ? { traceId: id } : { correlationId: id },
      );
      setTrace(recordsFromTraceLookup(r));
      setTraceReport(pickTraceReport(r));
      setTraceErr(null);
      syncParams({
        traceId: id,
        correlationId: mode === "correlation" ? id : correlation,
      });
    } catch (e) {
      setTrace([]);
      setTraceReport({});
      setTraceErr(e instanceof Error ? e.message : String(e));
    }
  }

  useEffect(() => {
    void loadList();
  }, [jobId]);

  useEffect(() => {
    const initialCorrelation = params.get("correlationId")?.trim() ?? "";
    const initialTrace = params.get("traceId")?.trim() ?? "";
    if (initialTrace && !initialCorrelation) {
      void loadTrace(initialTrace, "trace");
      return;
    }
    if (initialCorrelation) {
      void loadTrace(initialCorrelation, "correlation");
    }
  }, []);

  const distinctCategories = useMemo(
    () => new Set(records.map((record) => record.category)).size,
    [records],
  );
  const distinctCorrelations = useMemo(
    () =>
      new Set(records.map((record) => record.correlationId).filter(Boolean))
        .size,
    [records],
  );
  const failureCount = useMemo(
    () =>
      records.filter((record) =>
        ["failed", "error", "denied", "blocked"].includes(
          record.outcome.trim().toLowerCase(),
        ),
      ).length,
    [records],
  );

  return (
    <div className="forge-ops-board space-y-5">
      <header className="flex flex-col gap-3 lg:flex-row lg:items-end lg:justify-between">
        <div>
          <div className="forge-ops-label">Audit Trail</div>
          <h1 className="mt-2 text-2xl font-semibold tracking-normal text-forge-ash sm:text-3xl">
            Audit & trace
          </h1>
          <p className="mt-1 max-w-3xl text-sm leading-6 text-forge-mist/75">
            Append-only records with correlation pivots for reconstructing
            governed operations in sequence.
          </p>
        </div>
        <div className="flex flex-wrap items-center gap-2">
          <Link
            to={
              correlation
                ? `/inspectors?correlationId=${encodeURIComponent(correlation)}`
                : "/inspectors"
            }
            className="forge-btn forge-btn--ghost"
          >
            Open inspectors
          </Link>
          <GhostButton onClick={() => void loadList()}>
            Refresh list
          </GhostButton>
        </div>
      </header>

      <section className="grid gap-3 sm:grid-cols-2 xl:grid-cols-4">
        <AuditMetric
          label="Rows"
          value={records.length}
          detail="current list result"
          tone="muted"
        />
        <AuditMetric
          label="Categories"
          value={distinctCategories}
          detail="event families"
          tone="muted"
        />
        <AuditMetric
          label="Correlations"
          value={distinctCorrelations}
          detail="trace pivots"
          tone="ok"
        />
        <AuditMetric
          label="Failures"
          value={failureCount}
          detail="denied, blocked, or failed"
          tone={failureCount > 0 ? "bad" : "ok"}
        />
      </section>

      <section className="forge-ops-panel">
        <div className="forge-ops-panel__head">
          <div>
            <div className="forge-ops-title">Record Filters</div>
            <div className="mt-1 text-xs text-forge-mist/65">
              Find governed actions by category, outcome, job, or correlation
              id.
            </div>
          </div>
          <span className="font-mono text-[11px] text-forge-mist/60">
            limit 120
          </span>
        </div>
        <div className="forge-ops-panel__body">
          {jobId ? (
            <div className="mb-3 text-xs text-forge-mist">
              Filtered by job{" "}
              <span className="font-mono text-forge-ash">{jobId}</span>{" "}
              <button
                type="button"
                className="text-forge-emberSoft underline"
                onClick={() => {
                  setCategory("");
                  setOutcome("");
                  setCorrelation("");
                  setTraceId("");
                  setParams({});
                }}
              >
                clear
              </button>
            </div>
          ) : null}
          {err ? (
            <div className="rounded-md border border-forge-ember/30 bg-forge-ember/10 p-3 text-sm text-forge-ash">
              {err}
            </div>
          ) : null}
          <div className="mt-3 grid gap-2 md:grid-cols-3">
            <input
              className="forge-input"
              placeholder="category"
              value={category}
              onChange={(e) => setCategory(e.target.value)}
            />
            <input
              className="forge-input"
              placeholder="outcome"
              value={outcome}
              onChange={(e) => setOutcome(e.target.value)}
            />
            <input
              className="forge-input"
              placeholder="correlation id"
              value={correlation}
              onChange={(e) => setCorrelation(e.target.value)}
            />
          </div>
          <div className="mt-2">
            <PrimaryButton onClick={() => void loadList()}>
              Apply filters
            </PrimaryButton>
          </div>
        </div>
      </section>

      <section className="forge-ops-panel">
        <div className="forge-ops-panel__head">
          <div>
            <div className="forge-ops-title">Correlation Trace</div>
            <div className="mt-1 text-xs text-forge-mist/65">
              Loads oldest to newest for a single correlation id.
            </div>
          </div>
          {trace.length > 0 ? (
            <span className="forge-ops-status forge-ops-status--ok">
              {trace.length} events
            </span>
          ) : null}
        </div>
        <div className="forge-ops-panel__body">
          <div className="flex flex-wrap gap-2">
            <input
              className="forge-input min-w-[240px] flex-1"
              value={traceId}
              onChange={(e) => setTraceId(e.target.value)}
              placeholder="corr-…"
            />
            <PrimaryButton onClick={() => void loadTrace()}>
              Load trace
            </PrimaryButton>
          </div>
          {traceErr ? (
            <div className="mt-3 rounded-md border border-forge-ember/30 bg-forge-ember/10 p-3 text-sm text-forge-ash">
              {traceErr}
            </div>
          ) : null}
          {trace.length > 0 ? (
            <div className="mt-3 flex flex-wrap gap-2">
              <CountPill label="Events" value={trace.length} />
              <CountPill
                label="Correlation"
                value={trace[0]?.correlationId || traceId || "—"}
              />
              {trace[0]?.jobId ? (
                <CountPill label="Job" value={trace[0].jobId} />
              ) : null}
              {trace[0]?.approvalRequestId ? (
                <CountPill
                  label="Approval"
                  value={trace[0].approvalRequestId}
                />
              ) : null}
            </div>
          ) : null}
          <AuthorityChainPanel report={traceReport} />
          <div className="mt-4 space-y-2">
            {trace.length === 0 ? (
              <div className="text-sm text-forge-mist">No trace loaded.</div>
            ) : null}
            {trace.map((rec) => (
              <div
                key={rec.id}
                className="rounded border border-white/10 bg-black/20 p-3 text-xs text-forge-mist"
              >
                <div className="flex flex-wrap items-start justify-between gap-2">
                  <div className="font-mono text-forge-ash">
                    {rec.createdAtMs ? formatTime(rec.createdAtMs) : ""} ·{" "}
                    {rec.category}.{rec.action}
                  </div>
                  <div className="flex flex-wrap gap-2">
                    <span className={auditOutcomeClass(rec.outcome)}>
                      {rec.outcome || "unknown"}
                    </span>
                    <CountPill label="Risk" value={rec.riskClass} />
                  </div>
                </div>
                <div className="mt-2 text-sm text-forge-ash">{rec.summary}</div>
                <div className="mt-2 flex flex-wrap gap-2">
                  {rec.gatewayInvocationId ? (
                    <CountPill
                      label="Gateway"
                      value={rec.gatewayInvocationId}
                    />
                  ) : null}
                  {rec.approvalRequestId ? (
                    <CountPill label="Approval" value={rec.approvalRequestId} />
                  ) : null}
                  {rec.subjectType ? (
                    <CountPill
                      label="Subject"
                      value={`${rec.subjectType}:${rec.subjectId || "—"}`}
                    />
                  ) : null}
                </div>
                <div className="mt-3">
                  <JsonBlock
                    value={rec.payload}
                    empty="No details recorded for this trace event."
                    maxHeightClass="max-h-[180px]"
                  />
                </div>
              </div>
            ))}
          </div>
        </div>
      </section>

      <section className="forge-ops-panel">
        <div className="forge-ops-panel__head">
          <div>
            <div className="forge-ops-title">Recent Records</div>
            <div className="mt-1 text-xs text-forge-mist/65">
              Newest-first list from the audit table.
            </div>
          </div>
        </div>
        <div className="divide-y divide-white/10">
          {records.length === 0 ? (
            <div className="forge-ops-panel__body text-sm text-forge-mist">
              No records (or core offline).
            </div>
          ) : null}
          {records.map((rec) => (
            <article key={rec.id} className="p-4 text-xs text-forge-mist">
              <div className="flex flex-wrap justify-between gap-2">
                <div className="flex flex-wrap items-center gap-2">
                  <span className="font-mono text-forge-ash">
                    {rec.category}.{rec.action}
                  </span>
                  <span className={auditOutcomeClass(rec.outcome)}>
                    {rec.outcome || "unknown"}
                  </span>
                  <CountPill label="Risk" value={rec.riskClass} />
                </div>
                <span>{formatTime(rec.createdAtMs)}</span>
              </div>
              <div className="mt-2 text-sm text-forge-ash">{rec.summary}</div>
              <div className="mt-2 flex flex-wrap gap-2">
                <CountPill
                  label="Correlation"
                  value={rec.correlationId || "—"}
                />
                {rec.jobId ? <CountPill label="Job" value={rec.jobId} /> : null}
                {rec.gatewayInvocationId ? (
                  <CountPill label="Gateway" value={rec.gatewayInvocationId} />
                ) : null}
              </div>
              <div className="mt-3 flex flex-wrap gap-2 text-[11px]">
                {rec.correlationId ? (
                  <>
                    <button
                      type="button"
                      onClick={() => {
                        setCorrelation(rec.correlationId);
                        setTraceId(rec.correlationId);
                        void loadTrace(rec.correlationId, "correlation");
                      }}
                      className="rounded-full border border-white/15 bg-white/5 px-2.5 py-1 text-forge-mist transition hover:text-forge-ash"
                    >
                      Load correlation trace
                    </button>
                    <Link
                      to={`/inspectors?correlationId=${encodeURIComponent(rec.correlationId)}`}
                      className="rounded-full border border-white/15 bg-white/5 px-2.5 py-1 text-forge-mist transition hover:text-forge-ash"
                    >
                      Inspect evidence
                    </Link>
                  </>
                ) : null}
              </div>
            </article>
          ))}
        </div>
      </section>
    </div>
  );
}
