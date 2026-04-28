import type { SourceRow } from "@forge/shared";
import { GhostButton, Panel, PrimaryButton } from "@forge/ui";
import { useEffect, useMemo, useState } from "react";
import { useNavigate } from "react-router-dom";

import { api } from "../lib/api";
import { formatTime } from "../lib/format";
import { useUiStore } from "../stores/uiStore";

export function SourcesPage() {
  const navigate = useNavigate();
  const [sources, setSources] = useState<SourceRow[]>([]);
  const [path, setPath] = useState("");
  const [workspaceDir, setWorkspaceDir] = useState("");
  const [err, setErr] = useState<string | null>(null);
  const [adding, setAdding] = useState(false);
  const [reindexAllBusy, setReindexAllBusy] = useState(false);
  const [sourceBusyId, setSourceBusyId] = useState<number | null>(null);
  const setStatus = useUiStore((s) => s.setStatusLine);

  const trimmedPath = path.trim();
  const canAdd = trimmedPath.length > 0;

  async function refresh() {
    try {
      const [res, meta] = await Promise.all([api.sources.list(), api.meta()]);
      setSources(Array.isArray(res?.sources) ? res.sources : []);
      setWorkspaceDir(meta?.workspaceDir ?? "");
      setErr(null);
    } catch (e) {
      setErr(e instanceof Error ? e.message : String(e));
    }
  }

  useEffect(() => {
    void refresh();
    const id = window.setInterval(() => void refresh(), 3000);
    return () => window.clearInterval(id);
  }, []);

  const stats = useMemo(() => {
    let errors = 0;
    let indexing = 0;
    let ready = 0;
    let neverScanned = 0;
    for (const s of sources) {
      const state = sourceState(s);
      if (state === "error") errors++;
      if (state === "indexing") indexing++;
      if (state === "ready") ready++;
      if (state === "new") neverScanned++;
    }
    return { total: sources.length, errors, indexing, ready, neverScanned };
  }, [sources]);

  async function addSource() {
    if (!canAdd || adding) return;
    setAdding(true);
    try {
      const res = await api.sources.add(trimmedPath);
      setPath("");
      setStatus(`Source added: ${res.path}`);
      await refresh();
    } catch (e) {
      const msg = e instanceof Error ? e.message : String(e);
      setErr(msg);
      setStatus("Failed to add source.");
    } finally {
      setAdding(false);
    }
  }

  async function queueReindexAll() {
    if (reindexAllBusy) return;
    setReindexAllBusy(true);
    try {
      const res = await api.commands.execute("reindex", { via: "sources_page" });
      setStatus(res.jobId ? `Re-index all job queued: ${res.jobId}.` : "Re-index all submitted.");
      if (res.jobId) navigate(`/jobs/${res.jobId}`);
      await refresh();
    } catch (e) {
      setErr(e instanceof Error ? e.message : String(e));
    } finally {
      setReindexAllBusy(false);
    }
  }

  async function queueReindexSource(s: SourceRow) {
    setSourceBusyId(s.id);
    try {
      const res = await api.jobs.create({
        templateId: "reindex_sources",
        title: `Re-index source ${s.id}`,
        userRequest: `Re-index source ${s.path}`,
        objective: "Refresh indexed memory from one source folder.",
        initiatingSource: "sources_page",
        requestPayload: { sourceId: s.id, sourcePath: s.path },
      });
      setStatus(`Re-index job queued: ${res.job.id}.`);
      navigate(`/jobs/${res.job.id}`);
      await refresh();
    } catch (e) {
      setErr(e instanceof Error ? e.message : String(e));
    } finally {
      setSourceBusyId(null);
    }
  }

  async function removeSource(s: SourceRow) {
    const ok = window.confirm(
      `Remove source ${s.id}?\n\n${s.path}\n\nIndexed files/chunks for this source will be deleted.`,
    );
    if (!ok) return;
    setSourceBusyId(s.id);
    try {
      await api.sources.delete(s.id);
      setStatus(`Removed source ${s.id}.`);
      await refresh();
    } catch (e) {
      setErr(e instanceof Error ? e.message : String(e));
    } finally {
      setSourceBusyId(null);
    }
  }

  return (
    <div className="space-y-6">
      <Panel
        title="Source Folders"
        subtitle="Connect local directories, monitor index state, and queue bounded re-index jobs."
        actions={<GhostButton onClick={() => void refresh()}>Refresh</GhostButton>}
      >
        <div className="grid gap-3 md:grid-cols-4">
          <StatChip label="total" value={String(stats.total)} />
          <StatChip label="ready" value={String(stats.ready)} />
          <StatChip label="indexing" value={String(stats.indexing)} />
          <StatChip label="errors" value={String(stats.errors)} alert={stats.errors > 0} />
        </div>

        <div className="mt-4 flex flex-col gap-3 md:flex-row md:items-end">
          <div className="flex-1">
            <label className="text-xs font-semibold tracking-wide text-forge-mist">Folder path</label>
            <input
              className="forge-input mt-1"
              value={path}
              onChange={(e) => setPath(e.target.value)}
              placeholder="/home/you/docs/project"
            />
            <div className="mt-2 text-[11px] text-forge-mist/80">
              Use an absolute directory path on this machine.
              {workspaceDir ? ` Workspace root: ${workspaceDir}` : ""}
            </div>
          </div>
          <div className="flex flex-wrap gap-2">
            {workspaceDir ? (
              <GhostButton onClick={() => setPath(workspaceDir)}>Use Workspace Root</GhostButton>
            ) : null}
            <PrimaryButton onClick={() => void addSource()} disabled={!canAdd || adding}>
              {adding ? "Adding…" : "Add Source"}
            </PrimaryButton>
          </div>
        </div>

        <div className="mt-3 flex flex-wrap items-center gap-2">
          <GhostButton onClick={() => void queueReindexAll()} disabled={stats.total === 0 || reindexAllBusy}>
            {reindexAllBusy ? "Queueing…" : "Re-index All Sources"}
          </GhostButton>
          <button
            type="button"
            onClick={() => navigate("/jobs")}
            className="rounded border border-forge-platinum/10 bg-black/20 px-2.5 py-1 text-[11px] font-medium text-forge-mist transition hover:border-forge-ember/35 hover:text-forge-ash"
          >
            Open Jobs
          </button>
        </div>

        {err ? (
          <div className="mt-4 rounded-md border border-forge-ember/30 bg-forge-ember/10 p-3 text-sm text-forge-ash">{err}</div>
        ) : null}
      </Panel>

      <Panel title="Connected Sources" subtitle="Per-source state, scan timestamps, and actions.">
        {sources.length === 0 ? (
          <div className="text-sm text-forge-mist">No sources yet. Add a folder to begin ingestion.</div>
        ) : (
          <div className="space-y-3">
            {sources.map((s) => {
              const state = sourceState(s);
              const busy = sourceBusyId === s.id;
              return (
                <div key={s.id} className="rounded-lg border border-forge-platinum/10 bg-forge-slate/20 p-4">
                  <div className="flex flex-wrap items-start justify-between gap-3">
                    <div className="min-w-0 flex-1">
                      <div className="flex flex-wrap items-center gap-2">
                        <span className="rounded border border-forge-platinum/10 bg-black/25 px-1.5 py-0.5 font-mono text-[10px] text-forge-mist">
                          #{s.id}
                        </span>
                        <StatePill state={state} />
                      </div>
                      <div className="mt-2 break-all font-mono text-xs text-forge-ash">{s.path}</div>
                      <div className="mt-2 text-[11px] text-forge-mist">
                        added {formatTime(s.createdAtMs)} · started {safeTime(s.lastScanStartedMs)} · completed{" "}
                        {safeTime(s.lastScanCompletedMs)}
                      </div>
                      {s.lastError ? (
                        <div className="mt-2 rounded border border-forge-ember/20 bg-forge-ember/10 px-2 py-1 text-xs text-forge-emberSoft">
                          {s.lastError}
                        </div>
                      ) : null}
                    </div>
                    <div className="flex flex-wrap gap-2">
                      <GhostButton onClick={() => void queueReindexSource(s)} disabled={busy}>
                        {busy ? "Queueing…" : "Re-index"}
                      </GhostButton>
                      <GhostButton onClick={() => void removeSource(s)} disabled={busy}>
                        Remove
                      </GhostButton>
                    </div>
                  </div>
                </div>
              );
            })}
          </div>
        )}
      </Panel>
    </div>
  );
}

