import type { AdapterMetric, EvaluationRecord, JobRecord } from "@forge/shared";
import { GhostButton, PrimaryButton } from "@forge/ui";
import { useEffect, useState, type ReactNode } from "react";

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
    <div className="forge-ops-board space-y-5">
      <header className="flex flex-col gap-3 lg:flex-row lg:items-end lg:justify-between">
        <div>
          <div className="forge-ops-label">Outcome Review</div>
          <h1 className="mt-2 text-2xl font-semibold tracking-normal text-forge-ash sm:text-3xl">
            Evaluations board
          </h1>
          <p className="mt-1 max-w-3xl text-sm leading-6 text-forge-mist/75">
            Manual scoring feeds adapter comparison and routing insight without
            becoming automatic truth.
          </p>
        </div>
        <div className="flex flex-wrap items-center gap-2">
          <span className={statusPillClass(success ? "ok" : "warn")}>
            {success ? "success" : "needs review"}
          </span>
          <GhostButton onClick={() => void load()}>Refresh</GhostButton>
        </div>
      </header>

      {err ? (
        <div className="forge-ops-panel border-forge-ember/30 bg-forge-ember/10 p-3 text-sm text-forge-ash">
          {err}
        </div>
      ) : null}

      <section className="grid gap-3 sm:grid-cols-2 xl:grid-cols-4">
        <MetricTile
          label="Evaluations"
          value={String(evaluations.length)}
          detail="scoring records"
          tone="muted"
        />
        <MetricTile
          label="Adapters"
          value={String(metrics.length)}
          detail="metric groups"
          tone="ok"
        />
        <MetricTile
          label="Recent Jobs"
          value={String(jobs.length)}
          detail="selector pool"
          tone="muted"
        />
        <MetricTile
          label="Quality"
          value={quality}
          detail="draft rating"
          tone={Number(quality) >= 4 ? "ok" : "warn"}
        />
      </section>

      <OpsPanel
        title="Evaluation Intake"
        subtitle="Manual outcome scoring used for adapter comparison and routing insights."
      >
        <div className="forge-ops-card p-3">
          <div className="grid gap-3 md:grid-cols-3">
            <div>
              <label className="forge-ops-label">Job</label>
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
              <label className="mt-2 block text-[11px] text-forge-mist">
                Or enter job id
              </label>
              <input
                aria-label="Evaluation job id"
                className="forge-input mt-1 font-mono text-xs"
                value={jobId}
                onChange={(e) => setJobId(e.target.value)}
                placeholder="job_…"
              />
            </div>
            <div>
              <label className="text-xs font-semibold tracking-wide text-forge-mist">
                Dossier id (optional)
              </label>
              <input
                aria-label="Evaluation dossier id"
                className="forge-input mt-1"
                value={dossierId}
                onChange={(e) => setDossierId(e.target.value)}
              />
            </div>
            <div className="flex items-end gap-3">
              <label className="flex items-center gap-2 text-xs text-forge-mist">
                <input
                  aria-label="Evaluation success"
                  type="checkbox"
                  checked={success}
                  onChange={(e) => setSuccess(e.target.checked)}
                />
                Success
              </label>
              <label className="flex items-center gap-2 text-xs text-forge-mist">
                <input
                  aria-label="Retry recommended"
                  type="checkbox"
                  checked={retryRecommended}
                  onChange={(e) => setRetryRecommended(e.target.checked)}
                />
                Retry recommended
              </label>
              <label className="flex items-center gap-2 text-xs text-forge-mist">
                <input
                  aria-label="Influence routing"
                  type="checkbox"
                  checked={influenceRouting}
                  onChange={(e) => setInfluenceRouting(e.target.checked)}
                />
                Influence routing
              </label>
            </div>
          </div>
          <div className="mt-3 grid gap-3 md:grid-cols-5">
            <ScoreInput
              label="Quality"
              value={quality}
              onChange={setQuality}
              ariaLabel="Quality rating"
            />
            <ScoreInput
              label="Usefulness"
              value={usefulness}
              onChange={setUsefulness}
              ariaLabel="Usefulness rating"
            />
            <ScoreInput
              label="Correctness"
              value={correctness}
              onChange={setCorrectness}
              ariaLabel="Correctness confidence"
            />
            <ScoreInput
              label="Packet quality"
              value={packetQuality}
              onChange={setPacketQuality}
              ariaLabel="Packet quality rating"
            />
            <ScoreInput
              label="Adapter suitability"
              value={adapterSuitability}
              onChange={setAdapterSuitability}
              ariaLabel="Adapter suitability rating"
            />
          </div>
          <div className="mt-3">
            <label className="text-xs font-semibold tracking-wide text-forge-mist">
              Notes
            </label>
            <textarea
              aria-label="Evaluation notes"
              className="forge-input mt-1 min-h-[70px]"
              value={notes}
              onChange={(e) => setNotes(e.target.value)}
            />
          </div>
          <div className="mt-3 flex gap-2">
            <PrimaryButton
              className="w-full sm:w-auto"
              onClick={async () => {
                const d = dossierId.trim()
                  ? Number(dossierId.trim())
                  : undefined;
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
            <div className="mt-3 text-[11px] text-forge-mist">
              Recent jobs:{" "}
              {jobs
                .slice(0, 6)
                .map((j) => `${j.id} (${j.status})`)
                .join(" | ")}
            </div>
          ) : null}
        </div>
      </OpsPanel>

      <div className="grid gap-4 xl:grid-cols-2">
        <OpsPanel
          title="Adapter Metrics"
          subtitle="Comparison layer: success, quality, and retry tendencies."
        >
          {metrics.length === 0 ? (
            <EmptyState
              title="No adapter metrics yet"
              detail="Save evaluations with routing influence enabled to build adapter comparison data."
            />
          ) : (
            <div className="space-y-2">
              {metrics.map((m) => (
                <div
                  key={`${m.adapter}`}
                  className="forge-ops-card p-3 text-xs text-forge-mist"
                >
                  <div className="flex flex-wrap items-center justify-between gap-2">
                    <div className="font-semibold text-forge-ash">
                      {m.adapter}
                    </div>
                    <span
                      className={statusPillClass(
                        m.successRate >= 0.8 ? "ok" : "warn",
                      )}
                    >
                      {(m.successRate * 100).toFixed(1)}%
                    </span>
                  </div>
                  <div className="mt-2 grid gap-2 sm:grid-cols-3">
                    <MiniStat label="runs" value={String(m.runs)} />
                    <MiniStat
                      label="retry"
                      value={`${(m.retryRate * 100).toFixed(1)}%`}
                    />
                    <MiniStat label="quality" value={m.avgQuality.toFixed(2)} />
                  </div>
                  <div className="mt-2 text-[11px]">
                    usefulness {m.avgUsefulness.toFixed(2)} | suitability{" "}
                    {m.avgAdapterSuitability.toFixed(2)}
                  </div>
                </div>
              ))}
            </div>
          )}
        </OpsPanel>

        <OpsPanel
          title="Evaluation History"
          subtitle="Persistent scoring records used by insights and routing recommendations."
        >
          {evaluations.length === 0 ? (
            <EmptyState
              title="No evaluations recorded"
              detail="Choose a recent job, score the outcome, then save an operator evaluation."
            />
          ) : (
            <div className="space-y-2">
              {evaluations.map((e) => (
                <div
                  key={e.id}
                  className="forge-ops-card p-3 text-xs text-forge-mist"
                >
                  <div className="flex flex-wrap items-center justify-between gap-2">
                    <div className="font-semibold text-forge-ash">
                      #{e.id} - {e.jobId}
                    </div>
                    <span className={statusPillClass(e.success ? "ok" : "bad")}>
                      {e.success ? "success" : "failed"}
                    </span>
                  </div>
                  <div className="mt-1">
                    {formatTime(e.createdAtMs)} | success {String(e.success)} |
                    retry {String(e.retryRecommended)}
                  </div>
                  <div className="mt-1">
                    Q {e.qualityRating} | U {e.usefulnessRating} | C{" "}
                    {e.correctnessConfidence} | P {e.packetQualityRating} | A{" "}
                    {e.adapterSuitability}
                  </div>
                  <div className="mt-1">{e.notes || "(no notes)"}</div>
                </div>
              ))}
            </div>
          )}
        </OpsPanel>
      </div>
    </div>
  );
}

