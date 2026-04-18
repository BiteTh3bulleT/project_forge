import { GhostButton, Panel, PrimaryButton } from "@forge/ui";
import { useEffect, useState } from "react";

import { api } from "../lib/api";
import { useUiStore } from "../stores/uiStore";

type ChecklistItem = { id: string; title: string; status: string; detail: string; category: string };

export function ReleasePage() {
  const setStatus = useUiStore((s) => s.setStatusLine);
  const [items, setItems] = useState<ChecklistItem[]>([]);
  const [ready, setReady] = useState(false);
  const [firstRun, setFirstRun] = useState<Record<string, unknown> | null>(null);
  const [artifacts, setArtifacts] = useState<unknown[]>([]);
  const [err, setErr] = useState<string | null>(null);
  const [recKind, setRecKind] = useState("desktop_bundle");
  const [recVersion, setRecVersion] = useState("");
  const [recSummary, setRecSummary] = useState("");

  async function refresh() {
    try {
      const [c, f, a] = await Promise.all([api.release.readiness(), api.release.firstRun(), api.release.artifacts(40)]);
      const cl = c.checklist as Record<string, unknown>;
      setItems((cl.items as ChecklistItem[]) ?? []);
      setReady(Boolean(cl.ready));
      setFirstRun(f.firstRun as Record<string, unknown>);
      setArtifacts(a.artifacts);
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
      <Panel title="Release readiness" subtitle="Local packaging gate: filesystem, migrations, lanes, permissions, first-run." actions={<GhostButton onClick={() => void refresh()}>Refresh</GhostButton>}>
        {err ? <div className="rounded-md border border-forge-ember/30 bg-forge-ember/10 p-3 text-sm text-forge-ash">{err}</div> : null}
        <div className="mt-2 text-sm text-forge-mist">
          Overall:{" "}
          <span className={ready ? "text-forge-ash" : "text-forge-emberSoft"}>{ready ? "READY (reported)" : "NOT READY"}</span>
        </div>
      </Panel>

      {firstRun ? (
        <Panel title="First-run summary" subtitle="Operator onboarding state from the core.">
          <pre className="max-h-48 overflow-auto rounded border border-white/10 bg-black/25 p-3 font-mono text-[11px] text-forge-mist">{JSON.stringify(firstRun, null, 2)}</pre>
        </Panel>
      ) : null}

      <Panel title="Readiness checklist" subtitle="Each item is independently inspectable.">
        <div className="space-y-2">
          {items.map((it) => (
            <div key={it.id} className="rounded border border-white/10 bg-forge-iron/30 p-3 text-xs text-forge-mist">
              <div className="flex flex-wrap justify-between gap-2">
                <span className="font-semibold text-forge-ash">{it.title}</span>
                <span className={it.status === "ok" ? "text-forge-ash" : it.status === "fail" ? "text-forge-emberSoft" : ""}>{it.status}</span>
              </div>
              <div className="mt-1 text-[11px]">{it.detail}</div>
              <div className="mt-1 text-[10px] uppercase tracking-wide text-forge-mist/70">{it.category}</div>
            </div>
          ))}
        </div>
      </Panel>

      <Panel title="Record release artifact" subtitle="Use after running your local packaging script (e.g. tauri build); this is bookkeeping, not the build itself.">
        <div className="grid gap-2 md:grid-cols-2">
          <input className="forge-input" value={recKind} onChange={(e) => setRecKind(e.target.value)} placeholder="kind" />
          <input className="forge-input" value={recVersion} onChange={(e) => setRecVersion(e.target.value)} placeholder="version tag" />
        </div>
        <textarea className="forge-input mt-3 min-h-[72px]" value={recSummary} onChange={(e) => setRecSummary(e.target.value)} placeholder="summary" />
        <div className="mt-3">
          <PrimaryButton
            onClick={async () => {
              await api.release.recordArtifact({
                kind: recKind,
                versionTag: recVersion,
                channel: "local",
                summary: recSummary,
                notes: "",
                checklist: [],
              });
              setStatus("Recorded release artifact.");
              await refresh();
            }}
          >
            Record
          </PrimaryButton>
        </div>
      </Panel>

      <Panel title="Recorded artifacts" subtitle="release_artifacts table — audit-friendly packaging history.">
        <div className="space-y-2">
          {artifacts.map((row, idx) => (
            <pre key={idx} className="max-h-36 overflow-auto rounded border border-white/10 bg-black/25 p-2 font-mono text-[10px] text-forge-mist">
              {JSON.stringify(row, null, 2)}
            </pre>
          ))}
        </div>
      </Panel>
    </div>
  );
}
