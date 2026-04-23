import { emit } from "@tauri-apps/api/event";
import { WebviewWindow } from "@tauri-apps/api/webviewWindow";
import { availableMonitors, getAllWindows, getCurrentWindow, type Monitor, type Window } from "@tauri-apps/api/window";
import { invoke } from "@tauri-apps/api/core";

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

export type DesktopSystemDiagnostics = {
  hostName: string;
  osName: string;
  osVersion: string;
  kernelVersion?: string | null;
  architecture?: string | null;
  uptimeSeconds: number;
  cpuCount: number;
  totalMemoryBytes: number;
  availableMemoryBytes: number;
  usedMemoryBytes: number;
  totalSwapBytes: number;
  availableSwapBytes: number;
  usedSwapBytes: number;
  process?: {
    pid: number;
    name: string;
    memoryBytes: number;
    virtualMemoryBytes: number;
    cpuUsagePercent: number;
    runTimeSeconds: number;
  } | null;
  disks?: Array<{
    name: string;
    mountPoint: string;
    fileSystem: string;
    totalBytes: number;
    availableBytes: number;
    usedBytes: number;
  }> | null;
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

function parseDiagnosticsPayload(raw: unknown): DesktopSystemDiagnostics | null {
  if (!raw || typeof raw !== "object") return null;
  const value = raw as Partial<DesktopSystemDiagnostics>;
  const hostName = (value as Record<string, unknown>).hostName ?? (raw as Record<string, unknown>).host_name;
  const osName = (value as Record<string, unknown>).osName ?? (raw as Record<string, unknown>).os_name;
  const osVersion = (value as Record<string, unknown>).osVersion ?? (raw as Record<string, unknown>).os_version;
  const kernelVersion = (value as Record<string, unknown>).kernelVersion ?? (raw as Record<string, unknown>).kernel_version;
  const architecture = (value as Record<string, unknown>).architecture ?? (raw as Record<string, unknown>).architecture;
  if (typeof hostName !== "string" || typeof osName !== "string" || typeof osVersion !== "string") {
    return null;
  }
  return {
    hostName: String(hostName),
    osName: String(osName),
    osVersion: String(osVersion),
    kernelVersion: typeof kernelVersion === "string" ? kernelVersion : null,
    architecture: typeof architecture === "string" ? architecture : null,
    uptimeSeconds: Number((value as Record<string, unknown>).uptimeSeconds ?? (raw as Record<string, unknown>).uptime_seconds) || 0,
    cpuCount: Number((value as Record<string, unknown>).cpuCount ?? (raw as Record<string, unknown>).cpu_count) || 0,
    totalMemoryBytes: Number((value as Record<string, unknown>).totalMemoryBytes ?? (raw as Record<string, unknown>).total_memory_bytes) || 0,
    availableMemoryBytes: Number((value as Record<string, unknown>).availableMemoryBytes ?? (raw as Record<string, unknown>).available_memory_bytes) || 0,
    usedMemoryBytes: Number((value as Record<string, unknown>).usedMemoryBytes ?? (raw as Record<string, unknown>).used_memory_bytes) || 0,
    totalSwapBytes: Number((value as Record<string, unknown>).totalSwapBytes ?? (raw as Record<string, unknown>).total_swap_bytes) || 0,
    availableSwapBytes: Number((value as Record<string, unknown>).availableSwapBytes ?? (raw as Record<string, unknown>).available_swap_bytes) || 0,
    usedSwapBytes: Number((value as Record<string, unknown>).usedSwapBytes ?? (raw as Record<string, unknown>).used_swap_bytes) || 0,
    process: value.process && typeof value.process === "object" ? {
      pid: Number((value.process as { pid?: unknown }).pid || (value.process as { pid?: unknown })?.["pid"] || (value.process as { process_id?: unknown })?.["process_id"]) || 0,
      name: String((value.process as { name?: unknown }).name || ""),
      memoryBytes: Number((value.process as { memoryBytes?: unknown }).memoryBytes || (value.process as { memory_bytes?: unknown })?.["memory_bytes"]) || 0,
      virtualMemoryBytes: Number((value.process as { virtualMemoryBytes?: unknown }).virtualMemoryBytes || (value.process as { virtual_memory_bytes?: unknown })?.["virtual_memory_bytes"]) || 0,
      cpuUsagePercent: Number((value.process as { cpuUsagePercent?: unknown }).cpuUsagePercent || (value.process as { cpu_usage_percent?: unknown })?.["cpu_usage_percent"]) || 0,
      runTimeSeconds: Number((value.process as { runTimeSeconds?: unknown }).runTimeSeconds || (value.process as { run_time_seconds?: unknown })?.["run_time_seconds"]) || 0,
    } : null,
    disks: Array.isArray(value.disks)
      ? value.disks.map((disk) => ({
          name: String((disk as { name?: unknown })?.name || ""),
          mountPoint: String((disk as { mountPoint?: unknown })?.mountPoint || ""),
          fileSystem: String((disk as { fileSystem?: unknown })?.fileSystem || ""),
          totalBytes: Number((disk as { totalBytes?: unknown }).totalBytes || (disk as { total_bytes?: unknown })?.["total_bytes"]) || 0,
          availableBytes: Number((disk as { availableBytes?: unknown }).availableBytes || (disk as { available_bytes?: unknown })?.["available_bytes"]) || 0,
          usedBytes: Number((disk as { usedBytes?: unknown }).usedBytes || (disk as { used_bytes?: unknown })?.["used_bytes"]) || 0,
      }))
      : null,
  };
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

export async function getDesktopSystemDiagnostics(): Promise<DesktopSystemDiagnostics | null> {
  if (!isTauriDesktop()) return null;
  try {
    const diagnostics = await invoke("read_system_diagnostics");
    return parseDiagnosticsPayload(diagnostics);
  } catch {
    return null;
  }
}
