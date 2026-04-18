import { GhostButton, Panel, PrimaryButton } from "@forge/ui";
import { useCallback, useEffect, useState } from "react";

import { api } from "../lib/api";
import { formatTime } from "../lib/format";
import { useUiStore } from "../stores/uiStore";

/** Matches `lanes.Lane` from forge-core. */
export type LaneRow = {
  id: string;
  createdAtMs: number;
  updatedAtMs: number;
  name: string;
  description: string;
  actionType: string;
  allowedPaths: string[];
  forbiddenPaths: string[];
  writeIntent: boolean;
  requiresApproval: boolean;
  riskClass: string;
  maxBytes: number;
  expectedArtifacts: string[];
  builtin: boolean;
  enabled: boolean;
};

function parseJsonOrLines(s: string): string[] {
  const t = s.trim();
  if (!t) return [];
  if (t.startsWith("[")) {
    try {
      const v = JSON.parse(t) as unknown;
      return Array.isArray(v) ? v.map((x) => String(x)) : [];
    } catch {
      /* fall through */
    }
  }
  return t
    .split(/[\n,]+/)
    .map((x) => x.trim())
    .filter(Boolean);
}

function pathsToText(arr: string[]) {
  return (arr ?? []).join("\n");
}

export function ActionLanesPage() {
  const setStatus = useUiStore((s) => s.setStatusLine);
  const [lanes, setLanes] = useState<LaneRow[]>([]);
  const [err, setErr] = useState<string | null>(null);
  const [newLane, setNewLane] = useState({
    id: "",
    name: "",
    description: "",
    actionType: "fs.read",
    allowedPaths: "",
    forbiddenPaths: "",
    riskClass: "read_only",
    writeIntent: false,
    requiresApproval: false,
    maxBytes: "0",
    expectedArtifacts: "",
  });

  const refresh = useCallback(async () => {
    try {
      const r = await api.actionLanes.list();
      setLanes(r.lanes as LaneRow[]);
      setErr(null);
    } catch (e) {
      setErr(e instanceof Error ? e.message : String(e));
    }
  }, []);

  useEffect(() => {
    void refresh();
  }, [refresh]);

  async function saveLane(ln: LaneRow) {
    try {
      const body = {
        id: ln.id,
        createdAtMs: ln.createdAtMs,
        updatedAtMs: ln.updatedAtMs,
        name: ln.name,
        description: ln.description,
        actionType: ln.actionType,
        allowedPaths: ln.allowedPaths,
        forbiddenPaths: ln.forbiddenPaths,
        writeIntent: ln.writeIntent,
        requiresApproval: ln.requiresApproval,
        riskClass: ln.riskClass,
        maxBytes: ln.maxBytes,
        expectedArtifacts: ln.expectedArtifacts,
        builtin: ln.builtin,
        enabled: ln.enabled,
      };
      await api.actionLanes.save(body);
      setStatus(`Saved lane ${ln.id}`);
      await refresh();
    } catch (e) {
      setErr(e instanceof Error ? e.message : String(e));
    }
  }

  async function toggleEnabled(ln: LaneRow) {
    await saveLane({ ...ln, enabled: !ln.enabled });
  }

  async function createCustomLane() {
    const id = newLane.id.trim();
    if (!id) {
      setErr("Lane id is required.");
      return;
    }
    if (!/^[a-z0-9._-]+$/i.test(id)) {
      setErr("Use letters, numbers, dots, underscores, hyphens only.");
      return;
    }
    const mb = Number(newLane.maxBytes);
    try {
      await api.actionLanes.save({
        id,
        createdAtMs: 0,
        updatedAtMs: 0,
        name: newLane.name.trim() || id,
        description: newLane.description.trim(),
        actionType: newLane.actionType.trim() || "fs.read",
        allowedPaths: parseJsonOrLines(newLane.allowedPaths),
        forbiddenPaths: parseJsonOrLines(newLane.forbiddenPaths),
        writeIntent: newLane.writeIntent,
        requiresApproval: newLane.requiresApproval,
        riskClass: newLane.riskClass.trim() || "read_only",
        maxBytes: Number.isFinite(mb) ? mb : 0,
        expectedArtifacts: parseJsonOrLines(newLane.expectedArtifacts),
        builtin: false,
        enabled: true,
      });
      setStatus(`Created lane ${id}`);
      setNewLane({
        id: "",
        name: "",
        description: "",
        actionType: "fs.read",
        allowedPaths: "",
        forbiddenPaths: "",
        riskClass: "read_only",
        writeIntent: false,
        requiresApproval: false,
        maxBytes: "0",
        expectedArtifacts: "",
      });
      await refresh();
    } catch (e) {
      setErr(e instanceof Error ? e.message : String(e));
    }
  }

  async function deleteLane(ln: LaneRow) {
    if (ln.builtin) {
      setErr("Built-in lanes cannot be deleted — disable instead.");
      return;
    }
    if (!window.confirm(`Delete custom lane ${ln.id}?`)) return;
    try {
      await api.actionLanes.delete(ln.id);
      setStatus(`Deleted lane ${ln.id}`);
      await refresh();
    } catch (e) {
      setErr(e instanceof Error ? e.message : String(e));
    }
  }

  return (
    <div className="space-y-6">
      <Panel
        title="Action lanes"
        subtitle="Each lane binds gateway tools to path scopes and risk. Built-in lanes can be disabled; custom lanes can be added or removed."
        actions={<GhostButton onClick={() => void refresh()}>Refresh</GhostButton>}
      >
        {err ? <div className="rounded-md border border-forge-ember/30 bg-forge-ember/10 p-3 text-sm text-forge-ash">{err}</div> : null}
      </Panel>

      <Panel title="Add custom lane" subtitle="Non-builtin id — used for gateway + policy. Paths are usually absolute or workspace roots from core meta.">
        <div className="grid gap-3 md:grid-cols-2">
          <label className="text-xs text-forge-mist">
            Id
            <input className="forge-input mt-1 font-mono text-xs" value={newLane.id} onChange={(e) => setNewLane((n) => ({ ...n, id: e.target.value }))} />
          </label>
          <label className="text-xs text-forge-mist">
            Name
            <input className="forge-input mt-1" value={newLane.name} onChange={(e) => setNewLane((n) => ({ ...n, name: e.target.value }))} />
          </label>
          <label className="text-xs text-forge-mist md:col-span-2">
            Description
            <input className="forge-input mt-1" value={newLane.description} onChange={(e) => setNewLane((n) => ({ ...n, description: e.target.value }))} />
          </label>
          <label className="text-xs text-forge-mist">
            Action type
            <input className="forge-input mt-1 font-mono text-xs" value={newLane.actionType} onChange={(e) => setNewLane((n) => ({ ...n, actionType: e.target.value }))} />
          </label>
          <label className="text-xs text-forge-mist">
            Risk class
            <input className="forge-input mt-1" value={newLane.riskClass} onChange={(e) => setNewLane((n) => ({ ...n, riskClass: e.target.value }))} />
          </label>
          <label className="text-xs text-forge-mist md:col-span-2">
            Allowed paths (one per line)
            <textarea className="forge-input mt-1 min-h-[72px] font-mono text-[11px]" value={newLane.allowedPaths} onChange={(e) => setNewLane((n) => ({ ...n, allowedPaths: e.target.value }))} />
          </label>
          <label className="text-xs text-forge-mist md:col-span-2">
            Forbidden paths (optional)
            <textarea className="forge-input mt-1 min-h-[56px] font-mono text-[11px]" value={newLane.forbiddenPaths} onChange={(e) => setNewLane((n) => ({ ...n, forbiddenPaths: e.target.value }))} />
          </label>
          <label className="text-xs text-forge-mist md:col-span-2">
            Expected artifacts (labels, one per line)
            <textarea className="forge-input mt-1 min-h-[48px] font-mono text-[11px]" value={newLane.expectedArtifacts} onChange={(e) => setNewLane((n) => ({ ...n, expectedArtifacts: e.target.value }))} />
          </label>
          <label className="text-xs text-forge-mist">
            Max bytes
            <input className="forge-input mt-1 font-mono" value={newLane.maxBytes} onChange={(e) => setNewLane((n) => ({ ...n, maxBytes: e.target.value }))} />
          </label>
          <div className="flex flex-col gap-2 text-xs text-forge-mist md:pt-6">
            <label className="flex items-center gap-2">
              <input type="checkbox" checked={newLane.writeIntent} onChange={(e) => setNewLane((n) => ({ ...n, writeIntent: e.target.checked }))} />
              Write intent
            </label>
            <label className="flex items-center gap-2">
              <input type="checkbox" checked={newLane.requiresApproval} onChange={(e) => setNewLane((n) => ({ ...n, requiresApproval: e.target.checked }))} />
              Requires approval
            </label>
          </div>
        </div>
        <div className="mt-3">
          <PrimaryButton onClick={() => void createCustomLane()}>Create lane</PrimaryButton>
        </div>
      </Panel>

      <div className="space-y-3">
        {lanes.map((ln) => (
          <div key={ln.id} className="rounded-lg border border-white/10 bg-forge-iron/40 p-4">
            <div className="flex flex-wrap items-start justify-between gap-2">
              <div>
                <div className="font-mono text-sm text-forge-ash">{ln.id}</div>
                <div className="text-sm text-forge-mist">{ln.name}</div>
              </div>
              <div className="flex flex-wrap items-center gap-2">
                <label className="flex items-center gap-2 text-[11px] text-forge-mist">
                  <input type="checkbox" checked={ln.enabled} onChange={() => void toggleEnabled(ln)} />
                  Enabled
                </label>
                <span className="text-[11px] text-forge-mist/70">updated {formatTime(ln.updatedAtMs)}</span>
              </div>
            </div>
            <p className="mt-2 text-xs text-forge-mist">{ln.description}</p>
            <div className="mt-2 flex flex-wrap gap-2 text-[11px] text-forge-mist">
              <span className="rounded border border-white/10 px-2 py-0.5">type {ln.actionType}</span>
              <span className="rounded border border-white/10 px-2 py-0.5">risk {ln.riskClass}</span>
              <span className="rounded border border-white/10 px-2 py-0.5">{ln.writeIntent ? "write-intent" : "read-only"}</span>
              <span className="rounded border border-white/10 px-2 py-0.5">{ln.requiresApproval ? "approval gate" : "no lane approval"}</span>
              <span className="rounded border border-white/10 px-2 py-0.5">{ln.builtin ? "builtin" : "custom"}</span>
              <span className="rounded border border-white/10 px-2 py-0.5">maxBytes {ln.maxBytes}</span>
            </div>
            <div className="mt-3 grid gap-2 md:grid-cols-2">
              <div className="rounded border border-white/10 bg-black/20 p-2">
                <div className="text-[10px] font-semibold text-forge-ash">allowedPaths</div>
                <pre className="mt-1 max-h-28 overflow-auto whitespace-pre-wrap font-mono text-[10px] text-forge-mist">{JSON.stringify(ln.allowedPaths, null, 2)}</pre>
              </div>
              <div className="rounded border border-white/10 bg-black/20 p-2">
                <div className="text-[10px] font-semibold text-forge-ash">forbiddenPaths / expectedArtifacts</div>
                <pre className="mt-1 max-h-28 overflow-auto font-mono text-[10px] text-forge-mist">
                  {JSON.stringify({ forbiddenPaths: ln.forbiddenPaths, expectedArtifacts: ln.expectedArtifacts }, null, 2)}
                </pre>
              </div>
            </div>
            {!ln.builtin ? (
              <div className="mt-3 flex flex-wrap gap-2">
                <GhostButton
                  onClick={() => {
                    const allowed = window.prompt("Allowed paths (JSON array or one path per line)", pathsToText(ln.allowedPaths));
                    if (allowed == null) return;
                    const forbidden = window.prompt("Forbidden paths (JSON array or lines)", pathsToText(ln.forbiddenPaths));
                    if (forbidden == null) return;
                    void saveLane({
                      ...ln,
                      allowedPaths: parseJsonOrLines(allowed),
                      forbiddenPaths: parseJsonOrLines(forbidden),
                    });
                  }}
                >
                  Edit paths…
                </GhostButton>
                <GhostButton onClick={() => void deleteLane(ln)}>Delete lane</GhostButton>
              </div>
            ) : (
              <p className="mt-3 text-[11px] text-forge-mist/55">Built-in scope is fixed in core; only Enabled can be toggled here.</p>
            )}
          </div>
        ))}
      </div>
    </div>
  );
}
