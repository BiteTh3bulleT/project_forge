import { create } from "zustand";

import { assignableShellTools } from "../layout/shellConfig";
import {
  DETACHED_TAURI_TOOL_WINDOWS,
  emitWorkspaceSync,
  isForgeManagedWindowLabel,
  isTauriDesktop,
  listAvailableMonitors,
  monitorSignature,
  spanCurrentWindowAcrossMonitors,
  virtualDesktopBounds,
  type MonitorSnapshot,
} from "../lib/desktop";
import { hostLabelForMonitorOrdinal } from "../lib/desktopHostLabels";
import {
  bringCurrentWindowFront,
  bringWindowToFrontByLabel,
  closeWindowByLabel,
  createShellWindow,
  getCurrentWindowLabel,
  getCurrentWindowSnapshot,
  getWindowByLabel,
  listRuntimeWindows,
  navigateWindowByLabel,
  setCurrentWindowBounds,
  setCurrentWindowTitle,
  setWindowBoundsByLabel,
  setWindowTitleByLabel,
} from "../lib/windowManager";

const STORAGE_KEY = "forge.workspace.layouts.v2";
const STORAGE_KEY_LEGACY = "forge.workspace.layouts.v1";
const AUTO_RESTORE_TAURI_LAYOUTS = true;
const EXTENDED_DESKTOP_SINGLE_SHELL = false;

type WindowRole =
  | "chat"
  | "workbench"
  | "canvas"
  | "dossier"
  | "ops"
  | "review"
  | "settings"
  | "mixed";

type MonitorDesignation = {
  mainMonitorId: string | null;
  customLabels: Record<string, string>;
};

type MonitorRoleMap = Record<string, string>;

type LayoutWindowRecord = {
  id: string;
  runtimeLabel: string;
  title: string;
  role: WindowRole;
  assignedRoutes: string[];
  activeRoute: string;
  targetMonitorId: string | null;
  targetMonitorOrdinal: number;
  targetMonitorRole: string | null;
  bounds: { x: number; y: number; width: number; height: number } | null;
  fallbackReason: string | null;
};

type LayoutPreset = {
  id: string;
  name: string;
  windows: LayoutWindowRecord[];
  createdAtMs: number;
  updatedAtMs: number;
  lastActivatedAtMs: number | null;
};

type RuntimeWindowRecord = {
  runtimeLabel: string;
  layoutId: string | null;
  layoutWindowId: string | null;
  role: WindowRole;
  currentRoute: string;
  title: string;
  monitorId: string | null;
  isFocused: boolean;
  bounds: { x: number; y: number; width: number; height: number } | null;
  lastSeenAtMs: number;
};

type LayoutDoc = {
  version: 2;
  monitorDesignations: MonitorDesignation;
  activeLayoutId: string | null;
  selectedLayoutId: string | null;
  layouts: LayoutPreset[];
  runtimeWindows: RuntimeWindowRecord[];
  lastKnownMonitors: MonitorSnapshot[];
  lastMonitorSignature: string;
  fallbackNotice: string | null;
  lastRestoreAtMs: number | null;
};

type WorkspaceLayoutState = {
  ready: boolean;
  supported: boolean;
  currentWindowLabel: string;
  monitorDesignations: MonitorDesignation;
  monitorRoleMap: MonitorRoleMap;
  activeLayoutId: string | null;
  selectedLayoutId: string | null;
  layouts: LayoutPreset[];
  monitors: MonitorSnapshot[];
  runtimeWindows: RuntimeWindowRecord[];
  fallbackNotice: string | null;
  hydrate: (pathname: string) => Promise<void>;
  refreshEnvironment: () => Promise<void>;
  syncCurrentRoute: (pathname: string) => Promise<void>;
  setMainMonitor: (monitorId: string) => void;
  setMonitorRoleLabel: (monitorId: string, label: string) => void;
  createLayout: (name: string) => void;
  selectLayout: (layoutId: string) => void;
  renameLayout: (layoutId: string, name: string) => void;
  duplicateLayout: (layoutId: string) => void;
  deleteLayout: (layoutId: string) => Promise<void>;
  addLayoutWindow: (layoutId: string) => void;
  removeLayoutWindow: (layoutId: string, windowId: string) => void;
  updateLayoutWindow: (
    layoutId: string,
    windowId: string,
    patch: Partial<LayoutWindowRecord>,
  ) => void;
  activateLayout: (layoutId: string) => Promise<void>;
  captureRuntimeIntoLayout: (layoutId: string) => Promise<void>;
  clearFallbackNotice: () => void;
};

function nowMs() {
  return Date.now();
}

function parseMonitorRole(raw: unknown): string | null {
  if (typeof raw !== "string") return null;
  if (raw === "main") return "main";
  const secondary = /^secondary_(\d+)$/.exec(raw);
  if (!secondary) return null;
  const n = Number(secondary[1]);
  return Number.isFinite(n) && n > 0 ? `secondary_${n}` : null;
}

function uid(prefix: string) {
  return `${prefix}-${Math.random().toString(36).slice(2, 10)}`;
}

function clone<T>(value: T): T {
  return JSON.parse(JSON.stringify(value)) as T;
}

function defaultRoutesForRole(role: WindowRole): string[] {
  if (role === "chat") return ["/chat", "/jobs"];
  if (role === "workbench") return ["/workbench", "/jobs"];
  if (role === "canvas") return ["/canvas", "/dossiers"];
  if (role === "dossier") return ["/dossiers", "/memory"];
  if (role === "ops") return ["/jobs", "/approvals", "/events"];
  if (role === "review") return ["/reviews", "/approvals"];
  if (role === "settings") return ["/settings", "/layouts"];
  return ["/chat"];
}

function defaultWindowForLayout(input: {
  runtimeLabel: string;
  title: string;
  role: WindowRole;
  targetMonitorOrdinal: number;
  activeRoute?: string;
}): LayoutWindowRecord {
  const routes = defaultRoutesForRole(input.role);
  return {
    id: uid("window"),
    runtimeLabel: input.runtimeLabel,
    title: input.title,
    role: input.role,
    assignedRoutes: routes,
    activeRoute: input.activeRoute ?? routes[0] ?? "/chat",
    targetMonitorId: null,
    targetMonitorOrdinal: input.targetMonitorOrdinal,
    targetMonitorRole: null,
    bounds: null,
    fallbackReason: null,
  };
}