function OpsPanel(props: {
  title: string;
  subtitle: string;
  children: ReactNode;
}) {
  return (
    <section className="forge-ops-panel">
      <div className="forge-ops-panel__head">
        <div>
          <div className="forge-ops-title">{props.title}</div>
          <div className="mt-1 text-xs text-forge-mist/65">
            {props.subtitle}
          </div>
        </div>
      </div>
      <div className="forge-ops-panel__body">{props.children}</div>
    </section>
  );
}

function MetricTile(props: {
  label: string;
  value: string;
  detail: string;
  tone: string;
}) {
  return (
    <div className="forge-ops-card p-4">
      <div className="flex items-start justify-between gap-3">
        <div>
          <div className="forge-ops-label">{props.label}</div>
          <div className="mt-2 truncate text-2xl font-semibold tracking-normal text-forge-ash">
            {props.value}
          </div>
        </div>
        <span className={statusPillClass(props.tone)}>{props.tone}</span>
      </div>
      <div className="mt-3 text-xs text-forge-mist/65">{props.detail}</div>
    </div>
  );
}

function EmptyState(props: { title: string; detail: string }) {
  return (
    <div className="forge-ops-card border-dashed p-4 text-sm">
      <div className="font-semibold text-forge-ash">{props.title}</div>
      <div className="mt-1 text-xs leading-5 text-forge-mist/70">
        {props.detail}
      </div>
    </div>
  );
}

function MiniStat(props: { label: string; value: string }) {
  return (
    <div className="rounded border border-white/10 bg-black/25 px-2 py-1.5">
      <div className="forge-ops-label">{props.label}</div>
      <div className="mt-1 font-semibold text-forge-ash">{props.value}</div>
    </div>
  );
}

function ScoreInput(props: {
  label: string;
  value: string;
  onChange: (value: string) => void;
  ariaLabel: string;
}) {
  return (
    <div className="rounded border border-white/10 bg-black/20 p-2">
      <label className="forge-ops-label">{props.label}</label>
      <input
        aria-label={props.ariaLabel}
        className="forge-input mt-1 text-center font-mono"
        value={props.value}
        onChange={(e) => props.onChange(e.target.value)}
      />
    </div>
  );
}

function statusPillClass(status: string) {
  if (status === "ok" || status === "success") {
    return "forge-ops-status forge-ops-status--ok";
  }
  if (status === "bad" || status === "failed") {
    return "forge-ops-status forge-ops-status--bad";
  }
  if (status === "warn") {
    return "forge-ops-status forge-ops-status--warn";
  }
  return "forge-ops-status forge-ops-status--muted";
}
