import type { DashboardSummary, ForgeEvent } from "@forge/shared";
import { useEffect, useMemo, useRef, useState, type ReactNode } from "react";
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
  readHostPowerPolicy,
  requestHostPowerAction,
  subscribeToDesktopNotifications,
  type DesktopNotification,
  type ForgeHostPowerAction,
  type ForgeHostPowerPolicy,
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
import { FALLBACK_OPERATOR_APPS } from "../lib/operatorApps";

import {
  allShellTools,
  getShellTool,
  type ShellToolDefinition,
  type ShellToolId,
} from "./shellConfig";
import {
  DesktopWallpaper,
  DockContextMenuView,
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
  onForgeLock?: () => void | Promise<void>;
  onForgeLogout?: () => void | Promise<void>;
};

type AttentionLevel = "none" | "low" | "medium" | "high";
type RunningOperatorApp = {
  app: OperatorApp;
  pid: number | null;
  launchedAtMs: number;
  ignoredWindowIds: string[];
};
type ShellAuditRecord = {
  id?: number | string;
  createdAtMs?: number;
  category?: string;
  action?: string;
  outcome?: string;
  summary?: string;
  correlationId?: string;
};
type ShellNotificationItem = {
  id: string;
  createdAtMs: number;
  type: string;
  detail: string;
  body?: string;
};
type ShellContextSnapshot = {
  id?: string;
  createdAtMs?: number;
  workspaceId?: string;
  laneId?: string;
  snapshotKind?: string;
  label?: string;
  summary?: string;
};
type ShellTelemetry = {
  auditRecords: ShellAuditRecord[];
  eventRecords: ForgeEvent[];
  autonomyStatus: Record<string, unknown> | null;
  modelQueue: Record<string, unknown> | null;
  modelBackends: Array<Record<string, unknown>>;
  contextSnapshots: ShellContextSnapshot[];
  error: string | null;
};

const HOME_ROUTE = "/";
const EMPTY_TELEMETRY: ShellTelemetry = {
  auditRecords: [],
  eventRecords: [],
  autonomyStatus: null,
  modelQueue: null,
  modelBackends: [],
  contextSnapshots: [],
  error: null,
};
const NATIVE_LAUNCH_PENDING_TTL_MS = 30_000;

function getWorkspaceLabel(workspaceDir: string | null | undefined) {
  if (!workspaceDir) return "Workspace unavailable";
  const parts = workspaceDir.split(/[\\/]/).filter(Boolean);
  return parts.at(-1) ?? workspaceDir;
}

function numberField(
  record: Record<string, unknown> | null | undefined,
  key: string,
) {
  const value = record?.[key];
  return typeof value === "number" && Number.isFinite(value) ? value : null;
}

function auditSummary(record: ShellAuditRecord | null | undefined) {
  if (!record) return "none";
  const action = record.action?.trim() || record.category?.trim() || "event";
  const outcome = record.outcome?.trim();
  return outcome ? `${action} ${outcome}` : action;
}

function auditDetail(record: ShellAuditRecord | null | undefined) {
  if (!record) return "No audit events loaded.";
  return record.summary?.trim() || auditSummary(record);
}

function payloadTextField(payload: unknown, keys: string[]) {
  if (typeof payload !== "object" || payload == null) return "";
  const record = payload as Record<string, unknown>;
  for (const key of keys) {
    const value = record[key];
    if (typeof value === "string" && value.trim()) return value.trim();
    if (typeof value === "number" && Number.isFinite(value)) {
      return String(value);
    }
  }
  return "";
}

function isNotificationEvent(event: ForgeEvent) {
  const type = event.type.toLowerCase();
  if (type.includes("approval")) return true;
  if (type.includes("notification") || type.includes("notify")) return true;
  if (!type.includes("job")) return false;
  return /complete|done|fail|error|cancel|approval/i.test(type);
}

