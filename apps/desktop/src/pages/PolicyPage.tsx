import type { ApprovalPreset, Dossier, ExecutionStrategy, PacketGuidance, PolicyRecommendation } from "@forge/shared";
import { GhostButton, Panel, PrimaryButton } from "@forge/ui";
import { useEffect, useState } from "react";

import { api } from "../lib/api";
import { formatTime } from "../lib/format";
import { useUiStore } from "../stores/uiStore";

export function PolicyPage() {
  const setStatus = useUiStore((s) => s.setStatusLine);
  const [presets, setPresets] = useState<ApprovalPreset[]>([]);
  const [globalPreset, setGlobalPreset] = useState("");
  const [recommendations, setRecommendations] = useState<PolicyRecommendation[]>([]);
  const [strategies, setStrategies] = useState<ExecutionStrategy[]>([]);
  const [dossiers, setDossiers] = useState<Dossier[]>([]);
  const [guidance, setGuidance] = useState<PacketGuidance[]>([]);
  const [taskType, setTaskType] = useState("repo_analysis");
  const [dossierId, setDossierId] = useState("");
  const [strategyId, setStrategyId] = useState("");
  const [objective, setObjective] = useState("Investigate repeated failures and suggest safer strategy.");
  const [packetId, setPacketId] = useState("");
  const [guidanceJobId, setGuidanceJobId] = useState("");
  const [guidanceDossierId, setGuidanceDossierId] = useState("");
  const [presetEditorId, setPresetEditorId] = useState("");
  const [presetEditorName, setPresetEditorName] = useState("");
  const [presetEditorDescription, setPresetEditorDescription] = useState("");
  const [presetEditorProfile, setPresetEditorProfile] = useState("{\n  \"autoRun\": {\n    \"read_only\": true,\n    \"external_reasoning\": true,\n    \"write_files\": false,\n    \"run_commands\": false\n  }\n}");
  const [err, setErr] = useState<string | null>(null);

  async function load() {
    try {
      const [presetRes, globalRes, recRes, stratRes, dossierRes, guidanceRes] = await Promise.all([
        api.policy.listPresets(80),
        api.policy.getGlobalPreset(),
        api.policy.listRecommendations({ limit: 120 }),
        api.strategies.list({ limit: 240 }),
        api.dossiers.list(180),
        api.packetGuidance.list({ limit: 120 }),
      ]);
      setPresets(presetRes.presets);
      setGlobalPreset(globalRes.presetId || "");
      setRecommendations(recRes.recommendations);
      setStrategies(stratRes.strategies);
      setDossiers(dossierRes.dossiers);
      setGuidance(guidanceRes.guidance);
      setErr(null);
    } catch (e) {
      setErr(e instanceof Error ? e.message : String(e));
    }
  }

  useEffect(() => {
    void load();
  }, []);

  function loadPresetIntoEditor(p: ApprovalPreset) {
    setPresetEditorId(p.id);
    setPresetEditorName(p.name);
    setPresetEditorDescription(p.description);
    setPresetEditorProfile(JSON.stringify(p.profile, null, 2));
  }

  return (
    <div className="space-y-6">
      <Panel
        title="Policy & Approval Profiles"
        subtitle="Routing policy recommendations and approval presets with explicit confidence, reasons, and evidence."
        actions={<GhostButton onClick={() => void load()}>Refresh</GhostButton>}
      >
        {err ? <div className="rounded border border-forge-ember/30 bg-forge-ember/10 p-3 text-sm text-forge-ash">{err}</div> : null}

        <div className="grid gap-3 md:grid-cols-2">
          <div>
            <label className="text-xs font-semibold tracking-wide text-forge-mist">Global approval preset</label>
            <select className="forge-input mt-1" value={globalPreset} onChange={(e) => setGlobalPreset(e.target.value)}>
              <option value="">(none)</option>
              {presets.map((p) => (
                <option key={p.id} value={p.id}>
                  {p.name} ({p.id})
                </option>
              ))}
            </select>
          </div>
          <div className="flex items-end">
            <PrimaryButton
              onClick={async () => {
                await api.policy.setGlobalPreset(globalPreset);
                setStatus(`Global preset set to ${globalPreset || "(none)"}.`);
                await load();
              }}
            >
              Apply Global Preset
            </PrimaryButton>
          </div>
        </div>
      </Panel>

      <Panel title="Approval Presets" subtitle="Editable policy profiles controlling auto-run boundaries and approval gates.">
        <div className="mb-3 grid gap-3 md:grid-cols-2">
          <div>
            <label className="text-xs text-forge-mist">Preset editor id</label>
            <input className="forge-input mt-1" value={presetEditorId} onChange={(e) => setPresetEditorId(e.target.value)} placeholder="balanced_custom" />
          </div>
          <div>
            <label className="text-xs text-forge-mist">Preset name</label>
            <input className="forge-input mt-1" value={presetEditorName} onChange={(e) => setPresetEditorName(e.target.value)} placeholder="Balanced Custom" />
          </div>
        </div>
        <div className="mb-3">
          <label className="text-xs text-forge-mist">Description</label>
          <input className="forge-input mt-1" value={presetEditorDescription} onChange={(e) => setPresetEditorDescription(e.target.value)} />
        </div>
        <div className="mb-3">
          <label className="text-xs text-forge-mist">Profile JSON</label>
          <textarea className="forge-input mt-1 min-h-[140px] font-mono text-[12px]" value={presetEditorProfile} onChange={(e) => setPresetEditorProfile(e.target.value)} />
        </div>
        <div className="mb-3 flex gap-2">
          <PrimaryButton
            onClick={async () => {
              try {
                const parsed = JSON.parse(presetEditorProfile);
                if (!parsed || typeof parsed !== "object" || Array.isArray(parsed)) {
                  setErr("Preset profile must be a JSON object.");
                  return;
                }
                await api.policy.savePreset({
                  id: presetEditorId.trim(),
                  name: presetEditorName.trim(),
                  description: presetEditorDescription.trim(),
                  profile: parsed,
                  editable: true,
                });
                setStatus(`Preset saved: ${presetEditorId.trim()}`);
                await load();
              } catch (e) {
                setErr(e instanceof Error ? e.message : String(e));
              }
            }}
          >
            Save Preset
          </PrimaryButton>
          <GhostButton
            onClick={() => {
              setPresetEditorId("");
              setPresetEditorName("");
              setPresetEditorDescription("");
              setPresetEditorProfile("{\n  \"autoRun\": {\n    \"read_only\": true,\n    \"external_reasoning\": true,\n    \"write_files\": false,\n    \"run_commands\": false\n  }\n}");
            }}
          >
            New Preset Draft
          </GhostButton>
        </div>
        {presets.length === 0 ? (
          <div className="text-sm text-forge-mist">No presets found.</div>
        ) : (
          <div className="space-y-2">
            {presets.map((p) => (
              <button
                key={p.id}
                type="button"
                className="w-full rounded border border-forge-platinum/10 bg-black/20 p-3 text-left text-xs text-forge-mist hover:border-forge-ember/35"
                onClick={() => loadPresetIntoEditor(p)}
              >
                <div className="font-semibold text-forge-ash">
                  {p.name} · {p.id}
                </div>
                <div className="mt-1">{p.description}</div>
                <div className="mt-1">editable {String(p.editable)} · updated {formatTime(p.updatedAtMs)}</div>
                <pre className="mt-2 max-h-40 overflow-auto rounded border border-forge-platinum/10 bg-black/30 p-2 text-[11px] text-forge-mist">
                  {JSON.stringify(p.profile, null, 2)}
                </pre>
              </button>
            ))}
          </div>
        )}
      </Panel>

      <Panel title="Routing Recommendation" subtitle="Compute policy recommendation for task + dossier + strategy constraints.">
        <div className="grid gap-3 md:grid-cols-2">
          <div>
            <label className="text-xs text-forge-mist">Task type</label>
            <input className="forge-input mt-1" value={taskType} onChange={(e) => setTaskType(e.target.value)} />
          </div>
          <div>
            <label className="text-xs text-forge-mist">Dossier (optional)</label>
            <select className="forge-input mt-1" value={dossierId} onChange={(e) => setDossierId(e.target.value)}>
              <option value="">(none)</option>
              {dossiers.map((d) => (
                <option key={d.id} value={String(d.id)}>
                  {d.id} - {d.name}
                </option>
              ))}
            </select>
          </div>
        </div>

        <div className="mt-3 grid gap-3 md:grid-cols-2">
          <div>
            <label className="text-xs text-forge-mist">Strategy override (optional)</label>
            <select className="forge-input mt-1" value={strategyId} onChange={(e) => setStrategyId(e.target.value)}>
              <option value="">(none)</option>
              {strategies.map((s) => (
                <option key={s.id} value={s.id}>
                  {s.id} - {s.name}
                </option>
              ))}
            </select>
          </div>
          <div>
            <label className="text-xs text-forge-mist">Objective</label>
            <input className="forge-input mt-1" value={objective} onChange={(e) => setObjective(e.target.value)} />
          </div>
        </div>

        <div className="mt-3 flex gap-2">
          <PrimaryButton
            onClick={async () => {
              const d = dossierId.trim() ? Number(dossierId.trim()) : undefined;
              const res = await api.policy.recommend({
                taskType,
                dossierId: Number.isFinite(d) ? d : undefined,
                strategyId: strategyId.trim() || undefined,
                objective,
              });
              setStatus(`Policy recommendation created: ${res.recommendation.id}`);
              await load();
            }}
          >
            Generate Recommendation
          </PrimaryButton>
        </div>
      </Panel>

      <Panel title="Recommendation History" subtitle="Persisted advisory records; operator override remains allowed.">
        {recommendations.length === 0 ? (
          <div className="text-sm text-forge-mist">No recommendations recorded.</div>
        ) : (
          <div className="space-y-2">
            {recommendations.map((r) => (
              <div key={r.id} className="rounded border border-forge-platinum/10 bg-black/20 p-3 text-xs text-forge-mist">
                <div className="font-semibold text-forge-ash">
                  #{r.id} · {r.taskType} → {r.targetAdapter}
                </div>
                <div className="mt-1">
                  strategy {r.strategyId ?? "n/a"} · retrieval {r.retrievalMode} · approval preset {r.approvalPresetId ?? "n/a"}
                </div>
                <div className="mt-1">
                  confidence {(r.confidence * 100).toFixed(1)}% · inferred {String(r.inferred)} · override {String(r.operatorOverrideAllowed)}
                </div>
                <div className="mt-1">reasons: {r.reasons.join(" | ")}</div>
                <pre className="mt-2 max-h-44 overflow-auto rounded border border-forge-platinum/10 bg-black/30 p-2 text-[11px] text-forge-mist">
                  {JSON.stringify({ evidence: r.evidence, packetShape: r.packetShape }, null, 2)}
                </pre>
              </div>
            ))}
          </div>
        )}
      </Panel>

      <Panel title="Packet Optimization Guidance" subtitle="Analyze packet quality without mutating packet contracts automatically.">
        <div className="grid gap-3 md:grid-cols-3">
          <div>
            <label className="text-xs text-forge-mist">Packet id</label>
            <input className="forge-input mt-1" value={packetId} onChange={(e) => setPacketId(e.target.value)} placeholder="123" />
          </div>
          <div>
            <label className="text-xs text-forge-mist">Job id (optional)</label>
            <input className="forge-input mt-1" value={guidanceJobId} onChange={(e) => setGuidanceJobId(e.target.value)} />
          </div>
          <div>
            <label className="text-xs text-forge-mist">Dossier id (optional)</label>
            <input className="forge-input mt-1" value={guidanceDossierId} onChange={(e) => setGuidanceDossierId(e.target.value)} />
          </div>
        </div>

        <div className="mt-3 flex gap-2">
          <PrimaryButton
            onClick={async () => {
              try {
                const pid = Number(packetId.trim());
                if (!Number.isFinite(pid) || pid <= 0) {
                  setErr("Packet id is required for guidance analysis.");
                  return;
                }
                const did = guidanceDossierId.trim() ? Number(guidanceDossierId.trim()) : undefined;
                await api.packetGuidance.analyze({
                  packetId: pid,
                  jobId: guidanceJobId.trim() || undefined,
                  dossierId: Number.isFinite(did) ? did : undefined,
                });
                setStatus(`Packet guidance computed for packet ${pid}.`);
                await load();
              } catch (e) {
                setErr(e instanceof Error ? e.message : String(e));
              }
            }}
          >
            Analyze Packet
          </PrimaryButton>
          <GhostButton onClick={() => void load()}>Reload Guidance</GhostButton>
        </div>

        <div className="mt-3 space-y-2">
          {guidance.length === 0 ? (
            <div className="text-sm text-forge-mist">No packet guidance records yet.</div>
          ) : (
            guidance.map((g) => (
              <div key={g.id} className="rounded border border-forge-platinum/10 bg-black/20 p-3 text-xs text-forge-mist">
                <div className="font-semibold text-forge-ash">
                  guidance #{g.id} · packet {g.packetId ?? "n/a"} · score {g.guidanceScore.toFixed(2)}
                </div>
                <div className="mt-1">issues: {g.issues.join(" | ") || "none"}</div>
                <div className="mt-1">recommendations: {g.recommendations.join(" | ") || "none"}</div>
                <pre className="mt-2 max-h-36 overflow-auto rounded border border-forge-platinum/10 bg-black/30 p-2 text-[11px] text-forge-mist">
                  {JSON.stringify(g.evidence, null, 2)}
                </pre>
              </div>
            ))
          )}
        </div>
      </Panel>
    </div>
  );
}
