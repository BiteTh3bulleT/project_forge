import type { DashboardSummary } from "@forge/shared";
import {
  useEffect,
  useMemo,
  useRef,
  useState,
  type MouseEvent as ReactMouseEvent,
  type PointerEvent as ReactPointerEvent,
  type CSSProperties,
  type ReactNode,
} from "react";
import { useLocation, useNavigate } from "react-router-dom";

import { api } from "../lib/api";
import {
  DETACHED_TAURI_TOOL_WINDOWS,
  iconAssetUrl,
  focusLinuxWindow,
  isShellHostWindowLabel,
  isTauriDesktop,
  launchOperatorApp,
  listForgeWindows,
  listLinuxWindows,
  listOperatorApps,
  type LinuxWindowSnapshot,
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
  projectDesktopWindowToHost,
  resolveDesktopHostPlacement,
  type DesktopPlacement,
} from "./desktopGeometry";
import { getToolComponent } from "./toolRegistry";

type AppShellProps = {
  children: ReactNode;
  isMainWindow: boolean;
  hostLabel?: string;
};

type AttentionLevel = "none" | "low" | "medium" | "high";
type RunningOperatorApp = {
  app: OperatorApp;
  pid: number | null;
  launchedAtMs: number;
};

const HOME_ROUTE = "/";
const MIN_WINDOW_W = 360;
const MIN_WINDOW_H = 280;
const FALLBACK_OPERATOR_APPS: OperatorApp[] = [
  {
    id: "terminal",
    label: "Foot",
    description: "A Wayland native terminal emulator",
    executable: "foot",
    category: "Workspace",
    iconName: "foot",
    iconPath: "/run/current-system/sw/share/icons/hicolor/48x48/apps/foot.png",
    desktopFile: "/run/current-system/sw/share/applications/foot.desktop",
    native: true,
  },
  {
    id: "files",
    label: "PCMan File Manager",
    description: "Browse the file system and manage files",
    executable: "pcmanfm",
    category: "Workspace",
    iconName: "system-file-manager",
    iconPath: null,
    desktopFile: "/run/current-system/sw/share/applications/pcmanfm.desktop",
    native: true,
  },
  {
    id: "editor",
    label: "Editor",
    description: "Open the fixed operator editor wrapper",
    executable: "foot",
    category: "Workspace",
    iconName: "accessories-text-editor",
    iconPath: null,
    desktopFile: null,
    native: false,
  },
  {
    id: "browser",
    label: "Firefox",
    description: "Open local docs, web consoles, and model tooling",
    executable: "firefox",
    category: "Internet",
    iconName: "firefox",
    iconPath:
      "/run/current-system/sw/share/icons/hicolor/128x128/apps/firefox.png",
    desktopFile: "/run/current-system/sw/share/applications/firefox.desktop",
    native: true,
  },
  {
    id: "ollama-status",
    label: "Ollama Status",
    description: "Show local Ollama status without model mutation",
    executable: "foot",
    category: "AI Runtime",
    iconName: "utilities-terminal",
    iconPath: null,
    desktopFile: null,
    native: false,
  },
  {
    id: "system-monitor",
    label: "System Monitor",
    description: "Open the fixed btop/htop wrapper",
    executable: "foot",
    category: "System",
    iconName: "utilities-system-monitor",
    iconPath: null,
    desktopFile: null,
    native: false,
  },
  {
    id: "lazygit",
    label: "Git UI",
    description: "Open lazygit in the FORGE workspace",
    executable: "foot",
    category: "Developer",
    iconName: "git",
    iconPath: null,
    desktopFile: null,
    native: false,
  },
  {
    id: "forge-status",
    label: "FORGE Status",
    description: "Show local forge-core health",
    executable: "foot",
    category: "FORGE",
    iconName: "utilities-terminal",
    iconPath: null,
    desktopFile: null,
    native: false,
  },
];

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
  const isMainWindow = props.isMainWindow;
  const hostLabel = props.hostLabel?.trim() || "main";

  const isHome = pathname === HOME_ROUTE;

  // Deep links open confined in-shell desktop windows. The disabled detached
  // Tauri compatibility path owns tool routes in separate webviews.
  useEffect(() => {
    if (detachedTauriShell) return;
    if (isHome) return;
    if (!isMainWindow) return;
    const tool = currentTool;
    if (tool.id === "other" || tool.id === "job-detail") return;
    const existing = windows.find(
      (w) => w.toolId === tool.id && (w.hostLabel || "main") === hostLabel,
    );
    if (!existing) {
      void openWindow(tool.id, { hostLabel });
    } else if (existing.minimized || focusedId !== existing.id) {
      void restore(existing.id);
    }
    // intentionally omit windows / focusedId: this effect only reacts to route changes
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [pathname, isMainWindow, isHome, detachedTauriShell, hostLabel]);

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
    void openWindow(tool.id, { hostLabel });
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

  function resolveTransferredWindow(
    win: DesktopWindow,
    nextX: number,
    nextY: number,
  ) {
    return resolveDesktopHostPlacement(
      runtimeWindows,
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
      moveToHost(win.id, transferred.hostLabel, transferred.x, transferred.y);
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

  // Dock = pinned tools (in pinned order) + any open windows whose tool
  // isn't pinned (in open order).
  const dockTiles = useMemo<DockTile[]>(() => {
    const toolMap = new Map<ShellToolId, ShellToolDefinition>(
      allShellTools.map((t) => [t.id, t] as const),
    );
    const tiles: DockTile[] = [];
    for (const toolId of pinned) {
      const tool = toolMap.get(toolId);
      if (!tool) continue;
      const window_ =
        windows.find(
          (w) => w.toolId === toolId && (w.hostLabel || "main") === hostLabel,
        ) ?? null;
      tiles.push({ kind: "tile", tool, window: window_, pinned: true });
    }
    const pinnedSet = new Set(pinned);
    for (const window_ of windows) {
      if ((window_.hostLabel || "main") !== hostLabel) continue;
      if (pinnedSet.has(window_.toolId)) continue;
      const tool = toolMap.get(window_.toolId);
      if (!tool) continue;
      tiles.push({ kind: "tile", tool, window: window_, pinned: false });
    }
    return tiles;
  }, [hostLabel, pinned, windows]);

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
          <div className="forge-os-statusbar__right">
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

      {isMainWindow ? (
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
                  key={tile.tool.id}
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
            {linuxWindows.map((window_) => (
              <button
                key={window_.id}
                type="button"
                onClick={() => void focusLinuxWindow(window_.id)}
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
                <span className="forge-os-taskbar__name">{window_.title}</span>
              </button>
            ))}
            {pendingOperatorApps.map((item) => (
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
                <span className="forge-os-taskbar__name">{item.app.label}</span>
                {item.pid ? (
                  <span className="forge-os-taskbar__pid">PID {item.pid}</span>
                ) : null}
              </button>
            ))}
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
      ) : null}

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
            const isOpen = windows.some((w) => w.toolId === tool.id);
            setStartOpen(false);
            setContextMenu({
              x: event.clientX,
              y: event.clientY,
              tool,
              pinned: isPinned,
              open: isOpen,
            });
          }}
          activeTool={focusedTool}
          workspaceLabel={workspaceLabel}
          uiMode={uiMode}
          pinnedIds={pinned}
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
              launchTool(tool);
            } else if (action === "close") {
              void closeByTool(tool.id);
            } else if (action === "pin") {
              pin(tool.id);
            } else if (action === "unpin") {
              unpin(tool.id);
            }
            setContextMenu(null);
          }}
        />
      ) : null}
    </div>
  );
}

