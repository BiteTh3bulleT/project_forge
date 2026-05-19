import type {
  ApprovalPreset,
  AutomationRule,
  Dossier,
  DossierDetail,
  DossierMemoryView,
  DossierProfile,
  DossierVSASummary,
  ExecutionStrategy,
  ReviewRecord,
  SourceRow,
} from "@forge/shared";
import { GhostButton, Panel, PrimaryButton } from "@forge/ui";
import { useEffect, useMemo, useState } from "react";
import { useSearchParams } from "react-router-dom";

import { FoldSection } from "../components/FoldSection";
import { api } from "../lib/api";
import { formatTime } from "../lib/format";
import { useUiStore } from "../stores/uiStore";

function parseIDs(raw: string): number[] {
  return raw
    .split(",")
    .map((x) => Number(x.trim()))
    .filter((x) => Number.isFinite(x) && x > 0);
}

function parseCSV(raw: string): string[] {
  return raw
    .split(",")
    .map((x) => x.trim())
    .filter(Boolean);
}

function csv(raw: string[]): string {
  return raw.join(", ");
}

function arrayOrEmpty<T>(value: T[] | null | undefined): T[] {
  return Array.isArray(value) ? value : [];
}

function normalizeDossierDetail(detail: DossierDetail): DossierDetail {
  return {
    ...detail,
    sources: arrayOrEmpty(detail.sources),
    recentJobs: arrayOrEmpty(detail.recentJobs),
    briefs: arrayOrEmpty(detail.briefs),
  };
}

function normalizeMemoryView(view: DossierMemoryView): DossierMemoryView {
  return {
    ...view,
    recentObservations: arrayOrEmpty(view.recentObservations),
    recentSignals: arrayOrEmpty(view.recentSignals),
    recentAlignmentNotes: arrayOrEmpty(view.recentAlignmentNotes),
  };
}

function isOptionalEndpointMissing(error: unknown): boolean {
  const message =
    error instanceof Error
      ? error.message.toLowerCase()
      : String(error).toLowerCase();
  return message.includes("404") || message.includes("not found");
}