function ensureLayoutMonitorHosts(
  layout: LayoutPreset,
  monitors: MonitorSnapshot[],
  designations: MonitorDesignation,
) {
  if (monitors.length <= 1) return false;
  const catalog = buildMonitorRoleCatalog(monitors, designations);
  let changed = false;

  for (const monitor of monitors) {
    if (monitor.ordinal === 0) continue;
    const monitorRole = catalog.roleByMonitorId[monitor.id] ?? null;
    const hasHost = layout.windows.some((windowRecord) => {
      if (windowRecord.targetMonitorId === monitor.id) return true;
      if (monitorRole && windowRecord.targetMonitorRole === monitorRole) {
        return true;
      }
      return (
        !windowRecord.targetMonitorId &&
        !windowRecord.targetMonitorRole &&
        windowRecord.targetMonitorOrdinal === monitor.ordinal
      );
    });
    if (hasHost) continue;

    layout.windows.push({
      ...defaultWindowForLayout({
        runtimeLabel: hostLabelForMonitorOrdinal(monitor.ordinal),
        title: `FORGE Monitor ${monitor.ordinal + 1}`,
        role: "mixed",
        targetMonitorOrdinal: monitor.ordinal,
        activeRoute: "/operator-apps",
      }),
      assignedRoutes: ["/operator-apps", "/chat"],
      targetMonitorId: monitor.id,
      targetMonitorRole: monitorRole,
    });
    changed = true;
  }

  if (changed) {
    layout.updatedAtMs = nowMs();
  }
  return changed;
}

function normalizeMonitorDesignations(raw: unknown): MonitorDesignation {
  if (!raw || typeof raw !== "object") {
    return { mainMonitorId: null, customLabels: {} };
  }
  const input = raw as { mainMonitorId?: unknown; customLabels?: unknown };
  const rawMainMonitorId =
    typeof input.mainMonitorId === "string" ? input.mainMonitorId : null;
  const customLabels: Record<string, string> = {};
  if (typeof input.customLabels === "object" && input.customLabels !== null) {
    for (const [monitorId, value] of Object.entries(input.customLabels)) {
      if (typeof monitorId !== "string") continue;
      if (typeof value === "string" && value.trim()) {
        customLabels[monitorId] = value.trim();
      }
    }
  }
  return { mainMonitorId: rawMainMonitorId, customLabels };
}

function canonicalMonitorDesignations(
  monitors: MonitorSnapshot[],
  incoming: MonitorDesignation,
) {
  const monitorIds = new Set(monitors.map((monitor) => monitor.id));
  const keptLabels: Record<string, string> = {};
  for (const [monitorId, label] of Object.entries(incoming.customLabels)) {
    if (monitorIds.has(monitorId) && label.trim()) {
      keptLabels[monitorId] = label.trim();
    }
  }
  const preferredMain = incoming.mainMonitorId;
  const mainMonitorId =
    preferredMain && monitorIds.has(preferredMain)
      ? preferredMain
      : (monitors[0]?.id ?? null);
  return { mainMonitorId, customLabels: keptLabels };
}

function buildMonitorRoleCatalog(
  monitors: MonitorSnapshot[],
  designations: MonitorDesignation,
) {
  const sortedMonitors = [...monitors].sort((a, b) => a.ordinal - b.ordinal);
  const canonical = canonicalMonitorDesignations(sortedMonitors, designations);
  const roleByMonitorId: MonitorRoleMap = {};
  const monitorByRole: Record<string, MonitorSnapshot> = {};

  let secondary = 1;
  for (const monitor of sortedMonitors) {
    const role =
      monitor.id === canonical.mainMonitorId
        ? "main"
        : `secondary_${secondary++}`;
    roleByMonitorId[monitor.id] = role;
    monitorByRole[role] = monitor;
  }
  return { roleByMonitorId, monitorByRole, canonical };
}

function roleToMonitor(
  monitors: MonitorSnapshot[],
  designations: MonitorDesignation,
  role: string | null,
) {
  if (!role) return null;
  const normalizedRole = parseMonitorRole(role);
  if (!normalizedRole) return null;
  const catalog = buildMonitorRoleCatalog(monitors, designations);
  return catalog.monitorByRole[normalizedRole] ?? null;
}

function monitorStateFromDesignations(
  monitors: MonitorSnapshot[],
  designations: MonitorDesignation,
) {
  const catalog = buildMonitorRoleCatalog(monitors, designations);
  return {
    monitorDesignations: catalog.canonical,
    monitorRoleMap: catalog.roleByMonitorId,
  };
}

function ensureDocMonitors(doc: LayoutDoc, monitors: MonitorSnapshot[]) {
  const state = monitorStateFromDesignations(monitors, doc.monitorDesignations);
  doc.monitorDesignations = state.monitorDesignations;
  return state;
}

function deriveMonitorState(monitors: MonitorSnapshot[], doc: LayoutDoc) {
  const state = ensureDocMonitors(doc, monitors);
  return {
    monitorDesignations: state.monitorDesignations,
    monitorRoleMap: state.monitorRoleMap,
  };
}

function resolveWindowPlacement(
  windowRecord: LayoutWindowRecord,
  monitors: MonitorSnapshot[],
  designations: MonitorDesignation,
) {
  const roleMatch = roleToMonitor(
    monitors,
    designations,
    windowRecord.targetMonitorRole,
  );
  const preferred = windowRecord.targetMonitorId
    ? monitors.find((monitor) => monitor.id === windowRecord.targetMonitorId)
    : null;
  const ordinal = monitors[windowRecord.targetMonitorOrdinal] ?? null;
  const chosen = roleMatch ?? preferred ?? ordinal ?? monitors[0] ?? null;
  const fallbackReason = !chosen
    ? "No displays available."
    : windowRecord.targetMonitorRole && !roleMatch
      ? `Monitor role ${windowRecord.targetMonitorRole} unavailable for ${windowRecord.title}; placed on ${chosen.name ?? `display ${chosen.ordinal + 1}`}.`
      : preferred
        ? null
        : windowRecord.targetMonitorId
          ? `Target display unavailable for ${windowRecord.title}; placed on ${chosen.name ?? `display ${chosen.ordinal + 1}`}.`
          : monitors[windowRecord.targetMonitorOrdinal] == null &&
              monitors.length > 0
            ? `Expected display ${windowRecord.targetMonitorOrdinal + 1} unavailable; placed on ${chosen.name ?? `display ${chosen.ordinal + 1}`}.`
            : null;

  return {
    monitor: chosen,
    bounds:
      windowRecord.bounds ??
      (chosen
        ? logicalBoundsForMonitor(chosen, windowRecord.targetMonitorOrdinal)
        : { x: 60, y: 60, width: 1200, height: 780 }),
    fallbackReason,
  };
}

