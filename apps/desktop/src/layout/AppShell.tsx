import type { DashboardSummary } from "@forge/shared";
import { useEffect, useMemo, useState, type ReactNode } from "react";
import { useLocation, useNavigate } from "react-router-dom";

import { api } from "../lib/api";
import {
  DETACHED_TAURI_TOOL_WINDOWS,
  controlLinuxWindow,
  focusLinuxWindow,
  isShellHostWindowLabel,
  isTauriDesktop,
  launchOperatorApp,
  listForgeWindows,
  listLinuxWindows,
  listOperatorApps,
  requestHostPowerAction,
  type ForgeHostPowerAction,
  type LinuxWindowSnapshot,
  type LinuxWindowAction,
  type OperatorApp,
} from "../lib/desktop";
import { useUiStore } from "../stores/uiStore";
import { useWorkspaceLayoutStore } from "../stores/workspaceLayoutStore";
import { useWorkspaceStore } from "../stores/workspaceStore";
import {
  useDesktopWindowStore,
  type DesktopWindow,
} from "../stores/desktopWindowStore";

import {
  allShellTools,
  getShellTool,
  type ShellToolDefinition,
  type ShellToolId,
} from "./shellConfig";
import {
  DesktopWallpaper,
  DockContextMenuView,
  FALLBACK_OPERATOR_APPS,
  FloatingWindow,
  ForgeHero,
  LinuxWindowIcon,
  NativeWindowContextMenuView,
  OperatorAppIcon,
  StartMenu,
  type DockContextMenu,
  type DockTile,
  type NativeWindowContextMenu,
} from "./AppShellSurfaces";
import {
  buildDesktopHosts,
  globalToHostPoint,
  hostToGlobalPoint,
  projectDesktopWindowToHost,
  resolveDesktopHostPlacement,
  type DesktopPlacement,
} from "./desktopGeometry";

type AppShellProps = {
  children: ReactNode;
  isMainWindow: boolean;
  hostLabel?: string;
  onForgeLogout?: () => void;
};

type AttentionLevel = "none" | "low" | "medium" | "high";
type RunningOperatorApp = {
  app: OperatorApp;
  pid: number | null;
  launchedAtMs: number;
};

const HOME_ROUTE = "/";
function corePill(core: "online" | "offline" | "unknown") {
  if (core === "online") return "forge-chip--ok";
  if (core === "offline") return "forge-chip--warn";
  return "forge-chip--muted";
}

function getWorkspaceLabel(workspaceDir: string | null | undefined) {
  if (!workspaceDir) return "Workspace unavailable";
  const parts = workspaceDir.split(/[\\/]/).filter(Boolean);
  return parts.at(-1) ?? workspaceDir;
}

function attentionLevel(
  summary: DashboardSummary | null,
  core: "online" | "offline" | "unknown",
): AttentionLevel {
  if (core === "offline") return "high";
  const approvals = summary?.approvalsPending ?? 0;
  const reviews = summary?.reviewsPending ?? 0;
  const failures = Array.isArray(summary?.recentFailures)
    ? summary.recentFailures.length
    : 0;
  const active = Array.isArray(summary?.activeJobs)
    ? summary.activeJobs.length
    : 0;
  const score = approvals * 2 + reviews + failures * 2 + Math.min(active, 3);
  if (score <= 0) return "none";
  if (score <= 2) return "low";
  if (score <= 5) return "medium";
  return "high";
}

function cx(...parts: Array<string | false | null | undefined>) {
  return parts.filter(Boolean).join(" ");
}

