import { listen, type UnlistenFn } from "@tauri-apps/api/event";
import { invoke } from "@tauri-apps/api/core";
import {
  getCurrentWindow,
  LogicalPosition,
  LogicalSize,
} from "@tauri-apps/api/window";

import {
  createShellWindow as legacyCreateShellWindow,
  getCurrentWindowLabel as legacyGetCurrentWindowLabel,
  getCurrentWindowSnapshot as legacyGetCurrentWindowSnapshot,
  getWindowByLabel as legacyGetWindowByLabel,
  isTauriDesktop,
  isShellHostWindowLabel,
  listRuntimeWindows as legacyListRuntimeWindows,
  focusTauriWindow as legacyFocusTauriWindow,
  closeTauriWindow as legacyCloseTauriWindow,
  minimizeTauriWindow as legacyMinimizeTauriWindow,
  listForgeWindows as legacyListForgeWindows,
  WORKSPACE_LAYOUT_EVENT,
  WORKSPACE_NAVIGATE_EVENT,
  type ForgeWindowSnapshot,
  type RuntimeWindowSnapshot,
} from "./desktop";

export const FORGE_WINDOW_EVENTS = {
  opened: "forge://window/opened",
  closed: "forge://window/closed",
  focused: "forge://window/focused",
  hidden: "forge://window/hidden",
  shown: "forge://window/shown",
  registryUpdated: "forge://window/registry-updated",
  layoutRestored: "forge://layout/restored",
} as const;

export type ForgeWindowKind =
  | "main_shell"
  | "workspace"
  | "terminal"
  | "memory_panel"
  | "task_panel"
  | "system_panel"
  | "settings"
  | "inspector"
  | "artifact_viewer"
  | "debug_console"
  | "shell_host";

export type ForgeWindowBounds = {
  x: number;
  y: number;
  width: number;
  height: number;
};

export type ForgeWindowOpenRequest = {
  kind: ForgeWindowKind;
  workspaceId?: string | null;
  artifactId?: string | null;
  sessionId?: string | null;
  hostId?: string | null;
  route?: string | null;
  title?: string | null;
  bounds?: ForgeWindowBounds | null;
};

export type ForgeWindowRegistryEntry = {
  label: string;
  kind: ForgeWindowKind;
  route: string;
  title: string;
  visible: boolean;
  focused: boolean;
  minimized: boolean;
  singleton: boolean;
  bounds?: ForgeWindowBounds | null;
  workspaceId?: string | null;
  artifactId?: string | null;
  sessionId?: string | null;
  createdAtMs: number;
  updatedAtMs: number;
};

export type ForgeWindowRegistrySnapshot = {
  windows: ForgeWindowRegistryEntry[];
  timestampMs: number;
};

export type CreateForgeShellWindowOptions = {
  label: string;
  route: string;
  title: string;
  bounds: ForgeWindowBounds;
};

export type RuntimeWindowHandle = {
  label: string;
  close?: () => Promise<void>;
  emit?: (event: string, payload?: unknown) => Promise<void>;
  isMinimized?: () => Promise<boolean>;
  restore?: () => Promise<void>;
  setFocus?: () => Promise<void>;
  setPosition?: (position: LogicalPosition) => Promise<void>;
  setSize?: (size: LogicalSize) => Promise<void>;
  setTitle?: (title: string) => Promise<void>;
  show?: () => Promise<void>;
  unminimize?: () => Promise<void>;
};

function parseWindowKind(value: unknown): ForgeWindowKind | null {
  if (typeof value !== "string") return null;
  if (
    [
      "main_shell",
      "workspace",
      "terminal",
      "memory_panel",
      "task_panel",
      "system_panel",
      "settings",
      "inspector",
      "artifact_viewer",
      "debug_console",
      "shell_host",
    ].includes(value)
  ) {
    return value as ForgeWindowKind;
  }
  return null;
}

function parseBounds(raw: unknown): ForgeWindowBounds | null {
  if (!raw || typeof raw !== "object") return null;
  const value = raw as Record<string, unknown>;
  const x = Number(value.x);
  const y = Number(value.y);
  const width = Number(value.width);
  const height = Number(value.height);
  if (
    !Number.isFinite(x) ||
    !Number.isFinite(y) ||
    !Number.isFinite(width) ||
    !Number.isFinite(height)
  ) {
    return null;
  }
  return { x, y, width, height };
}