type DockTile = {
  kind: "tile";
  tool: ShellToolDefinition;
  window: DesktopWindow | null;
  pinned: boolean;
};

type DockContextMenu = {
  x: number;
  y: number;
  tool: ShellToolDefinition;
  pinned: boolean;
  open: boolean;
};

function DesktopWallpaper() {
  return (
    <div className="forge-os-wallpaper" aria-hidden>
      <picture className="forge-os-wallpaper__art">
        <source
          media="(orientation: portrait)"
          srcSet="/brand/forge-vertical.png"
        />
        <img src="/brand/forge-horizontal.png" alt="" draggable={false} />
      </picture>
      <div className="forge-os-wallpaper__gradient" />
      <div className="forge-os-wallpaper__grid" />
      <div className="forge-os-wallpaper__glow" />
    </div>
  );
}

function ForgeHero(props: { lastErr: string | null }) {
  if (!props.lastErr) return null;
  return (
    <div className="forge-os-hero">
      <div className="forge-os-hero__panel forge-os-hero__panel--error">
        <div className="forge-os-hero__error">{props.lastErr}</div>
      </div>
    </div>
  );
}

function FloatingWindow(props: {
  window: DesktopWindow;
  placement: DesktopPlacement;
  interactive: boolean;
  focused: boolean;
  onFocus: () => void;
  onMinimize: () => void;
  onClose: () => void;
  onToggleMaximize: () => void;
  onMove: (x: number, y: number) => void;
  onResize: (width: number, height: number) => void;
}) {
  const tool = useMemo(
    () => allShellTools.find((t) => t.id === props.window.toolId) ?? null,
    [props.window.toolId],
  );
  const Component = tool ? getToolComponent(tool.id) : null;
  const dragRef = useRef<{
    pointerStartX: number;
    pointerStartY: number;
    windowStartX: number;
    windowStartY: number;
  } | null>(null);
  const resizeRef = useRef<{
    startX: number;
    startY: number;
    startW: number;
    startH: number;
  } | null>(null);

  function startDrag(event: ReactPointerEvent<HTMLDivElement>) {
    if (!props.interactive) return;
    if (props.window.maximized) return;
    if (event.button !== 0) return;
    event.currentTarget.setPointerCapture?.(event.pointerId);
    event.preventDefault();
    dragRef.current = {
      pointerStartX: event.clientX,
      pointerStartY: event.clientY,
      windowStartX: props.window.x,
      windowStartY: props.window.y,
    };
    props.onFocus();
    const onMove = (e: PointerEvent) => {
      if (!dragRef.current) return;
      const x =
        dragRef.current.windowStartX +
        (e.clientX - dragRef.current.pointerStartX);
      const y =
        dragRef.current.windowStartY +
        (e.clientY - dragRef.current.pointerStartY);
      props.onMove(x, y);
    };
    const onUp = () => {
      dragRef.current = null;
      window.removeEventListener("pointermove", onMove);
      window.removeEventListener("pointerup", onUp);
    };
    window.addEventListener("pointermove", onMove);
    window.addEventListener("pointerup", onUp);
  }

  function startResize(event: ReactPointerEvent<HTMLDivElement>) {
    if (!props.interactive) return;
    if (props.window.maximized) return;
    if (event.button !== 0) return;
    event.currentTarget.setPointerCapture?.(event.pointerId);
    event.preventDefault();
    event.stopPropagation();
    resizeRef.current = {
      startX: event.clientX,
      startY: event.clientY,
      startW: props.window.width,
      startH: props.window.height,
    };
    props.onFocus();
    const onMove = (e: PointerEvent) => {
      if (!resizeRef.current) return;
      const dx = e.clientX - resizeRef.current.startX;
      const dy = e.clientY - resizeRef.current.startY;
      const w = Math.max(MIN_WINDOW_W, resizeRef.current.startW + dx);
      const h = Math.max(MIN_WINDOW_H, resizeRef.current.startH + dy);
      props.onResize(w, h);
    };
    const onUp = () => {
      resizeRef.current = null;
      window.removeEventListener("pointermove", onMove);
      window.removeEventListener("pointerup", onUp);
    };
    window.addEventListener("pointermove", onMove);
    window.addEventListener("pointerup", onUp);
  }

  if (props.window.minimized) return null;

  const style: CSSProperties = props.window.maximized
    ? {
        left: 0,
        top: 0,
        width: "100%",
        height: "100%",
        zIndex: props.window.z,
      }
    : {
        left: props.placement.x,
        top: props.placement.y,
        width: props.window.width,
        height: props.window.height,
        zIndex: props.window.z,
        pointerEvents: props.interactive ? undefined : "none",
      };

  return (
    <section
      className={cx(
        "forge-os-window",
        props.focused && "forge-os-window--focused",
        props.window.maximized && "forge-os-window--maximized",
      )}
      style={style}
      aria-hidden={props.interactive ? undefined : true}
      onPointerDown={() => {
        if (props.interactive && !props.focused) props.onFocus();
      }}
    >
      <div
        className="forge-os-window__chrome"
        onPointerDown={startDrag}
        onDoubleClick={props.onToggleMaximize}
      >
        <div className="forge-os-window__title">
          <span className="forge-os-window__sigil">
            {tool?.shortLabel ?? "??"}
          </span>
          <div className="min-w-0">
            <div className="forge-os-window__name">
              {tool?.label ?? "Unknown surface"}
            </div>
            <div className="forge-os-window__sub">
              {tool?.description ?? ""}
            </div>
          </div>
        </div>
        <div className="forge-os-window__buttons">
          <button
            type="button"
            className="forge-os-window__btn"
            onPointerDown={(e) => e.stopPropagation()}
            onClick={(e) => {
              e.stopPropagation();
              props.onMinimize();
            }}
            aria-label="Minimize"
            title="Minimize"
          >
            –
          </button>
          <button
            type="button"
            className="forge-os-window__btn"
            onPointerDown={(e) => e.stopPropagation()}
            onClick={(e) => {
              e.stopPropagation();
              props.onToggleMaximize();
            }}
            aria-label={props.window.maximized ? "Restore" : "Maximize"}
            title={props.window.maximized ? "Restore" : "Maximize"}
          >
            {props.window.maximized ? "❐" : "▢"}
          </button>
          <button
            type="button"
            className="forge-os-window__btn forge-os-window__btn--close"
            onPointerDown={(e) => e.stopPropagation()}
            onClick={(e) => {
              e.stopPropagation();
              props.onClose();
            }}
            aria-label="Close"
            title="Close"
          >
            ×
          </button>
        </div>
      </div>
      <div className="forge-os-window__body">
        <div className="forge-os-window__content">
          {Component ? (
            <Component />
          ) : (
            <UnsupportedToolNotice toolId={tool?.id} />
          )}
        </div>
      </div>
      {!props.window.maximized ? (
        <div
          className="forge-os-window__resize"
          onPointerDown={startResize}
          aria-hidden
        />
      ) : null}
    </section>
  );
}

