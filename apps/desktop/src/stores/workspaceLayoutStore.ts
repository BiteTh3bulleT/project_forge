import { create } from "zustand";
import { LogicalPosition, LogicalSize, getCurrentWindow } from "@tauri-apps/api/window";

import { assignableShellTools } from "../layout/shellConfig";
import {
  WORKSPACE_NAVIGATE_EVENT,
  createShellWindow,
  emitWorkspaceSync,
  getCurrentWindowLabel,
  getCurrentWindowSnapshot,
  getWindowByLabel,
  isTauriDesktop,
  listAvailableMonitors,
  listRuntimeWindows,
  monitorSignature,
  type MonitorSnapshot,
} from "../lib/desktop";

const STORAGE_KEY = "forge.workspace.layouts.v1";

type WindowRole = "chat" | "workbench" | "canvas" | "dossier" | "ops" | "review" | "settings" | "mixed";

type LayoutWindowRecord = {
  id: string;
  runtimeLabel: string;
  title: string;
  role: WindowRole;
  assignedRoutes: string[];
  activeRoute: string;
  targetMonitorId: string | null;
  targetMonitorOrdinal: number;
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
  version: 1;
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
  activeLayoutId: string | null;
  selectedLayoutId: string | null;
  layouts: LayoutPreset[];
  monitors: MonitorSnapshot[];
  runtimeWindows: RuntimeWindowRecord[];
  fallbackNotice: string | null;
  hydrate: (pathname: string) => Promise<void>;
  refreshEnvironment: () => Promise<void>;
  syncCurrentRoute: (pathname: string) => Promise<void>;
  createLayout: (name: string) => void;
  selectLayout: (layoutId: string) => void;
  renameLayout: (layoutId: string, name: string) => void;
  duplicateLayout: (layoutId: string) => void;
  deleteLayout: (layoutId: string) => Promise<void>;
  addLayoutWindow: (layoutId: string) => void;
  removeLayoutWindow: (layoutId: string, windowId: string) => void;
  updateLayoutWindow: (layoutId: string, windowId: string, patch: Partial<LayoutWindowRecord>) => void;
  activateLayout: (layoutId: string) => Promise<void>;
  captureRuntimeIntoLayout: (layoutId: string) => Promise<void>;
  clearFallbackNotice: () => void;
};

function nowMs() {
  return Date.now();
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
    bounds: null,
    fallbackReason: null,
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
        { ...defaultWindowForLayout({ runtimeLabel: "main", title: "FORGE Build", role: "chat", targetMonitorOrdinal: 0, activeRoute: "/chat" }), assignedRoutes: ["/chat", "/jobs", "/workbench"] },
        { ...defaultWindowForLayout({ runtimeLabel: "forge-build-workbench", title: "FORGE Workbench", role: "workbench", targetMonitorOrdinal: 1, activeRoute: "/workbench" }), assignedRoutes: ["/workbench", "/jobs"] },
      ],
    },
    {
      id: "research",
      name: "Research",
      createdAtMs,
      updatedAtMs: createdAtMs,
      lastActivatedAtMs: null,
      windows: [
        { ...defaultWindowForLayout({ runtimeLabel: "main", title: "FORGE Research", role: "chat", targetMonitorOrdinal: 0, activeRoute: "/chat" }), assignedRoutes: ["/chat", "/memory", "/dossiers"] },
        { ...defaultWindowForLayout({ runtimeLabel: "forge-research-canvas", title: "FORGE Canvas", role: "canvas", targetMonitorOrdinal: 1, activeRoute: "/canvas" }), assignedRoutes: ["/canvas", "/dossiers"] },
      ],
    },
    {
      id: "ops",
      name: "Ops",
      createdAtMs,
      updatedAtMs: createdAtMs,
      lastActivatedAtMs: null,
      windows: [
        { ...defaultWindowForLayout({ runtimeLabel: "main", title: "FORGE Ops", role: "ops", targetMonitorOrdinal: 0, activeRoute: "/jobs" }), assignedRoutes: ["/jobs", "/approvals", "/reviews", "/events"] },
        { ...defaultWindowForLayout({ runtimeLabel: "forge-ops-review", title: "FORGE Review", role: "review", targetMonitorOrdinal: 1, activeRoute: "/reviews" }), assignedRoutes: ["/reviews", "/approvals"] },
      ],
    },
    {
      id: "deep-work",
      name: "Deep Work",
      createdAtMs,
      updatedAtMs: createdAtMs,
      lastActivatedAtMs: null,
      windows: [
        { ...defaultWindowForLayout({ runtimeLabel: "main", title: "FORGE Deep Work", role: "chat", targetMonitorOrdinal: 0, activeRoute: "/chat" }), assignedRoutes: ["/chat", "/canvas"] },
        { ...defaultWindowForLayout({ runtimeLabel: "forge-deep-workbench", title: "FORGE Workbench", role: "workbench", targetMonitorOrdinal: 1, activeRoute: "/workbench" }), assignedRoutes: ["/workbench", "/dossiers"] },
      ],
    },
  ];
}