function seedLayouts(): LayoutPreset[] {
  const createdAtMs = nowMs();
  return [
    {
      id: "build",
      name: "Build",
      createdAtMs,
      updatedAtMs: createdAtMs,
      lastActivatedAtMs: null,
      windows: [
        {
          ...defaultWindowForLayout({
            runtimeLabel: "main",
            title: "FORGE Build",
            role: "chat",
            targetMonitorOrdinal: 0,
            activeRoute: "/chat",
          }),
          assignedRoutes: ["/chat", "/jobs", "/workbench"],
        },
        {
          ...defaultWindowForLayout({
            runtimeLabel: "forge-build-workbench",
            title: "FORGE Workbench",
            role: "workbench",
            targetMonitorOrdinal: 1,
            activeRoute: "/workbench",
          }),
          assignedRoutes: ["/workbench", "/jobs"],
        },
      ],
    },
    {
      id: "research",
      name: "Research",
      createdAtMs,
      updatedAtMs: createdAtMs,
      lastActivatedAtMs: null,
      windows: [
        {
          ...defaultWindowForLayout({
            runtimeLabel: "main",
            title: "FORGE Research",
            role: "chat",
            targetMonitorOrdinal: 0,
            activeRoute: "/chat",
          }),
          assignedRoutes: ["/chat", "/memory", "/dossiers"],
        },
        {
          ...defaultWindowForLayout({
            runtimeLabel: "forge-research-canvas",
            title: "FORGE Canvas",
            role: "canvas",
            targetMonitorOrdinal: 1,
            activeRoute: "/canvas",
          }),
          assignedRoutes: ["/canvas", "/dossiers"],
        },
      ],
    },
    {
      id: "ops",
      name: "Ops",
      createdAtMs,
      updatedAtMs: createdAtMs,
      lastActivatedAtMs: null,
      windows: [
        {
          ...defaultWindowForLayout({
            runtimeLabel: "main",
            title: "FORGE Ops",
            role: "ops",
            targetMonitorOrdinal: 0,
            activeRoute: "/jobs",
          }),
          assignedRoutes: ["/jobs", "/approvals", "/reviews", "/events"],
        },
        {
          ...defaultWindowForLayout({
            runtimeLabel: "forge-ops-review",
            title: "FORGE Review",
            role: "review",
            targetMonitorOrdinal: 1,
            activeRoute: "/reviews",
          }),
          assignedRoutes: ["/reviews", "/approvals"],
        },
      ],
    },
    {
      id: "deep-work",
      name: "Deep Work",
      createdAtMs,
      updatedAtMs: createdAtMs,
      lastActivatedAtMs: null,
      windows: [
        {
          ...defaultWindowForLayout({
            runtimeLabel: "main",
            title: "FORGE Deep Work",
            role: "chat",
            targetMonitorOrdinal: 0,
            activeRoute: "/chat",
          }),
          assignedRoutes: ["/chat", "/canvas"],
        },
        {
          ...defaultWindowForLayout({
            runtimeLabel: "forge-deep-workbench",
            title: "FORGE Workbench",
            role: "workbench",
            targetMonitorOrdinal: 1,
            activeRoute: "/workbench",
          }),
          assignedRoutes: ["/workbench", "/dossiers"],
        },
      ],
    },
  ];
}

function emptyDoc(): LayoutDoc {
  return {
    version: 2,
    monitorDesignations: { mainMonitorId: null, customLabels: {} },
    activeLayoutId: "build",
    selectedLayoutId: "build",
    layouts: seedLayouts(),
    runtimeWindows: [],
    lastKnownMonitors: [],
    lastMonitorSignature: "",
    fallbackNotice: null,
    lastRestoreAtMs: null,
  };
}

