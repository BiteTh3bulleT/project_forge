import { GhostButton, Panel, PrimaryButton } from "@forge/ui";
import { useEffect, useMemo, useState } from "react";
import { useSearchParams } from "react-router-dom";

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

export function AuditPage() {
  const [params, setParams] = useSearchParams();
  const jobId = useMemo(() => params.get("jobId") ?? "", [params]);
  const [category, setCategory] = useState("");
  const [outcome, setOutcome] = useState("");
  const [correlation, setCorrelation] = useState("");
  const [traceId, setTraceId] = useState("");
  const [records, setRecords] = useState<AuditRecord[]>([]);
  const [trace, setTrace] = useState<AuditRecord[]>([]);
  const [err, setErr] = useState<string | null>(null);

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
    } catch (e) {
      setErr(e instanceof Error ? e.message : String(e));
    }
  }

  useEffect(() => {
    void loadList();
  }, [jobId]);

  return (
    <div className="space-y-6">
      <Panel
        title="Audit & trace"
        subtitle="Append-only audit records with correlation ids. Use trace view to answer what happened, in order, for one logical operation."
        actions={<GhostButton onClick={() => void loadList()}>Refresh list</GhostButton>}
      >
        {jobId ? (
          <div className="mb-3 text-xs text-forge-mist">
            Filtered by job <span className="font-mono text-forge-ash">{jobId}</span>{" "}
            <button type="button" className="text-forge-emberSoft underline" onClick={() => setParams({})}>
                clear
              </button>
          </div>
        ) : null}
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
            onClick={async () => {
              if (!traceId.trim()) return;
              const r = await api.audit.trace(traceId.trim());
              setTrace(r.records as AuditRecord[]);
            }}
          >
            Load trace
          </PrimaryButton>
        </div>
        <div className="mt-4 space-y-2">
          {trace.map((rec) => (
            <div key={rec.id} className="rounded border border-white/10 bg-black/20 p-3 text-xs text-forge-mist">
              <div className="font-mono text-forge-ash">
                {rec.createdAtMs ? formatTime(rec.createdAtMs) : ""} · {rec.category}.{rec.action} · {rec.outcome}
              </div>
              <div className="mt-1">{rec.summary}</div>
              <pre className="mt-2 max-h-32 overflow-auto font-mono text-[10px]">{JSON.stringify(rec.payload, null, 2)}</pre>
            </div>
          ))}
        </div>
      </Panel>

      <Panel title="Recent records" subtitle="Newest-first list from the audit table.">
        <div className="space-y-2">
          {records.length === 0 ? <div className="text-sm text-forge-mist">No records (or core offline).</div> : null}
          {records.map((rec) => (
            <div key={rec.id} className="rounded border border-white/10 bg-forge-iron/30 p-3 text-xs text-forge-mist">
              <div className="flex flex-wrap justify-between gap-2">
                <span className="font-mono text-forge-ash">{rec.category}</span>
                <span>{formatTime(rec.createdAtMs)}</span>
              </div>
              <div className="mt-1">{rec.summary}</div>
              <div className="mt-1 text-[10px]">correlation: {rec.correlationId || "—"}</div>
            </div>
          ))}
        </div>
      </Panel>
    </div>
  );
}
