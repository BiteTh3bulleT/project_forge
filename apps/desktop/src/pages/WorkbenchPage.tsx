import type { JobDetail } from "@forge/shared";
import { GhostButton, PrimaryButton } from "@forge/ui";
import { useCallback, useEffect, useState } from "react";
import { Link, useSearchParams } from "react-router-dom";

import {
  AuditJobLink,
  AuditTraceLink,
  traceAuditTargetFrom,
} from "../components/AuditLinks";
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
  const [content, setContent] = useState<{
    text: string;
    textual: boolean;
    previewLimited: boolean;
  } | null>(null);
  const [compareId, setCompareId] = useState("");
  const [compare, setCompare] = useState<{
    artifact: ForgeArtifact;
    content: { text: string; textual: boolean; previewLimited: boolean };
  } | null>(null);
  const [jobDetail, setJobDetail] = useState<JobDetail | null>(null);
  const [err, setErr] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);
  const recentJobEvents = Array.isArray(jobDetail?.events)
    ? jobDetail.events.slice(-12)
    : [];

  const refreshList = useCallback(async () => {
    const res = await api.artifacts.list({
      limit: 200,
      jobId: jobId.trim() ? jobId.trim() : undefined,
    });
    setArtifacts(Array.isArray(res.artifacts) ? res.artifacts : []);
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
    <div className="forge-ops-board space-y-4">
      <header className="rounded-lg border border-forge-platinum/10 bg-forge-carbon/80 p-4 shadow-[0_18px_60px_rgba(0,0,0,0.32)] lg:flex lg:items-end lg:justify-between lg:gap-4">
        <div>
          <div className="forge-ops-label">Workbench</div>
          <h1 className="mt-2 text-2xl font-semibold tracking-normal text-forge-ash">
            Artifact builder
          </h1>
          <p className="mt-1 max-w-2xl text-sm leading-6 text-forge-mist/75">
            Inspect recorded outputs, compare revisions, and keep job evidence
            in view.
          </p>
        </div>
        <div className="flex flex-wrap items-center gap-2">
          <span className="forge-ops-status forge-ops-status--muted">
            {artifacts.length} artifacts
          </span>
          <GhostButton className="h-9 px-3" onClick={() => void refreshList()}>
            Refresh list
          </GhostButton>
        </div>
      </header>

      {err ? (
        <div className="forge-ops-panel border-forge-ember/30 bg-forge-ember/10 p-3 text-sm text-forge-ash">
          {err}
        </div>
      ) : null}

      <div className="grid min-h-[560px] gap-4 xl:grid-cols-[minmax(16rem,19rem)_minmax(0,1fr)_minmax(17rem,21rem)]">
        <aside className="forge-ops-panel min-w-0 bg-forge-carbon/90 shadow-[0_18px_50px_rgba(0,0,0,0.28)]">
          <div className="forge-ops-panel__head">
            <div>
              <div className="forge-ops-title">Artifact Index</div>
              <div className="mt-1 text-xs text-forge-mist/65">
                SQLite records and file-backed outputs.
              </div>
            </div>
          </div>
          <div className="forge-ops-panel__body">
            <label className="block text-xs text-forge-mist">
              <span className="forge-ops-label">Job filter</span>
              <input
                className="forge-input mt-2 w-full font-mono text-xs"
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
            <div className="mt-3 max-h-[min(62vh,680px)] space-y-1 overflow-auto pr-1">
              {artifacts.length === 0 ? (
                <div className="rounded border border-dashed border-forge-platinum/15 bg-black/35 p-4 text-xs text-forge-mist">
                  <div className="font-semibold text-forge-ash">
                    No artifacts found
                  </div>
                  <div className="mt-1 leading-5 text-forge-mist/75">
                    Clear the job filter or refresh after a job writes evidence.
                  </div>
                </div>
              ) : (
                artifacts.map((a) => (
                  <button
                    key={a.id}
                    type="button"
                    disabled={busy}
                    onClick={() => void openArtifact(a)}
                    className={[
                      "w-full rounded border px-2.5 py-2 text-left text-xs transition shadow-[inset_0_1px_0_rgba(255,255,255,0.03)]",
                      selected?.id === a.id
                        ? "border-forge-ember/50 bg-forge-ember/10 text-forge-ash shadow-[inset_3px_0_0_rgba(255,122,51,0.8)]"
                        : "border-forge-platinum/10 bg-black/25 text-forge-mist hover:border-forge-ember/30 hover:bg-forge-ember/5",
                    ].join(" ")}
                  >
                    <div className="flex min-w-0 items-center justify-between gap-2">
                      <span className="truncate font-semibold text-forge-ash">
                        {a.title}
                      </span>
                      <span className="shrink-0 font-mono text-[10px] text-forge-emberSoft">
                        #{a.id}
                      </span>
                    </div>
                    <div className="mt-1 truncate font-mono text-[10px] text-forge-mist/75">
                      {a.type} · {a.filePath}
                    </div>
                    {a.jobId ? (
                      <div className="mt-1 truncate text-[10px] text-forge-mist/65">
                        job {a.jobId}
                      </div>
                    ) : null}
                  </button>
                ))
              )}
            </div>
          </div>
        </aside>

        <main className="forge-ops-panel min-w-0 bg-forge-carbon/90 shadow-[0_22px_70px_rgba(0,0,0,0.34)]">
          <div className="forge-ops-panel__head">
            <div className="min-w-0">
              <div className="forge-ops-title truncate">
                {selected
                  ? `Inspect #${selected.id}: ${selected.title}`
                  : "Viewer"}
              </div>
              <div className="mt-1 text-xs text-forge-mist/65">
                Safe textual preview and line-level comparison.
              </div>
            </div>
            {selected?.jobId ? (
              <Link
                className="forge-btn forge-btn--primary inline-flex items-center"
                to={`/jobs/${encodeURIComponent(selected.jobId)}`}
              >
                Open job
              </Link>
            ) : null}
          </div>
          <div className="forge-ops-panel__body">
            {!selected ? (
              <div className="flex min-h-[460px] items-center justify-center rounded border border-dashed border-forge-platinum/15 bg-black/35 p-6 text-center text-sm text-forge-mist">
                <div className="max-w-sm">
                  <div className="text-base font-semibold text-forge-ash">
                    Select an artifact
                  </div>
                  <p className="mt-2 leading-6 text-forge-mist/75">
                    Open an output to inspect its textual preview, metadata, and
                    compare it against another artifact.
                  </p>
                  {artifacts[0] ? (
                    <PrimaryButton
                      className="mt-4 h-9 px-3"
                      onClick={() => void openArtifact(artifacts[0])}
                      disabled={busy}
                    >
                      Open latest
                    </PrimaryButton>
                  ) : null}
                </div>
              </div>
            ) : (
              <div className="space-y-3">
                {!content ? (
                  <div className="text-sm text-forge-mist">
                    Loading or unavailable…
                  </div>
                ) : content.previewLimited && !content.textual ? (
                  <div className="rounded border border-forge-platinum/10 bg-black/30 p-3 text-sm text-forge-mist">
                    Preview is not available for this file type. The artifact
                    exists on disk; use your editor or export tools outside
                    FORGE if needed.
                  </div>
                ) : (
                  <pre className="max-h-[min(60vh,640px)] overflow-auto whitespace-pre-wrap rounded border border-forge-platinum/10 bg-black/55 p-3 font-mono text-[11px] leading-relaxed text-forge-mist shadow-inner">
                    {content.text}
                  </pre>
                )}

                {selected ? (
                  <div className="rounded border border-forge-platinum/10 bg-black/30 p-3">
                    <div className="mb-2 flex flex-wrap items-center justify-between gap-2">
                      <div className="text-xs font-semibold text-forge-ash">
                        Compare artifact
                      </div>
                      {compare ? (
                        <span className="rounded-full border border-forge-ember/25 bg-forge-ember/10 px-2 py-0.5 font-mono text-[10px] text-forge-emberSoft">
                          #{selected.id} / #{compare.artifact.id}
                        </span>
                      ) : null}
                    </div>
                    <div className="flex flex-wrap gap-2">
                      <input
                        className="forge-input max-w-[12rem] font-mono text-xs"
                        value={compareId}
                        onChange={(e) => setCompareId(e.target.value)}
                        placeholder="artifact id"
                      />
                      <PrimaryButton
                        className="h-9 px-3"
                        onClick={async () => {
                          const id = Number(compareId);
                          if (!Number.isFinite(id) || id <= 0) {
                            setErr(
                              "Compare artifact id must be a positive number.",
                            );
                            return;
                          }
                          if (id === selected.id) {
                            setErr(
                              "Choose a different artifact id to compare.",
                            );
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
                        Load compare
                      </PrimaryButton>
                      {compare ? (
                        <GhostButton
                          className="h-9 px-3"
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
                        Comparing #{selected.id} ({selected.type}) with #
                        {compare.artifact.id} ({compare.artifact.type})
                      </div>
                    ) : null}
                  </div>
                ) : null}

                {selected && content?.textual && compare?.content.textual ? (
                  <div>
                    <div className="mb-2 text-xs font-semibold text-forge-ash">
                      Line diff
                    </div>
                    <pre className="max-h-[min(60vh,640px)] overflow-auto whitespace-pre-wrap rounded border border-forge-platinum/10 bg-black/55 p-3 font-mono text-[11px] leading-relaxed text-forge-mist shadow-inner">
                      {buildLineDiff(content.text, compare.content.text)}
                    </pre>
                  </div>
                ) : null}
              </div>
            )}
          </div>
        </main>

        <aside className="forge-ops-panel min-w-0 bg-forge-carbon/90 shadow-[0_18px_50px_rgba(0,0,0,0.28)]">
          <div className="forge-ops-panel__head">
            <div>
              <div className="forge-ops-title">Inspector</div>
              <div className="mt-1 text-xs text-forge-mist/65">
                Metadata, job projection, and evidence tail.
              </div>
            </div>
          </div>
          <div className="forge-ops-panel__body space-y-4">
            {selected ? (
              <div className="space-y-2 text-xs text-forge-mist">
                <div className="forge-ops-label">Selected artifact</div>
                <div className="rounded border border-forge-platinum/10 bg-black/25 p-3">
                  <div className="font-semibold text-forge-ash">
                    {selected.type} #{selected.id}
                  </div>
                  <div className="mt-2 break-all font-mono text-[11px]">
                    {selected.filePath}
                  </div>
                  <div className="mt-2 grid gap-1 text-[11px]">
                    <div>MIME: {selected.mimeType || "—"}</div>
                    <div>Created: {formatTime(selected.createdAtMs)}</div>
                    <div>
                      Preview:{" "}
                      {content?.textual
                        ? "text"
                        : content?.previewLimited
                          ? "limited"
                          : "pending"}
                    </div>
                  </div>
                </div>
              </div>
            ) : (
              <div className="rounded border border-dashed border-forge-platinum/15 bg-black/30 p-3 text-xs text-forge-mist">
                Select an artifact to reveal MIME, path, preview state, and job
                context.
              </div>
            )}

            {jobDetail ? (
              <div className="space-y-2 text-xs text-forge-mist">
                <div className="forge-ops-label">Job context</div>
                <div className="rounded border border-forge-platinum/10 bg-black/25 p-3">
                  <div className="font-semibold text-forge-ash">
                    {jobDetail.job.title}
                  </div>
                  <div className="mt-1">
                    {jobDetail.job.status} · {jobDetail.job.targetAdapter} ·
                    packet {jobDetail.job.taskPacketId ?? "—"}
                  </div>
                  <div className="mt-3 flex flex-wrap gap-2">
                    <Link
                      className="text-forge-emberSoft underline"
                      to={`/jobs/${encodeURIComponent(jobDetail.job.id)}`}
                    >
                      Job detail
                    </Link>
                    <AuditJobLink jobId={jobDetail.job.id} />
                    <Link
                      className="text-forge-emberSoft underline"
                      to={`/chat`}
                    >
                      Chat
                    </Link>
                  </div>
                </div>
              </div>
            ) : jobId.trim() ? (
              <div className="rounded border border-forge-platinum/10 bg-black/25 p-3 text-xs text-forge-mist">
                Could not load job context. Artifacts may still reference this
                job id.
              </div>
            ) : null}

            {jobDetail && recentJobEvents.length > 0 ? (
              <div>
                <div className="mb-2 forge-ops-label">Recent job events</div>
                <div className="max-h-80 space-y-1 overflow-auto rounded border border-forge-platinum/10 bg-black/25 p-2 text-[11px] text-forge-mist">
                  {recentJobEvents.map((ev) => {
                    const auditTarget = traceAuditTargetFrom(ev.payload);
                    return (
                      <div
                        key={ev.id}
                        className="border-b border-forge-platinum/5 py-1 last:border-0"
                      >
                        <span className="text-forge-ash">{ev.type}</span> ·{" "}
                        {formatTime(ev.createdAtMs)}
                        <div className="text-forge-mist/90">{ev.message}</div>
                        {auditTarget ? (
                          <div className="mt-1 font-semibold">
                            <AuditTraceLink target={auditTarget} />
                          </div>
                        ) : null}
                      </div>
                    );
                  })}
                </div>
              </div>
            ) : null}
          </div>
        </aside>
      </div>
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
