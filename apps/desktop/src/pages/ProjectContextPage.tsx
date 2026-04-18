import type { ProjectContextRecord } from "@forge/shared";
import { GhostButton, Panel, PrimaryButton } from "@forge/ui";
import { useEffect, useState } from "react";

import { api } from "../lib/api";
import { formatTime } from "../lib/format";
import { useUiStore } from "../stores/uiStore";

export function ProjectContextPage() {
  const setStatus = useUiStore((s) => s.setStatusLine);
  const [record, setRecord] = useState<ProjectContextRecord | null>(null);
  const [sourcePath, setSourcePath] = useState("");
  const [notes, setNotes] = useState("");
  const [err, setErr] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);

  async function refresh() {
    try {
      const res = await api.projectContext.get();
      setRecord(res.record);
      setErr(null);
    } catch (e) {
      setErr(e instanceof Error ? e.message : String(e));
    }
  }

  useEffect(() => {
    void refresh();
  }, []);

  return (
    <div className="space-y-6">
      <Panel
        title="Project Context"
        subtitle="Normalize source context into versioned FORGE records and durable agent guidance files."
        actions={<GhostButton onClick={() => void refresh()}>Refresh</GhostButton>}
      >
        <div className="grid gap-3 md:grid-cols-2">
          <div>
            <label className="text-xs font-semibold tracking-wide text-forge-mist">Source context file (optional)</label>
            <input
              className="forge-input mt-1"
              value={sourcePath}
              onChange={(e) => setSourcePath(e.target.value)}
              placeholder="/abs/path/FORGE_CONTEXT.md"
            />
            <div className="mt-2 text-[11px] text-forge-mist">Blank uses stored path or workspace `FORGE_CONTEXT.md`.</div>
          </div>
          <div>
            <label className="text-xs font-semibold tracking-wide text-forge-mist">Notes</label>
            <input className="forge-input mt-1" value={notes} onChange={(e) => setNotes(e.target.value)} placeholder="Optional normalization note" />
          </div>
        </div>
        <div className="mt-3 flex flex-wrap gap-2">
          <PrimaryButton
            disabled={busy}
            onClick={async () => {
              setBusy(true);
              try {
                const res = await api.projectContext.import(sourcePath.trim(), notes.trim());
                setRecord(res.record);
                setStatus(`Context imported into record ${res.record.id}.`);
                setErr(null);
              } catch (e) {
                setErr(e instanceof Error ? e.message : String(e));
              } finally {
                setBusy(false);
              }
            }}
          >
            {busy ? "Importing…" : "Import + Normalize"}
          </PrimaryButton>
          <GhostButton
            disabled={busy}
            onClick={async () => {
              setBusy(true);
              try {
                const res = await api.projectContext.regenerate();
                setRecord(res.record);
                setStatus(`Context regenerated in record ${res.record.id}.`);
                setErr(null);
              } catch (e) {
                setErr(e instanceof Error ? e.message : String(e));
              } finally {
                setBusy(false);
              }
            }}
          >
            Regenerate from Stored Source
          </GhostButton>
        </div>
        {err ? <div className="mt-4 rounded border border-forge-ember/30 bg-forge-ember/10 p-3 text-sm text-forge-ash">{err}</div> : null}
      </Panel>

      {!record ? (
        <Panel title="No normalized context" subtitle="Run an import to materialize project guidance.">
          <div className="text-sm text-forge-mist">No project context record exists yet.</div>
        </Panel>
      ) : (
        <>
          <Panel title="Latest Record" subtitle={`Record ${record.id} · version ${record.contextVersion} · generated ${formatTime(record.generatedAtMs)}`}>
            <div className="space-y-2 text-sm text-forge-mist">
              <div>
                Source: <span className="font-mono text-xs text-forge-ash">{record.sourcePath}</span>
              </div>
              <div>
                Hash: <span className="font-mono text-xs text-forge-ash">{record.sourceHash}</span>
              </div>
              <div>
                Generated files:
                <div className="mt-1 font-mono text-[11px] text-forge-ash">{record.generatedAgentsPath}</div>
                <div className="font-mono text-[11px] text-forge-ash">{record.generatedClaudePath}</div>
                <div className="font-mono text-[11px] text-forge-ash">{record.generatedBriefingPath}</div>
                <div className="font-mono text-[11px] text-forge-ash">{record.generatedCursorPath}</div>
              </div>
            </div>
          </Panel>

          <Panel title="Normalized Summary" subtitle="Structured digest extracted from context source.">
            <pre className="max-h-[360px] overflow-auto rounded border border-white/10 bg-black/30 p-3 text-[11px] text-forge-mist">
              {JSON.stringify(record.normalizedSummary, null, 2)}
            </pre>
          </Panel>

          <Panel title="Generated Briefing" subtitle="FORGE-owned durable briefing used in packet generation and handoffs.">
            <pre className="max-h-[420px] overflow-auto whitespace-pre-wrap rounded border border-white/10 bg-black/30 p-3 text-[11px] text-forge-mist">
              {record.briefingMarkdown}
            </pre>
          </Panel>
        </>
      )}
    </div>
  );
}
