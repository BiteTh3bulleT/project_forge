import { GhostButton, PrimaryButton } from "@forge/ui";
import { useCallback, useEffect, useState, type ReactNode } from "react";

import { HumanDataView } from "../components/HumanDataView";
import { api } from "../lib/api";
import { arrayOrEmpty } from "../lib/arrays";
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

type AuthorityState = {
  status: "loading" | "fresh" | "stale" | "unavailable";
  lastLoadedAtMs: number | null;
};

function linesToList(s: string): string[] {
  return s
    .split(/[\n,]+/)
    .map((x) => x.trim())
    .filter(Boolean);
}

function listToLines(arr: string[] | undefined): string {
  return arrayOrEmpty<string>(arr).join("\n");
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

function profileToForm(
  p: PermissionProfile,
): Record<string, string | boolean | number> {
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

function formToProfile(
  f: Record<string, string | boolean | number>,
  existing?: PermissionProfile,
): Record<string, unknown> {
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
  const [authority, setAuthority] = useState<AuthorityState>({
    status: "loading",
    lastLoadedAtMs: null,
  });
  const [form, setForm] =
    useState<Record<string, string | boolean | number>>(emptyForm);
  const [editingExistingId, setEditingExistingId] = useState<string | null>(
    null,
  );

  const refresh = useCallback(async () => {
    try {
      const r = await api.executionPermissions.profiles();
      setProfiles(arrayOrEmpty<PermissionProfile>(r.profiles));
      setActive(r.active as PermissionProfile | null);
      setSummary(r.summary);
      setErr(null);
      setAuthority({ status: "fresh", lastLoadedAtMs: Date.now() });
    } catch (e) {
      setErr(e instanceof Error ? e.message : String(e));
      setAuthority((current) => ({
        status: current.lastLoadedAtMs == null ? "unavailable" : "stale",
        lastLoadedAtMs: current.lastLoadedAtMs,
      }));
    }
  }, []);

  useEffect(() => {
    void refresh();
  }, [refresh]);

  const canMutateAuthority = authority.status === "fresh";
  const authorityLabel = authorityStatusLabel(authority, active);
  const authorityTone =
    authority.status === "fresh" && active
      ? "ok"
      : authority.status === "loading" || authority.status === "stale"
        ? "warn"
        : "bad";

  function startNew() {
    if (!canMutateAuthority) {
      setErr("Permission authority is not fresh; refresh before editing.");
      return;
    }
    setEditingExistingId(null);
    setForm(emptyForm());
    setStatus("New profile form — set id and paths, then Save.");
  }

  function startEdit(p: PermissionProfile) {
    if (!canMutateAuthority) {
      setErr("Permission authority is not fresh; refresh before editing.");
      return;
    }
    setEditingExistingId(p.id);
    setForm(profileToForm(p));
  }

  async function saveProfile() {
    if (!canMutateAuthority) {
      setErr("Permission authority is not fresh; refresh before saving.");
      return;
    }
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
    if (!canMutateAuthority) {
      setErr("Permission authority is not fresh; refresh before deleting.");
      return;
    }
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
    <div className="forge-ops-board space-y-5">
      <header className="flex flex-col gap-3 lg:flex-row lg:items-end lg:justify-between">
        <div>
          <div className="forge-ops-label">Gateway Authority</div>
          <h1 className="mt-2 text-2xl font-semibold tracking-normal text-forge-ash sm:text-3xl">
            Execution permissions board
          </h1>
          <p className="mt-1 max-w-3xl text-sm leading-6 text-forge-mist/75">
            Gateway path, network, tool, and write limits remain separate from
            routing policy. Exactly one profile should be active.
          </p>
        </div>
        <div className="flex flex-wrap items-center gap-2">
          <span className={statusPillClass(authorityTone)}>
            {authorityLabel}
          </span>
          <GhostButton onClick={() => void refresh()}>Refresh</GhostButton>
        </div>
      </header>

      {authority.status === "unavailable" || authority.status === "stale" ? (
        <div className="forge-ops-panel border-forge-ember/30 bg-forge-ember/10 p-3 text-sm text-forge-ash">
          {authority.status === "stale"
            ? "Permission authority stale"
            : "Permission authority unavailable"}
          {err ? `: ${err}` : ""}
        </div>
      ) : null}

      {err && authority.status === "fresh" ? (
        <div className="forge-ops-panel border-forge-ember/30 bg-forge-ember/10 p-3 text-sm text-forge-ash">
          {err}
        </div>
      ) : null}

      <section className="grid gap-3 sm:grid-cols-2 xl:grid-cols-4">
        <MetricTile
          label="Profiles"
          value={String(profiles.length)}
          detail="configured policies"
          tone="muted"
        />
        <MetricTile
          label="Active"
          value={authority.status === "unavailable" ? "unknown" : active ? "1" : "0"}
          detail={
            authority.status === "unavailable"
              ? "authority unavailable"
              : active?.name ?? "none"
          }
          tone={authority.status === "fresh" && active ? "ok" : "bad"}
        />
        <MetricTile
          label="Network"
          value={
            authority.status === "unavailable"
              ? "unknown"
              : active?.allowNetwork
                ? "allowed"
                : "denied"
          }
          detail={
            authority.status === "fresh"
              ? "active profile"
              : "authority not fresh"
          }
          tone={
            authority.status === "fresh"
              ? active?.allowNetwork
                ? "warn"
                : "ok"
              : "bad"
          }
        />
        <MetricTile
          label="Max Write"
          value={
            authority.status === "unavailable"
              ? "unknown"
              : String(active?.maxBytesPerWrite ?? 0)
          }
          detail={
            authority.status === "fresh"
              ? "bytes per write"
              : "authority not fresh"
          }
          tone="muted"
        />
      </section>

      <OpsPanel
        title="Execution permissions"
        subtitle="Gateway path and tool policy summary."
      >
        {summary ? (
          <div className="mt-3 max-h-32 overflow-auto rounded border border-white/10 bg-black/25 p-3 text-[11px] text-forge-mist">
            <HumanDataView value={summary} compact />
          </div>
        ) : null}
      </OpsPanel>

      <OpsPanel
        title="Active profile"
        subtitle="Used for every gateway invocation."
      >
        {active ? (
          <div className="forge-ops-card p-3 text-sm text-forge-mist">
            <div className="flex flex-wrap items-start justify-between gap-2">
              <div>
                <div className="font-mono text-forge-ash">{active.id}</div>
                <div className="mt-1">{active.name}</div>
              </div>
              <span className={statusPillClass("ok")}>active</span>
            </div>
            <div className="mt-2 text-xs">
              Updated {formatTime(active.updatedAtMs)}
            </div>
            <div className="text-xs">
              Network: {active.allowNetwork ? "allowed" : "denied"} · Max write:{" "}
              {active.maxBytesPerWrite} bytes
            </div>
            <button
              type="button"
              className="text-xs text-forge-emberSoft underline"
              disabled={!canMutateAuthority}
              onClick={() => startEdit(active)}
            >
              Edit in form below
            </button>
          </div>
        ) : authority.status !== "fresh" ? (
          <div className="text-sm text-forge-emberSoft">
            Permission authority {authority.status} — refresh before relying on
            profile state.
          </div>
        ) : (
          <div className="text-sm text-forge-emberSoft">
            No active profile — activate one from the list.
          </div>
        )}
      </OpsPanel>

      <OpsPanel
        title="Create or edit profile"
        subtitle="Path and tool lists: one entry per line (commas also work). Saving upserts; checking Active will deactivate others."
      >
        <div className="mb-3 flex flex-wrap items-center justify-between gap-2">
          <div>
            <div className="forge-ops-label">
              {editingExistingId ? "Editing profile" : "Draft profile"}
            </div>
            <div className="mt-1 text-xs text-forge-mist/70">
              {editingExistingId ??
                "Set an id, limits, and path boundaries before saving."}
            </div>
          </div>
          <GhostButton onClick={startNew} disabled={!canMutateAuthority}>
            New profile
          </GhostButton>
        </div>
        <div className="forge-ops-card p-3">
          <div className="grid gap-3 md:grid-cols-2">
            <label className="block text-xs font-semibold tracking-wide text-forge-mist">
              Id
              <input
                className="forge-input mt-1 font-mono text-xs"
                value={String(form.id)}
                onChange={(e) => setForm((f) => ({ ...f, id: e.target.value }))}
                disabled={editingExistingId != null || !canMutateAuthority}
              />
            </label>
            <label className="block text-xs font-semibold tracking-wide text-forge-mist">
              Name
              <input
                className="forge-input mt-1"
                value={String(form.name)}
                onChange={(e) =>
                  setForm((f) => ({ ...f, name: e.target.value }))
                }
                disabled={!canMutateAuthority}
              />
            </label>
          </div>
          <label className="mt-3 block text-xs font-semibold tracking-wide text-forge-mist">
            Description
            <input
              className="forge-input mt-1"
              value={String(form.description)}
              onChange={(e) =>
                setForm((f) => ({ ...f, description: e.target.value }))
              }
              disabled={!canMutateAuthority}
            />
          </label>
          <div className="mt-3 grid gap-3 md:grid-cols-2">
            <label className="block text-xs font-semibold tracking-wide text-forge-mist">
              Allowed read paths
              <textarea
                className="forge-input mt-1 min-h-[88px] font-mono text-[11px]"
                value={String(form.allowedReadPaths)}
                onChange={(e) =>
                  setForm((f) => ({ ...f, allowedReadPaths: e.target.value }))
                }
                disabled={!canMutateAuthority}
              />
            </label>
            <label className="block text-xs font-semibold tracking-wide text-forge-mist">
              Allowed write paths
              <textarea
                className="forge-input mt-1 min-h-[88px] font-mono text-[11px]"
                value={String(form.allowedWritePaths)}
                onChange={(e) =>
                  setForm((f) => ({ ...f, allowedWritePaths: e.target.value }))
                }
                disabled={!canMutateAuthority}
              />
            </label>
            <label className="block text-xs font-semibold tracking-wide text-forge-mist">
              Allowed execute paths
              <textarea
                className="forge-input mt-1 min-h-[88px] font-mono text-[11px]"
                value={String(form.allowedExecutePaths)}
                onChange={(e) =>
                  setForm((f) => ({
                    ...f,
                    allowedExecutePaths: e.target.value,
                  }))
                }
                disabled={!canMutateAuthority}
              />
            </label>
            <label className="block text-xs font-semibold tracking-wide text-forge-mist">
              Forbidden paths
              <textarea
                className="forge-input mt-1 min-h-[88px] font-mono text-[11px]"
                value={String(form.forbiddenPaths)}
                onChange={(e) =>
                  setForm((f) => ({ ...f, forbiddenPaths: e.target.value }))
                }
                disabled={!canMutateAuthority}
              />
            </label>
            <label className="block text-xs font-semibold tracking-wide text-forge-mist md:col-span-2">
              Allowed tools (ids)
              <textarea
                className="forge-input mt-1 min-h-[72px] font-mono text-[11px]"
                value={String(form.allowedTools)}
                onChange={(e) =>
                  setForm((f) => ({ ...f, allowedTools: e.target.value }))
                }
                disabled={!canMutateAuthority}
              />
            </label>
            <label className="block text-xs font-semibold tracking-wide text-forge-mist md:col-span-2">
              Approval required for risk classes
              <textarea
                className="forge-input mt-1 min-h-[56px] font-mono text-[11px]"
                value={String(form.approvalRequiredRisks)}
                onChange={(e) =>
                  setForm((f) => ({
                    ...f,
                    approvalRequiredRisks: e.target.value,
                  }))
                }
                placeholder="read_only, safe_write, scoped_execute, privileged, dangerous — one per line"
                disabled={!canMutateAuthority}
              />
            </label>
          </div>
          <div className="mt-3 flex flex-wrap items-end gap-4">
            <label className="text-xs text-forge-mist">
              Max bytes per write
              <input
                className="forge-input mt-1 max-w-[12rem] font-mono"
                value={String(form.maxBytesPerWrite)}
                onChange={(e) =>
                  setForm((f) => ({ ...f, maxBytesPerWrite: e.target.value }))
                }
                disabled={!canMutateAuthority}
              />
            </label>
            <label className="flex items-center gap-2 text-xs text-forge-mist">
              <input
                type="checkbox"
                checked={Boolean(form.allowNetwork)}
                onChange={(e) =>
                  setForm((f) => ({ ...f, allowNetwork: e.target.checked }))
                }
                disabled={!canMutateAuthority}
              />
              Allow network
            </label>
            <label className="flex items-center gap-2 text-xs text-forge-mist">
              <input
                type="checkbox"
                checked={Boolean(form.active)}
                onChange={(e) =>
                  setForm((f) => ({ ...f, active: e.target.checked }))
                }
                disabled={!canMutateAuthority}
              />
              Set active on save
            </label>
          </div>
          <div className="mt-4 flex flex-wrap gap-2">
            <PrimaryButton
              className="w-full sm:w-auto"
              onClick={() => void saveProfile()}
              disabled={!canMutateAuthority}
            >
              Save profile
            </PrimaryButton>
          </div>
        </div>
      </OpsPanel>

      <OpsPanel
        title="Profiles"
        subtitle="Activate, edit, or delete editable profiles."
      >
        {profiles.length === 0 ? (
          <EmptyState
            title={
              authority.status === "fresh"
                ? "No permission profiles"
                : "Permission profiles unavailable"
            }
            detail={
              authority.status === "fresh"
                ? "Create a profile to define gateway path, network, tool, and write limits."
                : "Refresh after core permission authority is reachable before creating or editing profiles."
            }
          />
        ) : (
          <div className="space-y-3">
            {profiles.map((p) => (
              <div
                key={p.id}
                className="forge-ops-card flex flex-wrap items-center justify-between gap-3 p-3"
              >
                <div className="min-w-0">
                  <div className="flex flex-wrap items-center gap-2">
                    <div className="font-mono text-sm text-forge-ash">
                      {p.id}
                    </div>
                    {p.active ? (
                      <span className={statusPillClass("ok")}>active</span>
                    ) : null}
                    {!p.editable ? (
                      <span className={statusPillClass("muted")}>built-in</span>
                    ) : null}
                  </div>
                  <div className="text-xs text-forge-mist">{p.name}</div>
                  <div className="mt-1 text-[10px] text-forge-mist/60">
                    Updated {formatTime(p.updatedAtMs)} · network{" "}
                    {p.allowNetwork ? "allowed" : "denied"}
                  </div>
                </div>
                <div className="flex flex-wrap gap-2">
                  <PrimaryButton
                    disabled={!canMutateAuthority}
                    onClick={async () => {
                      if (!canMutateAuthority) return;
                      await api.executionPermissions.activateProfile(p.id);
                      setStatus(`Activated ${p.id}`);
                      await refresh();
                    }}
                  >
                    Activate
                  </PrimaryButton>
                  <GhostButton
                    onClick={() => startEdit(p)}
                    disabled={!canMutateAuthority || !p.editable}
                  >
                    Edit
                  </GhostButton>
                  <GhostButton
                    onClick={() => void deleteProfile(p)}
                    disabled={!canMutateAuthority || !p.editable || p.active}
                  >
                    Delete
                  </GhostButton>
                </div>
              </div>
            ))}
          </div>
        )}
      </OpsPanel>
    </div>
  );
}

function OpsPanel(props: {
  title: string;
  subtitle: string;
  children: ReactNode;
}) {
  return (
    <section className="forge-ops-panel">
      <div className="forge-ops-panel__head">
        <div>
          <div className="forge-ops-title">{props.title}</div>
          <div className="mt-1 text-xs text-forge-mist/65">
            {props.subtitle}
          </div>
        </div>
      </div>
      <div className="forge-ops-panel__body">{props.children}</div>
    </section>
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
          <div className="mt-2 truncate text-2xl font-semibold tracking-normal text-forge-ash">
            {props.value}
          </div>
        </div>
        <span className={statusPillClass(props.tone)}>{props.tone}</span>
      </div>
      <div className="mt-3 truncate text-xs text-forge-mist/65">
        {props.detail}
      </div>
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

function authorityStatusLabel(
  authority: AuthorityState,
  active: PermissionProfile | null,
) {
  switch (authority.status) {
    case "loading":
      return "authority loading";
    case "unavailable":
      return "authority unavailable";
    case "stale":
      return "authority stale";
    case "fresh":
      return active ? active.id : "no active profile";
  }
}

function statusPillClass(status: string) {
  if (status === "ok" || status === "active") {
    return "forge-ops-status forge-ops-status--ok";
  }
  if (status === "bad") {
    return "forge-ops-status forge-ops-status--bad";
  }
  if (status === "warn") {
    return "forge-ops-status forge-ops-status--warn";
  }
  return "forge-ops-status forge-ops-status--muted";
}