function UnsupportedToolNotice(props: { toolId: string | undefined }) {
  return (
    <div className="forge-os-window__placeholder">
      <div className="forge-os-window__placeholder-title">
        Surface unavailable
      </div>
      <div className="forge-os-window__placeholder-body">
        No window component is registered for {props.toolId ?? "this surface"}.
      </div>
    </div>
  );
}

function StartMenu(props: {
  query: string;
  onQueryChange: (value: string) => void;
  onClose: () => void;
  onLaunch: (tool: ShellToolDefinition) => void;
  onContextMenu: (
    event: ReactMouseEvent<HTMLButtonElement>,
    tool: ShellToolDefinition,
  ) => void;
  activeTool: ShellToolDefinition | null;
  workspaceLabel: string;
  uiMode: "cognitive" | "metrics";
  pinnedIds: ShellToolId[];
  operatorApps: OperatorApp[];
  operatorAppStatus: string | null;
  onLaunchOperatorApp: (app: OperatorApp) => void;
}) {
  const query = props.query.trim().toLowerCase();
  const filteredTools = useMemo(() => {
    return allShellTools.filter((tool) => {
      const haystack =
        `${tool.label} ${tool.shortLabel} ${tool.description}`.toLowerCase();
      return !query || haystack.includes(query);
    });
  }, [query]);
  const filteredApps = useMemo(() => {
    return props.operatorApps.filter((app) => {
      const haystack =
        `${app.label} ${app.description} ${app.executable} ${app.category} ${app.iconName ?? ""}`.toLowerCase();
      return !query || haystack.includes(query);
    });
  }, [props.operatorApps, query]);

  const pinnedTools = useMemo(() => {
    const map = new Map<ShellToolId, ShellToolDefinition>(
      allShellTools.map((t) => [t.id, t] as const),
    );
    return props.pinnedIds
      .map((id) => map.get(id))
      .filter((t): t is ShellToolDefinition => Boolean(t));
  }, [props.pinnedIds]);
  const toolGroups = useMemo(
    () => groupToolsByCategory(filteredTools),
    [filteredTools],
  );
  const appGroups = useMemo(
    () => groupAppsByCategory(filteredApps),
    [filteredApps],
  );

  return (
    <>
      <button
        type="button"
        className="forge-os-startmenu__backdrop"
        aria-label="Close Start menu"
        onClick={props.onClose}
      />
      <section
        className="forge-os-startmenu"
        role="dialog"
        aria-label="FORGE Start"
      >
        <header className="forge-os-startmenu__head">
          <div className="forge-os-startmenu__operator">
            <div className="forge-os-startmenu__avatar">
              <img
                src="/brand/forge-start-button.png"
                alt=""
                draggable={false}
              />
            </div>
            <div className="min-w-0">
              <div className="forge-os-startmenu__operator-name">
                Forge Operator
              </div>
              <div className="forge-os-startmenu__operator-meta">
                {props.workspaceLabel} · Mode {props.uiMode}
              </div>
            </div>
          </div>
          <div className="forge-os-startmenu__search">
            <input
              type="text"
              value={props.query}
              onChange={(event) => props.onQueryChange(event.target.value)}
              placeholder="Search FORGE surfaces"
              className="forge-os-startmenu__input"
              autoFocus
            />
          </div>
        </header>

        {!query && pinnedTools.length > 0 ? (
          <div className="forge-os-startmenu__section forge-os-startmenu__section--pinned">
            <div className="forge-os-startmenu__section-label">
              Quick Launch
            </div>
            <div className="forge-os-startmenu__grid">
              {pinnedTools.map((tool) => (
                <button
                  key={tool.id}
                  type="button"
                  onClick={() => props.onLaunch(tool)}
                  onContextMenu={(event) => props.onContextMenu(event, tool)}
                  className={cx(
                    "forge-os-startmenu__tile",
                    props.activeTool?.id === tool.id &&
                      "forge-os-startmenu__tile--active",
                  )}
                  title={`${tool.description} (right-click to unpin)`}
                >
                  <span className="forge-os-startmenu__tile-short">
                    {tool.shortLabel}
                  </span>
                  <span className="forge-os-startmenu__tile-label">
                    {tool.label}
                  </span>
                </button>
              ))}
            </div>
          </div>
        ) : null}

        <div className="forge-os-startmenu__body">
          <div className="forge-os-startmenu__panel forge-os-startmenu__panel--native">
            <div className="forge-os-startmenu__panel-head">
              <div className="forge-os-startmenu__section-label">
                Native Apps
              </div>
              <div className="forge-os-startmenu__panel-meta">
                {filteredApps.length}{" "}
                {filteredApps.length === 1 ? "app" : "apps"}
              </div>
            </div>
            <div className="forge-os-startmenu__list forge-os-startmenu__list--native">
              {appGroups.length === 0 ? (
                <div className="forge-os-startmenu__empty">
                  {query
                    ? `No native apps match "${props.query}".`
                    : "No native operator apps are available."}
                </div>
              ) : (
                appGroups.map((group) => (
                  <div
                    key={group.category}
                    className="forge-os-startmenu__group"
                  >
                    <div className="forge-os-startmenu__group-head">
                      <div className="forge-os-startmenu__group-label">
                        {group.category}
                      </div>
                      <div className="forge-os-startmenu__group-count">
                        {group.items.length}{" "}
                        {group.items.length === 1 ? "app" : "apps"}
                      </div>
                    </div>
                    {group.items.map((app) => (
                      <button
                        key={app.id}
                        type="button"
                        onClick={() => props.onLaunchOperatorApp(app)}
                        className="forge-os-startmenu__row"
                      >
                        <OperatorAppIcon
                          app={app}
                          className="forge-os-startmenu__app-icon"
                        />
                        <span className="min-w-0 flex-1">
                          <span className="forge-os-startmenu__row-label">
                            {app.label}
                          </span>
                          <span className="forge-os-startmenu__row-desc">
                            {app.description}
                          </span>
                        </span>
                      </button>
                    ))}
                  </div>
                ))
              )}
            </div>
            {props.operatorAppStatus ? (
              <div className="forge-os-startmenu__status">
                {props.operatorAppStatus}
              </div>
            ) : null}
          </div>

          <div className="forge-os-startmenu__panel forge-os-startmenu__panel--surfaces">
            <div className="forge-os-startmenu__panel-head">
              <div className="forge-os-startmenu__section-label">
                {query ? "FORGE Results" : "FORGE Surfaces"}
              </div>
              <div className="forge-os-startmenu__panel-meta">
                {filteredTools.length}{" "}
                {filteredTools.length === 1 ? "surface" : "surfaces"}
              </div>
            </div>
            <div className="forge-os-startmenu__list">
              {toolGroups.length === 0 ? (
                <div className="forge-os-startmenu__empty">
                  No FORGE surfaces match "{props.query}".
                </div>
              ) : (
                toolGroups.map((group) => (
                  <div
                    key={group.category}
                    className="forge-os-startmenu__group"
                  >
                    <div className="forge-os-startmenu__group-head">
                      <div className="forge-os-startmenu__group-label">
                        {group.category}
                      </div>
                      <div className="forge-os-startmenu__group-count">
                        {group.items.length}{" "}
                        {group.items.length === 1 ? "item" : "items"}
                      </div>
                    </div>
                    {group.items.map((tool) => (
                      <button
                        key={tool.id}
                        type="button"
                        onClick={() => props.onLaunch(tool)}
                        onContextMenu={(event) =>
                          props.onContextMenu(event, tool)
                        }
                        className={cx(
                          "forge-os-startmenu__row",
                          props.activeTool?.id === tool.id &&
                            "forge-os-startmenu__row--active",
                        )}
                      >
                        <span className="forge-os-startmenu__row-short">
                          {tool.shortLabel}
                        </span>
                        <span className="min-w-0 flex-1">
                          <span className="forge-os-startmenu__row-label">
                            {tool.label}
                          </span>
                          <span className="forge-os-startmenu__row-desc">
                            {tool.description}
                          </span>
                        </span>
                      </button>
                    ))}
                  </div>
                ))
              )}
            </div>
          </div>
        </div>
      </section>
    </>
  );
}

