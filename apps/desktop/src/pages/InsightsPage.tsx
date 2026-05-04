import type {
  ImportedExecution,
  RoutingInsight,
  SourceEmbeddingStatus,
} from "@forge/shared";
import { GhostButton, Panel, PrimaryButton } from "@forge/ui";
import { useEffect, useState } from "react";

import { HumanDataView } from "../components/HumanDataView";
import { api } from "../lib/api";
import { formatTime } from "../lib/format";
import { useUiStore } from "../stores/uiStore";

export function InsightsPage() {
  const [insights, setInsights] = useState<RoutingInsight[]>([]);
  const [importsList, setImportsList] = useState<ImportedExecution[]>([]);
  const [embedStatus, setEmbedStatus] = useState<SourceEmbeddingStatus[]>([]);
  const [dossierId, setDossierId] = useState("");
  const [importAdapter, setImportAdapter] = useState("codex");
  const [importRunId, setImportRunId] = useState("");
  const [originJobId, setOriginJobId] = useState("");
  const [originPacketId, setOriginPacketId] = useState("");
  const [importSummary, setImportSummary] = useState("");
  const [importDiff, setImportDiff] = useState("");
  const [importNotes, setImportNotes] = useState("");
  const [err, setErr] = useState<string | null>(null);
  const setStatus = useUiStore((s) => s.setStatusLine);

  async function load() {
    try {
      const d = dossierId.trim() ? Number(dossierId.trim()) : undefined;
      const [ins, imp, emb] = await Promise.all([
        api.insights.list(120, Number.isFinite(d) ? d : undefined),
        api.imports.list(100, Number.isFinite(d) ? d : undefined),
        api.embeddings.status(),
      ]);
      setInsights(ins.insights);
      setImportsList(imp.imports);
      setEmbedStatus(emb.status);
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
      <Panel
        title="Insights"
        subtitle="Evidence-grounded routing advisories and imported execution memory. Dossier filter below applies when you click Refresh or Generate — change the number, then refresh to reload lists with that scope."
        actions={<GhostButton onClick={() => void load()}>Refresh</GhostButton>}
      >
        {err ? (
          <div className="rounded border border-forge-ember/30 bg-forge-ember/10 p-3 text-sm text-forge-ash">
            {err}
          </div>
        ) : null}
        <div className="grid gap-3 md:grid-cols-2">
          <div>
            <label className="text-xs font-semibold tracking-wide text-forge-mist">
              Dossier id filter (optional)
            </label>
            <input
              className="forge-input mt-1"
              value={dossierId}
              onChange={(e) => setDossierId(e.target.value)}
            />
          </div>
          <div className="flex items-end gap-2">
            <PrimaryButton
              onClick={async () => {
                const d = dossierId.trim()
                  ? Number(dossierId.trim())
                  : undefined;
                const res = await api.insights.generate(
                  Number.isFinite(d) ? d : undefined,
                );
                setStatus(
                  `Generated ${res.insights.length} insight record(s).`,
                );
                await load();
              }}
            >
              Generate Insights
            </PrimaryButton>
          </div>
        </div>
      </Panel>

      <Panel
        title="Routing Recommendations"
        subtitle="Advisory-only suggestions. Reasons and evidence are always visible."
      >
        {insights.length === 0 ? (
          <div className="text-sm text-forge-mist">No insights stored.</div>
        ) : (
          <div className="space-y-2">
            {insights.map((row) => (
              <div
                key={row.id}
                className="rounded border border-white/10 bg-black/20 p-3 text-xs text-forge-mist"
              >
                <div className="font-semibold text-forge-ash">
                  {row.adapterId} - {row.taskType}
                </div>
                <div className="mt-1">{row.recommendation}</div>
                <div className="mt-1">
                  confidence {(row.confidence * 100).toFixed(1)}% | dossier{" "}
                  {row.dossierId ?? "global"} | {formatTime(row.createdAtMs)}
                </div>
                <div className="mt-2 max-h-44 overflow-auto rounded border border-white/10 bg-black/30 p-2 text-[11px] text-forge-ash">
                  <HumanDataView
                    value={{ reasons: row.reasons, evidence: row.evidence }}
                    compact
                  />
                </div>
              </div>
            ))}
          </div>
        )}
      </Panel>

      <div className="grid gap-6 xl:grid-cols-2">
        <Panel
          title="Imported Executions"
          subtitle="Bring external Codex/Claude Code execution back into FORGE memory."
        >
          <div className="space-y-3">
            <div className="grid gap-3 md:grid-cols-2">
              <div>
                <label className="text-xs text-forge-mist">Adapter</label>
                <select
                  className="forge-input mt-1"
                  value={importAdapter}
                  onChange={(e) => setImportAdapter(e.target.value)}
                >
                  <option value="codex">codex</option>
                  <option value="claude_code">claude_code</option>
                  <option value="ollama">ollama</option>
                </select>
              </div>
              <div>
                <label className="text-xs text-forge-mist">
                  External run id
                </label>
                <input
                  className="forge-input mt-1"
                  value={importRunId}
                  onChange={(e) => setImportRunId(e.target.value)}
                />
              </div>
            </div>
            <div className="grid gap-3 md:grid-cols-2">
              <div>
                <label className="text-xs text-forge-mist">Origin job id</label>
                <input
                  className="forge-input mt-1"
                  value={originJobId}
                  onChange={(e) => setOriginJobId(e.target.value)}
                />
              </div>
              <div>
                <label className="text-xs text-forge-mist">
                  Origin packet id
                </label>
                <input
                  className="forge-input mt-1"
                  value={originPacketId}
                  onChange={(e) => setOriginPacketId(e.target.value)}
                />
              </div>
            </div>
            <div>
              <label className="text-xs text-forge-mist">Summary</label>
              <textarea
                className="forge-input mt-1 min-h-[70px]"
                value={importSummary}
                onChange={(e) => setImportSummary(e.target.value)}
              />
            </div>
            <div>
              <label className="text-xs text-forge-mist">
                Diff/patch summary
              </label>
              <textarea
                className="forge-input mt-1 min-h-[60px]"
                value={importDiff}
                onChange={(e) => setImportDiff(e.target.value)}
              />
            </div>
            <div>
              <label className="text-xs text-forge-mist">Execution notes</label>
              <textarea
                className="forge-input mt-1 min-h-[60px]"
                value={importNotes}
                onChange={(e) => setImportNotes(e.target.value)}
              />
            </div>
            <PrimaryButton
              onClick={async () => {
                const packet = originPacketId.trim()
                  ? Number(originPacketId.trim())
                  : undefined;
                const d = dossierId.trim()
                  ? Number(dossierId.trim())
                  : undefined;
                await api.imports.create({
                  adapterId: importAdapter,
                  externalRunId: importRunId,
                  originJobId: originJobId.trim() || undefined,
                  originPacketId: Number.isFinite(packet) ? packet : undefined,
                  dossierId: Number.isFinite(d) ? d : undefined,
                  summary: importSummary,
                  diffSummary: importDiff,
                  executionNotes: importNotes,
                  outputRefs: [],
                  evaluation: {},
                });
                setStatus("Imported execution result saved.");
                setImportRunId("");
                setImportSummary("");
                setImportDiff("");
                setImportNotes("");
                await load();
              }}
            >
              Import Execution Result
            </PrimaryButton>
          </div>
          <div className="mt-4 space-y-2">
            {importsList.length === 0 ? (
              <div className="text-sm text-forge-mist">No imports yet.</div>
            ) : (
              importsList.map((x) => (
                <div
                  key={x.id}
                  className="rounded border border-white/10 bg-black/20 p-3 text-xs text-forge-mist"
                >
                  <div className="font-semibold text-forge-ash">
                    #{x.id} - {x.adapterId}
                  </div>
                  <div className="mt-1">{x.summary}</div>
                  <div className="mt-1">
                    run {x.externalRunId || "n/a"} | job{" "}
                    {x.originJobId ?? "none"} | packet{" "}
                    {x.originPacketId ?? "none"}
                  </div>
                  <div className="mt-1">{formatTime(x.createdAtMs)}</div>
                </div>
              ))
            )}
          </div>
        </Panel>

        <Panel
          title="Embedding Status"
          subtitle="Semantic index readiness by source, with explicit re-embed controls."
        >
          <div className="mb-3 flex gap-2">
            <PrimaryButton
              onClick={async () => {
                await api.embeddings.reembed({});
                setStatus("Re-embed all triggered.");
                await load();
              }}
            >
              Re-embed All
            </PrimaryButton>
          </div>
          {embedStatus.length === 0 ? (
            <div className="text-sm text-forge-mist">
              No source embedding rows yet.
            </div>
          ) : (
            <div className="space-y-2">
              {embedStatus.map((srow) => (
                <div
                  key={srow.sourceId}
                  className="rounded border border-white/10 bg-black/20 p-3 text-xs text-forge-mist"
                >
                  <div className="font-semibold text-forge-ash">
                    source {srow.sourceId}
                  </div>
                  <div className="mt-1">{srow.path}</div>
                  <div className="mt-1">
                    ready {srow.readyChunks}/{srow.totalChunks} | failed{" "}
                    {srow.failedChunks}
                  </div>
                  <div className="mt-2">
                    <GhostButton
                      onClick={async () => {
                        await api.embeddings.reembed({
                          sourceId: srow.sourceId,
                        });
                        setStatus(
                          `Re-embed triggered for source ${srow.sourceId}.`,
                        );
                        await load();
                      }}
                    >
                      Re-embed Source
                    </GhostButton>
                  </div>
                </div>
              ))}
            </div>
          )}
        </Panel>
      </div>
    </div>
  );
}
