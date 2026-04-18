import type { AdapterMetric, EvaluationRecord, JobRecord } from "@forge/shared";
import { GhostButton, Panel, PrimaryButton } from "@forge/ui";
import { useEffect, useState } from "react";

import { api } from "../lib/api";
import { formatTime } from "../lib/format";
import { useUiStore } from "../stores/uiStore";

function intOr(v: string, fallback: number) {
  const n = Number(v);
  if (!Number.isFinite(n)) return fallback;
  const i = Math.round(n);
  return Math.min(5, Math.max(1, i));
}

export function EvaluationsPage() {
  const [evaluations, setEvaluations] = useState<EvaluationRecord[]>([]);
  const [metrics, setMetrics] = useState<AdapterMetric[]>([]);
  const [jobs, setJobs] = useState<JobRecord[]>([]);
  const [jobId, setJobId] = useState("");
  const [dossierId, setDossierId] = useState("");
  const [success, setSuccess] = useState(true);
  const [quality, setQuality] = useState("4");
  const [usefulness, setUsefulness] = useState("4");
  const [correctness, setCorrectness] = useState("4");
  const [packetQuality, setPacketQuality] = useState("4");
  const [adapterSuitability, setAdapterSuitability] = useState("4");
  const [retryRecommended, setRetryRecommended] = useState(false);
  const [influenceRouting, setInfluenceRouting] = useState(true);
  const [notes, setNotes] = useState("");
  const [err, setErr] = useState<string | null>(null);
  const setStatus = useUiStore((s) => s.setStatusLine);

  async function load() {
    try {
      const d = dossierId.trim() ? Number(dossierId.trim()) : undefined;
      const [ev, met, recent] = await Promise.all([
        api.evaluations.list(120, Number.isFinite(d) ? d : undefined),
        api.evaluations.metrics(Number.isFinite(d) ? d : undefined),
        api.jobs.list("", 60),
      ]);
      setEvaluations(ev.evaluations);
      setMetrics(met.metrics);
      setJobs(recent.jobs);
      if (!jobId && recent.jobs.length > 0) {
        setJobId(recent.jobs[0].id);
      }
      setErr(null);
    } catch (e) {
      setErr(e instanceof Error ? e.message : String(e));
    }
  }

  useEffect(() => {
    void load();
  }, []);

  return (
    <div className="space-y-6">
      <Panel title="Evaluations" subtitle="Manual outcome scoring used for adapter comparison and routing insights." actions={<GhostButton onClick={() => void load()}>Refresh</GhostButton>}>
        {err ? <div className="rounded border border-forge-ember/30 bg-forge-ember/10 p-3 text-sm text-forge-ash">{err}</div> : null}
        <div className="grid gap-3 md:grid-cols-3">
          <div>
            <label className="text-xs font-semibold tracking-wide text-forge-mist">Job</label>
            <select
              aria-label="Evaluation job selector"
              className="forge-input mt-1"
              value={jobs.some((j) => j.id === jobId) ? jobId : ""}
              onChange={(e) => {
                const v = e.target.value;
                if (v) setJobId(v);
              }}
            >
              <option value="">Select from recent jobs…</option>
              {jobs.map((j) => (
                <option key={j.id} value={j.id}>
                  {j.id} ({j.status})
                </option>
              ))}
            </select>
            <label className="mt-2 block text-[11px] text-forge-mist">Or enter job id</label>
            <input
              aria-label="Evaluation job id"
              className="forge-input mt-1 font-mono text-xs"
              value={jobId}
              onChange={(e) => setJobId(e.target.value)}
              placeholder="job_…"
            />
          </div>
          <div>
            <label className="text-xs font-semibold tracking-wide text-forge-mist">Dossier id (optional)</label>
            <input aria-label="Evaluation dossier id" className="forge-input mt-1" value={dossierId} onChange={(e) => setDossierId(e.target.value)} />
          </div>
          <div className="flex items-end gap-3">
            <label className="flex items-center gap-2 text-xs text-forge-mist">
              <input aria-label="Evaluation success" type="checkbox" checked={success} onChange={(e) => setSuccess(e.target.checked)} />
              Success
            </label>
            <label className="flex items-center gap-2 text-xs text-forge-mist">
              <input aria-label="Retry recommended" type="checkbox" checked={retryRecommended} onChange={(e) => setRetryRecommended(e.target.checked)} />
              Retry recommended
            </label>
            <label className="flex items-center gap-2 text-xs text-forge-mist">
              <input aria-label="Influence routing" type="checkbox" checked={influenceRouting} onChange={(e) => setInfluenceRouting(e.target.checked)} />
              Influence routing
            </label>
          </div>
        </div>
        <div className="mt-3 grid gap-3 md:grid-cols-5">
          <div>
            <label className="text-xs text-forge-mist">Quality</label>
            <input aria-label="Quality rating" className="forge-input mt-1" value={quality} onChange={(e) => setQuality(e.target.value)} />
          </div>
          <div>
            <label className="text-xs text-forge-mist">Usefulness</label>
            <input aria-label="Usefulness rating" className="forge-input mt-1" value={usefulness} onChange={(e) => setUsefulness(e.target.value)} />
          </div>
          <div>
            <label className="text-xs text-forge-mist">Correctness</label>
            <input aria-label="Correctness confidence" className="forge-input mt-1" value={correctness} onChange={(e) => setCorrectness(e.target.value)} />
          </div>
          <div>
            <label className="text-xs text-forge-mist">Packet quality</label>
            <input aria-label="Packet quality rating" className="forge-input mt-1" value={packetQuality} onChange={(e) => setPacketQuality(e.target.value)} />
          </div>
          <div>
            <label className="text-xs text-forge-mist">Adapter suitability</label>
            <input aria-label="Adapter suitability rating" className="forge-input mt-1" value={adapterSuitability} onChange={(e) => setAdapterSuitability(e.target.value)} />
          </div>
        </div>
        <div className="mt-3">
          <label className="text-xs font-semibold tracking-wide text-forge-mist">Notes</label>
          <textarea aria-label="Evaluation notes" className="forge-input mt-1 min-h-[70px]" value={notes} onChange={(e) => setNotes(e.target.value)} />
        </div>
        <div className="mt-3 flex gap-2">
          <PrimaryButton
            onClick={async () => {
              const d = dossierId.trim() ? Number(dossierId.trim()) : undefined;
              await api.evaluations.create({
                jobId,
                dossierId: Number.isFinite(d) ? d : undefined,
                success,
                qualityRating: intOr(quality, 4),
                usefulnessRating: intOr(usefulness, 4),
                correctnessConfidence: intOr(correctness, 4),
                packetQualityRating: intOr(packetQuality, 4),
                adapterSuitability: intOr(adapterSuitability, 4),
                retryRecommended,
                influenceRouting,
                notes,
                scorer: "operator",
              });
              setStatus(`Evaluation saved for ${jobId}.`);
              setNotes("");
              await load();
            }}
          >
            Save Evaluation
          </PrimaryButton>
        </div>
        {jobs.length > 0 ? (
          <div className="mt-3 text-[11px] text-forge-mist">Recent jobs: {jobs.slice(0, 6).map((j) => `${j.id} (${j.status})`).join(" | ")}</div>
        ) : null}
      </Panel>

      <div className="grid gap-6 xl:grid-cols-2">
        <Panel title="Adapter Metrics" subtitle="Comparison layer: success, quality, and retry tendencies.">
          {metrics.length === 0 ? (
            <div className="text-sm text-forge-mist">No metrics yet.</div>
          ) : (
            <div className="space-y-2">
              {metrics.map((m) => (
                <div key={`${m.adapter}`} className="rounded border border-white/10 bg-black/20 p-3 text-xs text-forge-mist">
                  <div className="font-semibold text-forge-ash">{m.adapter}</div>
                  <div className="mt-1">runs {m.runs} | success {(m.successRate * 100).toFixed(1)}% | retry {(m.retryRate * 100).toFixed(1)}%</div>
                  <div className="mt-1">quality {m.avgQuality.toFixed(2)} | usefulness {m.avgUsefulness.toFixed(2)} | suitability {m.avgAdapterSuitability.toFixed(2)}</div>
                </div>
              ))}
            </div>
          )}
        </Panel>

        <Panel title="Evaluation History" subtitle="Persistent scoring records used by insights and routing recommendations.">
          {evaluations.length === 0 ? (
            <div className="text-sm text-forge-mist">No evaluations recorded.</div>
          ) : (
            <div className="space-y-2">
              {evaluations.map((e) => (
                <div key={e.id} className="rounded border border-white/10 bg-black/20 p-3 text-xs text-forge-mist">
                  <div className="font-semibold text-forge-ash">#{e.id} - {e.jobId}</div>
                  <div className="mt-1">{formatTime(e.createdAtMs)} | success {String(e.success)} | retry {String(e.retryRecommended)}</div>
                  <div className="mt-1">Q {e.qualityRating} | U {e.usefulnessRating} | C {e.correctnessConfidence} | P {e.packetQualityRating} | A {e.adapterSuitability}</div>
                  <div className="mt-1">{e.notes || "(no notes)"}</div>
                </div>
              ))}
            </div>
          )}
        </Panel>
      </div>
    </div>
  );
}