function notificationDetail(event: ForgeEvent) {
  const summary = payloadTextField(event.payload, [
    "summary",
    "title",
    "message",
    "reason",
    "status",
  ]);
  if (summary) return summary;
  const identifier = payloadTextField(event.payload, [
    "jobId",
    "jobID",
    "requestId",
    "id",
  ]);
  return identifier ? `Reference ${identifier}` : "Core event notification";
}

function modelRuntimeSummary(telemetry: ShellTelemetry) {
  const queueDepth = numberField(telemetry.modelQueue, "depth") ?? 0;
  const healthyBackends = telemetry.modelBackends.filter(
    (backend) => backend.healthy === true,
  ).length;
  const unhealthyBackends = telemetry.modelBackends.filter(
    (backend) => backend.healthy === false,
  ).length;
  if (unhealthyBackends > 0) return `degraded · queue ${queueDepth}`;
  if (healthyBackends > 0) return `healthy · queue ${queueDepth}`;
  return `unknown · queue ${queueDepth}`;
}

function autonomySummary(status: Record<string, unknown> | null) {
  if (!status) return "unknown";
  const dream = status.dream;
  const maintenance = status.maintenanceLoop;
  const dreamActive =
    typeof dream === "object" &&
    dream != null &&
    (dream as Record<string, unknown>).active === true;
  const maintenanceActive =
    typeof maintenance === "object" &&
    maintenance != null &&
    (maintenance as Record<string, unknown>).active === true;
  if (dreamActive || maintenanceActive) return "active";
  if (status.available === false || status.enabled === false) return "disabled";
  if (status.available === true || status.enabled === true) return "idle";
  return "unknown";
}

function activeLoopSummary(status: Record<string, unknown> | null) {
  if (!status) return "loops unknown";
  const dream = status.dream;
  if (
    typeof dream === "object" &&
    dream != null &&
    (dream as Record<string, unknown>).active === true
  ) {
    return "dream active";
  }
  const maintenance = status.maintenanceLoop;
  if (
    typeof maintenance === "object" &&
    maintenance != null &&
    (maintenance as Record<string, unknown>).active === true
  ) {
    return "maintenance active";
  }
  const counts = status.counts;
  if (typeof counts === "object" && counts != null) {
    const activeIntents = numberField(
      counts as Record<string, unknown>,
      "activeIntents",
    );
    if (activeIntents != null && activeIntents > 0) {
      return `${activeIntents} active intent${activeIntents === 1 ? "" : "s"}`;
    }
  }
  return "no active loops";
}

function activeLinuxWindowSnapshots(
  windows: LinuxWindowSnapshot[],
): LinuxWindowSnapshot[] {
  return windows.filter((window_) => window_.lifecycle !== "closed");
}

function contextSnapshotLabel(
  snapshot: ShellContextSnapshot | null | undefined,
) {
  if (!snapshot) return "No context snapshot loaded.";
  return (
    snapshot.label?.trim() ||
    snapshot.summary?.trim() ||
    snapshot.snapshotKind?.trim() ||
    snapshot.id?.trim() ||
    "Context snapshot"
  );
}