function emptyDoc(): LayoutDoc {
  return {
    version: 1,
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

function loadDoc(): LayoutDoc {
  if (typeof window === "undefined") return emptyDoc();
  try {
    const raw = window.localStorage.getItem(STORAGE_KEY);
    if (!raw) return emptyDoc();
    const parsed = JSON.parse(raw) as LayoutDoc;
    if (!Array.isArray(parsed.layouts) || parsed.layouts.length === 0) return emptyDoc();
    return {
      ...emptyDoc(),
      ...parsed,
      layouts: parsed.layouts,
      runtimeWindows: Array.isArray(parsed.runtimeWindows) ? parsed.runtimeWindows : [],
      lastKnownMonitors: Array.isArray(parsed.lastKnownMonitors) ? parsed.lastKnownMonitors : [],
    };
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
  const x = workX + Math.max(20, Math.round((workWidth - width) * 0.5)) + index * 18;
  const y = workY + Math.max(20, Math.round((workHeight - height) * 0.08)) + index * 18;
  return { x, y, width: Math.min(width, workWidth), height: Math.min(height, workHeight) };
}

function resolveWindowPlacement(windowRecord: LayoutWindowRecord, monitors: MonitorSnapshot[]) {
  const preferred = monitors.find((monitor) => monitor.id === windowRecord.targetMonitorId);
  const ordinal = monitors[windowRecord.targetMonitorOrdinal] ?? null;
  const chosen = preferred ?? ordinal ?? monitors[0] ?? null;
  const fallbackReason = !chosen
    ? "No displays available."
    : preferred
      ? null
      : windowRecord.targetMonitorId
        ? `Target display unavailable for ${windowRecord.title}; placed on ${chosen.name ?? `display ${chosen.ordinal + 1}`}.`
        : monitors[windowRecord.targetMonitorOrdinal] == null && monitors.length > 0
          ? `Expected display ${windowRecord.targetMonitorOrdinal + 1} unavailable; placed on ${chosen.name ?? `display ${chosen.ordinal + 1}`}.`
          : null;
  return {
    monitor: chosen,
    bounds: windowRecord.bounds ?? (chosen ? logicalBoundsForMonitor(chosen, windowRecord.targetMonitorOrdinal) : { x: 60, y: 60, width: 1200, height: 780 }),
    fallbackReason,
  };
}

function mergeRuntimeWindow(doc: LayoutDoc, next: RuntimeWindowRecord) {
  const runtimeWindows = doc.runtimeWindows.filter((item) => item.runtimeLabel !== next.runtimeLabel);
  runtimeWindows.push(next);
  doc.runtimeWindows = runtimeWindows.sort((a, b) => a.runtimeLabel.localeCompare(b.runtimeLabel));
}

async function navigateWindow(runtimeLabel: string, route: string) {
  const target = await getWindowByLabel(runtimeLabel);
  if (!target) return;
  await target.emit(WORKSPACE_NAVIGATE_EVENT, { route });
}

async function syncCurrentRuntimeWindow(pathname: string) {
  const currentLabel = await getCurrentWindowLabel();
  const snapshot = await getCurrentWindowSnapshot();
  const doc = loadDoc();
  const activeLayout = findLayout(doc, doc.activeLayoutId);
  const layoutWindow = activeLayout?.windows.find((item) => item.runtimeLabel === currentLabel) ?? null;
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
  persistDoc(doc);
  await emitWorkspaceSync(currentLabel);
  return doc;
}

async function applyLayout(layoutId: string, markRestore = false) {
  if (!isTauriDesktop()) return loadDoc();
  const doc = loadDoc();
  const layout = findLayout(doc, layoutId);
  if (!layout) return doc;
  const currentLabel = await getCurrentWindowLabel();
  const monitors = await listAvailableMonitors();
  const fallbacks: string[] = [];
  doc.activeLayoutId = layoutId;
  doc.selectedLayoutId = layoutId;
  doc.lastKnownMonitors = monitors;
  doc.lastMonitorSignature = monitorSignature(monitors);
  doc.lastRestoreAtMs = markRestore ? nowMs() : doc.lastRestoreAtMs;

  for (const windowRecord of layout.windows) {
    if (!windowRecord.runtimeLabel) continue;
    const resolved = resolveWindowPlacement(windowRecord, monitors);
    windowRecord.fallbackReason = resolved.fallbackReason;
    if (resolved.fallbackReason) fallbacks.push(resolved.fallbackReason);

    const targetWindow = windowRecord.runtimeLabel === currentLabel ? null : await getWindowByLabel(windowRecord.runtimeLabel);
    if (windowRecord.runtimeLabel === currentLabel) {
      const appWindow = getCurrentWindow();
      await appWindow.setTitle(windowRecord.title);
      await appWindow.setPosition(new LogicalPosition(resolved.bounds.x, resolved.bounds.y)).catch(() => undefined);
      await appWindow.setSize(new LogicalSize(resolved.bounds.width, resolved.bounds.height)).catch(() => undefined);
      await navigateWindow(currentLabel, windowRecord.activeRoute);
    } else if (targetWindow) {
      await targetWindow.setTitle(windowRecord.title).catch(() => undefined);
      await targetWindow.setPosition(new LogicalPosition(resolved.bounds.x, resolved.bounds.y)).catch(() => undefined);
      await targetWindow.setSize(new LogicalSize(resolved.bounds.width, resolved.bounds.height)).catch(() => undefined);
      await navigateWindow(windowRecord.runtimeLabel, windowRecord.activeRoute);
      await targetWindow.show().catch(() => undefined);
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
      await runtimeWindow.close().catch(() => undefined);
    }
  }

  doc.fallbackNotice = fallbacks.length > 0 ? fallbacks.join(" ") : null;
  layout.lastActivatedAtMs = nowMs();
  layout.updatedAtMs = nowMs();
  persistDoc(doc);
  await emitWorkspaceSync(currentLabel);
  return doc;
}

export const useWorkspaceLayoutStore = create<WorkspaceLayoutState>((set, get) => ({
  ready: false,
  supported: false,
  currentWindowLabel: "main",
  activeLayoutId: null,
  selectedLayoutId: null,
  layouts: [],
  monitors: [],
  runtimeWindows: [],
  fallbackNotice: null,
  hydrate: async (pathname) => {
    const supported = isTauriDesktop();
    const currentWindowLabel = await getCurrentWindowLabel();
    let doc = loadDoc();
    if (supported) {
      doc = await syncCurrentRuntimeWindow(pathname);
      if (currentWindowLabel === "main" && doc.activeLayoutId) {
        doc = await applyLayout(doc.activeLayoutId, true);
      }
    }
    set({
      ready: true,
      supported,
      currentWindowLabel,
      activeLayoutId: doc.activeLayoutId,
      selectedLayoutId: doc.selectedLayoutId,
      layouts: clone(doc.layouts),
      runtimeWindows: clone(doc.runtimeWindows),
      monitors: clone(doc.lastKnownMonitors),
      fallbackNotice: doc.fallbackNotice,
    });
  },
  refreshEnvironment: async () => {
    const supported = get().supported;
    if (!supported) return;
    const currentWindowLabel = get().currentWindowLabel;
    const monitors = await listAvailableMonitors();
    const signature = monitorSignature(monitors);
    let doc = loadDoc();
    const changed = doc.lastMonitorSignature !== signature;
    doc.lastKnownMonitors = monitors;
    doc.lastMonitorSignature = signature;
    persistDoc(doc);
    if (changed && currentWindowLabel === "main" && doc.activeLayoutId) {
      doc = await applyLayout(doc.activeLayoutId, false);
    } else {
      doc = await syncCurrentRuntimeWindow(get().runtimeWindows.find((item) => item.runtimeLabel === currentWindowLabel)?.currentRoute ?? "/chat");
    }
    set({
      layouts: clone(doc.layouts),
      monitors: clone(doc.lastKnownMonitors),
      runtimeWindows: clone(doc.runtimeWindows),
      fallbackNotice: doc.fallbackNotice,
      activeLayoutId: doc.activeLayoutId,
      selectedLayoutId: doc.selectedLayoutId,
    });
  },
  syncCurrentRoute: async (pathname) => {
    const doc = await syncCurrentRuntimeWindow(pathname);
    set({
      activeLayoutId: doc.activeLayoutId,
      selectedLayoutId: doc.selectedLayoutId,
      layouts: clone(doc.layouts),
      runtimeWindows: clone(doc.runtimeWindows),
      fallbackNotice: doc.fallbackNotice,
    });
  },
  createLayout: (name) => {
    const doc = loadDoc();
    const label = name.trim() || "New Layout";
    const layoutId = uid("layout");
    const createdAtMs = nowMs();
    doc.layouts.push({
      id: layoutId,
      name: label,
      createdAtMs,
      updatedAtMs: createdAtMs,
      lastActivatedAtMs: null,
      windows: [defaultWindowForLayout({ runtimeLabel: "main", title: `FORGE ${label}`, role: "mixed", targetMonitorOrdinal: 0, activeRoute: "/chat" })],
    });
    doc.selectedLayoutId = layoutId;
    persistDoc(doc);
    set({ layouts: clone(doc.layouts), selectedLayoutId: layoutId });
  },
  selectLayout: (layoutId) => {
    const doc = loadDoc();
    doc.selectedLayoutId = layoutId;
    persistDoc(doc);
    set({ selectedLayoutId: layoutId });
  },
  renameLayout: (layoutId, name) => {
    const doc = loadDoc();
    const layout = findLayout(doc, layoutId);
    if (!layout) return;
    layout.name = name.trim() || layout.name;
    layout.updatedAtMs = nowMs();
    persistDoc(doc);
    set({ layouts: clone(doc.layouts) });
  },
  duplicateLayout: (layoutId) => {
    const doc = loadDoc();
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
    const doc = loadDoc();
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
      await applyLayout(doc.activeLayoutId);
    }
    set({
      layouts: clone(doc.layouts),
      activeLayoutId: doc.activeLayoutId,
      selectedLayoutId: doc.selectedLayoutId,
    });
  },
  addLayoutWindow: (layoutId) => {
    const doc = loadDoc();
    const layout = findLayout(doc, layoutId);
    if (!layout) return;
    const nextIndex = layout.windows.length;
    layout.windows.push(defaultWindowForLayout({ runtimeLabel: uid(`forge-${layout.id}`), title: `FORGE Window ${nextIndex + 1}`, role: "mixed", targetMonitorOrdinal: nextIndex, activeRoute: "/chat" }));
    layout.updatedAtMs = nowMs();
    persistDoc(doc);
    set({ layouts: clone(doc.layouts) });
  },
  removeLayoutWindow: (layoutId, windowId) => {
    const doc = loadDoc();
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
    const doc = loadDoc();
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
    persistDoc(doc);
    set({ layouts: clone(doc.layouts) });
  },
  activateLayout: async (layoutId) => {
    const doc = await applyLayout(layoutId);
    set({
      activeLayoutId: doc.activeLayoutId,
      selectedLayoutId: doc.selectedLayoutId,
      layouts: clone(doc.layouts),
      runtimeWindows: clone(doc.runtimeWindows),
      monitors: clone(doc.lastKnownMonitors),
      fallbackNotice: doc.fallbackNotice,
    });
  },
  captureRuntimeIntoLayout: async (layoutId) => {
    const doc = loadDoc();
    const layout = findLayout(doc, layoutId);
    if (!layout) return;
    for (const runtimeWindow of doc.runtimeWindows) {
      const layoutWindow = layout.windows.find((item) => item.runtimeLabel === runtimeWindow.runtimeLabel);
      if (!layoutWindow) continue;
      layoutWindow.bounds = runtimeWindow.bounds;
      layoutWindow.activeRoute = runtimeWindow.currentRoute;
      layoutWindow.targetMonitorId = runtimeWindow.monitorId;
      const matchedMonitor = doc.lastKnownMonitors.find((monitor) => monitor.id === runtimeWindow.monitorId);
      layoutWindow.targetMonitorOrdinal = matchedMonitor?.ordinal ?? layoutWindow.targetMonitorOrdinal;
      layoutWindow.title = runtimeWindow.title;
    }
    layout.updatedAtMs = nowMs();
    persistDoc(doc);
    set({ layouts: clone(doc.layouts) });
  },
  clearFallbackNotice: () => {
    const doc = loadDoc();
    doc.fallbackNotice = null;
    persistDoc(doc);
    set({ fallbackNotice: null });
  },
}));
