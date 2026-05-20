import { useMemo, type ReactNode } from "react";

import { assignableShellTools } from "../layout/shellConfig";
import { useWorkspaceLayoutStore } from "../stores/workspaceLayoutStore";
import type { DisplayArrangementMode } from "../stores/workspaceLayoutStore/types";

const roleOptions = [
  "chat",
  "workbench",
  "canvas",
  "dossier",
  "ops",
  "review",
  "settings",
  "mixed",
] as const;
const roleDisplay = (role: string | null) => {
  if (role === "main") return "Main";
  const secondary = /^secondary_(\d+)$/.exec(role ?? "");
  return secondary ? `Secondary ${secondary[1]}` : (role ?? "secondary");
};

export function WorkspaceLayoutsPage() {
  const activeLayoutId = useWorkspaceLayoutStore((s) => s.activeLayoutId);
  const selectedLayoutId = useWorkspaceLayoutStore((s) => s.selectedLayoutId);
  const layouts = useWorkspaceLayoutStore((s) => s.layouts);
  const monitors = useWorkspaceLayoutStore((s) => s.monitors);
  const monitorDesignations = useWorkspaceLayoutStore(
    (s) => s.monitorDesignations,
  );
  const monitorRoleMap = useWorkspaceLayoutStore((s) => s.monitorRoleMap);
  const runtimeWindows = useWorkspaceLayoutStore((s) => s.runtimeWindows);
  const displayIntent = useWorkspaceLayoutStore((s) => s.displayIntent);
  const fallbackNotice = useWorkspaceLayoutStore((s) => s.fallbackNotice);
  const supported = useWorkspaceLayoutStore((s) => s.supported);
  const createLayout = useWorkspaceLayoutStore((s) => s.createLayout);
  const selectLayout = useWorkspaceLayoutStore((s) => s.selectLayout);
  const renameLayout = useWorkspaceLayoutStore((s) => s.renameLayout);
  const duplicateLayout = useWorkspaceLayoutStore((s) => s.duplicateLayout);
  const deleteLayout = useWorkspaceLayoutStore((s) => s.deleteLayout);
  const addLayoutWindow = useWorkspaceLayoutStore((s) => s.addLayoutWindow);
  const removeLayoutWindow = useWorkspaceLayoutStore(
    (s) => s.removeLayoutWindow,
  );
  const updateLayoutWindow = useWorkspaceLayoutStore(
    (s) => s.updateLayoutWindow,
  );
  const activateLayout = useWorkspaceLayoutStore((s) => s.activateLayout);
  const captureRuntimeIntoLayout = useWorkspaceLayoutStore(
    (s) => s.captureRuntimeIntoLayout,
  );
  const clearFallbackNotice = useWorkspaceLayoutStore(
    (s) => s.clearFallbackNotice,
  );
  const setMainMonitor = useWorkspaceLayoutStore((s) => s.setMainMonitor);
  const setDisplayArrangementMode = useWorkspaceLayoutStore(
    (s) => s.setDisplayArrangementMode,
  );
  const setMonitorRoleLabel = useWorkspaceLayoutStore(
    (s) => s.setMonitorRoleLabel,
  );

  const selectedLayout = useMemo(
    () =>
      layouts.find((layout) => layout.id === selectedLayoutId) ??
      layouts[0] ??
      null,
    [layouts, selectedLayoutId],
  );
  const monitorRoleCatalog = useMemo(
    () =>
      [...monitors]
        .sort((a, b) => a.ordinal - b.ordinal)
        .map((monitor) => ({
          id: monitor.id,
          role: monitorRoleMap[monitor.id] || "secondary",
          label: monitor.name || `Display ${monitor.ordinal + 1}`,
          customLabel: monitorDesignations.customLabels[monitor.id] ?? "",
        })),
    [monitors, monitorRoleMap, monitorDesignations.customLabels],
  );

  return (
    <div className="forge-ops-board space-y-5">
      <header className="flex flex-col gap-3 lg:flex-row lg:items-end lg:justify-between">
        <div>
          <div className="forge-ops-label">Workspace Surfaces</div>
          <h1 className="mt-2 text-2xl font-semibold tracking-normal text-forge-ash sm:text-3xl">
            Layout command board
          </h1>
          <p className="mt-1 max-w-3xl text-sm leading-6 text-forge-mist/75">
            Monitor-aware window presets, role bindings, and runtime shell
            placement state.
          </p>
        </div>
        <div className="flex flex-wrap items-center gap-2">
          <span
            className={
              supported
                ? "forge-ops-status forge-ops-status--ok"
                : "forge-ops-status forge-ops-status--warn"
            }
          >
            {supported ? "Desktop runtime" : "Browser shell"}
          </span>
          <button
            type="button"
            className="forge-btn forge-btn--primary"
            onClick={() => createLayout(`Layout ${layouts.length + 1}`)}
          >
            New layout
          </button>
        </div>
      </header>

      <Panel
        title="Workspace Layouts"
        subtitle="Monitor-aware presets for confined in-shell FORGE desktop surfaces."
      >
        {!supported ? (
          <div className="rounded-xl border border-forge-platinum/10 bg-black/20 p-4 text-sm text-forge-mist">
            Monitor detection requires the Tauri desktop runtime. The browser
            shell can read saved presets but will not simulate attached
            displays.
          </div>
        ) : null}
        {fallbackNotice ? (
          <div className="mt-3 rounded-xl border border-forge-platinum/10 bg-black/20 p-4 text-sm text-forge-mist">
            <div className="font-semibold text-forge-ash">Fallback applied</div>
            <div className="mt-1">{fallbackNotice}</div>
            <button
              type="button"
              className="mt-3 forge-btn forge-btn--ghost"
              onClick={() => clearFallbackNotice()}
            >
              Dismiss notice
            </button>
          </div>
        ) : null}
        <div className="mt-4 grid gap-3 md:grid-cols-3">
          <MetricCard
            label="Detected displays"
            value={String(monitors.length)}
            detail={
              monitors.length > 0
                ? monitors
                    .map((m) => m.name ?? `Display ${m.ordinal + 1}`)
                    .join(" · ")
                : "No desktop monitor API available."
            }
          />
          <MetricCard
            label="Saved layouts"
            value={String(layouts.length)}
            detail={
              activeLayoutId
                ? `Active: ${layouts.find((layout) => layout.id === activeLayoutId)?.name ?? activeLayoutId}`
                : "No active layout."
            }
          />
          <MetricCard
            label="Open windows"
            value={String(runtimeWindows.length)}
            detail={
              runtimeWindows.length > 0
                ? runtimeWindows
                    .map((windowRecord) => windowRecord.runtimeLabel)
                    .join(" · ")
                : "No runtime windows registered yet."
            }
          />
        </div>
      </Panel>

      {supported ? (
        <div className="grid gap-6 xl:grid-cols-[minmax(0,1.1fr)_minmax(360px,0.9fr)]">
          <Panel
            title="Monitor Roles"
            subtitle="Assign one monitor as Main and add optional labels to the role list."
          >
            <div className="space-y-3">
              {monitorRoleCatalog.length === 0 ? (
                <div className="rounded-xl border border-forge-platinum/10 bg-black/20 p-3 text-sm text-forge-mist">
                  No displays detected yet.
                </div>
              ) : null}
              {monitorRoleCatalog.map((monitor) => {
                const isMain = monitor.id === monitorDesignations.mainMonitorId;
                return (
                  <div
                    key={monitor.id}
                    className="rounded-xl border border-forge-platinum/10 bg-black/20 p-3"
                  >
                    <div className="flex flex-wrap items-center justify-between gap-2">
                      <div>
                        <div className="text-sm font-semibold text-forge-ash">
                          {monitor.label}
                        </div>
                        <div className="text-xs text-forge-mist">
                          Role: {roleDisplay(monitor.role)}
                        </div>
                        <div className="mt-1 text-[11px] text-forge-mist">
                          Current monitor role target: {monitor.role}
                        </div>
                      </div>
                      <label className="text-xs text-forge-mist">
                        <input
                          type="radio"
                          checked={isMain}
                          onChange={() => setMainMonitor(monitor.id)}
                        />
                        <span className="ml-1">Main</span>
                      </label>
                    </div>
                    <label className="mt-3 block text-xs text-forge-mist">
                      Custom role label
                      <input
                        className="forge-input mt-1"
                        value={monitor.customLabel}
                        onChange={(e) =>
                          setMonitorRoleLabel(monitor.id, e.target.value)
                        }
                        placeholder="e.g. Focus +1"
                      />
                    </label>
                  </div>
                );
              })}
              <div className="rounded-xl border border-forge-ember/25 bg-forge-ember/5 p-3 text-xs leading-relaxed text-forge-mist">
                Role order is currently inferred from detected monitor index:{" "}
                <b>Main</b> first, then <b>Secondary N</b> by ordinal.
                <br />
                You can override layout bindings per-window using the role
                selector.
              </div>
            </div>
          </Panel>

          <Panel
            title="Display Layout Intent"
            subtitle="Saved display preference for the FORGE shell."
          >
            <div className="space-y-4">
              <label className="block text-xs text-forge-mist">
                Display arrangement intent
                <select
                  className="forge-input mt-1"
                  value={displayIntent.arrangementMode}
                  onChange={(e) =>
                    setDisplayArrangementMode(
                      e.target.value as DisplayArrangementMode,
                    )
                  }
                >
                  <option value="preserve">Preserve current layout</option>
                  <option value="extend">Extend desktop</option>
                  <option value="mirror">Mirror displays</option>
                </select>
              </label>
              <div className="grid gap-3 sm:grid-cols-2">
                <MetricCard
                  label="Primary display"
                  value={
                    monitors.find(
                      (monitor) =>
                        monitor.id === displayIntent.primaryMonitorId,
                    )?.name ?? "Unassigned"
                  }
                  detail={displayIntent.primaryMonitorId ?? "No saved display"}
                />
                <MetricCard
                  label="Output apply"
                  value="Apply deferred"
                  detail="Compositor output management gate pending."
                />
              </div>
              <div className="rounded-xl border border-forge-platinum/10 bg-black/20 p-3 text-xs leading-relaxed text-forge-mist">
                Saved order:{" "}
                {displayIntent.preferredOrder.length > 0
                  ? displayIntent.preferredOrder.join(" -> ")
                  : "No display order saved"}
              </div>
              <button type="button" className="forge-btn" disabled>
                Apply output changes
              </button>
            </div>
          </Panel>
        </div>
      ) : null}

      <div className="grid gap-6 xl:grid-cols-[320px_minmax(0,1fr)]">
        <Panel
          title="Preset List"
          subtitle="Create, select, duplicate, rename, delete, and activate named workspace layouts."
        >
          <div className="mb-3 text-xs text-forge-mist/65">
            Create and select layouts from the board header or this preset list.
          </div>
          <div className="space-y-2">
            {layouts.map((layout) => (
              <button
                key={layout.id}
                type="button"
                onClick={() => selectLayout(layout.id)}
                className={[
                  "w-full rounded-xl border px-3 py-3 text-left transition",
                  selectedLayout?.id === layout.id
                    ? "border-forge-platinum/20 bg-black/30"
                    : "border-forge-platinum/10 bg-black/20 hover:border-forge-platinum/20",
                ].join(" ")}
              >
                <div className="flex items-center justify-between gap-2">
                  <div className="text-sm font-semibold text-forge-ash">
                    {layout.name}
                  </div>
                  {activeLayoutId === layout.id ? (
                    <span className="rounded-full border border-forge-platinum/10 px-2 py-0.5 text-[10px] uppercase tracking-[0.14em] text-forge-mist">
                      active
                    </span>
                  ) : null}
                </div>
                <div className="mt-1 text-xs text-forge-mist">
                  {layout.windows.length} windows · updated{" "}
                  {new Date(layout.updatedAtMs).toLocaleString()}
                </div>
              </button>
            ))}
          </div>
        </Panel>

        {selectedLayout ? (
          <div className="space-y-6">
            <Panel
              title="Layout Editor"
              subtitle="Assign windows to real displays, choose per-window roles, routes, and default focus surfaces."
              actions={
                <div className="flex flex-wrap gap-2">
                  <button
                    type="button"
                    className="forge-btn forge-btn--ghost"
                    onClick={() => duplicateLayout(selectedLayout.id)}
                  >
                    Duplicate
                  </button>
                  <button
                    type="button"
                    className="forge-btn forge-btn--ghost"
                    onClick={() =>
                      void captureRuntimeIntoLayout(selectedLayout.id)
                    }
                  >
                    Capture running windows
                  </button>
                  <button
                    type="button"
                    className="forge-btn forge-btn--primary"
                    onClick={() => void activateLayout(selectedLayout.id)}
                  >
                    Activate layout
                  </button>
                </div>
              }
            >
              <label className="block text-xs text-forge-mist">
                Layout name
                <input
                  className="forge-input mt-1"
                  value={selectedLayout.name}
                  onChange={(e) =>
                    renameLayout(selectedLayout.id, e.target.value)
                  }
                />
              </label>
              <div className="mt-4 space-y-4">
                {selectedLayout.windows.map((windowRecord, index) => (
                  <div
                    key={windowRecord.id}
                    className="rounded-xl border border-forge-platinum/10 bg-black/20 p-4"
                  >
                    <div className="flex flex-wrap items-center justify-between gap-2">
                      <div>
                        <div className="text-sm font-semibold text-forge-ash">
                          {windowRecord.title}
                        </div>
                        <div className="mt-1 text-xs text-forge-mist">
                          Runtime label:{" "}
                          <span className="font-mono">
                            {windowRecord.runtimeLabel}
                          </span>
                        </div>
                      </div>
                      <div className="flex gap-2">
                        <button
                          type="button"
                          className="forge-btn forge-btn--ghost"
                          onClick={() => addLayoutWindow(selectedLayout.id)}
                        >
                          Add window
                        </button>
                        {windowRecord.runtimeLabel !== "main" ? (
                          <button
                            type="button"
                            className="forge-btn forge-btn--ghost"
                            onClick={() =>
                              removeLayoutWindow(
                                selectedLayout.id,
                                windowRecord.id,
                              )
                            }
                          >
                            Remove
                          </button>
                        ) : null}
                      </div>
                    </div>

                    <div className="mt-4 grid gap-3 md:grid-cols-2">
                      <label className="text-xs text-forge-mist">
                        Title
                        <input
                          className="forge-input mt-1"
                          value={windowRecord.title}
                          onChange={(e) =>
                            updateLayoutWindow(
                              selectedLayout.id,
                              windowRecord.id,
                              { title: e.target.value },
                            )
                          }
                        />
                      </label>
                      <label className="text-xs text-forge-mist">
                        Role
                        <select
                          className="forge-input mt-1"
                          value={windowRecord.role}
                          onChange={(e) =>
                            updateLayoutWindow(
                              selectedLayout.id,
                              windowRecord.id,
                              { role: e.target.value as never },
                            )
                          }
                        >
                          {roleOptions.map((role) => (
                            <option key={role} value={role}>
                              {role}
                            </option>
                          ))}
                        </select>
                      </label>
                      <label className="text-xs text-forge-mist">
                        Target monitor role
                        <select
                          className="forge-input mt-1"
                          value={windowRecord.targetMonitorRole ?? "none"}
                          onChange={(e) => {
                            const value = e.target.value;
                            updateLayoutWindow(
                              selectedLayout.id,
                              windowRecord.id,
                              {
                                targetMonitorRole:
                                  value === "none" ? null : value,
                              },
                            );
                          }}
                        >
                          <option value="none">No role binding</option>
                          {monitorRoleCatalog.map((roleEntry) => (
                            <option key={roleEntry.id} value={roleEntry.role}>
                              {roleDisplay(roleEntry.role)}
                              {roleEntry.customLabel
                                ? ` (${roleEntry.customLabel})`
                                : ""}
                              {" · "}
                              {roleEntry.label}
                            </option>
                          ))}
                        </select>
                      </label>
                      <label className="text-xs text-forge-mist">
                        Target display override
                        <select
                          className="forge-input mt-1"
                          value={
                            windowRecord.targetMonitorId ??
                            `ordinal:${windowRecord.targetMonitorOrdinal}`
                          }
                          onChange={(e) => {
                            const raw = e.target.value;
                            const monitor =
                              monitors.find((item) => item.id === raw) ?? null;
                            updateLayoutWindow(
                              selectedLayout.id,
                              windowRecord.id,
                              {
                                targetMonitorId: monitor?.id ?? null,
                                targetMonitorOrdinal:
                                  monitor?.ordinal ??
                                  (Number(raw.replace("ordinal:", "")) || 0),
                              },
                            );
                          }}
                        >
                          {monitors.length === 0 ? (
                            <option
                              value={`ordinal:${windowRecord.targetMonitorOrdinal}`}
                            >
                              No live display list
                            </option>
                          ) : null}
                          {monitors.map((monitor) => (
                            <option key={monitor.id} value={monitor.id}>
                              {monitor.name ?? `Display ${monitor.ordinal + 1}`}{" "}
                              · {monitor.workArea.width}x
                              {monitor.workArea.height}
                              {monitor.id === monitorDesignations.mainMonitorId
                                ? " · main role"
                                : ""}
                              {monitorRoleMap[monitor.id] ===
                              `secondary_${monitor.ordinal + 1}`
                                ? " · secondary role"
                                : ""}
                            </option>
                          ))}
                        </select>
                      </label>
                      <label className="text-xs text-forge-mist">
                        Default surface
                        <select
                          className="forge-input mt-1"
                          value={windowRecord.activeRoute}
                          onChange={(e) =>
                            updateLayoutWindow(
                              selectedLayout.id,
                              windowRecord.id,
                              { activeRoute: e.target.value },
                            )
                          }
                        >
                          {windowRecord.assignedRoutes.map((route) => (
                            <option key={route} value={route}>
                              {assignableShellTools.find(
                                (tool) => tool.route === route,
                              )?.label ?? route}
                            </option>
                          ))}
                        </select>
                      </label>
                    </div>

                    <div className="mt-4">
                      <div className="text-xs font-semibold tracking-[0.16em] text-forge-mist">
                        Assigned surfaces
                      </div>
                      <div className="mt-2 grid gap-2 md:grid-cols-2 xl:grid-cols-3">
                        {assignableShellTools.map((tool) => {
                          const checked = windowRecord.assignedRoutes.includes(
                            tool.route,
                          );
                          return (
                            <label
                              key={tool.route}
                              className="flex items-start gap-2 rounded-lg border border-forge-platinum/10 bg-black/20 px-3 py-2 text-sm text-forge-mist"
                            >
                              <input
                                type="checkbox"
                                checked={checked}
                                onChange={(e) => {
                                  const nextRoutes = e.target.checked
                                    ? [
                                        ...windowRecord.assignedRoutes,
                                        tool.route,
                                      ]
                                    : windowRecord.assignedRoutes.filter(
                                        (route) => route !== tool.route,
                                      );
                                  updateLayoutWindow(
                                    selectedLayout.id,
                                    windowRecord.id,
                                    { assignedRoutes: nextRoutes },
                                  );
                                }}
                              />
                              <span>
                                <span className="block font-semibold text-forge-ash">
                                  {tool.label}
                                </span>
                                <span className="block text-[11px] leading-relaxed text-forge-mist">
                                  {tool.description}
                                </span>
                              </span>
                            </label>
                          );
                        })}
                      </div>
                    </div>

                    {windowRecord.fallbackReason ? (
                      <div className="mt-3 text-xs text-forge-mist">
                        Last fallback: {windowRecord.fallbackReason}
                      </div>
                    ) : null}
                    {index === 0 ? (
                      <div className="mt-3 text-xs text-forge-mist">
                        The first window is pinned to the real Tauri `main`
                        shell so restore always has a recovery anchor.
                      </div>
                    ) : null}
                  </div>
                ))}
              </div>
              <div className="mt-4 flex gap-2">
                <button
                  type="button"
                  className="forge-btn forge-btn--ghost"
                  onClick={() => void deleteLayout(selectedLayout.id)}
                  disabled={layouts.length <= 1}
                >
                  Delete layout
                </button>
              </div>
            </Panel>

            <Panel
              title="Runtime Windows"
              subtitle="Current shell registration for the main desktop host."
            >
              <div className="space-y-2">
                {runtimeWindows.length === 0 ? (
                  <div className="text-sm text-forge-mist">
                    No runtime windows registered yet.
                  </div>
                ) : (
                  runtimeWindows.map((windowRecord) => (
                    <div
                      key={windowRecord.runtimeLabel}
                      className="rounded-xl border border-forge-platinum/10 bg-black/20 p-3 text-sm text-forge-mist"
                    >
                      <div className="flex flex-wrap items-center justify-between gap-2">
                        <div className="font-semibold text-forge-ash">
                          {windowRecord.title}
                        </div>
                        <span className="text-[11px]">
                          {windowRecord.isFocused ? "focused" : "background"}
                        </span>
                      </div>
                      <div className="mt-1 font-mono text-[11px]">
                        {windowRecord.runtimeLabel}
                      </div>
                      <div className="mt-1 text-[11px]">
                        route {windowRecord.currentRoute} · monitor{" "}
                        {windowRecord.monitorId ?? "unknown"}
                      </div>
                      {windowRecord.bounds ? (
                        <div className="mt-1 text-[11px]">
                          {windowRecord.bounds.x},{windowRecord.bounds.y} ·{" "}
                          {windowRecord.bounds.width}x
                          {windowRecord.bounds.height}
                        </div>
                      ) : null}
                    </div>
                  ))
                )}
              </div>
            </Panel>
          </div>
        ) : null}
      </div>
    </div>
  );
}

function MetricCard(props: { label: string; value: string; detail: string }) {
  return (
    <div className="forge-ops-card p-4">
      <div className="forge-ops-label">{props.label}</div>
      <div className="mt-2 text-lg font-semibold text-forge-ash">
        {props.value}
      </div>
      <div className="mt-1 text-xs leading-relaxed text-forge-mist">
        {props.detail}
      </div>
    </div>
  );
}

function Panel(props: {
  title: string;
  subtitle?: string;
  actions?: ReactNode;
  children: ReactNode;
}) {
  return (
    <section className="forge-ops-panel">
      <div className="forge-ops-panel__head">
        <div>
          <div className="forge-ops-title">{props.title}</div>
          {props.subtitle ? (
            <div className="mt-1 text-xs text-forge-mist/65">
              {props.subtitle}
            </div>
          ) : null}
        </div>
        {props.actions ? (
          <div className="flex flex-wrap items-center gap-2">
            {props.actions}
          </div>
        ) : null}
      </div>
      <div className="forge-ops-panel__body">{props.children}</div>
    </section>
  );
}