function parseRegistryEntry(raw: unknown): ForgeWindowRegistryEntry | null {
  if (!raw || typeof raw !== "object") return null;
  const value = raw as Record<string, unknown>;
  const kind = parseWindowKind(value.kind);
  if (
    typeof value.label !== "string" ||
    typeof value.route !== "string" ||
    typeof value.title !== "string" ||
    typeof value.visible !== "boolean" ||
    typeof value.focused !== "boolean" ||
    typeof value.minimized !== "boolean" ||
    typeof value.singleton !== "boolean" ||
    !kind
  ) {
    return null;
  }
  return {
    label: value.label,
    kind,
    route: value.route,
    title: value.title,
    visible: value.visible,
    focused: value.focused,
    minimized: value.minimized,
    singleton: value.singleton,
    bounds: parseBounds(value.bounds),
    workspaceId:
      typeof value.workspaceId === "string"
        ? value.workspaceId
        : typeof value.workspace_id === "string"
          ? value.workspace_id
          : null,
    artifactId:
      typeof value.artifactId === "string"
        ? value.artifactId
        : typeof value.artifact_id === "string"
          ? value.artifact_id
          : null,
    sessionId:
      typeof value.sessionId === "string"
        ? value.sessionId
        : typeof value.session_id === "string"
          ? value.session_id
          : null,
    createdAtMs: Number(value.createdAtMs ?? value.created_at_ms) || 0,
    updatedAtMs: Number(value.updatedAtMs ?? value.updated_at_ms) || 0,
  };
}

function parseRegistrySnapshot(raw: unknown): ForgeWindowRegistrySnapshot {
  if (!raw || typeof raw !== "object") {
    return { windows: [], timestampMs: Date.now() };
  }
  const value = raw as Record<string, unknown>;
  return {
    windows: Array.isArray(value.windows)
      ? value.windows
          .map((entry) => parseRegistryEntry(entry))
          .filter((entry): entry is ForgeWindowRegistryEntry => Boolean(entry))
      : [],
    timestampMs: Number(value.timestampMs ?? value.timestamp_ms) || Date.now(),
  };
}

function supportedBackendRequest(
  options: CreateForgeShellWindowOptions,
): ForgeWindowOpenRequest | null {
  if (isShellHostWindowLabel(options.label)) {
    if (options.label === "main") {
      return {
        kind: "main_shell",
        route: options.route,
        title: options.title,
        bounds: options.bounds,
      };
    }
    return {
      kind: "shell_host",
      hostId: options.label,
      route: options.route,
      title: options.title,
      bounds: options.bounds,
    };
  }
  if (options.label === "settings") {
    return { kind: "settings", route: options.route, title: options.title, bounds: options.bounds };
  }
  if (options.label === "memory-panel") {
    return { kind: "memory_panel", route: options.route, title: options.title, bounds: options.bounds };
  }
  if (options.label === "task-panel") {
    return { kind: "task_panel", route: options.route, title: options.title, bounds: options.bounds };
  }
  if (options.label === "system-panel") {
    return { kind: "system_panel", route: options.route, title: options.title, bounds: options.bounds };
  }
  if (options.label === "inspector") {
    return { kind: "inspector", route: options.route, title: options.title, bounds: options.bounds };
  }
  if (options.label.startsWith("artifact-")) {
    return {
      kind: "artifact_viewer",
      artifactId: options.label.slice("artifact-".length),
      route: options.route,
      title: options.title,
      bounds: options.bounds,
    };
  }
  if (options.label.startsWith("terminal-")) {
    return {
      kind: "terminal",
      sessionId: options.label.slice("terminal-".length),
      route: options.route,
      title: options.title,
      bounds: options.bounds,
    };
  }
  if (options.label.startsWith("workspace-")) {
    const workspaceId =
      options.label === "workspace-main"
        ? null
        : options.label.slice("workspace-".length);
    return {
      kind: "workspace",
      workspaceId,
      route: options.route,
      title: options.title,
      bounds: options.bounds,
    };
  }
  return null;
}

export async function openForgeWindow(
  request: ForgeWindowOpenRequest,
): Promise<ForgeWindowRegistryEntry | null> {
  if (!isTauriDesktop()) return null;
  const raw = await invoke("forge_window_open", { request });
  const parsed = parseRegistryEntry(raw);
  if (!parsed) throw new Error("invalid FORGE window open response");
  return parsed;
}