function groupToolsByCategory(tools: ShellToolDefinition[]) {
  const groups = new Map<string, ShellToolDefinition[]>();
  for (const tool of tools) {
    const category = shellToolCategory(tool.id);
    groups.set(category, [...(groups.get(category) ?? []), tool]);
  }
  return Array.from(groups, ([category, items]) => ({ category, items }));
}

function groupAppsByCategory(apps: OperatorApp[]) {
  const groups = new Map<string, OperatorApp[]>();
  for (const app of apps) {
    const category = app.category?.trim() || "Tools";
    groups.set(category, [...(groups.get(category) ?? []), app]);
  }
  return Array.from(groups, ([category, items]) => ({ category, items }));
}

function shellToolCategory(id: ShellToolId) {
  if (
    [
      "chat",
      "start",
      "dashboard",
      "system",
      "settings",
      "operator-apps",
    ].includes(id)
  ) {
    return "Core";
  }
  if (
    ["workbench", "canvas", "dossiers", "jobs", "reviews", "layouts"].includes(
      id,
    )
  ) {
    return "Workspace";
  }
  if (
    [
      "memory",
      "project-context",
      "sources",
      "insights",
      "lineage",
      "retrieval-runs",
      "evaluations",
    ].includes(id)
  ) {
    return "Knowledge";
  }
  if (
    [
      "models",
      "gateway",
      "adapters",
      "command",
      "action-lanes",
      "automation",
    ].includes(id)
  ) {
    return "Runtime";
  }
  if (
    [
      "approvals",
      "autonomy",
      "audit",
      "policy",
      "strategies",
      "execution-permissions",
      "inspectors",
    ].includes(id)
  ) {
    return "Governance";
  }
  if (["logs", "backup", "release"].includes(id)) {
    return "Operations";
  }
  return "Tools";
}

