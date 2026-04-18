import type { JobDetail } from "@forge/shared";
import { GhostButton, Panel } from "@forge/ui";
import { useCallback, useEffect, useState } from "react";
import { Link, useSearchParams } from "react-router-dom";

import { api, type ForgeArtifact } from "../lib/api";
import { formatTime } from "../lib/format";
import { useUiStore } from "../stores/uiStore";

export function WorkbenchPage() {
  const [params, setParams] = useSearchParams();
  const setStatus = useUiStore((s) => s.setStatusLine);
  const jobId = params.get("jobId") ?? "";
  const artifactIdParam = params.get("artifactId");
  const [artifacts, setArtifacts] = useState<ForgeArtifact[]>([]);
  const [selected, setSelected] = useState<ForgeArtifact | null>(null);
  const [content, setContent] = useState<{ text: string; textual: boolean; previewLimited: boolean } | null>(null);
  const [compareId, setCompareId] = useState("");
  const [compare, setCompare] = useState<{
    artifact: ForgeArtifact;
    content: { text: string; textual: boolean; previewLimited: boolean };
  } | null>(null);
  const [jobDetail, setJobDetail] = useState<JobDetail | null>(null);
  const [err, setErr] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);

  const refreshList = useCallback(async () => {
    const res = await api.artifacts.list({
      limit: 200,
      jobId: jobId.trim() ? jobId.trim() : undefined,
    });
    setArtifacts(res.artifacts);
  }, [jobId]);

  useEffect(() => {
    let cancelled = false;
    async function run() {
      try {
        await refreshList();
        if (jobId.trim()) {
          const d = await api.jobs.detail(jobId.trim(), 0);
          if (!cancelled) setJobDetail(d);
        } else if (!cancelled) {
          setJobDetail(null);
        }
        setErr(null);
      } catch (e) {
        if (!cancelled) {
          setErr(e instanceof Error ? e.message : String(e));
          setJobDetail(null);
        }
      }
    }
    void run();
    return () => {
      cancelled = true;
    };
  }, [jobId, refreshList]);

  async function openArtifact(a: ForgeArtifact) {
    setBusy(true);
    setSelected(a);
    setParams((prev) => {
      const p = new URLSearchParams(prev);
      p.set("artifactId", String(a.id));
      if (a.jobId) p.set("jobId", a.jobId);
      return p;
    });
    try {
      const c = await api.artifacts.content(a.id);
      setContent({
        text: c.content,
        textual: c.textual,
        previewLimited: c.previewLimited,
      });
      if (compare?.artifact.id === a.id) {
        setCompare(null);
        setCompareId("");
      }
      setErr(null);
      setStatus(`Opened artifact #${a.id}.`);
    } catch (e) {
      setContent(null);
      setErr(e instanceof Error ? e.message : String(e));
    } finally {
      setBusy(false);
    }
  }

  async function openCompareArtifact(id: number) {
    const a = await api.artifacts.get(id);
    const c = await api.artifacts.content(id);
    setCompare({
      artifact: a,
      content: {
        text: c.content,
        textual: c.textual,
        previewLimited: c.previewLimited,
      },
    });
    setStatus(`Loaded compare artifact #${id}.`);
  }

  useEffect(() => {
    const id = artifactIdParam ? Number(artifactIdParam) : NaN;
    if (!Number.isFinite(id) || id <= 0) return;
    let cancelled = false;
    async function autoOpen() {
      try {
        const a = await api.artifacts.get(id);
        if (!cancelled) await openArtifact(a);
      } catch {
        /* list may still load */
      }
    }
    void autoOpen();
    return () => {
      cancelled = true;
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps -- openArtifact stable enough; avoid loop
  }, [artifactIdParam]);

  return (
    <div className="grid min-h-[560px] gap-4 xl:grid-cols-[minmax(0,340px)_minmax(0,1fr)]">
      <div className="space-y-4">
        <Panel
          title="Artifact index"
          subtitle="Files recorded in SQLite with paths under the core artifact directory."
          actions={<GhostButton onClick={() => void refreshList()}>Refresh list</GhostButton>}
        >
          <label className="block text-xs text-forge-mist">
            Filter by job id
            <input
              className="forge-input mt-1 w-full font-mono text-xs"
              value={jobId}
              onChange={(e) => {
                const v = e.target.value;
                setParams((prev) => {
                  const p = new URLSearchParams(prev);
                  if (v.trim()) p.set("jobId", v.trim());
                  else p.delete("jobId");
                  return p;
                });
              }}
              placeholder="job id (optional)"
            />
          </label>
          {err ? <div className="mt-2 rounded border border-forge-ember/30 bg-forge-ember/10 p-2 text-xs text-forge-ash">{err}</div> : null}
          <div className="mt-3 max-h-[min(52vh,520px)] space-y-1 overflow-auto">
            {artifacts.length === 0 ? (
              <div className="text-sm text-forge-mist">No artifacts match this filter.</div>
            ) : (
              artifacts.map((a) => (
                <button
                  key={a.id}
                  type="button"
                  disabled={busy}
                  onClick={() => void openArtifact(a)}
                  className={[
                    "w-full rounded border px-2 py-2 text-left text-xs",
                    selected?.id === a.id ? "border-forge-ember/40 bg-forge-slate/40" : "border-white/10 bg-black/20 hover:border-forge-ember/25",
                  ].join(" ")}
                >
                  <div className="truncate font-semibold text-forge-ash">
                    #{a.id} · {a.type}: {a.title}
                  </div>
                  <div className="mt-0.5 font-mono text-[10px] text-forge-mist/80">{a.filePath}</div>
                  {a.jobId ? <div className="mt-0.5 text-[10px] text-forge-mist">job {a.jobId}</div> : null}
                </button>
              ))
            )}
          </div>
        </Panel>

        {jobDetail ? (
          <Panel title="Job context" subtitle="Latest projection for the filtered job id.">
            <div className="space-y-2 text-xs text-forge-mist">
              <div className="font-semibold text-forge-ash">{jobDetail.job.title}</div>
              <div>
                {jobDetail.job.status} · {jobDetail.job.targetAdapter} · packet {jobDetail.job.taskPacketId ?? "—"}
              </div>
              <div className="flex flex-wrap gap-2">
                <Link className="text-forge-emberSoft underline" to={`/jobs/${encodeURIComponent(jobDetail.job.id)}`}>
                  Job detail
                </Link>
                <Link className="text-forge-emberSoft underline" to={`/chat`}>
                  Chat
                </Link>
              </div>
            </div>
          </Panel>
        ) : jobId.trim() ? (
          <Panel title="Job context" subtitle="Could not load job (check id), or core offline.">
            <div className="text-xs text-forge-mist">Artifacts may still list if they reference this job id.</div>
          </Panel>
        ) : null}
      </div>

      <Panel
        title={selected ? `Inspect · #${selected.id}` : "Viewer"}
        subtitle="Textual artifacts load file contents from disk when path and MIME are considered safe. Binary files show metadata only."
        actions={
          selected?.jobId ? (
            <Link className="forge-btn forge-btn--primary inline-flex items-center" to={`/jobs/${encodeURIComponent(selected.jobId)}`}>
              Open job
            </Link>
          ) : null
        }
      >
        {!selected ? (
          <div className="text-sm text-forge-mist">Select an artifact from the index.</div>
        ) : (
          <div className="space-y-3">
            <div className="rounded border border-white/10 bg-black/25 p-3 text-xs text-forge-mist">
              <div className="font-mono text-[11px]">{selected.filePath}</div>
              <div className="mt-1">MIME: {selected.mimeType || "—"}</div>
              <div className="mt-1">Created: {formatTime(selected.createdAtMs)}</div>
            </div>

            {!content ? (
              <div className="text-sm text-forge-mist">Loading or unavailable…</div>
            ) : content.previewLimited && !content.textual ? (
              <div className="rounded border border-white/10 bg-black/30 p-3 text-sm text-forge-mist">
                Preview is not available for this file type. The artifact exists on disk; use your editor or export tools outside FORGE if needed.
              </div>
            ) : (
              <pre className="max-h-[min(60vh,640px)] overflow-auto whitespace-pre-wrap rounded border border-white/10 bg-black/40 p-3 font-mono text-[11px] leading-relaxed text-forge-mist">
                {content.text}
              </pre>
            )}

            {selected ? (
              <div className="rounded border border-white/10 bg-black/25 p-3">
                <div className="mb-2 text-xs font-semibold text-forge-ash">Compare with another artifact</div>
                <div className="flex flex-wrap gap-2">
                  <input
                    className="forge-input max-w-[12rem] font-mono text-xs"
                    value={compareId}
                    onChange={(e) => setCompareId(e.target.value)}
                    placeholder="artifact id"
                  />
                  <GhostButton
                    onClick={async () => {
                      const id = Number(compareId);
                      if (!Number.isFinite(id) || id <= 0) {
                        setErr("Compare artifact id must be a positive number.");
                        return;
                      }
                      if (id === selected.id) {
                        setErr("Choose a different artifact id to compare.");
                        return;
                      }
                      try {
                        await openCompareArtifact(id);
                        setErr(null);
                      } catch (e) {
                        setErr(e instanceof Error ? e.message : String(e));
                      }
                    }}
                    disabled={busy}
                  >
                    Load compare artifact
                  </GhostButton>
                  {compare ? (
                    <GhostButton
                      onClick={() => {
                        setCompare(null);
                        setCompareId("");
                      }}
                      disabled={busy}
                    >
                      Clear compare
                    </GhostButton>
                  ) : null}
                </div>
                {compare ? (
                  <div className="mt-2 text-[11px] text-forge-mist">
                    Comparing #{selected.id} ({selected.type}) with #{compare.artifact.id} ({compare.artifact.type})
                  </div>
                ) : null}
              </div>
            ) : null}

            {selected && content?.textual && compare?.content.textual ? (
              <div>
                <div className="mb-2 text-xs font-semibold text-forge-ash">Line diff</div>
                <pre className="max-h-[min(60vh,640px)] overflow-auto whitespace-pre-wrap rounded border border-white/10 bg-black/40 p-3 font-mono text-[11px] leading-relaxed text-forge-mist">
                  {buildLineDiff(content.text, compare.content.text)}
                </pre>
              </div>
            ) : null}

            {jobDetail && jobDetail.events.length > 0 ? (
              <div>
                <div className="mb-2 text-xs font-semibold text-forge-ash">Recent job events (tail)</div>
                <div className="max-h-56 space-y-1 overflow-auto rounded border border-white/10 bg-black/25 p-2 text-[11px] text-forge-mist">
                  {jobDetail.events.slice(-12).map((ev) => (
                    <div key={ev.id} className="border-b border-white/5 py-1 last:border-0">
                      <span className="text-forge-ash">{ev.type}</span> · {formatTime(ev.createdAtMs)}
                      <div className="text-forge-mist/90">{ev.message}</div>
                    </div>
                  ))}
                </div>
              </div>
            ) : null}
          </div>
        )}
      </Panel>
    </div>
  );
}

function buildLineDiff(left: string, right: string): string {
  const a = left.replace(/\r\n/g, "\n").split("\n");
  const b = right.replace(/\r\n/g, "\n").split("\n");
  const n = Math.max(a.length, b.length);
  const out: string[] = [];
  for (let i = 0; i < n; i += 1) {
    const l = a[i];
    const r = b[i];
    if (l === r) {
      out.push(`  ${l ?? ""}`);
      continue;
    }
    if (l !== undefined) out.push(`- ${l}`);
    if (r !== undefined) out.push(`+ ${r}`);
  }
  return out.join("\n");
}
