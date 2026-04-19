import { emit } from "@tauri-apps/api/event";
import { WebviewWindow } from "@tauri-apps/api/webviewWindow";
import { availableMonitors, getAllWindows, getCurrentWindow, type Monitor, type Window } from "@tauri-apps/api/window";

export const WORKSPACE_LAYOUT_EVENT = "forge://workspace-layouts-updated";
export const WORKSPACE_NAVIGATE_EVENT = "forge://workspace-navigate";

export type MonitorSnapshot = {
  id: string;
  ordinal: number;
  name: string | null;
  position: { x: number; y: number };
  size: { width: number; height: number };
  workArea: { x: number; y: number; width: number; height: number };
  scaleFactor: number;
};

export type RuntimeWindowSnapshot = {
  label: string;
  title: string;
  isFocused: boolean;
  monitorId: string | null;
  bounds: { x: number; y: number; width: number; height: number } | null;
};

export function isTauriDesktop() {
  return typeof window !== "undefined" && "__TAURI_INTERNALS__" in window;
}

export function monitorIdFromMonitor(monitor: Monitor) {
  return [
    monitor.name ?? "unnamed",
    monitor.position.x,
    monitor.position.y,
    monitor.size.width,
    monitor.size.height,
    monitor.scaleFactor,
  ].join("|");
}

export function snapshotMonitor(monitor: Monitor, ordinal: number): MonitorSnapshot {
  return {
    id: monitorIdFromMonitor(monitor),
    ordinal,
    name: monitor.name,
    position: { x: monitor.position.x, y: monitor.position.y },
    size: { width: monitor.size.width, height: monitor.size.height },
    workArea: {
      x: monitor.workArea.position.x,
      y: monitor.workArea.position.y,
      width: monitor.workArea.size.width,
      height: monitor.workArea.size.height,
    },
    scaleFactor: monitor.scaleFactor,
  };
}

export function monitorSignature(monitors: MonitorSnapshot[]) {
  return monitors
    .map((monitor) => `${monitor.id}:${monitor.workArea.x},${monitor.workArea.y},${monitor.workArea.width},${monitor.workArea.height}`)
    .sort()
    .join(";");
}

function areaIntersection(a: { x: number; y: number; width: number; height: number }, b: { x: number; y: number; width: number; height: number }) {
  const x1 = Math.max(a.x, b.x);
  const y1 = Math.max(a.y, b.y);
  const x2 = Math.min(a.x + a.width, b.x + b.width);
  const y2 = Math.min(a.y + a.height, b.y + b.height);
  const w = Math.max(0, x2 - x1);
  const h = Math.max(0, y2 - y1);
  return w * h;
}

function resolveMonitorFromBounds(monitors: Monitor[], bounds: { x: number; y: number; width: number; height: number }) {
  if (monitors.length === 0) return null;
  let best: Monitor | null = null;
  let bestOverlap = -1;
  for (const monitor of monitors) {
    const monitorRect = {
      x: monitor.position.x,
      y: monitor.position.y,
      width: monitor.size.width,
      height: monitor.size.height,
    };
    const overlap = areaIntersection(bounds, monitorRect);
    if (overlap > bestOverlap) {
      best = monitor;
      bestOverlap = overlap;
    }
  }
  if (best && bestOverlap > 0) return best;
  const cx = bounds.x + bounds.width / 2;
  const cy = bounds.y + bounds.height / 2;
  let nearest = monitors[0] ?? null;
  let nearestDist = Number.POSITIVE_INFINITY;
  for (const monitor of monitors) {
    const mx = monitor.position.x + monitor.size.width / 2;
    const my = monitor.position.y + monitor.size.height / 2;
    const dist = Math.hypot(cx - mx, cy - my);
    if (dist < nearestDist) {
      nearest = monitor;
      nearestDist = dist;
    }
  }
  return nearest;
}

export async function listAvailableMonitors(): Promise<MonitorSnapshot[]> {
  if (!isTauriDesktop()) return [];
  try {
    const monitors = await availableMonitors();
    return monitors.map((monitor, index) => snapshotMonitor(monitor, index));
  } catch (error) {
    if (typeof console !== "undefined") {
      console.error("[FORGE] failed to query monitors", error);
    }
    return [];
  }
}

export async function getCurrentWindowLabel() {
  if (!isTauriDesktop()) return "main";
  return getCurrentWindow().label;
}

export async function getCurrentWindowSnapshot(): Promise<RuntimeWindowSnapshot> {
  if (!isTauriDesktop()) {
    return {
      label: "main",
      title: document.title || "FORGE",
      isFocused: true,
      monitorId: null,
      bounds: null,
    };
  }
  const appWindow = getCurrentWindow();
  const getMonitor = async () => {
    const maybeWindow = appWindow as unknown as { currentMonitor?: () => Promise<Monitor | null> };
    if (typeof maybeWindow.currentMonitor === "function") {
      return maybeWindow.currentMonitor();
    }
    return null;
  };
  const [title, focused, monitor, position, size, monitors] = await Promise.all([
    appWindow.title(),
    appWindow.isFocused(),
    getMonitor(),
    appWindow.outerPosition(),
    appWindow.outerSize(),
    availableMonitors().catch(() => [] as Monitor[]),
  ]);
  const bounds = {
    x: position.x,
    y: position.y,
    width: size.width,
    height: size.height,
  };
  const resolvedMonitor = monitor ?? resolveMonitorFromBounds(monitors, bounds);
  return {
    label: appWindow.label,
    title,
    isFocused: focused,
    monitorId: resolvedMonitor ? monitorIdFromMonitor(resolvedMonitor) : null,
    bounds,
  };
}

function baseWindowUrl() {
  const href = window.location.href;
  const hashIndex = href.indexOf("#");
  return hashIndex >= 0 ? href.slice(0, hashIndex) : href;
}

export function buildWindowUrl(route: string) {
  return `${baseWindowUrl()}#${route}`;
}

export async function createShellWindow(options: {
  label: string;
  route: string;
  title: string;
  bounds: { x: number; y: number; width: number; height: number };
}) {
  if (!isTauriDesktop()) return null;
  if (!Number.isFinite(options.bounds.x) || !Number.isFinite(options.bounds.y) || !Number.isFinite(options.bounds.width) || !Number.isFinite(options.bounds.height)) {
    options.bounds = { x: 60, y: 60, width: 1200, height: 780 };
  }
  try {
    const window = new WebviewWindow(options.label, {
      url: buildWindowUrl(options.route),
      title: options.title,
      x: options.bounds.x,
      y: options.bounds.y,
      width: options.bounds.width,
      height: options.bounds.height,
      visible: true,
      focus: false,
      resizable: true,
      preventOverflow: true,
    });
    try {
      await window.show();
    } catch (error) {
      if (typeof console !== "undefined") {
        console.error(`[FORGE] failed to show new window ${options.label}`, error);
      }
    }
    return window;
  } catch (error) {
    if (typeof console !== "undefined") {
      console.error(`[FORGE] failed to create window ${options.label}`, error);
    }
    return null;
  }
}

export async function getWindowByLabel(label: string) {
  if (!isTauriDesktop()) return null;
  return WebviewWindow.getByLabel(label);
}

export async function listRuntimeWindows(): Promise<Window[]> {
  if (!isTauriDesktop()) return [];
  return getAllWindows();
}

export async function emitWorkspaceSync(origin: string) {
  if (!isTauriDesktop()) return;
  await emit(WORKSPACE_LAYOUT_EVENT, { origin, atMs: Date.now() });
}
