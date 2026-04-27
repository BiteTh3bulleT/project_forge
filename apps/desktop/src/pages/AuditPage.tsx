import { GhostButton, Panel, PrimaryButton } from "@forge/ui";
import { useEffect, useMemo, useState } from "react";
import { Link, useSearchParams } from "react-router-dom";

import { api } from "../lib/api";
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

function JsonBlock(props: { value: unknown; empty?: string; maxHeightClass?: string }) {
  const text = useMemo(() => {
    if (props.value == null) return "";
    try {
      return JSON.stringify(props.value, null, 2);
    } catch {
      return String(props.value);
    }
  }, [props.value]);

  if (!text || text === "{}" || text === "[]" || text === "null") {
    return <div className="text-xs text-forge-mist/75">{props.empty ?? "No recorded payload."}</div>;
  }

  return (
    <pre
      className={cx(
        "overflow-auto rounded border border-forge-platinum/10 bg-black/25 p-3 font-mono text-[11px] text-forge-mist",
        props.maxHeightClass ?? "max-h-[220px]",
      )}
    >
      {text}
    </pre>
  );
}

function CountPill(props: { label: string; value: string | number }) {
  return (
    <div className="rounded-full border border-forge-platinum/10 bg-forge-platinum/5 px-2.5 py-1 text-[11px] text-forge-mist">
      <span className="text-forge-mist/65">{props.label}</span> {props.value}
    </div>
  );
}