export function AppShell(props: AppShellProps) {
  const navigate = useNavigate();
  const location = useLocation();
  const pathname = location.pathname;
  const currentTool = useMemo(() => getShellTool(pathname), [pathname]);
  const detachedTauriShell = isTauriDesktop() && DETACHED_TAURI_TOOL_WINDOWS;

  const core = useWorkspaceStore((s) => s.core);
  const meta = useWorkspaceStore((s) => s.meta);
  const lastErr = useWorkspaceStore((s) => s.lastCoreError);
  const fallbackNotice = useWorkspaceLayoutStore((s) => s.fallbackNotice);
  const monitors = useWorkspaceLayoutStore((s) => s.monitors);
  const runtimeWindows = useWorkspaceLayoutStore((s) => s.runtimeWindows);
  const clearFallbackNotice = useWorkspaceLayoutStore(
    (s) => s.clearFallbackNotice,
  );
  const uiMode = useUiStore((s) => s.uiMode);

  const pinned = useDesktopWindowStore((s) => s.pinned);
  const windows = useDesktopWindowStore((s) => s.windows);
  const focusedId = useDesktopWindowStore((s) => s.focusedId);
  const openWindow = useDesktopWindowStore((s) => s.openWindow);
  const closeWindow = useDesktopWindowStore((s) => s.closeWindow);
  const closeByTool = useDesktopWindowStore((s) => s.closeByTool);
  const minimize = useDesktopWindowStore((s) => s.minimize);
  const restore = useDesktopWindowStore((s) => s.restore);
  const focus = useDesktopWindowStore((s) => s.focus);
  const toggleMaximize = useDesktopWindowStore((s) => s.toggleMaximize);
  const move = useDesktopWindowStore((s) => s.move);
  const moveToHost = useDesktopWindowStore((s) => s.moveToHost);
  const resize = useDesktopWindowStore((s) => s.resize);
  const reconcileHostAvailability = useDesktopWindowStore(
    (s) => s.reconcileHostAvailability,
  );
  const pin = useDesktopWindowStore((s) => s.pin);
  const unpin = useDesktopWindowStore((s) => s.unpin);

  const [summary, setSummary] = useState<DashboardSummary | null>(null);
  const [shellErr, setShellErr] = useState<string | null>(null);
  const [startOpen, setStartOpen] = useState(false);
  const [startQuery, setStartQuery] = useState("");
  const [operatorApps, setOperatorApps] = useState<OperatorApp[]>(
    FALLBACK_OPERATOR_APPS,
  );
  const [operatorAppStatus, setOperatorAppStatus] = useState<string | null>(
    null,
  );
  const [runningOperatorApps, setRunningOperatorApps] = useState<
    RunningOperatorApp[]
  >([]);
  const [linuxWindows, setLinuxWindows] = useState<LinuxWindowSnapshot[]>([]);
  const [now, setNow] = useState(() => new Date());
  const [contextMenu, setContextMenu] = useState<DockContextMenu | null>(null);
  const [nativeContextMenu, setNativeContextMenu] =
    useState<NativeWindowContextMenu | null>(null);
  const isMainWindow = props.isMainWindow;
  const hostLabel = props.hostLabel?.trim() || "main";
  const desktopHosts = useMemo(
    () =>
      buildDesktopHosts(
        monitors.map((monitor) => ({
          id: monitor.id,
          ordinal: monitor.ordinal,
          workArea: monitor.workArea,
        })),
        runtimeWindows,
      ),
    [monitors, runtimeWindows],
  );
  const desktopHostBounds = useMemo(
    () =>
      desktopHosts.map((host) => ({
        runtimeLabel: host.hostLabel,
        monitorId: host.monitorId,
        bounds: host.bounds,
      })),
    [desktopHosts],
  );
  const currentDesktopHost =
    desktopHosts.find((host) => host.hostLabel === hostLabel) ?? null;
  const currentHostMonitorId =
    currentDesktopHost?.monitorId ??
    runtimeWindows.find((window_) => window_.runtimeLabel === hostLabel)
      ?.monitorId ??
    null;

  useEffect(() => {
    if (desktopHosts.length === 0) return;
    reconcileHostAvailability(
      desktopHosts.map((host) => ({
        hostLabel: host.hostLabel,
        monitorId: host.monitorId,
        primary: host.role === "main",
      })),
    );
  }, [desktopHosts, reconcileHostAvailability]);

  const isHome = pathname === HOME_ROUTE;

  // Deep links open confined in-shell desktop windows for every monitor host.
  // The disabled detached Tauri compatibility path owns tool routes in
  // separate webviews.
  useEffect(() => {
    if (detachedTauriShell) return;
    if (isHome) return;
    const tool = currentTool;
    if (tool.id === "other" || tool.id === "job-detail") return;
    const existing = windows.find(
      (w) => w.toolId === tool.id && (w.hostLabel || "main") === hostLabel,
    );
    if (!existing) {
      void openWindow(tool.id, { hostLabel, monitorId: currentHostMonitorId });
    } else if (existing.minimized || focusedId !== existing.id) {
      void restore(existing.id);
    }
    // intentionally omit windows / focusedId: this effect only reacts to route changes
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [pathname, isHome, detachedTauriShell, hostLabel]);

  // Detached Tauri compatibility only: reconcile the desktop window store
  // against real Tauri windows. Normal Tauri uses confined in-shell windows.
  useEffect(() => {
    if (!detachedTauriShell) return;
    if (!isMainWindow) return;
    let cancelled = false;
    const reconcile = useDesktopWindowStore.getState().reconcileFromTauri;
    async function tick() {
      const snapshots = await listForgeWindows();
      if (cancelled) return;
      reconcile(snapshots);
    }
    void tick();
    const id = window.setInterval(() => void tick(), 1000);
    return () => {
      cancelled = true;
      window.clearInterval(id);
    };
  }, [isMainWindow, detachedTauriShell]);

  useEffect(() => {
    if (!isMainWindow) return;
    let cancelled = false;
    async function loadShellState() {
      try {
        const nextSummary = await api.dashboard.summary();
        if (cancelled) return;
        setSummary(nextSummary);
        setShellErr(null);
      } catch (e) {
        if (cancelled) return;
        setShellErr(e instanceof Error ? e.message : String(e));
      }
    }
    void loadShellState();
    const id = window.setInterval(() => void loadShellState(), 5000);
    return () => {
      cancelled = true;
      window.clearInterval(id);
    };
  }, [isMainWindow]);

  useEffect(() => {
    if (!isMainWindow || !isTauriDesktop()) return;
    let cancelled = false;
    async function loadLinuxWindows() {
      const nextWindows = await listLinuxWindows();
      if (cancelled) return;
      setLinuxWindows(nextWindows);
    }
    void loadLinuxWindows();
    const id = window.setInterval(() => void loadLinuxWindows(), 1500);
    return () => {
      cancelled = true;
      window.clearInterval(id);
    };
  }, [isMainWindow]);

  useEffect(() => {
    const onStorage = (event: StorageEvent) => {
      if (
        event.key === "forge.os.windows.v1" ||
        event.key === "forge.os.focus.v1"
      ) {
        useDesktopWindowStore.getState().hydrate();
      }
    };
    window.addEventListener("storage", onStorage);
    return () => window.removeEventListener("storage", onStorage);
  }, []);

  useEffect(() => {
    if (!isMainWindow) return;
    const id = window.setInterval(() => setNow(new Date()), 30_000);
    return () => window.clearInterval(id);
  }, [isMainWindow]);

  useEffect(() => {
    if (!isMainWindow) return;
    let cancelled = false;
    async function loadOperatorApps() {
      try {
        const apps = await listOperatorApps();
        if (cancelled) return;
        setOperatorApps(apps.length > 0 ? apps : FALLBACK_OPERATOR_APPS);
        setOperatorAppStatus(null);
      } catch (error) {
        if (cancelled) return;
        setOperatorApps(FALLBACK_OPERATOR_APPS);
        setOperatorAppStatus(
          error instanceof Error ? error.message : String(error),
        );
      }
    }
    void loadOperatorApps();
    return () => {
      cancelled = true;
    };
  }, [isMainWindow]);

  useEffect(() => {
    const onKey = (event: KeyboardEvent) => {
      const target = event.target as HTMLElement | null;
      const tag = target?.tagName?.toLowerCase();
      const editing = tag === "input" || tag === "textarea" || tag === "select";
      if (event.key === "Escape") {
        if (contextMenu) {
          setContextMenu(null);
          return;
        }
        if (startOpen) {
          setStartOpen(false);
          return;
        }
        // Minimize the focused window via Esc.
        if (focusedId) {
          void minimize(focusedId);
        }
        return;
      }
      if (editing) return;
      if (event.key === "Meta" || (event.ctrlKey && event.code === "Space")) {
        event.preventDefault();
        setStartOpen((value) => !value);
      }
    };
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, [startOpen, focusedId, minimize, contextMenu]);

  useEffect(() => {
    setStartOpen(false);
    setStartQuery("");
  }, [pathname]);

  useEffect(() => {
    if (!contextMenu) return;
    const onDown = () => setContextMenu(null);
    window.addEventListener("mousedown", onDown);
    return () => window.removeEventListener("mousedown", onDown);
  }, [contextMenu]);

  function launchTool(tool: ShellToolDefinition) {
    setStartOpen(false);
    void openWindow(tool.id, { hostLabel, monitorId: currentHostMonitorId });
    // Confined shell mode navigates after opening so deep links and reloads
    // rehydrate the matching in-shell window.
    if (!detachedTauriShell) {
      navigate(tool.route);
    }
  }

  async function launchNativeApp(app: OperatorApp) {
    setStartOpen(false);
    try {
      const result = await launchOperatorApp(app.id);
      setRunningOperatorApps((items) => {
        const next: RunningOperatorApp = {
          app,
          pid: result.pid ?? null,
          launchedAtMs: Date.now(),
        };
        return [next, ...items.filter((item) => item.app.id !== app.id)];
      });
      setOperatorAppStatus(
        result.pid
          ? `${result.label} launch requested. PID ${result.pid}.`
          : result.message,
      );
    } catch (error) {
      setOperatorAppStatus(
        error instanceof Error ? error.message : String(error),
      );
    }
  }

  async function runNativeWindowAction(
    window_: LinuxWindowSnapshot,
    action: LinuxWindowAction,
  ) {
    const ok =
      action === "focus"
        ? await focusLinuxWindow(window_.id)
        : await controlLinuxWindow(window_.id, action);
    if (!ok) return;
    const nextWindows = await listLinuxWindows();
    setLinuxWindows(nextWindows);
  }

  function resolveTransferredWindow(
    win: DesktopWindow,
    nextX: number,
    nextY: number,
  ) {
    return resolveDesktopHostPlacement(
      desktopHostBounds,
      hostLabel,
      win,
      nextX,
      nextY,
      isShellHostWindowLabel,
    );
  }

  function moveAcrossShellHosts(
    win: DesktopWindow,
    nextX: number,
    nextY: number,
  ) {
    const transferred = resolveTransferredWindow(win, nextX, nextY);
    if (transferred.hostLabel !== (win.hostLabel || "main")) {
      const targetHost =
        desktopHosts.find((host) => host.hostLabel === transferred.hostLabel) ??
        null;
      const targetMonitorId =
        targetHost?.monitorId ??
        runtimeWindows.find(
          (window_) => window_.runtimeLabel === transferred.hostLabel,
        )?.monitorId ??
        null;
      const targetPlacement =
        targetHost && currentDesktopHost
          ? globalToHostPoint(
              targetHost,
              hostToGlobalPoint(currentDesktopHost, {
                x: nextX,
                y: nextY,
              }),
            )
          : { x: transferred.x, y: transferred.y };
      moveToHost(
        win.id,
        transferred.hostLabel,
        targetPlacement.x,
        targetPlacement.y,
        targetMonitorId,
      );
      return;
    }
    move(win.id, transferred.x, transferred.y);
  }

  function focusFromDock(window_: DesktopWindow) {
    if (window_.minimized) {
      void restore(window_.id);
    } else if (focusedId === window_.id) {
      if (detachedTauriShell) {
        void focus(window_.id);
      } else {
        void minimize(window_.id);
      }
    } else {
      void focus(window_.id);
    }
    if (!detachedTauriShell) {
      const tool = allShellTools.find((t) => t.id === window_.toolId);
      if (tool) navigate(tool.route);
    }
  }

  async function handleStartPowerAction(
    action: "logout" | ForgeHostPowerAction,
  ) {
    setStartOpen(false);
    setStartQuery("");

    if (action === "logout") {
      props.onForgeLogout?.();
      if (!props.onForgeLogout) {
        navigate("/login");
      }
      return;
    }

    const confirmed = window.confirm(
      action === "reboot"
        ? "Reboot the FORGE host now?"
        : "Shut down the FORGE host now?",
    );
    if (!confirmed) return;

    try {
      const result = await requestHostPowerAction(action);
      setOperatorAppStatus(result.message);
    } catch (error) {
      setOperatorAppStatus(
        error instanceof Error
          ? error.message
          : `Unable to request host ${action}.`,
      );
    }
  }

  const approvalsPending = summary?.approvalsPending ?? 0;
  const reviewsPending = summary?.reviewsPending ?? 0;
  const recentFailures = Array.isArray(summary?.recentFailures)
    ? summary.recentFailures
    : [];
  const attentionCount =
    approvalsPending + reviewsPending + recentFailures.length;
  const level = attentionLevel(summary, core);
  const workspaceLabel = getWorkspaceLabel(meta?.workspaceDir);
  const runtimeState =
    core === "offline" ? "offline" : shellErr ? "degraded" : "online";

  // Dock = pinned tools (in pinned order) + global open windows. Hosts render
  // only their local desktop windows, but the taskbar remains globally aware.
  const dockTiles = useMemo<DockTile[]>(() => {
    const toolMap = new Map<ShellToolId, ShellToolDefinition>(
      allShellTools.map((t) => [t.id, t] as const),
    );
    const tiles: DockTile[] = [];
    const representedWindowIds = new Set<string>();
    for (const toolId of pinned) {
      const tool = toolMap.get(toolId);
      if (!tool) continue;
      const window_ =
        (focusedId
          ? windows.find((w) => w.id === focusedId && w.toolId === toolId)
          : null) ??
        windows.find(
          (w) => w.toolId === toolId && (w.hostLabel || "main") === hostLabel,
        ) ??
        windows.find((w) => w.toolId === toolId) ??
        null;
      if (window_) representedWindowIds.add(window_.id);
      tiles.push({
        kind: "tile",
        key: `pinned:${tool.id}`,
        tool,
        window: window_,
        pinned: true,
      });
    }
    const pinnedSet = new Set(pinned);
    for (const window_ of windows) {
      if (representedWindowIds.has(window_.id)) continue;
      const tool = toolMap.get(window_.toolId);
      if (!tool) continue;
      if (pinnedSet.has(window_.toolId)) {
        tiles.push({
          kind: "tile",
          key: `window:${window_.id}`,
          tool,
          window: window_,
          pinned: true,
        });
        continue;
      }
      tiles.push({
        kind: "tile",
        key: `window:${window_.id}`,
        tool,
        window: window_,
        pinned: false,
      });
    }
    return tiles;
  }, [focusedId, hostLabel, pinned, windows]);

  // Active foreground tool (focused, non-minimized).
  const focusedWindow = useMemo<DesktopWindow | null>(() => {
    if (!focusedId) return null;
    const w = windows.find((w_) => w_.id === focusedId);
    if (!w || w.minimized) return null;
    if ((w.hostLabel || "main") !== hostLabel) return null;
    return w;
  }, [focusedId, hostLabel, windows]);
  const focusedTool = useMemo<ShellToolDefinition | null>(() => {
    if (!focusedWindow) return null;
    return allShellTools.find((t) => t.id === focusedWindow.toolId) ?? null;
  }, [focusedWindow]);

  // Sort windows so the focused one renders last (= on top via DOM order).
  const sortedWindows = useMemo(
    () => [...windows].sort((a, b) => a.z - b.z),
    [windows],
  );
  const visibleWindows = useMemo<
    Array<{ window: DesktopWindow; placement: DesktopPlacement }>
  >(
    () =>
      sortedWindows.flatMap((window_) => {
        if (window_.minimized) return [];
        if ((window_.hostLabel || "main") !== hostLabel) return [];
        const placement = projectDesktopWindowToHost(
          runtimeWindows,
          hostLabel,
          window_,
        );
        return placement ? [{ window: window_, placement }] : [];
      }),
    [hostLabel, runtimeWindows, sortedWindows],
  );
  const shellRenderedWindows = useMemo(
    () =>
      detachedTauriShell
        ? visibleWindows.filter(({ window }) => !window.tauri)
        : visibleWindows,
    [detachedTauriShell, visibleWindows],
  );
  const linuxWindowKeys = useMemo(
    () =>
      new Set(
        linuxWindows.flatMap((window_) => [
          window_.appId.toLowerCase(),
          window_.title.toLowerCase(),
        ]),
      ),
    [linuxWindows],
  );
  const pendingOperatorApps = useMemo(
    () =>
      runningOperatorApps.filter((item) => {
        const appKey = item.app.executable.toLowerCase();
        const labelKey = item.app.label.toLowerCase();
        return !linuxWindowKeys.has(appKey) && !linuxWindowKeys.has(labelKey);
      }),
    [linuxWindowKeys, runningOperatorApps],
  );

  return (
    <div
      className={cx(
        "forge-os-shell flex h-full min-h-0 flex-col text-forge-ash",
      )}
    >
      <DesktopWallpaper />

      {isMainWindow ? (
        <header className="forge-os-statusbar">
          <div className="forge-os-statusbar__left">
            <span className="forge-os-statusbar__crumb">{workspaceLabel}</span>
            <span className="forge-os-statusbar__sep" aria-hidden>
              ›
            </span>
            <span className="forge-os-statusbar__crumb forge-os-statusbar__crumb--active">
              {focusedTool ? focusedTool.label : "Desktop"}
            </span>
          </div>
          <div
            className="forge-os-statusbar__right"
            role="status"
            aria-label="FORGE shell status"
            aria-live="polite"
            aria-atomic="true"
          >
            <span
              className={cx(
                "forge-chip px-2.5 py-1 text-[10px] font-semibold uppercase tracking-[0.12em]",
                corePill(core),
              )}
            >
              Core:{" "}
              {core === "online"
                ? "online"
                : core === "offline"
                  ? "offline"
                  : "checking"}
            </span>
            <span
              className={cx(
                "forge-chip px-2.5 py-1 text-[10px] font-semibold uppercase tracking-[0.12em]",
                runtimeState === "degraded"
                  ? "forge-chip--warn"
                  : "forge-chip--muted",
              )}
            >
              Runtime: {runtimeState}
            </span>
            <span
              className={cx(
                "forge-chip px-2.5 py-1 text-[10px] font-semibold uppercase tracking-[0.12em]",
                level === "none"
                  ? "forge-chip--muted"
                  : level === "high"
                    ? "forge-chip--warn"
                    : "forge-chip--info",
              )}
              title={
                level === "none"
                  ? "Queue clear"
                  : `Approvals ${approvalsPending} · Reviews ${reviewsPending} · Failures ${recentFailures.length}`
              }
            >
              Queue:{" "}
              {level === "none" ? "clear" : `attention ${attentionCount}`}
            </span>
            <span className="forge-os-statusbar__mode forge-chip forge-chip--muted px-2.5 py-1 text-[10px] font-semibold uppercase tracking-[0.12em]">
              Mode: {uiMode}
            </span>
          </div>
        </header>
      ) : null}

      <div className="forge-os-stage">
        {fallbackNotice ? (
          <div className="forge-os-banner">
            <span>{fallbackNotice}</span>
            <button
              type="button"
              onClick={() => clearFallbackNotice()}
              className="forge-inline-link"
            >
              Dismiss
            </button>
          </div>
        ) : null}

        <main className="forge-os-desktop">
          {/* Wallpaper-only when no visible FORGE windows are present. */}
          {shellRenderedWindows.length === 0 ? (
            <ForgeHero lastErr={lastErr} />
          ) : null}

          {!detachedTauriShell
            ? shellRenderedWindows.map(({ window: win, placement }) => (
                <FloatingWindow
                  key={win.id}
                  window={win}
                  placement={placement}
                  interactive={(win.hostLabel || "main") === hostLabel}
                  focused={focusedId === win.id}
                  onFocus={() => void focus(win.id)}
                  onMinimize={() => void minimize(win.id)}
                  onClose={() => void closeWindow(win.id)}
                  onToggleMaximize={() => toggleMaximize(win.id)}
                  onMove={(x, y) => moveAcrossShellHosts(win, x, y)}
                  onResize={(w, h) => resize(win.id, w, h)}
                />
              ))
            : null}

          {/* Hidden router-driven children for deep-link routes. Detached
              Tauri compatibility renders routes in separate webviews. */}
          {!detachedTauriShell ? (
            <div className="forge-os-router-sink" aria-hidden>
              {props.children}
            </div>
          ) : null}
        </main>
      </div>

      <footer className="forge-os-taskbar">
        <button
          type="button"
          onClick={() => setStartOpen((value) => !value)}
          className={cx(
            "forge-os-taskbar__start",
            startOpen && "forge-os-taskbar__start--active",
          )}
          aria-label="Open Start menu"
          aria-expanded={startOpen}
        >
          <img
            className="forge-os-taskbar__anvil"
            src="/brand/forge-start-button.png"
            alt=""
            draggable={false}
          />
          <span className="forge-os-taskbar__label">FORGE</span>
        </button>

        <div className="forge-os-taskbar__items">
          {dockTiles.map((tile) => {
            const isActive =
              tile.window != null &&
              !tile.window.minimized &&
              focusedId === tile.window.id;
            const isOpen = tile.window != null;
            const isMinimized = tile.window?.minimized ?? false;
            return (
              <button
                key={tile.key}
                type="button"
                onClick={() => {
                  if (tile.window) {
                    focusFromDock(tile.window);
                  } else {
                    launchTool(tile.tool);
                  }
                }}
                onContextMenu={(event) => {
                  event.preventDefault();
                  setContextMenu({
                    x: event.clientX,
                    y: event.clientY,
                    tool: tile.tool,
                    window: tile.window,
                    pinned: tile.pinned,
                    open: isOpen,
                  });
                }}
                onAuxClick={(event) => {
                  if (event.button === 1 && tile.window) {
                    event.preventDefault();
                    void closeWindow(tile.window.id);
                  }
                }}
                className={cx(
                  "forge-os-taskbar__item",
                  isActive && "forge-os-taskbar__item--active",
                  !isActive && isOpen && "forge-os-taskbar__item--open",
                  isMinimized && "forge-os-taskbar__item--minimized",
                )}
                aria-current={isActive ? "page" : undefined}
                title={
                  isActive
                    ? detachedTauriShell
                      ? `${tile.tool.label} (active)`
                      : `${tile.tool.label} (click to minimize)`
                    : isMinimized
                      ? `${tile.tool.label} (minimized)`
                      : isOpen
                        ? `${tile.tool.label} (open)`
                        : tile.tool.label
                }
              >
                <span className="forge-os-taskbar__short">
                  {tile.tool.shortLabel}
                </span>
                <span className="forge-os-taskbar__name">
                  {tile.tool.label}
                </span>
              </button>
            );
          })}
          {isMainWindow
            ? linuxWindows.map((window_) => (
                <button
                  key={window_.id}
                  type="button"
                  onClick={() => void focusLinuxWindow(window_.id)}
                  onContextMenu={(event) => {
                    event.preventDefault();
                    setNativeContextMenu({
                      x: event.clientX,
                      y: event.clientY,
                      window: window_,
                    });
                  }}
                  onAuxClick={(event) => {
                    if (event.button === 1) {
                      event.preventDefault();
                      void runNativeWindowAction(window_, "minimize");
                    }
                  }}
                  className={cx(
                    "forge-os-taskbar__item",
                    "forge-os-taskbar__item--native",
                    "forge-os-taskbar__item--open",
                    window_.focused && "forge-os-taskbar__item--active",
                    window_.minimized && "forge-os-taskbar__item--minimized",
                  )}
                  aria-label={`${window_.title} linux app`}
                  title={
                    window_.appId
                      ? `${window_.title} · ${window_.appId}`
                      : window_.title
                  }
                >
                  <LinuxWindowIcon
                    window={window_}
                    className="forge-os-taskbar__native-icon"
                  />
                  <span className="forge-os-taskbar__name">
                    {window_.title}
                  </span>
                </button>
              ))
            : null}
          {isMainWindow
            ? pendingOperatorApps.map((item) => (
                <button
                  key={item.app.id}
                  type="button"
                  onClick={() => void launchNativeApp(item.app)}
                  onAuxClick={(event) => {
                    if (event.button === 1) {
                      event.preventDefault();
                      setRunningOperatorApps((items) =>
                        items.filter(
                          (candidate) => candidate.app.id !== item.app.id,
                        ),
                      );
                    }
                  }}
                  className="forge-os-taskbar__item forge-os-taskbar__item--native forge-os-taskbar__item--open"
                  aria-label={`${item.app.label} native app`}
                  title={
                    item.pid
                      ? `${item.app.label} native app · PID ${item.pid}`
                      : `${item.app.label} native app`
                  }
                >
                  <OperatorAppIcon
                    app={item.app}
                    className="forge-os-taskbar__native-icon"
                  />
                  <span className="forge-os-taskbar__name">
                    {item.app.label}
                  </span>
                  {item.pid ? (
                    <span className="forge-os-taskbar__pid">
                      PID {item.pid}
                    </span>
                  ) : null}
                </button>
              ))
            : null}
        </div>

        <div className="forge-os-taskbar__system">
          <span>
            {now.toLocaleTimeString([], {
              hour: "numeric",
              minute: "2-digit",
            })}
          </span>
          <span>
            {now.toLocaleDateString([], {
              day: "numeric",
              month: "short",
            })}
          </span>
        </div>
      </footer>

      {startOpen ? (
        <StartMenu
          query={startQuery}
          onQueryChange={setStartQuery}
          onClose={() => {
            setStartOpen(false);
            setStartQuery("");
          }}
          onLaunch={launchTool}
          onContextMenu={(event, tool) => {
            event.preventDefault();
            const isPinned = pinned.includes(tool.id);
            const openWindow =
              windows.find(
                (w) =>
                  w.toolId === tool.id && (w.hostLabel || "main") === hostLabel,
              ) ??
              windows.find((w) => w.toolId === tool.id) ??
              null;
            setStartOpen(false);
            setContextMenu({
              x: event.clientX,
              y: event.clientY,
              tool,
              window: openWindow,
              pinned: isPinned,
              open: openWindow != null,
            });
          }}
          activeTool={focusedTool}
          workspaceLabel={workspaceLabel}
          uiMode={uiMode}
          pinnedIds={pinned}
          onPowerAction={(action) => void handleStartPowerAction(action)}
          operatorApps={operatorApps}
          operatorAppStatus={operatorAppStatus}
          onLaunchOperatorApp={(app) => void launchNativeApp(app)}
        />
      ) : null}

      {contextMenu ? (
        <DockContextMenuView
          menu={contextMenu}
          onClose={() => setContextMenu(null)}
          onAction={(action) => {
            const tool = contextMenu.tool;
            if (action === "open") {
              if (contextMenu.window) {
                focusFromDock(contextMenu.window);
              } else {
                launchTool(tool);
              }
            } else if (action === "close") {
              if (contextMenu.window) {
                void closeWindow(contextMenu.window.id);
              } else {
                void closeByTool(tool.id);
              }
            } else if (action === "pin") {
              pin(tool.id);
            } else if (action === "unpin") {
              unpin(tool.id);
            }
            setContextMenu(null);
          }}
        />
      ) : null}
      {nativeContextMenu ? (
        <NativeWindowContextMenuView
          menu={nativeContextMenu}
          onClose={() => setNativeContextMenu(null)}
          onAction={(action) => {
            void runNativeWindowAction(nativeContextMenu.window, action);
            setNativeContextMenu(null);
          }}
        />
      ) : null}
    </div>
  );
}