function OperatorAppIcon(props: { app: OperatorApp; className?: string }) {
  const iconPath = props.app.iconPath?.trim();
  if (iconPath) {
    return (
      <img
        src={iconAssetUrl(iconPath)}
        alt={`${props.app.label} icon`}
        className={props.className}
        draggable={false}
      />
    );
  }
  const short = props.app.label
    .split(/\s+/)
    .filter(Boolean)
    .slice(0, 2)
    .map((part) => part[0]?.toUpperCase() ?? "")
    .join("");
  return (
    <span className={cx("forge-os-startmenu__app-icon", props.className)}>
      {short || "AP"}
    </span>
  );
}

function LinuxWindowIcon(props: {
  window: LinuxWindowSnapshot;
  className?: string;
}) {
  const iconPath = props.window.iconPath?.trim();
  if (iconPath) {
    return (
      <img
        src={iconAssetUrl(iconPath)}
        alt={`${props.window.title} icon`}
        className={props.className}
        draggable={false}
      />
    );
  }
  const label = props.window.title || props.window.appId;
  const short = label
    .split(/\s+/)
    .filter(Boolean)
    .slice(0, 2)
    .map((part) => part[0]?.toUpperCase() ?? "")
    .join("");
  return (
    <span className={cx("forge-os-startmenu__app-icon", props.className)}>
      {short || "LN"}
    </span>
  );
}