function desktopHeroError(message: string | null | undefined) {
  const trimmed = message?.trim();
  if (!trimmed) return null;
  const lower = trimmed.toLowerCase();
  if (
    lower.includes("unauthorized") ||
    lower.includes("bearer token") ||
    lower.includes("missing or invalid")
  ) {
    return null;
  }
  if (trimmed.startsWith("{") || trimmed.startsWith("[")) {
    return "FORGE core reported an API error.";
  }
  return trimmed.length > 180 ? `${trimmed.slice(0, 177)}...` : trimmed;
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

function normalizeNativeKey(value: string | null | undefined) {
  return (value ?? "")
    .trim()
    .toLowerCase()
    .replace(/\.desktop$/, "");
}

function operatorAppIdentityKeys(app: OperatorApp) {
  const keys = new Set<string>();
  for (const value of [
    app.id,
    app.label,
    app.executable,
    app.iconName ?? "",
    app.desktopFile?.split(/[\\/]/).at(-1) ?? "",
  ]) {
    const key = normalizeNativeKey(value);
    if (key) keys.add(key);
  }
  return keys;
}

function linuxWindowMatchesOperatorApp(
  window_: LinuxWindowSnapshot,
  app: OperatorApp,
) {
  const windowAppId = normalizeNativeKey(window_.appId);
  const windowTitle = normalizeNativeKey(window_.title);
  const appKeys = operatorAppIdentityKeys(app);

  for (const key of appKeys) {
    if (windowAppId && windowAppId === key) return true;
    if (
      windowAppId &&
      (windowAppId.includes(key) || key.includes(windowAppId))
    ) {
      return true;
    }
    if (key !== "foot" && windowTitle.includes(key)) return true;
  }

  return false;
}

function linuxWindowResolvesOperatorLaunch(
  item: RunningOperatorApp,
  window_: LinuxWindowSnapshot,
) {
  if (item.ignoredWindowIds.includes(window_.id)) return false;
  if (!linuxWindowMatchesOperatorApp(window_, item.app)) return false;
  const seenAtMs = window_.firstSeenMs ?? window_.lastSeenMs;
  return seenAtMs == null || seenAtMs >= item.launchedAtMs - 1500;
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
  const themePreference = useUiStore((s) => s.themePreference);
  const toggleThemePreference = useUiStore((s) => s.toggleThemePreference);
  const accentPreference = useUiStore((s) => s.accentPreference);
  const setAccentPreference = useUiStore((s) => s.setAccentPreference);

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
  const moveLive = useDesktopWindowStore((s) => s.moveLive);
  const moveToHostLive = useDesktopWindowStore((s) => s.moveToHostLive);
  const resizeLive = useDesktopWindowStore((s) => s.resizeLive);
  const commitGeometry = useDesktopWindowStore((s) => s.commitGeometry);
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
  const [hostPowerPolicy, setHostPowerPolicy] =
    useState<ForgeHostPowerPolicy | null>(null);
  const [runningOperatorApps, setRunningOperatorApps] = useState<
    RunningOperatorApp[]
  >([]);
  const launchingOperatorAppIdsRef = useRef<Set<string>>(new Set());
  const [linuxWindows, setLinuxWindows] = useState<LinuxWindowSnapshot[]>([]);
  const [desktopNotifications, setDesktopNotifications] = useState<
    DesktopNotification[]
  >([]);
  const [telemetry, setTelemetry] = useState<ShellTelemetry>(EMPTY_TELEMETRY);
  const [statusDetailsOpen, setStatusDetailsOpen] = useState(false);
  const [activityOpen, setActivityOpen] = useState(false);
  const [notificationOpen, setNotificationOpen] = useState(false);
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
  const routedDetailOpen =
    pathname.startsWith("/jobs/") || pathname.startsWith("/memory/chunk/");
  const routedDetailBackRoute = pathname.startsWith("/jobs/")
    ? "/jobs"
    : "/memory";

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
    if (routedDetailOpen) return;
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
  }, [pathname, isHome, routedDetailOpen, detachedTauriShell, hostLabel]);

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
    if (!isMainWindow) return;
    let cancelled = false;
    async function loadTelemetry() {
      const [
        auditRes,
        eventRes,
        autonomyRes,
        queueRes,
        backendsRes,
        contextRes,
      ] = await Promise.allSettled([
        api.audit.list({ limit: 20 }),
        api.events(20),
        api.autonomy.status(),
        api.modelRuntime.queue(),
        api.modelRuntime.backends(),
        api.contextInspector.listSnapshots({ limit: 3 }),
      ]);
      if (cancelled) return;

      const rejected = [
        auditRes,
        eventRes,
        autonomyRes,
        queueRes,
        backendsRes,
        contextRes,
      ].find((result) => result.status === "rejected");
      setTelemetry({
        auditRecords:
          auditRes.status === "fulfilled"
            ? (auditRes.value.records as ShellAuditRecord[])
            : [],
        eventRecords:
          eventRes.status === "fulfilled" &&
          Array.isArray(eventRes.value.events)
            ? (eventRes.value.events as ForgeEvent[])
            : [],
        autonomyStatus:
          autonomyRes.status === "fulfilled"
            ? (autonomyRes.value as Record<string, unknown>)
            : null,
        modelQueue:
          queueRes.status === "fulfilled"
            ? (queueRes.value.queue as Record<string, unknown>)
            : null,
        modelBackends:
          backendsRes.status === "fulfilled"
            ? (backendsRes.value.backends as Array<Record<string, unknown>>)
            : [],
        contextSnapshots:
          contextRes.status === "fulfilled"
            ? (contextRes.value.snapshots as ShellContextSnapshot[])
            : [],
        error:
          rejected?.status === "rejected"
            ? rejected.reason instanceof Error
              ? rejected.reason.message
              : String(rejected.reason)
            : null,
      });
    }
    void loadTelemetry();
    const id = window.setInterval(() => void loadTelemetry(), 7500);
    return () => {
      cancelled = true;
      window.clearInterval(id);
    };
  }, [isMainWindow]);

  useEffect(() => {
    document.documentElement.dataset.theme = themePreference;
    document.documentElement.dataset.forgeAccent = accentPreference;
  }, [accentPreference, themePreference]);

  useEffect(() => {
    if (!isMainWindow || !isTauriDesktop()) return;
    let cancelled = false;
    async function loadLinuxWindows() {
      const nextWindows = await listLinuxWindows();
      if (cancelled) return;
      setLinuxWindows(activeLinuxWindowSnapshots(nextWindows));
    }
    void loadLinuxWindows();
    const id = window.setInterval(() => void loadLinuxWindows(), 1500);
    return () => {
      cancelled = true;
      window.clearInterval(id);
    };
  }, [isMainWindow]);

  useEffect(() => {
    if (!isMainWindow) return;
    let cancelled = false;
    let unlisten: (() => void) | null = null;
    void subscribeToDesktopNotifications((notification) => {
      setDesktopNotifications((current) => {
        const withoutReplacement = current.filter(
          (entry) => entry.id !== notification.id,
        );
        return [notification, ...withoutReplacement].slice(0, 20);
      });
    })
      .then((nextUnlisten) => {
        if (cancelled) {
          nextUnlisten?.();
        } else {
          unlisten = nextUnlisten;
        }
      })
      .catch(() => {
        // Notification service is opportunistic; core event polling remains active.
      });
    return () => {
      cancelled = true;
      unlisten?.();
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
    async function loadHostPowerPolicy() {
      try {
        const policy = await readHostPowerPolicy();
        if (!cancelled) setHostPowerPolicy(policy);
      } catch (error) {
        if (!cancelled) {
          setHostPowerPolicy({
            directSystemControlEnabled: false,
            message:
              error instanceof Error
                ? error.message
                : "Unable to read host power policy.",
          });
        }
      }
    }
    void loadHostPowerPolicy();
    return () => {
      cancelled = true;
    };
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
        if (statusDetailsOpen) {
          setStatusDetailsOpen(false);
          return;
        }
        if (activityOpen) {
          setActivityOpen(false);
          return;
        }
        if (notificationOpen) {
          setNotificationOpen(false);
          return;
        }
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
  }, [
    activityOpen,
    contextMenu,
    focusedId,
    minimize,
    notificationOpen,
    startOpen,
    statusDetailsOpen,
  ]);

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
    if (
      launchingOperatorAppIdsRef.current.has(app.id) ||
      runningOperatorApps.some((item) => item.app.id === app.id)
    ) {
      setOperatorAppStatus(`${app.label} launch is already pending.`);
      return;
    }
    launchingOperatorAppIdsRef.current.add(app.id);
    const ignoredWindowIds = linuxWindows
      .filter((window_) => linuxWindowMatchesOperatorApp(window_, app))
      .map((window_) => window_.id);
    try {
      const result = await launchOperatorApp(app.id);
      if (!result.launched) {
        setOperatorAppStatus(result.message);
        return;
      }
      setRunningOperatorApps((items) => {
        const next: RunningOperatorApp = {
          app,
          pid: result.pid ?? null,
          launchedAtMs: Date.now(),
          ignoredWindowIds,
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
    } finally {
      launchingOperatorAppIdsRef.current.delete(app.id);
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
    if (!ok) {
      setOperatorAppStatus(
        `${window_.title} did not accept the ${action} request.`,
      );
      return;
    }
    const nextWindows = await listLinuxWindows();
    setLinuxWindows(activeLinuxWindowSnapshots(nextWindows));
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
      moveToHostLive(
        win.id,
        transferred.hostLabel,
        targetPlacement.x,
        targetPlacement.y,
        targetMonitorId,
      );
      return;
    }
    moveLive(win.id, transferred.x, transferred.y);
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
    action: "lock" | "logout" | ForgeHostPowerAction,
  ) {
    setStartOpen(false);
    setStartQuery("");

    if (action === "lock") {
      try {
        await props.onForgeLock?.();
      } catch (error) {
        setOperatorAppStatus(
          error instanceof Error
            ? error.message
            : "Unable to lock the FORGE operator session.",
        );
      }
      return;
    }

    if (action === "logout") {
      try {
        await props.onForgeLogout?.();
        if (!props.onForgeLogout) {
          navigate("/login");
        }
      } catch (error) {
        setOperatorAppStatus(
          error instanceof Error
            ? error.message
            : "Unable to exit the FORGE operator session.",
        );
      }
      return;
    }

    if (hostPowerPolicy?.directSystemControlEnabled !== true) {
      setOperatorAppStatus(
        hostPowerPolicy?.message ??
          "Host shutdown and reboot policy is still loading.",
      );
      return;
    }

    const confirmed = window.confirm(
      action === "reboot"
        ? "Restart the FORGE host now?"
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
  const latestAudit = telemetry.auditRecords[0] ?? null;
  const latestContextSnapshot = telemetry.contextSnapshots[0] ?? null;
  const modelRuntimeText = modelRuntimeSummary(telemetry);
  const autonomyText = autonomySummary(telemetry.autonomyStatus);
  const activeLoopText = activeLoopSummary(telemetry.autonomyStatus);
  const pendingApprovalsText = `${approvalsPending} pending`;
  const notifications = useMemo<ShellNotificationItem[]>(() => {
    const coreNotifications = telemetry.eventRecords
      .filter(isNotificationEvent)
      .slice(0, 20)
      .map((event) => ({
        id: `core:${event.id}`,
        createdAtMs: event.createdAtMs,
        type: event.type,
        detail: notificationDetail(event),
      }));
    const nativeNotifications = desktopNotifications.map((notification) => ({
      id: `desktop:${notification.id}`,
      createdAtMs: notification.createdAtMs,
      type: notification.appName,
      detail: notification.summary,
      body: notification.body,
    }));
    return [...nativeNotifications, ...coreNotifications]
      .sort((a, b) => b.createdAtMs - a.createdAtMs)
      .slice(0, 20);
  }, [desktopNotifications, telemetry.eventRecords]);
  const queueText = level === "none" ? "clear" : `attention ${attentionCount}`;
  const statusSummary = `Core: ${
    core === "online" ? "online" : core === "offline" ? "offline" : "checking"
  } · Runtime: ${runtimeState} · Queue: ${queueText}`;
  const heroError = desktopHeroError(lastErr);

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

  useEffect(() => {
    if (linuxWindows.length === 0) return;
    setRunningOperatorApps((items) =>
      items.filter(
        (item) =>
          !linuxWindows.some((window_) =>
            linuxWindowResolvesOperatorLaunch(item, window_),
          ),
      ),
    );
  }, [linuxWindows]);

  useEffect(() => {
    const pending = runningOperatorApps.filter(
      (item) =>
        !linuxWindows.some((window_) =>
          linuxWindowResolvesOperatorLaunch(item, window_),
        ),
    );
    if (pending.length === 0) return;
    const nextExpirationDelay = Math.max(
      0,
      Math.min(
        ...pending.map(
          (item) =>
            NATIVE_LAUNCH_PENDING_TTL_MS - (Date.now() - item.launchedAtMs),
        ),
      ),
    );
    const timeout = window.setTimeout(() => {
      const expiredLabels: string[] = [];
      setRunningOperatorApps((items) =>
        items.filter((item) => {
          const resolved = linuxWindows.some((window_) =>
            linuxWindowResolvesOperatorLaunch(item, window_),
          );
          if (resolved) return false;
          const expired =
            Date.now() - item.launchedAtMs >= NATIVE_LAUNCH_PENDING_TTL_MS;
          if (expired) expiredLabels.push(item.app.label);
          return !expired;
        }),
      );
      if (expiredLabels.length > 0) {
        setOperatorAppStatus(
          `${expiredLabels[0]} launch did not report a compositor window.`,
        );
      }
    }, nextExpirationDelay);
    return () => window.clearTimeout(timeout);
  }, [linuxWindows, runningOperatorApps]);

  const pendingOperatorApps = useMemo(
    () =>
      runningOperatorApps.filter(
        (item) =>
          Date.now() - item.launchedAtMs < NATIVE_LAUNCH_PENDING_TTL_MS &&
          !linuxWindows.some((window_) =>
            linuxWindowResolvesOperatorLaunch(item, window_),
          ),
      ),
    [linuxWindows, runningOperatorApps],
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
            <span className="forge-os-statusbar__sep" aria-hidden="true">
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
            <span className="forge-os-statusbar__summary">{statusSummary}</span>
            <span className="forge-os-statusbar__mode forge-os-statusbar__chip forge-chip forge-chip--muted">
              Mode: {uiMode}
            </span>
            {statusDetailsOpen ? (
              <button
                type="button"
                className="forge-os-statusbar__button"
                onClick={() => setStatusDetailsOpen((value) => !value)}
                aria-label="Open shell status details"
                aria-expanded="true"
              >
                Status
              </button>
            ) : (
              <button
                type="button"
                className="forge-os-statusbar__button"
                onClick={() => setStatusDetailsOpen((value) => !value)}
                aria-label="Open shell status details"
                aria-expanded="false"
              >
                Status
              </button>
            )}
            {activityOpen ? (
              <button
                type="button"
                className="forge-os-statusbar__button"
                onClick={() => setActivityOpen((value) => !value)}
                aria-label="Open activity log"
                aria-expanded="true"
              >
                Activity
              </button>
            ) : (
              <button
                type="button"
                className="forge-os-statusbar__button"
                onClick={() => setActivityOpen((value) => !value)}
                aria-label="Open activity log"
                aria-expanded="false"
              >
                Activity
              </button>
            )}
            <button
              type="button"
              className="forge-os-statusbar__button forge-os-statusbar__button--notify"
              onClick={() => setNotificationOpen((value) => !value)}
              aria-label="Open notification center"
              aria-expanded={notificationOpen ? "true" : "false"}
            >
              Alerts {notifications.length}
            </button>
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
            <ForgeHero lastErr={heroError} />
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
                  onMoveCommit={() => commitGeometry(win.id)}
                  onResize={(w, h) => resizeLive(win.id, w, h)}
                  onResizeCommit={() => commitGeometry(win.id)}
                />
              ))
            : null}

          {routedDetailOpen && !detachedTauriShell ? (
            <section
              className="forge-os-window forge-os-window--focused forge-os-route-window"
              aria-label={currentTool.label}
            >
              <div className="forge-os-window__chrome">
                <div className="forge-os-window__title">
                  <span className="forge-os-window__sigil">
                    {currentTool.shortLabel}
                  </span>
                  <div className="min-w-0">
                    <div className="forge-os-window__name">
                      {currentTool.label}
                    </div>
                    <div className="forge-os-window__sub">
                      {currentTool.description}
                    </div>
                  </div>
                </div>
                <div className="forge-os-window__buttons">
                  <button
                    type="button"
                    className="forge-os-window__btn"
                    onClick={() => navigate(routedDetailBackRoute)}
                    aria-label={`Back to ${routedDetailBackRoute === "/jobs" ? "Jobs" : "Memory"}`}
                    title="Back"
                  >
                    ‹
                  </button>
                  <button
                    type="button"
                    className="forge-os-window__btn forge-os-window__btn--close"
                    onClick={() => navigate(routedDetailBackRoute)}
                    aria-label="Close"
                    title="Close"
                  >
                    ×
                  </button>
                </div>
              </div>
              <div className="forge-os-window__body">
                <div className="forge-os-window__content">{props.children}</div>
              </div>
            </section>
          ) : null}

          {/* Hidden router-driven children keep route effects mounted for normal
              tool pages. Detail routes render above as focused shell windows. */}
          {!detachedTauriShell &&
          !routedDetailOpen &&
          shellRenderedWindows.length === 0 ? (
            <div className="forge-os-router-sink" aria-hidden="true">
              {props.children}
            </div>
          ) : null}
        </main>
        {isMainWindow && statusDetailsOpen ? (
          <aside
            className="forge-os-context-inspector"
            role="complementary"
            aria-label="Shell context inspector"
          >
            <div className="forge-os-context-inspector__header">
              <div>
                <div className="forge-os-context-inspector__eyebrow">Shell</div>
                <h2>Context Inspector</h2>
              </div>
              <span className="forge-os-context-inspector__pill">
                {telemetry.error ? "degraded" : "live"}
              </span>
            </div>
            <section className="forge-os-context-inspector__section">
              <h3>Shell Status</h3>
              <p>{statusSummary}</p>
              <dl>
                <div>
                  <dt>Workspace</dt>
                  <dd>{workspaceLabel}</dd>
                </div>
                <div>
                  <dt>Modelruntime</dt>
                  <dd>{modelRuntimeText}</dd>
                </div>
                <div>
                  <dt>Autonomy</dt>
                  <dd>{autonomyText}</dd>
                </div>
                <div>
                  <dt>Audit</dt>
                  <dd>{auditSummary(latestAudit)}</dd>
                </div>
              </dl>
            </section>
            <section className="forge-os-context-inspector__section">
              <h3>Compiling Context</h3>
              <p>{contextSnapshotLabel(latestContextSnapshot)}</p>
              <dl>
                <div>
                  <dt>Workspace</dt>
                  <dd>
                    {latestContextSnapshot?.workspaceId?.trim() ||
                      workspaceLabel}
                  </dd>
                </div>
                <div>
                  <dt>Lane</dt>
                  <dd>{latestContextSnapshot?.laneId?.trim() || "default"}</dd>
                </div>
              </dl>
            </section>
            <section className="forge-os-context-inspector__section">
              <h3>Journal / Audit</h3>
              <p>{auditDetail(latestAudit)}</p>
              <dl>
                <div>
                  <dt>Last action</dt>
                  <dd>{auditSummary(latestAudit)}</dd>
                </div>
                <div>
                  <dt>Loaded</dt>
                  <dd>{telemetry.auditRecords.length} events</dd>
                </div>
              </dl>
            </section>
            <section className="forge-os-context-inspector__section">
              <h3>Loops / Approvals</h3>
              <dl>
                <div>
                  <dt>Loops</dt>
                  <dd>{activeLoopText}</dd>
                </div>
                <div>
                  <dt>Approvals</dt>
                  <dd>{pendingApprovalsText}</dd>
                </div>
              </dl>
            </section>
            <section className="forge-os-context-inspector__section">
              <h3>Appearance</h3>
              <div className="forge-os-context-inspector__controls">
                <button
                  type="button"
                  className="forge-os-statusbar__button"
                  onClick={() => toggleThemePreference()}
                  aria-label="Switch shell theme"
                  title={`Theme: ${themePreference}`}
                >
                  {themePreference === "dark" ? "Dark" : "Light"}
                </button>
                <label className="forge-os-statusbar__select-label">
                  <span>Accent</span>
                  <select
                    aria-label="Shell accent"
                    value={accentPreference}
                    onChange={(event) =>
                      setAccentPreference(
                        event.target.value as "cyan" | "amber" | "mint",
                      )
                    }
                  >
                    <option value="cyan">Cyan</option>
                    <option value="amber">Amber</option>
                    <option value="mint">Mint</option>
                  </select>
                </label>
                <button
                  type="button"
                  className="forge-os-statusbar__button"
                  onClick={() => setStatusDetailsOpen(false)}
                >
                  Close
                </button>
              </div>
            </section>
          </aside>
        ) : null}
      </div>

      {isMainWindow && activityOpen ? (
        <section
          className="forge-os-activity-log"
          role="dialog"
          aria-label="Activity log"
        >
          <div className="forge-os-activity-log__header">
            <div>
              <div className="forge-os-context-inspector__eyebrow">Last 20</div>
              <h2>Activity log</h2>
            </div>
            <button
              type="button"
              className="forge-os-statusbar__button"
              onClick={() => setActivityOpen(false)}
            >
              Close
            </button>
          </div>
          <div className="forge-os-activity-log__list">
            {telemetry.auditRecords.length > 0 ? (
              telemetry.auditRecords.slice(0, 20).map((record, index) => (
                <article
                  key={`${record.id ?? "audit"}:${index}`}
                  className="forge-os-activity-log__item"
                >
                  <div className="forge-os-activity-log__meta">
                    <span>{record.action?.trim() || "audit.event"}</span>
                    <span>{record.outcome?.trim() || "unknown"}</span>
                  </div>
                  <p>{auditDetail(record)}</p>
                </article>
              ))
            ) : (
              <p className="forge-os-activity-log__empty">
                No audit events loaded.
              </p>
            )}
          </div>
        </section>
      ) : null}

      {isMainWindow && notificationOpen ? (
        <section
          className="forge-os-notification-center"
          role="dialog"
          aria-label="Notification center"
        >
          <div className="forge-os-activity-log__header">
            <div>
              <div className="forge-os-context-inspector__eyebrow">
                Core events
              </div>
              <h2>Notification center</h2>
            </div>
            <button
              type="button"
              className="forge-os-statusbar__button"
              onClick={() => setNotificationOpen(false)}
            >
              Close
            </button>
          </div>
          <div className="forge-os-activity-log__list">
            {notifications.length > 0 ? (
              notifications.map((item) => (
                <article key={item.id} className="forge-os-activity-log__item">
                  <div className="forge-os-activity-log__meta">
                    <span>{item.type}</span>
                    <span>
                      {new Date(item.createdAtMs).toLocaleTimeString([], {
                        hour: "numeric",
                        minute: "2-digit",
                      })}
                    </span>
                  </div>
                  <p>{item.detail}</p>
                  {item.body ? <p>{item.body}</p> : null}
                </article>
              ))
            ) : (
              <p className="forge-os-activity-log__empty">
                No operator notifications loaded.
              </p>
            )}
          </div>
        </section>
      ) : null}

      <footer className="forge-os-taskbar">
        {startOpen ? (
          <button
            type="button"
            onClick={() => setStartOpen((value) => !value)}
            className="forge-os-taskbar__start forge-os-taskbar__start--active"
            aria-label="Open Start menu"
            aria-expanded="true"
          >
            <img
              className="forge-os-taskbar__anvil"
              src="/brand/forge-start-button.png"
              alt=""
              draggable={false}
            />
            <span className="forge-os-taskbar__label">FORGE</span>
          </button>
        ) : (
          <button
            type="button"
            onClick={() => setStartOpen((value) => !value)}
            className="forge-os-taskbar__start"
            aria-label="Open Start menu"
            aria-expanded="false"
          >
            <img
              className="forge-os-taskbar__anvil"
              src="/brand/forge-start-button.png"
              alt=""
              draggable={false}
            />
            <span className="forge-os-taskbar__label">FORGE</span>
          </button>
        )}

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
                  onClick={() =>
                    void runNativeWindowAction(
                      window_,
                      window_.focused && !window_.minimized
                        ? "minimize"
                        : "focus",
                    )
                  }
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
                      void runNativeWindowAction(window_, "close");
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
          hostPowerPolicy={hostPowerPolicy}
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
