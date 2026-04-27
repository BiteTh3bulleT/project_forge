import { GhostButton, Panel, PrimaryButton } from "@forge/ui";
import { useCallback, useEffect, useState } from "react";

import { HumanDataView } from "../components/HumanDataView";
import { api } from "../lib/api";
import { formatTime } from "../lib/format";
import { useUiStore } from "../stores/uiStore";

/** Matches `permissions.Profile` from forge-core. */
export type PermissionProfile = {
  id: string;
  createdAtMs: number;
  updatedAtMs: number;
  name: string;
  description: string;
  allowedReadPaths: string[];
  allowedWritePaths: string[];
  allowedExecutePaths: string[];
  forbiddenPaths: string[];
  allowedTools: string[];
  approvalRequiredRisks: string[];
  maxBytesPerWrite: number;
  allowNetwork: boolean;
  editable: boolean;
  active: boolean;
};

function linesToList(s: string): string[] {
  return s
    .split(/[\n,]+/)
    .map((x) => x.trim())
    .filter(Boolean);
}

function listToLines(arr: string[] | undefined): string {
  return (arr ?? []).join("\n");
}

function emptyForm(): Record<string, string | boolean | number> {
  return {
    id: "",
    name: "",
    description: "",
    allowedReadPaths: "",
    allowedWritePaths: "",
    allowedExecutePaths: "",
    forbiddenPaths: "",
    allowedTools: "",
    approvalRequiredRisks: "",
    maxBytesPerWrite: String(512 * 1024),
    allowNetwork: false,
    active: false,
  };
}

function profileToForm(p: PermissionProfile): Record<string, string | boolean | number> {
  return {
    id: p.id,
    name: p.name,
    description: p.description,
    allowedReadPaths: listToLines(p.allowedReadPaths),
    allowedWritePaths: listToLines(p.allowedWritePaths),
    allowedExecutePaths: listToLines(p.allowedExecutePaths),
    forbiddenPaths: listToLines(p.forbiddenPaths),
    allowedTools: listToLines(p.allowedTools),
    approvalRequiredRisks: listToLines(p.approvalRequiredRisks),
    maxBytesPerWrite: String(p.maxBytesPerWrite ?? 0),
    allowNetwork: p.allowNetwork,
    active: p.active,
  };
}

function formToProfile(f: Record<string, string | boolean | number>, existing?: PermissionProfile): Record<string, unknown> {
  const max = Number(String(f.maxBytesPerWrite));
  return {
    id: String(f.id).trim(),
    name: String(f.name).trim(),
    description: String(f.description).trim(),
    allowedReadPaths: linesToList(String(f.allowedReadPaths)),
    allowedWritePaths: linesToList(String(f.allowedWritePaths)),
    allowedExecutePaths: linesToList(String(f.allowedExecutePaths)),
    forbiddenPaths: linesToList(String(f.forbiddenPaths)),
    allowedTools: linesToList(String(f.allowedTools)),
    approvalRequiredRisks: linesToList(String(f.approvalRequiredRisks)),
    maxBytesPerWrite: Number.isFinite(max) ? max : 0,
    allowNetwork: Boolean(f.allowNetwork),
    editable: existing?.editable ?? true,
    active: Boolean(f.active),
    createdAtMs: existing?.createdAtMs ?? 0,
    updatedAtMs: existing?.updatedAtMs ?? 0,
  };
}