function normalizeLayoutDoc(
  raw: LayoutDoc | null,
  monitors: MonitorSnapshot[] = [],
) {
  const doc = {
    ...emptyDoc(),
    ...(raw ?? {}),
    version: 2,
  } as LayoutDoc;

  if (!Array.isArray(doc.layouts) || doc.layouts.length === 0) {
    doc.layouts = seedLayouts();
  }
  doc.layouts = doc.layouts.map((layout) => {
    const source = layout as LayoutPreset & { windows?: unknown };
    return {
      id:
        typeof source.id === "string" && source.id ? source.id : uid("layout"),
      name:
        typeof source.name === "string" && source.name
          ? source.name
          : "Recovered Layout",
      createdAtMs:
        typeof source.createdAtMs === "number" &&
        Number.isFinite(source.createdAtMs)
          ? source.createdAtMs
          : nowMs(),
      updatedAtMs:
        typeof source.updatedAtMs === "number" &&
        Number.isFinite(source.updatedAtMs)
          ? source.updatedAtMs
          : nowMs(),
      lastActivatedAtMs:
        typeof source.lastActivatedAtMs === "number" &&
        Number.isFinite(source.lastActivatedAtMs)
          ? source.lastActivatedAtMs
          : null,
      windows:
        Array.isArray(source.windows) && source.windows.length > 0
          ? source.windows.map((windowRecord) => {
              const windowSource = windowRecord as LayoutWindowRecord & {
                fallbackReason?: unknown;
                assignedRoutes?: unknown;
                targetMonitorRole?: unknown;
              };
              return {
                id:
                  typeof windowSource.id === "string" && windowSource.id
                    ? windowSource.id
                    : uid("window"),
                runtimeLabel:
                  typeof windowSource.runtimeLabel === "string" &&
                  windowSource.runtimeLabel
                    ? windowSource.runtimeLabel
                    : uid("window"),
                title:
                  typeof windowSource.title === "string" && windowSource.title
                    ? windowSource.title
                    : "FORGE Window",
                role:
                  windowSource.role === "chat" ||
                  windowSource.role === "workbench" ||
                  windowSource.role === "canvas" ||
                  windowSource.role === "dossier" ||
                  windowSource.role === "ops" ||
                  windowSource.role === "review" ||
                  windowSource.role === "settings" ||
                  windowSource.role === "mixed"
                    ? windowSource.role
                    : "mixed",
                assignedRoutes: sanitizeRoutes(
                  Array.isArray(windowSource.assignedRoutes)
                    ? (windowSource.assignedRoutes.filter(
                        (route) => typeof route === "string",
                      ) as string[])
                    : defaultRoutesForRole(
                        windowSource.role === "chat" ||
                          windowSource.role === "workbench" ||
                          windowSource.role === "canvas" ||
                          windowSource.role === "dossier" ||
                          windowSource.role === "ops" ||
                          windowSource.role === "review" ||
                          windowSource.role === "settings" ||
                          windowSource.role === "mixed"
                          ? windowSource.role
                          : "mixed",
                      ),
                ),
                activeRoute:
                  typeof windowSource.activeRoute === "string" &&
                  windowSource.activeRoute
                    ? windowSource.activeRoute
                    : "/chat",
                targetMonitorId:
                  typeof windowSource.targetMonitorId === "string" &&
                  windowSource.targetMonitorId.length > 0
                    ? windowSource.targetMonitorId
                    : null,
                targetMonitorOrdinal:
                  typeof windowSource.targetMonitorOrdinal === "number" &&
                  Number.isFinite(windowSource.targetMonitorOrdinal)
                    ? windowSource.targetMonitorOrdinal
                    : 0,
                targetMonitorRole: parseMonitorRole(
                  windowSource.targetMonitorRole,
                ),
                bounds:
                  windowSource.bounds &&
                  typeof windowSource.bounds === "object" &&
                  Number.isFinite(windowSource.bounds.x) &&
                  Number.isFinite(windowSource.bounds.y) &&
                  Number.isFinite(windowSource.bounds.width) &&
                  Number.isFinite(windowSource.bounds.height)
                    ? windowSource.bounds
                    : null,
                fallbackReason:
                  typeof windowSource.fallbackReason === "string" &&
                  windowSource.fallbackReason.length > 0
                    ? windowSource.fallbackReason
                    : null,
              };
            })
          : [
              defaultWindowForLayout({
                runtimeLabel: "main",
                title: "FORGE Window",
                role: "mixed",
                targetMonitorOrdinal: 0,
                activeRoute: "/chat",
              }),
            ],
    };
  });

  doc.monitorDesignations = normalizeMonitorDesignations(
    doc.monitorDesignations,
  );
  ensureDocMonitors(doc, monitors);
  doc.runtimeWindows = Array.isArray(doc.runtimeWindows)
    ? doc.runtimeWindows
    : [];
  doc.lastKnownMonitors = Array.isArray(doc.lastKnownMonitors)
    ? doc.lastKnownMonitors
    : [];
  doc.lastMonitorSignature =
    typeof doc.lastMonitorSignature === "string"
      ? doc.lastMonitorSignature
      : "";
  doc.fallbackNotice =
    typeof doc.fallbackNotice === "string" ? doc.fallbackNotice : null;
  doc.lastRestoreAtMs =
    typeof doc.lastRestoreAtMs === "number" &&
    Number.isFinite(doc.lastRestoreAtMs)
      ? doc.lastRestoreAtMs
      : null;
  return ensureActiveLayout(doc);
}

function parseStoredDoc(raw: string | null): LayoutDoc | null {
  if (!raw) return null;
  try {
    const parsed = JSON.parse(raw) as LayoutDoc;
    if (!parsed || typeof parsed !== "object") return null;
    return parsed;
  } catch {
    return null;
  }
}

function loadDoc(monitors: MonitorSnapshot[] = []): LayoutDoc {
  if (typeof window === "undefined") return emptyDoc();
  try {
    const latest = parseStoredDoc(window.localStorage.getItem(STORAGE_KEY));
    const legacy = latest
      ? null
      : parseStoredDoc(window.localStorage.getItem(STORAGE_KEY_LEGACY));
    return normalizeLayoutDoc(latest ?? legacy, monitors);
  } catch {
    return emptyDoc();
  }
}

function persistDoc(doc: LayoutDoc) {
  if (typeof window === "undefined") return;
  window.localStorage.setItem(STORAGE_KEY, JSON.stringify(doc));
}

function findLayout(doc: LayoutDoc, layoutId: string | null) {
  if (!layoutId) return null;
  return doc.layouts.find((layout) => layout.id === layoutId) ?? null;
}

function ensureActiveLayout(doc: LayoutDoc): LayoutDoc {
  const hasActiveLayout = findLayout(doc, doc.activeLayoutId) !== null;
  const fallbackLayoutId = doc.layouts[0]?.id ?? null;
  if (!hasActiveLayout && fallbackLayoutId) {
    doc.activeLayoutId = fallbackLayoutId;
  }
  if (!doc.selectedLayoutId || !findLayout(doc, doc.selectedLayoutId)) {
    doc.selectedLayoutId = doc.activeLayoutId ?? fallbackLayoutId;
  }
  return doc;
}

function isInvalidWindowHandleError(error: unknown) {
  const message = error instanceof Error ? error.message : String(error ?? "");
  return (
    message.includes("Invalid window handle") ||
    message.includes("is not found") ||
    message.includes("not found")
  );
}

async function reclaimWindowLabel(runtimeLabel: string) {
  await closeWindowByLabel(runtimeLabel).catch(() => undefined);
}