export function AuditPage() {
  const [params, setParams] = useSearchParams();
  const jobId = useMemo(() => params.get("jobId") ?? "", [params]);
  const [category, setCategory] = useState(() => params.get("category") ?? "");
  const [outcome, setOutcome] = useState(() => params.get("outcome") ?? "");
  const [correlation, setCorrelation] = useState(() => params.get("correlationId") ?? "");
  const [traceId, setTraceId] = useState(() => params.get("traceId") ?? "");
  const [records, setRecords] = useState<AuditRecord[]>([]);
  const [trace, setTrace] = useState<AuditRecord[]>([]);
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
      setRecords(r.records as AuditRecord[]);
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

  async function loadTrace(nextTraceId = traceId) {
    const id = nextTraceId.trim() || correlation.trim();
    if (!id) {
      setTraceErr("Provide a correlation id to load a trace.");
      setTrace([]);
      return;
    }
    try {
      const r = await api.audit.trace(id);
      setTrace(r.records as AuditRecord[]);
      setTraceErr(null);
      syncParams({ traceId: id, correlationId: correlation });
    } catch (e) {
      setTrace([]);
      setTraceErr(e instanceof Error ? e.message : String(e));
    }
  }

  useEffect(() => {
    void loadList();
  }, [jobId]);

  useEffect(() => {
    if (traceId.trim()) {
      void loadTrace(traceId);
    }
  }, []);

  const distinctCategories = useMemo(() => new Set(records.map((record) => record.category)).size, [records]);
  const distinctCorrelations = useMemo(() => new Set(records.map((record) => record.correlationId).filter(Boolean)).size, [records]);

  return (
    <div className="space-y-6">
      <Panel
        title="Audit & trace"
        subtitle="Append-only audit records with correlation ids. Use trace view to answer what happened, in order, for one logical operation."
        actions={
          <div className="flex flex-wrap gap-2">
            <Link
              to={correlation ? `/inspectors?correlationId=${encodeURIComponent(correlation)}` : "/inspectors"}
              className="rounded border border-forge-platinum/15 bg-forge-platinum/5 px-2.5 py-1 text-[11px] text-forge-mist transition hover:text-forge-ash"
            >
              Open inspectors
            </Link>
            <GhostButton onClick={() => void loadList()}>Refresh list</GhostButton>
          </div>
        }
      >
        {jobId ? (
          <div className="mb-3 text-xs text-forge-mist">
            Filtered by job <span className="font-mono text-forge-ash">{jobId}</span>{" "}
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
        <div className="mb-4 grid gap-3 lg:grid-cols-[minmax(0,1.2fr)_minmax(0,0.8fr)]">
          <div className="rounded-xl border border-forge-platinum/10 bg-black/20 p-3 text-sm text-forge-mist">
            Use the list to find a governed action, then pivot into a single correlation trace to reconstruct the exact sequence of persisted audit events.
          </div>
          <div className="rounded-xl border border-forge-platinum/10 bg-black/20 p-3">
            <div className="text-[10px] uppercase tracking-[0.14em] text-forge-mist/65">Current view</div>
            <div className="mt-2 flex flex-wrap gap-2">
              <CountPill label="Rows" value={records.length} />
              <CountPill label="Categories" value={distinctCategories} />
              <CountPill label="Correlations" value={distinctCorrelations} />
            </div>
          </div>
        </div>
        {err ? <div className="rounded-md border border-forge-ember/30 bg-forge-ember/10 p-3 text-sm text-forge-ash">{err}</div> : null}
        <div className="mt-3 grid gap-2 md:grid-cols-3">
          <input className="forge-input" placeholder="category" value={category} onChange={(e) => setCategory(e.target.value)} />
          <input className="forge-input" placeholder="outcome" value={outcome} onChange={(e) => setOutcome(e.target.value)} />
          <input className="forge-input" placeholder="correlation id" value={correlation} onChange={(e) => setCorrelation(e.target.value)} />
        </div>
        <div className="mt-2">
          <PrimaryButton onClick={() => void loadList()}>Apply filters</PrimaryButton>
        </div>
      </Panel>

      <Panel title="Correlation trace" subtitle="Loads oldest→newest for a single correlation id.">
        <div className="flex flex-wrap gap-2">
          <input className="forge-input min-w-[240px] flex-1" value={traceId} onChange={(e) => setTraceId(e.target.value)} placeholder="corr-…" />
          <PrimaryButton
            onClick={() => void loadTrace()}
          >
            Load trace
          </PrimaryButton>
        </div>
        {traceErr ? <div className="mt-3 rounded-md border border-forge-ember/30 bg-forge-ember/10 p-3 text-sm text-forge-ash">{traceErr}</div> : null}
        {trace.length > 0 ? (
          <div className="mt-3 flex flex-wrap gap-2">
            <CountPill label="Events" value={trace.length} />
            <CountPill label="Correlation" value={trace[0]?.correlationId || traceId || "—"} />
            {trace[0]?.jobId ? <CountPill label="Job" value={trace[0].jobId} /> : null}
            {trace[0]?.approvalRequestId ? <CountPill label="Approval" value={trace[0].approvalRequestId} /> : null}
          </div>
        ) : null}
        <div className="mt-4 space-y-2">
          {trace.length === 0 ? <div className="text-sm text-forge-mist">No trace loaded.</div> : null}
          {trace.map((rec) => (
            <div key={rec.id} className="rounded-xl border border-forge-platinum/10 bg-black/20 p-3 text-xs text-forge-mist">
              <div className="flex flex-wrap items-start justify-between gap-2">
                <div className="font-mono text-forge-ash">
                  {rec.createdAtMs ? formatTime(rec.createdAtMs) : ""} · {rec.category}.{rec.action}
                </div>
                <div className="flex flex-wrap gap-2">
                  <CountPill label="Outcome" value={rec.outcome} />
                  <CountPill label="Risk" value={rec.riskClass} />
                </div>
              </div>
              <div className="mt-2 text-sm text-forge-ash">{rec.summary}</div>
              <div className="mt-2 flex flex-wrap gap-2">
                {rec.gatewayInvocationId ? <CountPill label="Gateway" value={rec.gatewayInvocationId} /> : null}
                {rec.approvalRequestId ? <CountPill label="Approval" value={rec.approvalRequestId} /> : null}
                {rec.subjectType ? <CountPill label="Subject" value={`${rec.subjectType}:${rec.subjectId || "—"}`} /> : null}
              </div>
              <div className="mt-3">
                <JsonBlock value={rec.payload} empty="No payload recorded for this trace event." maxHeightClass="max-h-[180px]" />
              </div>
            </div>
          ))}
        </div>
      </Panel>

      <Panel title="Recent records" subtitle="Newest-first list from the audit table.">
        <div className="space-y-2">
          {records.length === 0 ? <div className="text-sm text-forge-mist">No records (or core offline).</div> : null}
          {records.map((rec) => (
            <div key={rec.id} className="rounded-xl border border-forge-platinum/10 bg-forge-iron/30 p-3 text-xs text-forge-mist">
              <div className="flex flex-wrap justify-between gap-2">
                <div className="flex flex-wrap items-center gap-2">
                  <span className="font-mono text-forge-ash">{rec.category}.{rec.action}</span>
                  <CountPill label="Outcome" value={rec.outcome} />
                  <CountPill label="Risk" value={rec.riskClass} />
                </div>
                <span>{formatTime(rec.createdAtMs)}</span>
              </div>
              <div className="mt-2 text-sm text-forge-ash">{rec.summary}</div>
              <div className="mt-2 flex flex-wrap gap-2">
                <CountPill label="Correlation" value={rec.correlationId || "—"} />
                {rec.jobId ? <CountPill label="Job" value={rec.jobId} /> : null}
                {rec.gatewayInvocationId ? <CountPill label="Gateway" value={rec.gatewayInvocationId} /> : null}
              </div>
              <div className="mt-3 flex flex-wrap gap-2 text-[11px]">
                {rec.correlationId ? (
                  <>
                    <button
                      type="button"
                      onClick={() => {
                        setTraceId(rec.correlationId);
                        void loadTrace(rec.correlationId);
                      }}
                      className="rounded-full border border-forge-platinum/15 bg-forge-platinum/5 px-2.5 py-1 text-forge-mist transition hover:text-forge-ash"
                    >
                      Load correlation trace
                    </button>
                    <Link
                      to={`/inspectors?correlationId=${encodeURIComponent(rec.correlationId)}`}
                      className="rounded-full border border-forge-platinum/15 bg-forge-platinum/5 px-2.5 py-1 text-forge-mist transition hover:text-forge-ash"
                    >
                      Inspect evidence
                    </Link>
                  </>
                ) : null}
              </div>
            </div>
          ))}
        </div>
      </Panel>
    </div>
  );
}