function sourceState(s: SourceRow): "new" | "indexing" | "ready" | "error" {
  if (s.lastError) return "error";
  const started = s.lastScanStartedMs ?? 0;
  const completed = s.lastScanCompletedMs ?? 0;
  if (started > 0 && started > completed) return "indexing";
  if (completed > 0) return "ready";
  return "new";
}

function safeTime(ms: number | null | undefined): string {
  return typeof ms === "number" && ms > 0 ? formatTime(ms) : "—";
}

function StatePill(props: { state: "new" | "indexing" | "ready" | "error" }) {
  const cls =
    props.state === "ready"
      ? "border-forge-ultramarine/35 bg-forge-ultramarine/10 text-forge-platinum"
      : props.state === "indexing"
        ? "border-forge-ultramarine/40 bg-forge-ultramarine/10 text-forge-platinum"
        : props.state === "error"
          ? "border-forge-ember/40 bg-forge-ember/10 text-forge-emberSoft"
          : "border-forge-platinum/15 bg-forge-platinum/5 text-forge-mist";
  return <span className={`rounded border px-2 py-0.5 text-[10px] font-semibold uppercase tracking-wide ${cls}`}>{props.state}</span>;
}

function StatChip(props: { label: string; value: string; alert?: boolean }) {
  return (
    <div className={`rounded border bg-black/20 p-2.5 ${props.alert ? "border-forge-ember/35" : "border-forge-platinum/10"}`}>
      <div className="text-[10px] uppercase tracking-wide text-forge-mist">{props.label}</div>
      <div className={`mt-1 text-lg font-semibold ${props.alert ? "text-forge-emberSoft" : "text-forge-ash"}`}>{props.value}</div>
    </div>
  );
}