async function syncOrRecreateWindow(
  layoutWindow: LayoutWindowRecord,
  bounds: { x: number; y: number; width: number; height: number },
  options: { route: string; setFocus?: boolean },
) {
  const targetWindow = await getWindowByLabel(layoutWindow.runtimeLabel);
  if (!targetWindow) {
    return createShellWindow({
      label: layoutWindow.runtimeLabel,
      route: options.route,
      title: layoutWindow.title,
      bounds,
    });
  }
  try {
    await setWindowTitleByLabel(layoutWindow.runtimeLabel, layoutWindow.title);
    await setWindowBoundsByLabel(layoutWindow.runtimeLabel, bounds);
    await navigateWindow(layoutWindow.runtimeLabel, options.route);
    await bringWindowToFrontByLabel(
      layoutWindow.runtimeLabel,
      options.setFocus === true,
    );
    return targetWindow;
  } catch (error) {
    if (!isInvalidWindowHandleError(error)) {
      throw error;
    }
    await reclaimWindowLabel(layoutWindow.runtimeLabel).catch(() => undefined);
    if (typeof console !== "undefined") {
      console.warn(
        `[FORGE] window ${layoutWindow.runtimeLabel} has invalid handle, recreating`,
        error,
      );
    }
    return createShellWindow({
      label: layoutWindow.runtimeLabel,
      route: options.route,
      title: layoutWindow.title,
      bounds,
    });
  }
}

function sanitizeRoutes(routes: string[]) {
  const allowed = new Set(assignableShellTools.map((tool) => tool.route));
  const sanitized = routes.filter((route) => allowed.has(route));
  return sanitized.length > 0 ? Array.from(new Set(sanitized)) : ["/chat"];
}

function logicalBoundsForMonitor(monitor: MonitorSnapshot, index: number) {
  const workX = Math.round(monitor.workArea.x / monitor.scaleFactor);
  const workY = Math.round(monitor.workArea.y / monitor.scaleFactor);
  const workWidth = Math.round(monitor.workArea.width / monitor.scaleFactor);
  const workHeight = Math.round(monitor.workArea.height / monitor.scaleFactor);
  const width = Math.max(920, Math.round(workWidth * 0.78));
  const height = Math.max(640, Math.round(workHeight * 0.84));
  const x =
    workX + Math.max(20, Math.round((workWidth - width) * 0.5)) + index * 18;
  const y =
    workY + Math.max(20, Math.round((workHeight - height) * 0.08)) + index * 18;
  return {
    x,
    y,
    width: Math.min(width, workWidth),
    height: Math.min(height, workHeight),
  };
}

function mergeRuntimeWindow(doc: LayoutDoc, next: RuntimeWindowRecord) {
  const runtimeWindows = doc.runtimeWindows.filter(
    (item) => item.runtimeLabel !== next.runtimeLabel,
  );
  runtimeWindows.push(next);
  doc.runtimeWindows = runtimeWindows.sort((a, b) =>
    a.runtimeLabel.localeCompare(b.runtimeLabel),
  );
}

async function syncRuntimeWindowRegistry(doc: LayoutDoc) {
  const runtimeWindows = await listRuntimeWindows();
  const liveLabels = new Set(runtimeWindows.map((item) => item.label));
  doc.runtimeWindows = doc.runtimeWindows.filter(
    (item) => liveLabels.has(item.runtimeLabel) || item.runtimeLabel === "main",
  );
}

async function closeSecondaryShellHosts() {
  const runtimeWindows = await listRuntimeWindows();
  for (const runtimeWindow of runtimeWindows) {
    if (runtimeWindow.label === "main") continue;
    if (isForgeManagedWindowLabel(runtimeWindow.label)) {
      await closeWindowByLabel(runtimeWindow.label).catch(() => undefined);
    }
  }
}

async function navigateWindow(runtimeLabel: string, route: string) {
  await navigateWindowByLabel(runtimeLabel, route);
}

async function syncCurrentRuntimeWindow(
  pathname: string,
  monitors: MonitorSnapshot[] = [],
) {
  const currentLabel = await getCurrentWindowLabel();
  const snapshot = await getCurrentWindowSnapshot();
  const doc = loadDoc(monitors);
  const activeLayout = findLayout(doc, doc.activeLayoutId);
  const layoutWindow =
    activeLayout?.windows.find((item) => item.runtimeLabel === currentLabel) ??
    null;
  const next: RuntimeWindowRecord = {
    runtimeLabel: currentLabel,
    layoutId: activeLayout?.id ?? null,
    layoutWindowId: layoutWindow?.id ?? null,
    role: layoutWindow?.role ?? "mixed",
    currentRoute: pathname,
    title: snapshot.title,
    monitorId: snapshot.monitorId,
    isFocused: snapshot.isFocused,
    bounds: snapshot.bounds,
    lastSeenAtMs: nowMs(),
  };
  mergeRuntimeWindow(doc, next);
  if (layoutWindow) {
    layoutWindow.activeRoute = pathname;
    layoutWindow.bounds = snapshot.bounds;
    layoutWindow.fallbackReason = null;
    activeLayout!.updatedAtMs = nowMs();
  }
  await syncRuntimeWindowRegistry(doc);
  persistDoc(doc);
  await emitWorkspaceSync(currentLabel);
  return doc;
}