export async function createShellWindow(
  options: CreateForgeShellWindowOptions,
) {
  if (isTauriDesktop()) {
    const request = supportedBackendRequest(options);
    if (request) {
      try {
        const opened = await openForgeWindow(request);
        if (!opened || opened.label === options.label) {
          return legacyGetWindowByLabel(options.label);
        }
        await closeForgeWindow(opened.label).catch(() => undefined);
        throw new Error(
          `backend window manager returned ${opened.label} for requested label ${options.label}`,
        );
      } catch (error) {
        if (typeof console !== "undefined") {
          console.warn(
            `[FORGE] backend window manager could not open ${options.label}`,
            error,
          );
        }
        throw error;
      }
    }
    return null;
  }
  return legacyCreateShellWindow(options);
}

export async function getWindowByLabel(
  label: string,
): Promise<RuntimeWindowHandle | null> {
  return legacyGetWindowByLabel(label);
}

export async function listRuntimeWindows(): Promise<RuntimeWindowHandle[]> {
  return legacyListRuntimeWindows();
}

export async function getCurrentWindowLabel() {
  return legacyGetCurrentWindowLabel();
}

export async function getCurrentWindowSnapshot(): Promise<RuntimeWindowSnapshot> {
  return legacyGetCurrentWindowSnapshot();
}

export async function setCurrentWindowTitle(title: string): Promise<void> {
  if (!isTauriDesktop()) {
    if (typeof document !== "undefined") document.title = title;
    return;
  }
  await getCurrentWindow().setTitle(title);
}

export async function setCurrentWindowBounds(
  bounds: ForgeWindowBounds,
): Promise<void> {
  if (!isTauriDesktop()) return;
  const current = getCurrentWindow();
  await current
    .setPosition(new LogicalPosition(bounds.x, bounds.y))
    .catch(() => undefined);
  await current
    .setSize(new LogicalSize(bounds.width, bounds.height))
    .catch(() => undefined);
}

export async function bringCurrentWindowFront(
  setFocus = false,
): Promise<void> {
  if (!isTauriDesktop()) return;
  const current = getCurrentWindow();
  const maybeWindow = current as RuntimeWindowHandle;
  const isMinimized =
    typeof maybeWindow.isMinimized === "function"
      ? await maybeWindow.isMinimized().catch(() => false)
      : false;
  if (isMinimized && typeof maybeWindow.unminimize === "function") {
    await maybeWindow.unminimize().catch(() => undefined);
  } else if (isMinimized && typeof maybeWindow.restore === "function") {
    await maybeWindow.restore().catch(() => undefined);
  }
  await maybeWindow.show?.().catch(() => undefined);
  if (setFocus) {
    await maybeWindow.setFocus?.().catch(() => undefined);
  }
}

export async function setWindowTitleByLabel(
  label: string,
  title: string,
): Promise<boolean> {
  const target = await getWindowByLabel(label);
  if (!target?.setTitle) return false;
  await target.setTitle(title);
  return true;
}

export async function setWindowBoundsByLabel(
  label: string,
  bounds: ForgeWindowBounds,
): Promise<boolean> {
  const target = await getWindowByLabel(label);
  if (!target) return false;
  await target
    .setPosition?.(new LogicalPosition(bounds.x, bounds.y))
    .catch(() => undefined);
  await target
    .setSize?.(new LogicalSize(bounds.width, bounds.height))
    .catch(() => undefined);
  return true;
}

export async function navigateWindowByLabel(
  label: string,
  route: string,
): Promise<boolean> {
  const target = await getWindowByLabel(label);
  if (!target?.emit) return false;
  await target.emit(WORKSPACE_NAVIGATE_EVENT, { route });
  return true;
}

export async function closeWindowByLabel(label: string): Promise<boolean> {
  if (!isTauriDesktop()) return false;
  try {
    return await closeForgeWindow(label);
  } catch {
    const target = await getWindowByLabel(label);
    if (!target?.close) return false;
    await target.close();
    return true;
  }
}

export async function showWindowByLabel(label: string): Promise<boolean> {
  if (!isTauriDesktop()) return false;
  try {
    const result = await invoke("forge_window_show", { label });
    return Boolean(result);
  } catch {
    const target = await getWindowByLabel(label);
    if (!target?.show) return false;
    await target.show();
    return true;
  }
}