export function DossiersPage() {
  const [params, setParams] = useSearchParams();
  const uiMode = useUiStore((s) => s.uiMode);
  const [dossiers, setDossiers] = useState<Dossier[]>([]);
  const [detail, setDetail] = useState<DossierDetail | null>(null);
  const [sources, setSources] = useState<SourceRow[]>([]);
  const [selectedID, setSelectedID] = useState<number | null>(() => {
    const raw = Number(params.get("dossierId"));
    return Number.isFinite(raw) && raw > 0 ? raw : null;
  });
  const [name, setName] = useState("");
  const [description, setDescription] = useState("");
  const [sourceIDsRaw, setSourceIDsRaw] = useState("");

  const [profile, setProfile] = useState<DossierProfile | null>(null);
  const [memoryView, setMemoryView] = useState<DossierMemoryView | null>(null);
  const [vsaSummary, setVSASummary] = useState<DossierVSASummary | null>(null);
  const [presets, setPresets] = useState<ApprovalPreset[]>([]);
  const [strategies, setStrategies] = useState<ExecutionStrategy[]>([]);
  const [rules, setRules] = useState<AutomationRule[]>([]);
  const [reviews, setReviews] = useState<ReviewRecord[]>([]);

  const [preferredStrategiesRaw, setPreferredStrategiesRaw] = useState("");
  const [preferredAdaptersRaw, setPreferredAdaptersRaw] = useState("");
  const [approvalPresetId, setApprovalPresetId] = useState("balanced");
  const [retrievalMode, setRetrievalMode] = useState("hybrid");
  const [highValueRaw, setHighValueRaw] = useState("");
  const [noisyRaw, setNoisyRaw] = useState("");
  const [routingNotes, setRoutingNotes] = useState("");
  const [automationBindingsRaw, setAutomationBindingsRaw] = useState("");

  const [err, setErr] = useState<string | null>(null);
  const setStatus = useUiStore((s) => s.setStatusLine);
  const [showAdvancedProfile, setShowAdvancedProfile] = useState(false);
  const [dossiersView, setDossiersView] = useState<
    "all" | "create" | "detail" | "policy"
  >("all");

  async function loadList() {
    try {
      const [d, src, p, st, ar, rv] = await Promise.all([
        api.dossiers.list(180),
        api.sources.list(),
        api.policy.listPresets(80),
        api.strategies.list({ limit: 220 }),
        api.automation.listRules({ limit: 220 }),
        api.reviews.list({ limit: 260 }),
      ]);
      const nextDossiers = arrayOrEmpty(d.dossiers);
      setDossiers(nextDossiers);
      setSources(arrayOrEmpty(src.sources));
      setPresets(arrayOrEmpty(p.presets));
      setStrategies(arrayOrEmpty(st.strategies));
      setRules(arrayOrEmpty(ar.rules));
      setReviews(arrayOrEmpty(rv.reviews));
      setErr(null);
      if (selectedID == null && nextDossiers.length > 0) {
        const firstId = nextDossiers[0].id;
        setSelectedID(firstId);
        setParams({ dossierId: String(firstId) });
      }
    } catch (e) {
      setErr(e instanceof Error ? e.message : String(e));
    }
  }

  async function loadDetail(id: number) {
    try {
      const [d, p, mem, vsa] = await Promise.all([
        api.dossiers.detail(id),
        api.policy.getDossierProfile(id),
        api.memory.dossierView(id, 30),
        api.memory.dossierVSASummary(id).catch((error: unknown) => {
          if (isOptionalEndpointMissing(error)) {
            return { summary: null as DossierVSASummary | null };
          }
          throw error;
        }),
      ]);
      const nextDetail = normalizeDossierDetail(d.detail);
      const nextMemoryView = normalizeMemoryView(mem.view);
      setDetail(nextDetail);
      setProfile(p.profile);
      setMemoryView(nextMemoryView);
      setVSASummary(
        vsa.summary ?? nextMemoryView.vsaSummary ?? nextDetail.vsaSummary ?? null,
      );
      if (p.profile) {
        setPreferredStrategiesRaw(csv(arrayOrEmpty(p.profile.preferredStrategies)));
        setPreferredAdaptersRaw(csv(arrayOrEmpty(p.profile.preferredAdapters)));
        setApprovalPresetId(p.profile.approvalPresetId ?? "");
        setRetrievalMode(
          String((p.profile.retrievalDefaults?.mode as string) ?? "hybrid"),
        );
        setHighValueRaw(csv(arrayOrEmpty(p.profile.highValueFiles)));
        setNoisyRaw(csv(arrayOrEmpty(p.profile.noisyFiles)));
        setRoutingNotes(p.profile.routingNotes || "");
        setAutomationBindingsRaw(
          arrayOrEmpty(p.profile.automationBindings)
            .map((x) => String(x))
            .join(", "),
        );
      } else {
        setPreferredStrategiesRaw("");
        setPreferredAdaptersRaw("");
        setApprovalPresetId("balanced");
        setRetrievalMode("hybrid");
        setHighValueRaw("");
        setNoisyRaw("");
        setRoutingNotes("");
        setAutomationBindingsRaw("");
      }
      setErr(null);
    } catch (e) {
      setErr(e instanceof Error ? e.message : String(e));
      setDetail(null);
      setProfile(null);
      setMemoryView(null);
      setVSASummary(null);
    }
  }

  useEffect(() => {
    void loadList();
  }, []);

  useEffect(() => {
    const fromUrl = Number(params.get("dossierId"));
    if (Number.isFinite(fromUrl) && fromUrl > 0 && fromUrl !== selectedID) {
      setSelectedID(fromUrl);
      return;
    }
    if (selectedID != null) {
      void loadDetail(selectedID);
    }
  }, [params, selectedID]);

  const sourceHint = useMemo(
    () => sources.map((s) => `${s.id}:${s.path}`).join(" | "),
    [sources],
  );
  const strategyHint = useMemo(
    () => strategies.map((s) => s.id).join(", "),
    [strategies],
  );
  const rulesHint = useMemo(
    () => rules.map((r) => `${r.id}:${r.name}`).join(" | "),
    [rules],
  );
  const dossierReviews = useMemo(() => {
    if (!detail) return [];
    return reviews.filter((r) => r.dossierId === detail.dossier.id);
  }, [reviews, detail]);

  return (
    <div className="forge-ops-board space-y-5">
      <Panel
        title="Dossiers"
        subtitle={
          uiMode === "cognitive"
            ? "Project memory profiles. Create a dossier, link sources, and review recent work."
            : "Durable project memory profiles that scope retrieval, packets, jobs, and policy behavior."
        }
        actions={
          <div className="flex items-center gap-2">
            <label className="text-[11px] text-forge-mist">
              View
              <select
                className="forge-input ml-2 px-2 py-1 text-[11px]"
                value={dossiersView}
                onChange={(e) =>
                  setDossiersView(
                    e.target.value as "all" | "create" | "detail" | "policy",
                  )
                }
              >
                <option value="all">All</option>
                <option value="create">Create + list</option>
                <option value="detail">Detail</option>
                <option value="policy">Policy profile</option>
              </select>
            </label>
            <GhostButton onClick={() => void loadList()}>Refresh</GhostButton>
          </div>
        }
      >
        {err ? (
          <div className="rounded border border-forge-ember/30 bg-forge-ember/10 p-3 text-sm text-forge-ash">
            {err}
          </div>
        ) : null}
        <FoldSection
          title="Create dossier"
          subtitle="Define a scoped project memory profile and source links."
          defaultOpen
        >
          <div className="grid gap-3 md:grid-cols-2">
            <div>
              <label className="text-xs font-semibold tracking-wide text-forge-mist">
                Name
              </label>
              <input
                aria-label="Dossier name"
                className="forge-input mt-1"
                value={name}
                onChange={(e) => setName(e.target.value)}
                placeholder="e.g. ProjectForge Core"
              />
            </div>
            <div>
              <label className="text-xs font-semibold tracking-wide text-forge-mist">
                Source ids (comma separated)
              </label>
              <input
                aria-label="Source ids"
                className="forge-input mt-1"
                value={sourceIDsRaw}
                onChange={(e) => setSourceIDsRaw(e.target.value)}
                placeholder="1,2"
              />
            </div>
          </div>
          <div className="mt-3">
            <label className="text-xs font-semibold tracking-wide text-forge-mist">
              Description
            </label>
            <textarea
              aria-label="Dossier description"
              className="forge-input mt-1 min-h-[80px]"
              value={description}
              onChange={(e) => setDescription(e.target.value)}
            />
          </div>
          <div className="mt-3 flex flex-wrap gap-2">
            <PrimaryButton
              onClick={async () => {
                const sourceIds = parseIDs(sourceIDsRaw);
                const res = await api.dossiers.create({
                  name,
                  description,
                  sourceIds,
                  primaryPaths: [],
                  relatedRepos: [],
                  constraints: [],
                  preferredAdapters: ["ollama", "codex", "claude_code"],
                  importantFiles: [],
                });
                setStatus(`Dossier created: ${res.dossier.name}`);
                setName("");
                setDescription("");
                setSourceIDsRaw("");
                await loadList();
                setSelectedID(res.dossier.id);
                setParams({ dossierId: String(res.dossier.id) });
              }}
            >
              Create Dossier
            </PrimaryButton>
          </div>
          {sourceHint ? (
            <div className="mt-3 text-[11px] text-forge-mist">
              Known sources: {sourceHint}
            </div>
          ) : null}
        </FoldSection>
      </Panel>

      {dossiersView === "all" || dossiersView === "create" ? (
        <Panel
          title="Dossier List"
          subtitle="Choose a dossier to inspect linked sources, recent jobs, briefs, reviews, and policy profile."
        >
          {dossiers.length === 0 ? (
            <div className="text-sm text-forge-mist">No dossiers yet.</div>
          ) : (
            <div className="space-y-2">
              {dossiers.map((d) => (
                <button
                  key={d.id}
                  type="button"
                  className={[
                    "w-full rounded border px-3 py-2 text-left",
                    selectedID === d.id
                      ? "border-forge-ember/40 bg-black/30"
                      : "border-forge-platinum/10 bg-black/20 hover:border-forge-ember/35",
                  ].join(" ")}
                  onClick={() => {
                    setSelectedID(d.id);
                    setParams({ dossierId: String(d.id) });
                  }}
                >
                  <div className="text-sm font-semibold text-forge-ash">
                    {d.name}
                  </div>
                  <div className="mt-1 text-xs text-forge-mist">
                    {d.description || "(no description)"}
                  </div>
                  <div className="mt-1 text-[11px] text-forge-mist">
                    updated {formatTime(d.updatedAtMs)}
                  </div>
                </button>
              ))}
            </div>
          )}
        </Panel>
      ) : null}

      {dossiersView === "all" || dossiersView === "detail" ? (
        <Panel
          title="Dossier Detail"
          subtitle="Project brief, scope anchors, linked reviews, and recent execution memory."
          actions={
            detail ? (
              <PrimaryButton
                onClick={async () => {
                  await api.dossiers.generateBrief(
                    detail.dossier.id,
                    "Regenerated from dossier view",
                  );
                  setStatus(
                    `Brief regenerated for dossier ${detail.dossier.id}.`,
                  );
                  await loadDetail(detail.dossier.id);
                }}
              >
                Generate Brief
              </PrimaryButton>
            ) : null
          }
        >
          {!detail ? (
            <div className="text-sm text-forge-mist">
              Select a dossier to inspect details.
            </div>
          ) : (
            <div className="forge-ops-board space-y-5">
              <div className="rounded border border-forge-platinum/10 bg-black/20 p-3 text-xs text-forge-mist">
                <div className="text-sm font-semibold text-forge-ash">
                  {detail.dossier.name}
                </div>
                <div className="mt-2">
                  {detail.dossier.description || "No description"}
                </div>
                <div className="mt-2">
                  Routing notes: {detail.dossier.routingNotes || "none"}
                </div>
              </div>
              <div className="grid gap-3 md:grid-cols-2">
                <div className="rounded border border-forge-platinum/10 bg-black/20 p-3">
                  <div className="text-xs font-semibold tracking-wide text-forge-mist">
                    Linked Sources
                  </div>
                  {detail.sources.length === 0 ? (
                    <div className="mt-2 text-xs text-forge-mist">
                      No source links.
                    </div>
                  ) : (
                    <div className="mt-2 space-y-1 text-xs text-forge-mist">
                      {detail.sources.map((s) => (
                        <div key={s.sourceId}>
                          {s.sourceId} - {s.path}
                        </div>
                      ))}
                    </div>
                  )}
                </div>
                <div className="rounded border border-forge-platinum/10 bg-black/20 p-3">
                  <div className="text-xs font-semibold tracking-wide text-forge-mist">
                    Recent Jobs
                  </div>
                  {detail.recentJobs.length === 0 ? (
                    <div className="mt-2 text-xs text-forge-mist">
                      No linked jobs.
                    </div>
                  ) : (
                    <div className="mt-2 space-y-1 text-xs text-forge-mist">
                      {detail.recentJobs.slice(0, 8).map((j) => (
                        <div key={j.jobId}>
                          {j.jobId} - {j.status} - {j.targetAdapter}
                        </div>
                      ))}
                    </div>
                  )}
                </div>
              </div>
              <div className="rounded border border-forge-platinum/10 bg-black/20 p-3">
                <div className="text-xs font-semibold tracking-wide text-forge-mist">
                  Brief History
                </div>
                {detail.briefs.length === 0 ? (
                  <div className="mt-2 text-xs text-forge-mist">
                    No brief snapshots yet.
                  </div>
                ) : (
                  <div className="mt-2 space-y-2">
                    {detail.briefs.slice(0, 6).map((b) => (
                      <div
                        key={b.id}
                        className="rounded border border-forge-platinum/10 bg-black/30 p-2"
                      >
                        <div className="text-[11px] text-forge-mist">
                          brief #{b.id} - {formatTime(b.createdAtMs)}
                        </div>
                        <pre className="mt-1 max-h-44 overflow-auto whitespace-pre-wrap text-[11px] text-forge-ash">
                          {b.summaryMarkdown}
                        </pre>
                      </div>
                    ))}
                  </div>
                )}
              </div>

              <div className="rounded border border-forge-platinum/10 bg-black/20 p-3">
                <div className="text-xs font-semibold tracking-wide text-forge-mist">
                  Dossier Reviews
                </div>
                {dossierReviews.length === 0 ? (
                  <div className="mt-2 text-xs text-forge-mist">
                    No review records linked to this dossier.
                  </div>
                ) : (
                  <div className="mt-2 space-y-1 text-xs text-forge-mist">
                    {dossierReviews.slice(0, 10).map((r) => (
                      <div key={r.id}>
                        #{r.id} - {r.status} - {r.targetType}:{r.targetId}
                      </div>
                    ))}
                  </div>
                )}
              </div>

              <div className="rounded border border-forge-platinum/10 bg-black/20 p-3">
                <div className="text-xs font-semibold tracking-wide text-forge-mist">
                  Memory View
                </div>
                {!memoryView ? (
                  <div className="mt-2 text-xs text-forge-mist">
                    No dossier memory view loaded.
                  </div>
                ) : (
                  <div className="mt-2 space-y-2 text-xs text-forge-mist">
                    <div>
                      observations {memoryView.observationCount} · stale{" "}
                      {memoryView.staleObservationCount} · signals{" "}
                      {memoryView.recentSignals.length}
                    </div>
                    <div className="rounded border border-forge-platinum/10 bg-black/30 p-2">
                      <div className="text-[11px] font-semibold text-forge-ash">
                        VSA Coverage + Health
                      </div>
                      {!vsaSummary ? (
                        <div className="mt-1 text-[11px]">
                          No VSA summary available.
                        </div>
                      ) : (
                        <div className="mt-1 space-y-1 text-[11px]">
                          <div>
                            health {vsaSummary.health || "unknown"} · coverage{" "}
                            {(vsaSummary.coverageScore * 100).toFixed(1)}%
                          </div>
                          <div>
                            pointers {vsaSummary.pointerCount} · bindings{" "}
                            {vsaSummary.bindingCount} · associations{" "}
                            {vsaSummary.associationCount}
                          </div>
                          <div>
                            last reindex run{" "}
                            {vsaSummary.lastReindexRunId ?? "none"} ·{" "}
                            {vsaSummary.lastReindexAtMs
                              ? formatTime(vsaSummary.lastReindexAtMs)
                              : "never"}
                          </div>
                        </div>
                      )}
                    </div>
                    <div className="rounded border border-forge-platinum/10 bg-black/30 p-2">
                      <div className="text-[11px] font-semibold text-forge-ash">
                        Recent observations
                      </div>
                      {memoryView.recentObservations.length === 0 ? (
                        <div className="mt-1 text-[11px]">
                          No observations yet.
                        </div>
                      ) : (
                        <div className="mt-1 space-y-1">
                          {memoryView.recentObservations
                            .slice(0, 8)
                            .map((o) => (
                              <div key={o.id}>
                                #{o.id} · {o.type} · useful {o.usefulnessCount}{" "}
                                / noisy {o.noiseCount} · stale {String(o.stale)}
                              </div>
                            ))}
                        </div>
                      )}
                    </div>
                    <div className="rounded border border-forge-platinum/10 bg-black/30 p-2">
                      <div className="text-[11px] font-semibold text-forge-ash">
                        Recent packet alignment notes
                      </div>
                      {memoryView.recentAlignmentNotes.length === 0 ? (
                        <div className="mt-1 text-[11px]">
                          No packet alignment notes yet.
                        </div>
                      ) : (
                        <div className="mt-1 space-y-1">
                          {memoryView.recentAlignmentNotes
                            .slice(0, 6)
                            .map((n) => (
                              <div key={n.id}>
                                packet {n.packetId} · result{" "}
                                {n.retrievalResultId ?? "n/a"} · {n.note}
                              </div>
                            ))}
                        </div>
                      )}
                    </div>
                  </div>
                )}
              </div>
            </div>
          )}
        </Panel>
      ) : null}

      {dossiersView === "all" || dossiersView === "policy" ? (
        <Panel
          title="Dossier Policy Profile"
          subtitle="Dossier-specific strategy, adapter, retrieval, approval, and automation preferences."
        >
          {!detail ? (
            <div className="text-sm text-forge-mist">
              Select a dossier first.
            </div>
          ) : uiMode === "cognitive" && !showAdvancedProfile ? (
            <div className="space-y-3">
              <div className="rounded border border-forge-platinum/10 bg-black/20 p-3 text-xs text-forge-mist">
                Advanced policy profile controls are hidden in cognitive mode to
                reduce noise.
              </div>
              <PrimaryButton onClick={() => setShowAdvancedProfile(true)}>
                Show Advanced Controls
              </PrimaryButton>
            </div>
          ) : (
            <div className="space-y-3">
              <div className="grid gap-3 md:grid-cols-2">
                <div>
                  <label className="text-xs text-forge-mist">
                    Preferred strategies (ids, comma separated)
                  </label>
                  <input
                    aria-label="Preferred strategies"
                    className="forge-input mt-1"
                    value={preferredStrategiesRaw}
                    onChange={(e) => setPreferredStrategiesRaw(e.target.value)}
                    placeholder="repo_analysis, codex_implementation_handoff"
                  />
                </div>
                <div>
                  <label className="text-xs text-forge-mist">
                    Preferred adapters (comma separated)
                  </label>
                  <input
                    aria-label="Preferred adapters"
                    className="forge-input mt-1"
                    value={preferredAdaptersRaw}
                    onChange={(e) => setPreferredAdaptersRaw(e.target.value)}
                    placeholder="ollama, codex"
                  />
                </div>
              </div>

              <div className="grid gap-3 md:grid-cols-3">
                <div>
                  <label className="text-xs text-forge-mist">
                    Approval preset override
                  </label>
                  <select
                    aria-label="Approval preset override"
                    className="forge-input mt-1"
                    value={approvalPresetId}
                    onChange={(e) => setApprovalPresetId(e.target.value)}
                  >
                    <option value="">(none)</option>
                    {presets.map((p) => (
                      <option key={p.id} value={p.id}>
                        {p.name} ({p.id})
                      </option>
                    ))}
                  </select>
                </div>
                <div>
                  <label className="text-xs text-forge-mist">
                    Retrieval default mode
                  </label>
                  <select
                    aria-label="Retrieval default mode"
                    className="forge-input mt-1"
                    value={retrievalMode}
                    onChange={(e) => setRetrievalMode(e.target.value)}
                  >
                    <option value="keyword">keyword</option>
                    <option value="semantic">semantic</option>
                    <option value="hybrid">hybrid</option>
                  </select>
                </div>
                <div>
                  <label className="text-xs text-forge-mist">
                    Automation bindings (rule ids)
                  </label>
                  <input
                    aria-label="Automation bindings"
                    className="forge-input mt-1"
                    value={automationBindingsRaw}
                    onChange={(e) => setAutomationBindingsRaw(e.target.value)}
                    placeholder="1, 4"
                  />
                </div>
              </div>

              <div className="grid gap-3 md:grid-cols-2">
                <div>
                  <label className="text-xs text-forge-mist">
                    High-value files (comma separated)
                  </label>
                  <textarea
                    aria-label="High-value files"
                    className="forge-input mt-1 min-h-[70px]"
                    value={highValueRaw}
                    onChange={(e) => setHighValueRaw(e.target.value)}
                  />
                </div>
                <div>
                  <label className="text-xs text-forge-mist">
                    Noisy files (comma separated)
                  </label>
                  <textarea
                    aria-label="Noisy files"
                    className="forge-input mt-1 min-h-[70px]"
                    value={noisyRaw}
                    onChange={(e) => setNoisyRaw(e.target.value)}
                  />
                </div>
              </div>

              <div>
                <label className="text-xs text-forge-mist">Routing notes</label>
                <textarea
                  aria-label="Routing notes"
                  className="forge-input mt-1 min-h-[70px]"
                  value={routingNotes}
                  onChange={(e) => setRoutingNotes(e.target.value)}
                />
              </div>

              <div className="flex gap-2">
                <PrimaryButton
                  onClick={async () => {
                    const bindings = parseIDs(automationBindingsRaw);
                    await api.policy.saveDossierProfile(detail.dossier.id, {
                      preferredStrategies: parseCSV(preferredStrategiesRaw),
                      preferredAdapters: parseCSV(preferredAdaptersRaw),
                      approvalPresetId: approvalPresetId.trim() || undefined,
                      retrievalDefaults: { mode: retrievalMode },
                      highValueFiles: parseCSV(highValueRaw),
                      noisyFiles: parseCSV(noisyRaw),
                      routingNotes,
                      automationBindings: bindings,
                    });
                    setStatus(
                      `Dossier profile saved for ${detail.dossier.id}.`,
                    );
                    await loadDetail(detail.dossier.id);
                  }}
                >
                  Save Profile
                </PrimaryButton>
                <GhostButton onClick={() => void loadDetail(detail.dossier.id)}>
                  Reload Profile
                </GhostButton>
              </div>

              {profile ? (
                <div className="rounded border border-forge-platinum/10 bg-black/20 p-3 text-[11px] text-forge-mist">
                  Profile updated {formatTime(profile.updatedAtMs)}
                </div>
              ) : (
                <div className="rounded border border-forge-platinum/10 bg-black/20 p-3 text-[11px] text-forge-mist">
                  No profile yet. Save to create one.
                </div>
              )}

              <div className="text-[11px] text-forge-mist">
                Known strategy ids: {strategyHint || "none"}
              </div>
              <div className="text-[11px] text-forge-mist">
                Known automation rules: {rulesHint || "none"}
              </div>
              {uiMode === "cognitive" ? (
                <GhostButton onClick={() => setShowAdvancedProfile(false)}>
                  Hide Advanced Controls
                </GhostButton>
              ) : null}
            </div>
          )}
        </Panel>
      ) : null}
    </div>
  );
}