async function applyLayout(
  layoutId: string,
  markRestore = false,
  monitors: MonitorSnapshot[] = [],
) {
  if (!isTauriDesktop()) return loadDoc(monitors);
  const resolvedMonitors =
    monitors.length > 0 ? monitors : await listAvailableMonitors();
  const doc = loadDoc(resolvedMonitors);
  const layout = findLayout(doc, layoutId);
  if (!layout) return doc;
  const currentLabel = await getCurrentWindowLabel();
  const resolvedMonitorState = ensureDocMonitors(doc, resolvedMonitors);
  const fallbacks: string[] = [];
  doc.activeLayoutId = layoutId;
  doc.selectedLayoutId = layoutId;
  doc.lastKnownMonitors = resolvedMonitors;
  doc.lastMonitorSignature = monitorSignature(resolvedMonitors);
  doc.monitorDesignations = resolvedMonitorState.monitorDesignations;
  doc.lastRestoreAtMs = markRestore ? nowMs() : doc.lastRestoreAtMs;
  ensureLayoutMonitorHosts(layout, resolvedMonitors, doc.monitorDesignations);

  if (EXTENDED_DESKTOP_SINGLE_SHELL && currentLabel === "main") {
    const mainRecord =
      layout.windows.find((item) => item.runtimeLabel === "main") ??
      layout.windows[0] ??
      defaultWindowForLayout({
        runtimeLabel: "main",
        title: "FORGE",
        role: "mixed",
        targetMonitorOrdinal: 0,
        activeRoute: "/chat",
      });
    const resolved = resolveWindowPlacement(
      mainRecord,
      resolvedMonitors,
      doc.monitorDesignations,
    );
    if (resolved.fallbackReason) fallbacks.push(resolved.fallbackReason);

    await setCurrentWindowTitle(mainRecord.title || "FORGE");
    let shellBounds = resolved.bounds;
    if (resolvedMonitors.length > 1) {
      const spanned = await spanCurrentWindowAcrossMonitors(resolvedMonitors);
      shellBounds = virtualDesktopBounds(resolvedMonitors) ?? resolved.bounds;
      if (!spanned) {
        fallbacks.push(
          "Unable to span the desktop shell across all displays; using the main display.",
        );
        await setCurrentWindowBounds(resolved.bounds);
        shellBounds = resolved.bounds;
      }
    } else {
      await setCurrentWindowBounds(resolved.bounds);
    }
    await navigateWindow("main", mainRecord.activeRoute || "/chat");
    await bringCurrentWindowFront(true).catch(() => undefined);
    await closeSecondaryShellHosts();

    const runtimeRecord: RuntimeWindowRecord = {
      runtimeLabel: "main",
      layoutId: layout.id,
      layoutWindowId: mainRecord.id,
      role: mainRecord.role,
      currentRoute: mainRecord.activeRoute || "/chat",
      title: mainRecord.title || "FORGE",
      monitorId: resolved.monitor?.id ?? null,
      isFocused: true,
      bounds: shellBounds,
      lastSeenAtMs: nowMs(),
    };
    mergeRuntimeWindow(doc, runtimeRecord);
    doc.runtimeWindows = doc.runtimeWindows.filter(
      (item) => item.runtimeLabel === "main",
    );
    doc.fallbackNotice = fallbacks.length > 0 ? fallbacks.join(" ") : null;
    layout.lastActivatedAtMs = nowMs();
    layout.updatedAtMs = nowMs();
    persistDoc(doc);
    await emitWorkspaceSync(currentLabel);
    return doc;
  }

  for (const windowRecord of layout.windows) {
    if (!windowRecord.runtimeLabel) continue;
    const resolved = resolveWindowPlacement(
      windowRecord,
      resolvedMonitors,
      doc.monitorDesignations,
    );
    windowRecord.fallbackReason = resolved.fallbackReason;
    if (resolved.fallbackReason) fallbacks.push(resolved.fallbackReason);

    const targetWindow =
      windowRecord.runtimeLabel === currentLabel
        ? null
        : await getWindowByLabel(windowRecord.runtimeLabel);
    if (windowRecord.runtimeLabel === currentLabel) {
      await setCurrentWindowTitle(windowRecord.title);
      await setCurrentWindowBounds(resolved.bounds);
      await navigateWindow(currentLabel, windowRecord.activeRoute);
      await bringCurrentWindowFront(true).catch(() => undefined);
    } else if (targetWindow) {
      try {
        await syncOrRecreateWindow(windowRecord, resolved.bounds, {
          route: windowRecord.activeRoute,
          setFocus: false,
        });
      } catch (error) {
        if (isInvalidWindowHandleError(error)) {
          await reclaimWindowLabel(windowRecord.runtimeLabel).catch(
            () => undefined,
          );
          if (typeof console !== "undefined") {
            console.warn(
              `[FORGE] window ${windowRecord.runtimeLabel} could not be restored`,
              error,
            );
          }
          await createShellWindow({
            label: windowRecord.runtimeLabel,
            route: windowRecord.activeRoute,
            title: windowRecord.title,
            bounds: resolved.bounds,
          });
        }
      }
    } else {
      await createShellWindow({
        label: windowRecord.runtimeLabel,
        route: windowRecord.activeRoute,
        title: windowRecord.title,
        bounds: resolved.bounds,
      });
    }
  }

  const desired = new Set(layout.windows.map((item) => item.runtimeLabel));
  const runtimeWindows = await listRuntimeWindows();
  for (const runtimeWindow of runtimeWindows) {
    if (runtimeWindow.label === "main") continue;
    if (!desired.has(runtimeWindow.label)) {
      if (
        DETACHED_TAURI_TOOL_WINDOWS ||
        isForgeManagedWindowLabel(runtimeWindow.label)
      ) {
        await closeWindowByLabel(runtimeWindow.label).catch(() => undefined);
      }
    }
  }
  await syncRuntimeWindowRegistry(doc);

  doc.fallbackNotice = fallbacks.length > 0 ? fallbacks.join(" ") : null;
  layout.lastActivatedAtMs = nowMs();
  layout.updatedAtMs = nowMs();
  persistDoc(doc);
  await emitWorkspaceSync(currentLabel);
  return doc;
}

