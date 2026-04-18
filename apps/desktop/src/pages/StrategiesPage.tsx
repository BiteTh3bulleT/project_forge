import type { ApprovalPreset, ExecutionStrategy } from "@forge/shared";
import { GhostButton, Panel, PrimaryButton } from "@forge/ui";
import { useEffect, useMemo, useState } from "react";

import { api } from "../lib/api";
import { formatTime } from "../lib/format";
import { useUiStore } from "../stores/uiStore";

function toCSV(v: string[]) {
  return v.join(", ");
}

function parseCSV(v: string): string[] {
  return v
    .split(",")
    .map((s) => s.trim())
    .filter(Boolean);
}

function jsonPretty(v: unknown) {
  return JSON.stringify(v ?? {}, null, 2);
}

function parseJSON(raw: string, field: string): Record<string, unknown> {
  try {
    const parsed = JSON.parse(raw);
    if (!parsed || typeof parsed !== "object" || Array.isArray(parsed)) {
      throw new Error(`${field} must be a JSON object`);
    }
    return parsed as Record<string, unknown>;
  } catch (e) {
    throw new Error(`${field} invalid JSON: ${e instanceof Error ? e.message : String(e)}`);
  }
}

export function StrategiesPage() {
  const setStatus = useUiStore((s) => s.setStatusLine);
  const [strategies, setStrategies] = useState<ExecutionStrategy[]>([]);
  const [presets, setPresets] = useState<ApprovalPreset[]>([]);
  const [selected, setSelected] = useState<string>("");
  const [id, setId] = useState("");
  const [name, setName] = useState("");
  const [taskType, setTaskType] = useState("");
  const [targetAdapter, setTargetAdapter] = useState("ollama");
  const [retrievalMode, setRetrievalMode] = useState("hybrid");
  const [approvalRequired, setApprovalRequired] = useState(true);
  const [approvalPresetId, setApprovalPresetId] = useState("balanced");
  const [expectedArtifacts, setExpectedArtifacts] = useState("task_packet, adapter_output, job_result");
  const [packetRules, setPacketRules] = useState("{\n  \"targetItems\": 8,\n  \"maxItems\": 14\n}");
  const [successCriteria, setSuccessCriteria] = useState("{\n  \"requiresSummary\": true\n}");
  const [retryGuidance, setRetryGuidance] = useState("{\n  \"maxRetries\": 2,\n  \"adjustment\": \"tighten_scope\"\n}");
  const [enabled, setEnabled] = useState(true);
  const [err, setErr] = useState<string | null>(null);

  async function load() {
    try {
      const [s, p] = await Promise.all([api.strategies.list({ limit: 240 }), api.policy.listPresets(80)]);
      setStrategies(s.strategies);
      setPresets(p.presets);
      setErr(null);
      if (!selected && s.strategies.length > 0) {
        selectStrategy(s.strategies[0]);
      }
    } catch (e) {
      setErr(e instanceof Error ? e.message : String(e));
    }
  }

  function selectStrategy(row: ExecutionStrategy) {
    setSelected(row.id);
    setId(row.id);
    setName(row.name);
    setTaskType(row.taskType);
    setTargetAdapter(row.targetAdapter);
    setRetrievalMode(row.retrievalMode);
    setApprovalRequired(row.approvalRequired);
    setApprovalPresetId(row.approvalPresetId ?? "");
    setExpectedArtifacts(toCSV(row.expectedArtifacts));
    setPacketRules(jsonPretty(row.packetRules));
    setSuccessCriteria(jsonPretty(row.successCriteria));
    setRetryGuidance(jsonPretty(row.retryGuidance));
    setEnabled(row.enabled);
  }

  useEffect(() => {
    void load();
  }, []);

  const presetOptions = useMemo(() => presets.map((p) => p.id), [presets]);

  return (
    <div className="space-y-6">
      <Panel
        title="Execution Strategies"
        subtitle="Reusable execution contracts that define adapter, retrieval, packet rules, approval needs, and success expectations."
        actions={<GhostButton onClick={() => void load()}>Refresh</GhostButton>}
      >
        {err ? <div className="rounded border border-forge-ember/30 bg-forge-ember/10 p-3 text-sm text-forge-ash">{err}</div> : null}
        <div className="grid gap-3 md:grid-cols-3">
          <div>
            <label className="text-xs font-semibold tracking-wide text-forge-mist">Strategy id</label>
            <input className="forge-input mt-1" value={id} onChange={(e) => setId(e.target.value)} placeholder="e.g. repo_analysis" />
          </div>
          <div>
            <label className="text-xs font-semibold tracking-wide text-forge-mist">Name</label>
            <input className="forge-input mt-1" value={name} onChange={(e) => setName(e.target.value)} />
          </div>
          <div>
            <label className="text-xs font-semibold tracking-wide text-forge-mist">Task type</label>
            <input className="forge-input mt-1" value={taskType} onChange={(e) => setTaskType(e.target.value)} placeholder="repo_analysis" />
          </div>
        </div>

        <div className="mt-3 grid gap-3 md:grid-cols-4">
          <div>
            <label className="text-xs text-forge-mist">Adapter</label>
            <input className="forge-input mt-1" value={targetAdapter} onChange={(e) => setTargetAdapter(e.target.value)} />
          </div>
          <div>
            <label className="text-xs text-forge-mist">Retrieval mode</label>
            <select className="forge-input mt-1" value={retrievalMode} onChange={(e) => setRetrievalMode(e.target.value)}>
              <option value="keyword">keyword</option>
              <option value="semantic">semantic</option>
              <option value="hybrid">hybrid</option>
            </select>
          </div>
          <div>
            <label className="text-xs text-forge-mist">Approval preset</label>
            <select className="forge-input mt-1" value={approvalPresetId} onChange={(e) => setApprovalPresetId(e.target.value)}>
              <option value="">(none)</option>
              {presetOptions.map((p) => (
                <option key={p} value={p}>
                  {p}
                </option>
              ))}
            </select>
          </div>
          <div className="flex items-end gap-3 pb-2">
            <label className="flex items-center gap-2 text-xs text-forge-mist">
              <input type="checkbox" checked={approvalRequired} onChange={(e) => setApprovalRequired(e.target.checked)} />
              Approval required
            </label>
            <label className="flex items-center gap-2 text-xs text-forge-mist">
              <input type="checkbox" checked={enabled} onChange={(e) => setEnabled(e.target.checked)} />
              Enabled
            </label>
          </div>
        </div>

        <div className="mt-3">
          <label className="text-xs text-forge-mist">Expected artifacts (comma separated)</label>
          <input className="forge-input mt-1" value={expectedArtifacts} onChange={(e) => setExpectedArtifacts(e.target.value)} />
        </div>

        <div className="mt-3 grid gap-3 md:grid-cols-3">
          <JsonField label="Packet rules" value={packetRules} onChange={setPacketRules} />
          <JsonField label="Success criteria" value={successCriteria} onChange={setSuccessCriteria} />
          <JsonField label="Retry guidance" value={retryGuidance} onChange={setRetryGuidance} />
        </div>

        <div className="mt-3 flex gap-2">
          <PrimaryButton
            onClick={async () => {
              try {
                const payload = {
                  id: id.trim(),
                  name,
                  taskType,
                  targetAdapter,
                  retrievalMode,
                  packetRules: parseJSON(packetRules, "packetRules"),
                  approvalRequired,
                  approvalPresetId: approvalPresetId.trim() || undefined,
                  expectedArtifacts: parseCSV(expectedArtifacts),
                  successCriteria: parseJSON(successCriteria, "successCriteria"),
                  retryGuidance: parseJSON(retryGuidance, "retryGuidance"),
                  enabled,
                };
                const res = await api.strategies.save(payload);
                setStatus(`Strategy saved: ${res.strategy.id}`);
                await load();
                selectStrategy(res.strategy);
              } catch (e) {
                setErr(e instanceof Error ? e.message : String(e));
              }
            }}
          >
            Save Strategy
          </PrimaryButton>
          <GhostButton
            onClick={() => {
              setSelected("");
              setId("");
              setName("");
              setTaskType("");
              setTargetAdapter("ollama");
              setRetrievalMode("hybrid");
              setApprovalRequired(true);
              setApprovalPresetId("balanced");
              setExpectedArtifacts("task_packet, adapter_output, job_result");
              setPacketRules("{\n  \"targetItems\": 8,\n  \"maxItems\": 14\n}");
              setSuccessCriteria("{\n  \"requiresSummary\": true\n}");
              setRetryGuidance("{\n  \"maxRetries\": 2\n}");
              setEnabled(true);
            }}
          >
            New Strategy Draft
          </GhostButton>
        </div>
      </Panel>

      <Panel title="Strategy Inventory" subtitle="Persisted strategy records used by routing policy and operator selection.">
        {strategies.length === 0 ? (
          <div className="text-sm text-forge-mist">No strategies found.</div>
        ) : (
          <div className="space-y-2">
            {strategies.map((s) => (
              <button
                key={s.id}
                type="button"
                onClick={() => selectStrategy(s)}
                className={[
                  "w-full rounded border px-3 py-2 text-left",
                  selected === s.id ? "border-forge-ember/40 bg-black/30" : "border-white/10 bg-black/20 hover:border-forge-ember/35",
                ].join(" ")}
              >
                <div className="text-sm font-semibold text-forge-ash">{s.name}</div>
                <div className="mt-1 text-xs text-forge-mist">
                  {s.id} · {s.taskType} · {s.targetAdapter} · {s.retrievalMode}
                </div>
                <div className="mt-1 text-[11px] text-forge-mist">
                  approval {String(s.approvalRequired)} · preset {s.approvalPresetId ?? "none"} · enabled {String(s.enabled)}
                </div>
                <div className="mt-1 text-[11px] text-forge-mist">updated {formatTime(s.updatedAtMs)}</div>
              </button>
            ))}
          </div>
        )}
      </Panel>
    </div>
  );
}

function JsonField(props: { label: string; value: string; onChange: (v: string) => void }) {
  return (
    <div>
      <label className="text-xs text-forge-mist">{props.label}</label>
      <textarea className="forge-input mt-1 min-h-[140px] font-mono text-[12px]" value={props.value} onChange={(e) => props.onChange(e.target.value)} />
    </div>
  );
}