export async function bringWindowToFrontByLabel(
  label: string,
  setFocus = false,
): Promise<boolean> {
  const shown = await showWindowByLabel(label);
  if (!setFocus) return shown;
  return focusForgeWindow(label);
}

export async function focusForgeWindow(label: string): Promise<boolean> {
  if (!isTauriDesktop()) return false;
  try {
    const result = await invoke("forge_window_focus", { label });
    return Boolean(result);
  } catch {
    return legacyFocusTauriWindow(label);
  }
}

export async function closeForgeWindow(label: string): Promise<boolean> {
  if (!isTauriDesktop()) return false;
  try {
    const result = await invoke("forge_window_close", { label });
    return Boolean(result);
  } catch {
    return legacyCloseTauriWindow(label);
  }
}

export async function minimizeForgeWindow(label: string): Promise<boolean> {
  if (!isTauriDesktop()) return false;
  try {
    const result = await invoke("forge_window_minimize", { label });
    return Boolean(result);
  } catch {
    return legacyMinimizeTauriWindow(label);
  }
}

export const focusTauriWindow = focusForgeWindow;
export const closeTauriWindow = closeForgeWindow;
export const minimizeTauriWindow = minimizeForgeWindow;

export async function listForgeWindows(): Promise<ForgeWindowSnapshot[]> {
  return legacyListForgeWindows();
}

export async function listForgeWindowRegistry(): Promise<
  ForgeWindowRegistryEntry[]
> {
  if (!isTauriDesktop()) return [];
  try {
    const raw = await invoke("forge_window_list");
    if (!Array.isArray(raw)) return [];
    return raw
      .map((entry) => parseRegistryEntry(entry))
      .filter((entry): entry is ForgeWindowRegistryEntry => Boolean(entry));
  } catch {
    return [];
  }
}

export async function snapshotForgeWindows(): Promise<ForgeWindowRegistrySnapshot> {
  if (!isTauriDesktop()) return { windows: [], timestampMs: Date.now() };
  try {
    return parseRegistrySnapshot(await invoke("forge_window_snapshot"));
  } catch {
    return { windows: [], timestampMs: Date.now() };
  }
}

export async function restoreForgeWindowLayout(
  layoutId?: string,
): Promise<ForgeWindowRegistrySnapshot> {
  if (!isTauriDesktop()) return { windows: [], timestampMs: Date.now() };
  const raw = await invoke("forge_window_restore_layout", { layoutId });
  return parseRegistrySnapshot(raw);
}

export async function syncForgeWindowState(): Promise<ForgeWindowRegistrySnapshot> {
  if (!isTauriDesktop()) return { windows: [], timestampMs: Date.now() };
  const raw = await invoke("forge_window_sync_state");
  return parseRegistrySnapshot(raw);
}

export async function subscribeToForgeWindowEvents(
  onEvent: (eventName: string, payload: unknown) => void,
): Promise<UnlistenFn[]> {
  if (!isTauriDesktop()) return [];
  const entries = Object.values(FORGE_WINDOW_EVENTS);
  return Promise.all(
    entries.map((eventName) =>
      listen(eventName, (event) => onEvent(eventName, event.payload)),
    ),
  );
}

export async function subscribeToWorkspaceNavigation(
  onNavigate: (route: string) => void,
): Promise<UnlistenFn | null> {
  if (!isTauriDesktop()) return null;
  return getCurrentWindow().listen<{ route?: string }>(
    WORKSPACE_NAVIGATE_EVENT,
    (event) => {
      const route = event.payload?.route;
      if (route) onNavigate(route);
    },
  );
}

export async function subscribeToCurrentWindowLifecycle(
  onChange: () => void,
): Promise<UnlistenFn[]> {
  if (!isTauriDesktop()) return [];
  const current = getCurrentWindow();
  return Promise.all([
    current.onMoved(() => onChange()),
    current.onResized(() => onChange()),
    current.onFocusChanged(() => onChange()),
  ]);
}

export async function subscribeToWorkspaceLayoutSync(
  onSync: (payload: { origin?: string } | null) => void,
): Promise<UnlistenFn | null> {
  if (!isTauriDesktop()) return null;
  return listen<{ origin?: string }>(WORKSPACE_LAYOUT_EVENT, (event) => {
    onSync(event.payload ?? null);
  });
}