export function ExecutionPermissionsPage() {
  const setStatus = useUiStore((s) => s.setStatusLine);
  const [profiles, setProfiles] = useState<PermissionProfile[]>([]);
  const [active, setActive] = useState<PermissionProfile | null>(null);
  const [summary, setSummary] = useState<Record<string, unknown> | null>(null);
  const [err, setErr] = useState<string | null>(null);
  const [form, setForm] = useState<Record<string, string | boolean | number>>(emptyForm);
  const [editingExistingId, setEditingExistingId] = useState<string | null>(null);

  const refresh = useCallback(async () => {
    try {
      const r = await api.executionPermissions.profiles();
      setProfiles(r.profiles as PermissionProfile[]);
      setActive(r.active as PermissionProfile | null);
      setSummary(r.summary);
      setErr(null);
    } catch (e) {
      setErr(e instanceof Error ? e.message : String(e));
    }
  }, []);

  useEffect(() => {
    void refresh();
  }, [refresh]);

  function startNew() {
    setEditingExistingId(null);
    setForm(emptyForm());
    setStatus("New profile form — set id and paths, then Save.");
  }

  function startEdit(p: PermissionProfile) {
    setEditingExistingId(p.id);
    setForm(profileToForm(p));
  }

  async function saveProfile() {
    const id = String(form.id).trim();
    if (!id) {
      setErr("Profile id is required.");
      return;
    }
    const existing = profiles.find((x) => x.id === id);
    if (existing && !existing.editable) {
      setErr(`Profile ${id} is not editable in the database.`);
      return;
    }
    try {
      const body = formToProfile(form, existing ?? undefined);
      await api.executionPermissions.saveProfile(body);
      setStatus(`Saved profile ${id}.`);
      setErr(null);
      await refresh();
      setEditingExistingId(id);
    } catch (e) {
      setErr(e instanceof Error ? e.message : String(e));
    }
  }

  async function deleteProfile(p: PermissionProfile) {
    if (!p.editable) {
      setErr("Cannot delete a non-editable profile.");
      return;
    }
    if (p.active) {
      setErr("Deactivate or activate another profile before deleting.");
      return;
    }
    if (!window.confirm(`Delete permission profile ${p.id}?`)) return;
    try {
      await api.executionPermissions.deleteProfile(p.id);
      setStatus(`Deleted profile ${p.id}.`);
      if (editingExistingId === p.id) {
        setForm(emptyForm());
        setEditingExistingId(null);
      }
      await refresh();
    } catch (e) {
      setErr(e instanceof Error ? e.message : String(e));
    }
  }

  return (
    <div className="space-y-6">
      <Panel
        title="Execution permissions"
        subtitle="Gateway path and tool policy (distinct from routing policy). Exactly one profile should be active."
        actions={<GhostButton onClick={() => void refresh()}>Refresh</GhostButton>}
      >
        {err ? <div className="rounded-md border border-forge-ember/30 bg-forge-ember/10 p-3 text-sm text-forge-ash">{err}</div> : null}
        {summary ? (
          <div className="mt-3 max-h-32 overflow-auto rounded border border-white/10 bg-black/25 p-3 text-[11px] text-forge-mist">
            <HumanDataView value={summary} compact />
          </div>
        ) : null}
      </Panel>

      <Panel title="Active profile" subtitle="Used for every gateway invocation.">
        {active ? (
          <div className="space-y-2 text-sm text-forge-mist">
            <div className="font-mono text-forge-ash">{active.id}</div>
            <div>{active.name}</div>
            <div className="text-xs">Updated {formatTime(active.updatedAtMs)}</div>
            <div className="text-xs">Network: {active.allowNetwork ? "allowed" : "denied"} · Max write: {active.maxBytesPerWrite} bytes</div>
            <button type="button" className="text-xs text-forge-emberSoft underline" onClick={() => startEdit(active)}>
              Edit in form below
            </button>
          </div>
        ) : (
          <div className="text-sm text-forge-emberSoft">No active profile — activate one from the list.</div>
        )}
      </Panel>

      <Panel
        title="Create or edit profile"
        subtitle="Path and tool lists: one entry per line (commas also work). Saving upserts; checking Active will deactivate others."
        actions={
          <div className="flex flex-wrap gap-2">
            <GhostButton onClick={startNew}>New profile</GhostButton>
          </div>
        }
      >
        <div className="grid gap-3 md:grid-cols-2">
          <label className="block text-xs text-forge-mist">
            Id
            <input
              className="forge-input mt-1 font-mono text-xs"
              value={String(form.id)}
              onChange={(e) => setForm((f) => ({ ...f, id: e.target.value }))}
              disabled={editingExistingId != null}
            />
          </label>
          <label className="block text-xs text-forge-mist">
            Name
            <input className="forge-input mt-1" value={String(form.name)} onChange={(e) => setForm((f) => ({ ...f, name: e.target.value }))} />
          </label>
        </div>
        <label className="mt-3 block text-xs text-forge-mist">
          Description
          <input className="forge-input mt-1" value={String(form.description)} onChange={(e) => setForm((f) => ({ ...f, description: e.target.value }))} />
        </label>
        <div className="mt-3 grid gap-3 md:grid-cols-2">
          <label className="block text-xs text-forge-mist">
            Allowed read paths
            <textarea className="forge-input mt-1 min-h-[88px] font-mono text-[11px]" value={String(form.allowedReadPaths)} onChange={(e) => setForm((f) => ({ ...f, allowedReadPaths: e.target.value }))} />
          </label>
          <label className="block text-xs text-forge-mist">
            Allowed write paths
            <textarea className="forge-input mt-1 min-h-[88px] font-mono text-[11px]" value={String(form.allowedWritePaths)} onChange={(e) => setForm((f) => ({ ...f, allowedWritePaths: e.target.value }))} />
          </label>
          <label className="block text-xs text-forge-mist">
            Allowed execute paths
            <textarea className="forge-input mt-1 min-h-[88px] font-mono text-[11px]" value={String(form.allowedExecutePaths)} onChange={(e) => setForm((f) => ({ ...f, allowedExecutePaths: e.target.value }))} />
          </label>
          <label className="block text-xs text-forge-mist">
            Forbidden paths
            <textarea className="forge-input mt-1 min-h-[88px] font-mono text-[11px]" value={String(form.forbiddenPaths)} onChange={(e) => setForm((f) => ({ ...f, forbiddenPaths: e.target.value }))} />
          </label>
          <label className="block text-xs text-forge-mist md:col-span-2">
            Allowed tools (ids)
            <textarea className="forge-input mt-1 min-h-[72px] font-mono text-[11px]" value={String(form.allowedTools)} onChange={(e) => setForm((f) => ({ ...f, allowedTools: e.target.value }))} />
          </label>
          <label className="block text-xs text-forge-mist md:col-span-2">
            Approval required for risk classes
            <textarea className="forge-input mt-1 min-h-[56px] font-mono text-[11px]" value={String(form.approvalRequiredRisks)} onChange={(e) => setForm((f) => ({ ...f, approvalRequiredRisks: e.target.value }))} placeholder="read_only, safe_write, scoped_execute, privileged, dangerous — one per line"
            />
          </label>
        </div>
        <div className="mt-3 flex flex-wrap items-end gap-4">
          <label className="text-xs text-forge-mist">
            Max bytes per write
            <input
              className="forge-input mt-1 max-w-[12rem] font-mono"
              value={String(form.maxBytesPerWrite)}
              onChange={(e) => setForm((f) => ({ ...f, maxBytesPerWrite: e.target.value }))}
            />
          </label>
          <label className="flex items-center gap-2 text-xs text-forge-mist">
            <input type="checkbox" checked={Boolean(form.allowNetwork)} onChange={(e) => setForm((f) => ({ ...f, allowNetwork: e.target.checked }))} />
            Allow network
          </label>
          <label className="flex items-center gap-2 text-xs text-forge-mist">
            <input type="checkbox" checked={Boolean(form.active)} onChange={(e) => setForm((f) => ({ ...f, active: e.target.checked }))} />
            Set active on save
          </label>
        </div>
        <div className="mt-4 flex flex-wrap gap-2">
          <PrimaryButton onClick={() => void saveProfile()}>Save profile</PrimaryButton>
        </div>
      </Panel>

      <Panel title="Profiles" subtitle="Activate, edit, or delete editable profiles.">
        <div className="space-y-3">
          {profiles.map((p) => (
            <div key={p.id} className="flex flex-wrap items-center justify-between gap-3 rounded-lg border border-white/10 bg-forge-slate/20 p-3">
              <div className="min-w-0">
                <div className="font-mono text-sm text-forge-ash">{p.id}</div>
                <div className="text-xs text-forge-mist">{p.name}</div>
                {p.active ? <div className="mt-1 text-[11px] text-forge-emberSoft">ACTIVE</div> : null}
                {!p.editable ? <div className="mt-1 text-[10px] text-forge-mist/60">Built-in · not deletable</div> : null}
              </div>
              <div className="flex flex-wrap gap-2">
                <PrimaryButton
                  onClick={async () => {
                    await api.executionPermissions.activateProfile(p.id);
                    setStatus(`Activated ${p.id}`);
                    await refresh();
                  }}
                >
                  Activate
                </PrimaryButton>
                <GhostButton onClick={() => startEdit(p)} disabled={!p.editable}>
                  Edit
                </GhostButton>
                <GhostButton onClick={() => void deleteProfile(p)} disabled={!p.editable || p.active}>
                  Delete
                </GhostButton>
              </div>
            </div>
          ))}
        </div>
      </Panel>
    </div>
  );
}
