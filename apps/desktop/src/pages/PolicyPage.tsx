import type {
  ApprovalPreset,
  Dossier,
  ExecutionStrategy,
  PacketGuidance,
  PolicyRecommendation,
} from "@forge/shared";
import { GhostButton, PrimaryButton } from "@forge/ui";
import { useEffect, useState } from "react";

import { AsyncState } from "../components/AsyncState";
import { HumanDataView } from "../components/HumanDataView";
import { OpsPanel } from "../components/OpsPanel";
import { api } from "../lib/api";
import { arrayOrEmpty } from "../lib/arrays";
import { formatTime } from "../lib/format";
import { useUiStore } from "../stores/uiStore";

export function PolicyPage() {
  const setStatus = useUiStore((s) => s.setStatusLine);
  const [presets, setPresets] = useState<ApprovalPreset[]>([]);
  const [globalPreset, setGlobalPreset] = useState("");
  const [recommendations, setRecommendations] = useState<
    PolicyRecommendation[]
  >([]);
  const [strategies, setStrategies] = useState<ExecutionStrategy[]>([]);
  const [dossiers, setDossiers] = useState<Dossier[]>([]);
  const [guidance, setGuidance] = useState<PacketGuidance[]>([]);
  const [taskType, setTaskType] = useState("repo_analysis");
  const [dossierId, setDossierId] = useState("");
  const [strategyId, setStrategyId] = useState("");
  const [objective, setObjective] = useState(
    "Investigate repeated failures and suggest safer strategy.",
  );
  const [packetId, setPacketId] = useState("");
  const [guidanceJobId, setGuidanceJobId] = useState("");
  const [guidanceDossierId, setGuidanceDossierId] = useState("");
  const [presetEditorId, setPresetEditorId] = useState("");
  const [presetEditorName, setPresetEditorName] = useState("");
  const [presetEditorDescription, setPresetEditorDescription] = useState("");
  const [presetEditorProfile, setPresetEditorProfile] = useState(
    defaultPresetProfileText(),
  );
  const [err, setErr] = useState<string | null>(null);

  async function load() {
    try {
      const [presetRes, globalRes, recRes, stratRes, dossierRes, guidanceRes] =
        await Promise.all([
          api.policy.listPresets(80),
          api.policy.getGlobalPreset(),
          api.policy.listRecommendations({ limit: 120 }),
          api.strategies.list({ limit: 240 }),
          api.dossiers.list(180),
          api.packetGuidance.list({ limit: 120 }),
        ]);
      setPresets(arrayOrEmpty<ApprovalPreset>(presetRes.presets));
      setGlobalPreset(globalRes.presetId || "");
      setRecommendations(
        arrayOrEmpty<PolicyRecommendation>(recRes.recommendations),
      );
      setStrategies(arrayOrEmpty<ExecutionStrategy>(stratRes.strategies));
      setDossiers(arrayOrEmpty<Dossier>(dossierRes.dossiers));
      setGuidance(arrayOrEmpty<PacketGuidance>(guidanceRes.guidance));
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
    setPresetEditorProfile(profileToReadableText(p.profile));
  }

  return (
    <div className="forge-ops-board space-y-5">
      <header className="flex flex-col gap-3 lg:flex-row lg:items-end lg:justify-between">
        <div>
          <div className="forge-ops-label">Governance Policy</div>
          <h1 className="mt-2 text-2xl font-semibold tracking-normal text-forge-ash sm:text-3xl">
            Policy command board
          </h1>
          <p className="mt-1 max-w-3xl text-sm leading-6 text-forge-mist/75">
            Approval presets, routing recommendations, and packet guidance stay
            advisory until operator or kernel admission.
          </p>
        </div>
        <div className="flex flex-wrap items-center gap-2">
          <span className={statusPillClass(globalPreset ? "ok" : "muted")}>
            {globalPreset || "no global preset"}
          </span>
          <GhostButton onClick={() => void load()}>Refresh</GhostButton>
        </div>
      </header>

      {err ? (
        <div className="forge-ops-panel p-3">
          <AsyncState error={err} />
        </div>
      ) : null}

      <section className="grid gap-3 sm:grid-cols-2 xl:grid-cols-4">
        <MetricTile
          label="Presets"
          value={String(presets.length)}
          detail="approval profiles"
          tone="muted"
        />
        <MetricTile
          label="Recommendations"
          value={String(recommendations.length)}
          detail="routing history"
          tone="ok"
        />
        <MetricTile
          label="Strategies"
          value={String(strategies.length)}
          detail="eligible plans"
          tone="muted"
        />
        <MetricTile
          label="Guidance"
          value={String(guidance.length)}
          detail="packet analyses"
          tone={guidance.length > 0 ? "warn" : "muted"}
        />
      </section>

      <OpsPanel
        title="Policy & Approval Profiles"
        subtitle="Routing policy recommendations and approval presets with explicit confidence, reasons, and evidence."
      >
        <div className="forge-ops-card grid gap-3 p-3 md:grid-cols-[minmax(0,1fr)_auto]">
          <div>
            <label className="forge-ops-label">Global approval preset</label>
            <select
              className="forge-input mt-1"
              value={globalPreset}
              onChange={(e) => setGlobalPreset(e.target.value)}
            >
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
              className="w-full md:w-auto"
              onClick={async () => {
                try {
                  await api.policy.setGlobalPreset(globalPreset);
                  setStatus(
                    `Global preset set to ${globalPreset || "(none)"}.`,
                  );
                  setErr(null);
                  await load();
                } catch (e) {
                  setErr(e instanceof Error ? e.message : String(e));
                }
              }}
            >
              Apply Global Preset
            </PrimaryButton>
          </div>
        </div>
      </OpsPanel>

      <OpsPanel
        title="Approval Presets"
        subtitle="Editable policy profiles controlling auto-run boundaries and approval gates."
      >
        <div className="forge-ops-card mb-3 p-3">
          <div className="grid gap-3 md:grid-cols-2">
            <div>
              <label className="forge-ops-label">Preset editor id</label>
              <input
                className="forge-input mt-1"
                value={presetEditorId}
                onChange={(e) => setPresetEditorId(e.target.value)}
                placeholder="balanced_custom"
              />
            </div>
            <div>
              <label className="forge-ops-label">Preset name</label>
              <input
                className="forge-input mt-1"
                value={presetEditorName}
                onChange={(e) => setPresetEditorName(e.target.value)}
                placeholder="Balanced Custom"
              />
            </div>
          </div>
          <div className="mt-3">
            <label className="forge-ops-label">Description</label>
            <input
              className="forge-input mt-1"
              value={presetEditorDescription}
              onChange={(e) => setPresetEditorDescription(e.target.value)}
            />
          </div>
          <div className="mt-3">
            <label className="forge-ops-label">Profile rules</label>
            <textarea
              className="forge-input mt-1 min-h-[140px] font-mono text-[12px]"
              value={presetEditorProfile}
              onChange={(e) => setPresetEditorProfile(e.target.value)}
            />
            <div className="mt-1 text-[11px] text-forge-mist/75">
              Use one rule per line, for example: autoRun.read_only: yes
            </div>
          </div>
          <div className="mt-3 flex flex-wrap gap-2">
            <PrimaryButton
              className="w-full sm:w-auto"
              onClick={async () => {
                try {
                  const parsed = parseReadableProfile(presetEditorProfile);
                  if (
                    !parsed ||
                    typeof parsed !== "object" ||
                    Array.isArray(parsed)
                  ) {
                    setErr(
                      "Preset profile must contain one or more profile rules.",
                    );
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
                setPresetEditorProfile(defaultPresetProfileText());
              }}
            >
              New Preset Draft
            </GhostButton>
          </div>
        </div>
        {presets.length === 0 ? (
          <EmptyState
            title="No approval presets"
            detail="Draft and save a preset to make policy boundaries selectable."
          />
        ) : (
          <div className="space-y-2">
            {presets.map((p) => (
              <button
                key={p.id}
                type="button"
                className="forge-ops-card w-full p-3 text-left text-xs text-forge-mist hover:border-forge-ember/35"
                onClick={() => loadPresetIntoEditor(p)}
              >
                <div className="font-semibold text-forge-ash">
                  {p.name} · {p.id}
                </div>
                <div className="mt-1">{p.description}</div>
                <div className="mt-1">
                  editable {String(p.editable)} · updated{" "}
                  {formatTime(p.updatedAtMs)}
                </div>
                <div className="mt-2 max-h-40 overflow-auto rounded border border-white/10 bg-black/30 p-2 text-[11px] text-forge-mist">
                  <HumanDataView value={p.profile} compact />
                </div>
              </button>
            ))}
          </div>
        )}
      </OpsPanel>

      <OpsPanel
        title="Routing Recommendation"
        subtitle="Compute policy recommendation for task + dossier + strategy constraints."
      >
        <div className="forge-ops-card p-3">
          <div className="grid gap-3 md:grid-cols-2">
            <div>
              <label className="forge-ops-label">Task type</label>
              <input
                className="forge-input mt-1"
                value={taskType}
                onChange={(e) => setTaskType(e.target.value)}
              />
            </div>
            <div>
              <label className="forge-ops-label">Dossier (optional)</label>
              <select
                className="forge-input mt-1"
                value={dossierId}
                onChange={(e) => setDossierId(e.target.value)}
              >
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
              <label className="forge-ops-label">
                Strategy override (optional)
              </label>
              <select
                className="forge-input mt-1"
                value={strategyId}
                onChange={(e) => setStrategyId(e.target.value)}
              >
                <option value="">(none)</option>
                {strategies.map((s) => (
                  <option key={s.id} value={s.id}>
                    {s.id} - {s.name}
                  </option>
                ))}
              </select>
            </div>
            <div>
              <label className="forge-ops-label">Objective</label>
              <input
                className="forge-input mt-1"
                value={objective}
                onChange={(e) => setObjective(e.target.value)}
              />
            </div>
          </div>

          <div className="mt-3 flex gap-2">
            <PrimaryButton
              className="w-full sm:w-auto"
              onClick={async () => {
                try {
                  const d = dossierId.trim()
                    ? Number(dossierId.trim())
                    : undefined;
                  const res = await api.policy.recommend({
                    taskType,
                    dossierId: Number.isFinite(d) ? d : undefined,
                    strategyId: strategyId.trim() || undefined,
                    objective,
                  });
                  setStatus(
                    `Policy recommendation created: ${res.recommendation.id}`,
                  );
                  setErr(null);
                  await load();
                } catch (e) {
                  setErr(e instanceof Error ? e.message : String(e));
                }
              }}
            >
              Generate Recommendation
            </PrimaryButton>
          </div>
        </div>
      </OpsPanel>

      <OpsPanel
        title="Recommendation History"
        subtitle="Persisted advisory records; operator override remains allowed."
      >
        {recommendations.length === 0 ? (
          <EmptyState
            title="No recommendations recorded"
            detail="Generate a routing recommendation to capture policy evidence and confidence."
          />
        ) : (
          <div className="space-y-2">
            {recommendations.map((r) => (
              <div
                key={r.id}
                className="forge-ops-card p-3 text-xs text-forge-mist"
              >
                <div className="font-semibold text-forge-ash">
                  #{r.id} · {r.taskType} → {r.targetAdapter}
                </div>
                <div className="mt-1">
                  strategy {r.strategyId ?? "n/a"} · retrieval {r.retrievalMode}{" "}
                  · approval preset {r.approvalPresetId ?? "n/a"}
                </div>
                <div className="mt-1">
                  confidence {(r.confidence * 100).toFixed(1)}% · inferred{" "}
                  {String(r.inferred)} · override{" "}
                  {String(r.operatorOverrideAllowed)}
                </div>
                <div className="mt-1">reasons: {r.reasons.join(" | ")}</div>
                <div className="mt-2 max-h-44 overflow-auto rounded border border-white/10 bg-black/30 p-2 text-[11px] text-forge-mist">
                  <HumanDataView
                    value={{ evidence: r.evidence, packetShape: r.packetShape }}
                    compact
                  />
                </div>
              </div>
            ))}
          </div>
        )}
      </OpsPanel>

      <OpsPanel
        title="Packet Optimization Guidance"
        subtitle="Analyze packet quality without mutating packet contracts automatically."
      >
        <div className="forge-ops-card p-3">
          <div className="grid gap-3 md:grid-cols-3">
            <div>
              <label className="forge-ops-label">Packet id</label>
              <input
                className="forge-input mt-1"
                value={packetId}
                onChange={(e) => setPacketId(e.target.value)}
                placeholder="123"
              />
            </div>
            <div>
              <label className="forge-ops-label">Job id (optional)</label>
              <input
                className="forge-input mt-1"
                value={guidanceJobId}
                onChange={(e) => setGuidanceJobId(e.target.value)}
              />
            </div>
            <div>
              <label className="forge-ops-label">Dossier id (optional)</label>
              <input
                className="forge-input mt-1"
                value={guidanceDossierId}
                onChange={(e) => setGuidanceDossierId(e.target.value)}
              />
            </div>
          </div>

          <div className="mt-3 flex gap-2">
            <PrimaryButton
              className="w-full sm:w-auto"
              onClick={async () => {
                try {
                  const pid = Number(packetId.trim());
                  if (!Number.isFinite(pid) || pid <= 0) {
                    setErr("Packet id is required for guidance analysis.");
                    return;
                  }
                  const did = guidanceDossierId.trim()
                    ? Number(guidanceDossierId.trim())
                    : undefined;
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
            <GhostButton onClick={() => void load()}>
              Reload Guidance
            </GhostButton>
          </div>
        </div>

        <div className="mt-3 space-y-2">
          {guidance.length === 0 ? (
            <EmptyState
              title="No packet guidance records"
              detail="Analyze a packet to record quality issues, recommendations, and supporting evidence."
            />
          ) : (
            guidance.map((g) => (
              <div
                key={g.id}
                className="forge-ops-card p-3 text-xs text-forge-mist"
              >
                <div className="font-semibold text-forge-ash">
                  guidance #{g.id} · packet {g.packetId ?? "n/a"} · score{" "}
                  {g.guidanceScore.toFixed(2)}
                </div>
                <div className="mt-1">
                  issues: {g.issues.join(" | ") || "none"}
                </div>
                <div className="mt-1">
                  recommendations: {g.recommendations.join(" | ") || "none"}
                </div>
                <div className="mt-2 max-h-36 overflow-auto rounded border border-white/10 bg-black/30 p-2 text-[11px] text-forge-mist">
                  <HumanDataView value={g.evidence} compact />
                </div>
              </div>
            ))
          )}
        </div>
      </OpsPanel>
    </div>
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
          <div className="mt-2 text-2xl font-semibold tracking-normal text-forge-ash">
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

function statusPillClass(status: string) {
  if (status === "ok" || status === "approved") {
    return "forge-ops-status forge-ops-status--ok";
  }
  if (status === "bad" || status === "rejected") {
    return "forge-ops-status forge-ops-status--bad";
  }
  if (status === "warn" || status === "pending" || status === "deferred") {
    return "forge-ops-status forge-ops-status--warn";
  }
  return "forge-ops-status forge-ops-status--muted";
}

function defaultPresetProfileText() {
  return [
    "autoRun.read_only: yes",
    "autoRun.external_reasoning: yes",
    "autoRun.write_files: no",
    "autoRun.run_commands: no",
  ].join("\n");
}

function profileToReadableText(value: unknown, prefix = ""): string {
  if (!value || typeof value !== "object" || Array.isArray(value)) return "";
  return Object.entries(value as Record<string, unknown>)
    .flatMap(([key, item]) => {
      const path = prefix ? `${prefix}.${key}` : key;
      if (item && typeof item === "object" && !Array.isArray(item)) {
        return profileToReadableText(item, path).split("\n").filter(Boolean);
      }
      return `${path}: ${formatProfileValue(item)}`;
    })
    .join("\n");
}

function formatProfileValue(value: unknown) {
  if (typeof value === "boolean") return value ? "yes" : "no";
  if (Array.isArray(value)) return value.join(", ");
  if (value == null) return "none";
  return String(value);
}

function parseReadableProfile(text: string): Record<string, unknown> {
  const root: Record<string, unknown> = {};
  const lines = text
    .split(/\r?\n/)
    .map((line) => line.trim())
    .filter((line) => line && !line.startsWith("#"));

  for (const line of lines) {
    const splitAt = line.indexOf(":");
    if (splitAt < 1)
      throw new Error(`Profile rule is missing a colon: ${line}`);
    const path = line.slice(0, splitAt).trim().split(".").filter(Boolean);
    if (path.length === 0)
      throw new Error(`Profile rule is missing a key: ${line}`);
    let cursor = root;
    for (const segment of path.slice(0, -1)) {
      const next = cursor[segment];
      if (!next || typeof next !== "object" || Array.isArray(next)) {
        cursor[segment] = {};
      }
      cursor = cursor[segment] as Record<string, unknown>;
    }
    cursor[path[path.length - 1]] = parseProfileValue(
      line.slice(splitAt + 1).trim(),
    );
  }

  return root;
}

function parseProfileValue(value: string): unknown {
  const normalized = value.trim().toLowerCase();
  if (normalized === "yes" || normalized === "true") return true;
  if (normalized === "no" || normalized === "false") return false;
  if (normalized === "none" || normalized === "null") return null;
  const numberValue = Number(value);
  if (value !== "" && Number.isFinite(numberValue)) return numberValue;
  if (value.includes(",")) {
    return value
      .split(",")
      .map((part) => part.trim())
      .filter(Boolean);
  }
  return value;
}