export const useWorkspaceLayoutStore = create<WorkspaceLayoutState>(
  (set, get) => ({
    ready: false,
    supported: false,
    currentWindowLabel: "main",
    monitorDesignations: { mainMonitorId: null, customLabels: {} },
    monitorRoleMap: {},
    activeLayoutId: null,
    selectedLayoutId: null,
    layouts: [],
    monitors: [],
    runtimeWindows: [],
    fallbackNotice: null,
    hydrate: async (pathname) => {
      const supported = isTauriDesktop();
      const currentWindowLabel = await getCurrentWindowLabel();
      const monitors = supported ? await listAvailableMonitors() : [];
      let doc = loadDoc(monitors);
      doc = ensureActiveLayout(doc);
      if (supported) {
        doc = await syncCurrentRuntimeWindow(pathname, monitors);
        if (
          AUTO_RESTORE_TAURI_LAYOUTS &&
          currentWindowLabel === "main" &&
          doc.activeLayoutId
        ) {
          doc = await applyLayout(doc.activeLayoutId, true, monitors);
        }
      }
      const monitorState = deriveMonitorState(monitors, doc);
      set({
        ready: true,
        supported,
        currentWindowLabel,
        activeLayoutId: doc.activeLayoutId,
        selectedLayoutId: doc.selectedLayoutId,
        layouts: clone(doc.layouts),
        runtimeWindows: clone(doc.runtimeWindows),
        monitors: clone(monitors),
        monitorDesignations: clone(monitorState.monitorDesignations),
        monitorRoleMap: monitorState.monitorRoleMap,
        fallbackNotice: doc.fallbackNotice,
      });
    },
    refreshEnvironment: async () => {
      const supported = get().supported;
      if (!supported) return;
      const currentWindowLabel = get().currentWindowLabel;
      const monitors = await listAvailableMonitors();
      const signature = monitorSignature(monitors);
      const doc = loadDoc(monitors);
      const changed = doc.lastMonitorSignature !== signature;
      doc.lastKnownMonitors = monitors;
      doc.lastMonitorSignature = signature;
      persistDoc(doc);
      if (
        AUTO_RESTORE_TAURI_LAYOUTS &&
        changed &&
        currentWindowLabel === "main" &&
        doc.activeLayoutId
      ) {
        const refreshed = await applyLayout(
          doc.activeLayoutId,
          false,
          monitors,
        );
        doc.layouts = refreshed.layouts;
        doc.runtimeWindows = refreshed.runtimeWindows;
        doc.fallbackNotice = refreshed.fallbackNotice;
        doc.lastRestoreAtMs = refreshed.lastRestoreAtMs;
        doc.monitorDesignations = refreshed.monitorDesignations;
      } else {
        const route =
          doc.runtimeWindows.find(
            (item) => item.runtimeLabel === currentWindowLabel,
          )?.currentRoute ?? "/chat";
        const refreshed = await syncCurrentRuntimeWindow(route, monitors);
        doc.layouts = refreshed.layouts;
        doc.runtimeWindows = refreshed.runtimeWindows;
        doc.fallbackNotice = refreshed.fallbackNotice;
        doc.lastRestoreAtMs = refreshed.lastRestoreAtMs;
      }
      const monitorState = deriveMonitorState(monitors, doc);
      set({
        layouts: clone(doc.layouts),
        monitorDesignations: monitorState.monitorDesignations,
        monitorRoleMap: monitorState.monitorRoleMap,
        monitors: clone(doc.lastKnownMonitors),
        runtimeWindows: clone(doc.runtimeWindows),
        fallbackNotice: doc.fallbackNotice,
        activeLayoutId: doc.activeLayoutId,
        selectedLayoutId: doc.selectedLayoutId,
      });
    },
    syncCurrentRoute: async (pathname) => {
      const doc = await syncCurrentRuntimeWindow(pathname, get().monitors);
      const monitorState = deriveMonitorState(get().monitors, doc);
      set({
        activeLayoutId: doc.activeLayoutId,
        selectedLayoutId: doc.selectedLayoutId,
        layouts: clone(doc.layouts),
        monitorDesignations: monitorState.monitorDesignations,
        monitorRoleMap: monitorState.monitorRoleMap,
        runtimeWindows: clone(doc.runtimeWindows),
        fallbackNotice: doc.fallbackNotice,
      });
    },
    createLayout: (name) => {
      const doc = loadDoc(get().monitors);
      const label = name.trim() || "New Layout";
      const layoutId = uid("layout");
      const createdAtMs = nowMs();
      doc.layouts.push({
        id: layoutId,
        name: label,
        createdAtMs,
        updatedAtMs: createdAtMs,
        lastActivatedAtMs: null,
        windows: [
          defaultWindowForLayout({
            runtimeLabel: "main",
            title: `FORGE ${label}`,
            role: "mixed",
            targetMonitorOrdinal: 0,
            activeRoute: "/chat",
          }),
        ],
      });
      doc.selectedLayoutId = layoutId;
      persistDoc(doc);
      set({ layouts: clone(doc.layouts), selectedLayoutId: layoutId });
    },
    selectLayout: (layoutId) => {
      const doc = loadDoc(get().monitors);
      doc.selectedLayoutId = layoutId;
      persistDoc(doc);
      set({ selectedLayoutId: layoutId });
    },
    renameLayout: (layoutId, name) => {
      const doc = loadDoc(get().monitors);
      const layout = findLayout(doc, layoutId);
      if (!layout) return;
      layout.name = name.trim() || layout.name;
      layout.updatedAtMs = nowMs();
      persistDoc(doc);
      set({ layouts: clone(doc.layouts) });
    },
    duplicateLayout: (layoutId) => {
      const doc = loadDoc(get().monitors);
      const layout = findLayout(doc, layoutId);
      if (!layout) return;
      const createdAtMs = nowMs();
      const cloneLayout: LayoutPreset = clone(layout);
      cloneLayout.id = uid("layout");
      cloneLayout.name = `${layout.name} Copy`;
      cloneLayout.createdAtMs = createdAtMs;
      cloneLayout.updatedAtMs = createdAtMs;
      cloneLayout.lastActivatedAtMs = null;
      cloneLayout.windows = cloneLayout.windows.map((windowRecord, index) => ({
        ...windowRecord,
        id: uid("window"),
        runtimeLabel: index === 0 ? "main" : uid(`forge-${cloneLayout.id}`),
      }));
      doc.layouts.push(cloneLayout);
      doc.selectedLayoutId = cloneLayout.id;
      persistDoc(doc);
      set({ layouts: clone(doc.layouts), selectedLayoutId: cloneLayout.id });
    },
    deleteLayout: async (layoutId) => {
      const doc = loadDoc(get().monitors);
      if (doc.layouts.length <= 1) return;
      doc.layouts = doc.layouts.filter((layout) => layout.id !== layoutId);
      if (doc.activeLayoutId === layoutId) {
        doc.activeLayoutId = doc.layouts[0]?.id ?? null;
      }
      if (doc.selectedLayoutId === layoutId) {
        doc.selectedLayoutId = doc.layouts[0]?.id ?? null;
      }
      persistDoc(doc);
      if (doc.activeLayoutId) {
        await applyLayout(doc.activeLayoutId, false, get().monitors);
      }
      set({
        layouts: clone(doc.layouts),
        activeLayoutId: doc.activeLayoutId,
        selectedLayoutId: doc.selectedLayoutId,
      });
    },
    addLayoutWindow: (layoutId) => {
      const doc = loadDoc(get().monitors);
      const layout = findLayout(doc, layoutId);
      if (!layout) return;
      const nextIndex = layout.windows.length;
      layout.windows.push(
        defaultWindowForLayout({
          runtimeLabel: uid(`forge-${layout.id}`),
          title: `FORGE Window ${nextIndex + 1}`,
          role: "mixed",
          targetMonitorOrdinal: nextIndex,
          activeRoute: "/chat",
        }),
      );
      layout.updatedAtMs = nowMs();
      persistDoc(doc);
      set({ layouts: clone(doc.layouts) });
    },
    removeLayoutWindow: (layoutId, windowId) => {
      const doc = loadDoc(get().monitors);
      const layout = findLayout(doc, layoutId);
      if (!layout) return;
      if (layout.windows.length <= 1) return;
      const target = layout.windows.find((item) => item.id === windowId);
      if (!target || target.runtimeLabel === "main") return;
      layout.windows = layout.windows.filter((item) => item.id !== windowId);
      layout.updatedAtMs = nowMs();
      persistDoc(doc);
      set({ layouts: clone(doc.layouts) });
    },
    updateLayoutWindow: (layoutId, windowId, patch) => {
      const doc = loadDoc(get().monitors);
      const layout = findLayout(doc, layoutId);
      if (!layout) return;
      const target = layout.windows.find((item) => item.id === windowId);
      if (!target) return;
      if (patch.assignedRoutes) {
        patch.assignedRoutes = sanitizeRoutes(patch.assignedRoutes);
      }
      Object.assign(target, patch);
      target.assignedRoutes = sanitizeRoutes(target.assignedRoutes);
      if (!target.assignedRoutes.includes(target.activeRoute)) {
        target.activeRoute = target.assignedRoutes[0] ?? "/chat";
      }
      if (target.runtimeLabel === "main") {
        target.runtimeLabel = "main";
      }
      layout.updatedAtMs = nowMs();
      const monitorState = deriveMonitorState(get().monitors, doc);
      persistDoc(doc);
      set({
        layouts: clone(doc.layouts),
        monitorDesignations: monitorState.monitorDesignations,
        monitorRoleMap: monitorState.monitorRoleMap,
      });
    },
    setMainMonitor: (monitorId) => {
      const currentMonitors = get().monitors;
      const doc = loadDoc(currentMonitors);
      const hasMonitor = currentMonitors.some(
        (monitor) => monitor.id === monitorId,
      );
      if (!hasMonitor) return;
      doc.monitorDesignations.mainMonitorId = monitorId;
      doc.monitorDesignations.customLabels = normalizeMonitorDesignations(
        doc.monitorDesignations,
      ).customLabels;
      const next = monitorStateFromDesignations(
        currentMonitors,
        doc.monitorDesignations,
      );
      doc.monitorDesignations = next.monitorDesignations;
      persistDoc(doc);
      set({
        monitorDesignations: next.monitorDesignations,
        monitorRoleMap: next.monitorRoleMap,
      });
      const activeLayoutId = doc.activeLayoutId;
      if (activeLayoutId) {
        void applyLayout(activeLayoutId, false, currentMonitors).catch(
          () => undefined,
        );
      }
    },
    setMonitorRoleLabel: (monitorId, label) => {
      const currentMonitors = get().monitors;
      const doc = loadDoc(currentMonitors);
      const cleanLabel = label.trim();
      if (!currentMonitors.some((monitor) => monitor.id === monitorId)) return;
      if (cleanLabel.length === 0) {
        delete doc.monitorDesignations.customLabels[monitorId];
      } else {
        doc.monitorDesignations.customLabels[monitorId] = cleanLabel;
      }
      const next = monitorStateFromDesignations(
        currentMonitors,
        doc.monitorDesignations,
      );
      doc.monitorDesignations = next.monitorDesignations;
      persistDoc(doc);
      set({
        monitorDesignations: next.monitorDesignations,
        monitorRoleMap: next.monitorRoleMap,
      });
    },
    activateLayout: async (layoutId) => {
      const doc = await applyLayout(layoutId, false, get().monitors);
      const monitorState = deriveMonitorState(get().monitors, doc);
      set({
        activeLayoutId: doc.activeLayoutId,
        selectedLayoutId: doc.selectedLayoutId,
        layouts: clone(doc.layouts),
        monitorDesignations: monitorState.monitorDesignations,
        monitorRoleMap: monitorState.monitorRoleMap,
        runtimeWindows: clone(doc.runtimeWindows),
        monitors: clone(doc.lastKnownMonitors),
        fallbackNotice: doc.fallbackNotice,
      });
    },
    captureRuntimeIntoLayout: async (layoutId) => {
      const doc = loadDoc(get().monitors);
      const layout = findLayout(doc, layoutId);
      if (!layout) return;
      for (const runtimeWindow of doc.runtimeWindows) {
        const layoutWindow = layout.windows.find(
          (item) => item.runtimeLabel === runtimeWindow.runtimeLabel,
        );
        if (!layoutWindow) continue;
        layoutWindow.bounds = runtimeWindow.bounds;
        layoutWindow.activeRoute = runtimeWindow.currentRoute;
        layoutWindow.targetMonitorId = runtimeWindow.monitorId;
        const matchedMonitor = doc.lastKnownMonitors.find(
          (monitor) => monitor.id === runtimeWindow.monitorId,
        );
        layoutWindow.targetMonitorOrdinal =
          matchedMonitor?.ordinal ?? layoutWindow.targetMonitorOrdinal;
        layoutWindow.targetMonitorRole = matchedMonitor
          ? (monitorStateFromDesignations(
              doc.lastKnownMonitors,
              doc.monitorDesignations,
            ).monitorRoleMap[matchedMonitor.id] ?? null)
          : layoutWindow.targetMonitorRole;
        layoutWindow.title = runtimeWindow.title;
      }
      layout.updatedAtMs = nowMs();
      const monitorState = deriveMonitorState(get().monitors, doc);
      persistDoc(doc);
      set({
        layouts: clone(doc.layouts),
        monitorDesignations: monitorState.monitorDesignations,
        monitorRoleMap: monitorState.monitorRoleMap,
      });
    },
    clearFallbackNotice: () => {
      const doc = loadDoc(get().monitors);
      doc.fallbackNotice = null;
      const monitorState = deriveMonitorState(get().monitors, doc);
      persistDoc(doc);
      set({
        fallbackNotice: null,
        monitorDesignations: monitorState.monitorDesignations,
        monitorRoleMap: monitorState.monitorRoleMap,
      });
    },
  }),
);