function DockContextMenuView(props: {
  menu: DockContextMenu;
  onClose: () => void;
  onAction: (action: "open" | "close" | "pin" | "unpin") => void;
}) {
  // Clamp position so the menu doesn't overflow the viewport.
  const left = Math.min(
    props.menu.x,
    typeof window !== "undefined" ? window.innerWidth - 220 : props.menu.x,
  );
  const top = Math.max(
    8,
    typeof window !== "undefined"
      ? Math.min(props.menu.y - 8, window.innerHeight - 220)
      : props.menu.y - 8,
  );
  return (
    <div
      role="menu"
      className="forge-os-context-menu"
      style={{ left, top }}
      onMouseDown={(event) => event.stopPropagation()}
    >
      <div className="forge-os-context-menu__title">
        {props.menu.tool.label}
      </div>
      <button
        type="button"
        role="menuitem"
        className="forge-os-context-menu__item"
        onClick={() => props.onAction("open")}
      >
        {props.menu.open ? "Focus window" : "Open"}
      </button>
      {props.menu.open ? (
        <button
          type="button"
          role="menuitem"
          className="forge-os-context-menu__item"
          onClick={() => props.onAction("close")}
        >
          Close window
        </button>
      ) : null}
      {props.menu.pinned ? (
        <button
          type="button"
          role="menuitem"
          className="forge-os-context-menu__item"
          onClick={() => props.onAction("unpin")}
        >
          Unpin from taskbar
        </button>
      ) : (
        <button
          type="button"
          role="menuitem"
          className="forge-os-context-menu__item"
          onClick={() => props.onAction("pin")}
        >
          Pin to taskbar
        </button>
      )}
      <button
        type="button"
        role="menuitem"
        className="forge-os-context-menu__item forge-os-context-menu__item--muted"
        onClick={props.onClose}
      >
        Cancel
      </button>
    </div>
  );
}
