import { create } from "zustand";

import {
  DETACHED_TAURI_TOOL_WINDOWS,
  closeTauriWindow,
  createShellWindow,
  focusTauriWindow,
  isTauriDesktop,
  minimizeTauriWindow,
  type ForgeWindowSnapshot,
} from "../lib/desktop";
import { allShellTools, type ShellToolId } from "../layout/shellConfig";

const PINNED_KEY = "forge.os.pinned.v1";
const WINDOWS_KEY = "forge.os.windows.v1";
const FOCUS_KEY = "forge.os.focus.v1";

const DEFAULT_PINNED: ShellToolId[] = [
  "chat",
  "jobs",
  "memory",
  "models",
  "approvals",
  "settings",
];

const APP_LABEL_PREFIX = "forge-app-";

export function tauriLabelForTool(toolId: ShellToolId): string {
  return `${APP_LABEL_PREFIX}${toolId}`;
}

export function toolIdFromTauriLabel(label: string): ShellToolId | null {
  if (!label.startsWith(APP_LABEL_PREFIX)) return null;
  const id = label.slice(APP_LABEL_PREFIX.length);
  // Validate against the known tool-id set.
  if (allShellTools.some((t) => t.id === id)) return id as ShellToolId;
  return null;
}

// Tool surfaces are in-shell MDI windows by default, including in Tauri. The
// `tauri` flag is kept only for the disabled detached-window compatibility path.
export type DesktopWindow = {
  id: string;
  toolId: ShellToolId;
  hostLabel: string;
  monitorId?: string | null;
  createdAtMs?: number;
  updatedAtMs?: number;
  // in-shell geometry; ignored only by the disabled detached Tauri path
  x: number;
  y: number;
  width: number;
  height: number;
  z: number;
  minimized: boolean;
  maximized: boolean;
  // when true, this entry mirrors a real Tauri OS window labeled
  // tauriLabelForTool(toolId)
  tauri: boolean;
};

type DesktopWindowState = {
  pinned: ShellToolId[];
  windows: DesktopWindow[];
  focusedId: string | null;
  // Pinned management
  pin: (toolId: ShellToolId) => void;
  unpin: (toolId: ShellToolId) => void;
  // Window lifecycle (always go through these — they pick the right backend)
  openWindow: (toolId: ShellToolId, opts?: OpenOpts) => Promise<string | null>;
  closeWindow: (id: string) => Promise<void>;
  closeByTool: (toolId: ShellToolId) => Promise<void>;
  minimize: (id: string) => Promise<void>;
  restore: (id: string) => Promise<void>;
  toggleMaximize: (id: string) => void;
  focus: (id: string) => Promise<void>;
  // Geometry (browser fallback only)
  move: (id: string, x: number, y: number) => void;
  moveToHost: (
    id: string,
    hostLabel: string,
    x: number,
    y: number,
    monitorId?: string | null,
  ) => void;
  resize: (id: string, width: number, height: number) => void;
  reconcileHostAvailability: (hosts: DesktopWindowHostTarget[]) => void;
  // Internal: rehydrate from storage
  hydrate: () => void;
  // Reconcile windows[] against Tauri's real window list
  reconcileFromTauri: (snapshots: ForgeWindowSnapshot[]) => void;
};

export type DesktopWindowHostTarget = {
  hostLabel: string;
  monitorId?: string | null;
  primary?: boolean;
};

type OpenOpts = {
  hostLabel?: string;
  monitorId?: string | null;
  x?: number;
  y?: number;
  width?: number;
  height?: number;
};

function safeParseArray<T>(
  raw: string | null,
  validate: (v: unknown) => v is T,
): T[] {
  if (!raw) return [];
  try {
    const value = JSON.parse(raw);
    if (!Array.isArray(value)) return [];
    return value.filter(validate);
  } catch {
    return [];
  }
}

function isShellToolId(value: unknown): value is ShellToolId {
  return (
    typeof value === "string" && allShellTools.some((t) => t.id === value)
  );
}

function isDesktopWindow(value: unknown): value is DesktopWindow {
  if (!value || typeof value !== "object") return false;
  const v = value as Record<string, unknown>;
  return (
    typeof v.id === "string" &&
    typeof v.toolId === "string" &&
    typeof v.x === "number" &&
    typeof v.y === "number" &&
    typeof v.width === "number" &&
    typeof v.height === "number" &&
    typeof v.z === "number" &&
    typeof v.minimized === "boolean" &&
    typeof v.maximized === "boolean"
  );
}

function loadPinned(): ShellToolId[] {
  if (typeof window === "undefined") return DEFAULT_PINNED;
  const raw = window.localStorage.getItem(PINNED_KEY);
  if (raw === null) return DEFAULT_PINNED;
  const parsed = safeParseArray<ShellToolId>(raw, isShellToolId);
  return parsed;
}

function loadWindows(): DesktopWindow[] {
  if (typeof window === "undefined") return [];
  const raw = window.localStorage.getItem(WINDOWS_KEY);
  const list = safeParseArray<DesktopWindow>(raw, isDesktopWindow);
  // Drop any persisted Tauri-flagged windows on load — Tauri owns those and
  // the reconciler will repopulate.
  return list.filter((w) => !w.tauri).map(normalizeBrowserWindow);
}

function loadFocus(): string | null {
  if (typeof window === "undefined") return null;
  const raw = window.localStorage.getItem(FOCUS_KEY);
  return raw && raw.length > 0 ? raw : null;
}

function validFocusForWindows(
  windows: DesktopWindow[],
  focusedId: string | null,
): string | null {
  if (focusedId && windows.some((window_) => window_.id === focusedId)) {
    return focusedId;
  }
  return topOf(windows.filter((window_) => !window_.minimized))?.id ?? null;
}

function loadValidFocus(windows: DesktopWindow[]): string | null {
  return validFocusForWindows(windows, loadFocus());
}

function persist(state: {
  pinned: ShellToolId[];
  windows: DesktopWindow[];
  focusedId: string | null;
}) {
  if (typeof window === "undefined") return;
  try {
    window.localStorage.setItem(PINNED_KEY, JSON.stringify(state.pinned));
    // Don't persist Tauri-mirror windows; only browser-fallback ones.
    const persistable = state.windows.filter((w) => !w.tauri);
    window.localStorage.setItem(WINDOWS_KEY, JSON.stringify(persistable));
    if (state.focusedId) {
      window.localStorage.setItem(FOCUS_KEY, state.focusedId);
    } else {
      window.localStorage.removeItem(FOCUS_KEY);
    }
  } catch {
    // Storage failures are cosmetic.
  }
}

function nextZ(windows: DesktopWindow[]): number {
  let max = 0;
  for (const w of windows) {
    if (w.z > max) max = w.z;
  }
  return max + 1;
}

function defaultGeometry(index: number): {
  x: number;
  y: number;
  width: number;
  height: number;
} {
  const base = { x: 120, y: 92, width: 960, height: 640 };
  const offset = (index % 5) * 28;
  return {
    x: base.x + offset,
    y: base.y + offset,
    width: base.width,
    height: base.height,
  };
}

function browserViewportBounds(): {
  maxWidth: number;
  maxHeight: number;
} {
  if (typeof window === "undefined") {
    return { maxWidth: 1280, maxHeight: 760 };
  }
  // The browser fallback paints inside a desktop shell with statusbar and
  // taskbar chrome. Keep restored windows inside the usable center area.
  return {
    maxWidth: Math.max(420, window.innerWidth - 24),
    maxHeight: Math.max(320, window.innerHeight - 132),
  };
}

function normalizeHostLabel(hostLabel: string | null | undefined): string {
  const clean = hostLabel?.trim();
  return clean && clean.length > 0 ? clean : "main";
}

function normalizeMonitorId(
  monitorId: string | null | undefined,
): string | null {
  const clean = monitorId?.trim();
  return clean && clean.length > 0 ? clean : null;
}

function validTimestamp(value: unknown): value is number {
  return typeof value === "number" && Number.isFinite(value) && value >= 0;
}

function normalizeBrowserWindow(window_: DesktopWindow): DesktopWindow {
  const now = Date.now();
  const createdAtMs = validTimestamp(window_.createdAtMs)
    ? window_.createdAtMs
    : now;
  const updatedAtMs = validTimestamp(window_.updatedAtMs)
    ? window_.updatedAtMs
    : createdAtMs;
  const bounds = browserViewportBounds();
  const width = Math.min(Math.max(window_.width || 960, 420), bounds.maxWidth);
  const height = Math.min(
    Math.max(window_.height || 640, 320),
    bounds.maxHeight,
  );
  const maxX = Math.max(0, bounds.maxWidth - width);
  const maxY = Math.max(0, bounds.maxHeight - height);
  return {
    ...window_,
    hostLabel: normalizeHostLabel(window_.hostLabel),
    monitorId: normalizeMonitorId(window_.monitorId),
    createdAtMs,
    updatedAtMs,
    x: Math.min(
      Math.max(Number.isFinite(window_.x) ? window_.x : 0, 0),
      maxX,
    ),
    y: Math.min(
      Math.max(Number.isFinite(window_.y) ? window_.y : 0, 0),
      maxY,
    ),
    width,
    height,
  };
}

let idSeq = 1;
function makeId(): string {
  idSeq += 1;
  return `w-${Date.now().toString(36)}-${idSeq}`;
}

function withUpdatedAt(
  window_: DesktopWindow,
  updatedAtMs = Date.now(),
): DesktopWindow {
  return {
    ...window_,
    monitorId: normalizeMonitorId(window_.monitorId),
    createdAtMs: validTimestamp(window_.createdAtMs)
      ? window_.createdAtMs
      : updatedAtMs,
    updatedAtMs,
  };
}

function topOf(windows: DesktopWindow[]): DesktopWindow | null {
  if (windows.length === 0) return null;
  let top = windows[0]!;
  for (const w of windows) {
    if (w.z > top.z) top = w;
  }
  return top;
}

function primaryHostTarget(
  hosts: DesktopWindowHostTarget[],
): DesktopWindowHostTarget {
  return (
    hosts.find((host) => host.primary) ??
    hosts.find((host) => normalizeHostLabel(host.hostLabel) === "main") ??
    hosts[0] ?? { hostLabel: "main", monitorId: null, primary: true }
  );
}

function toolTitle(toolId: ShellToolId): string {
  const tool = allShellTools.find((t) => t.id === toolId);
  return tool ? `FORGE — ${tool.label}` : "FORGE";
}

function toolRoute(toolId: ShellToolId): string {
  const tool = allShellTools.find((t) => t.id === toolId);
  return tool?.route ?? "/";
}

export const useDesktopWindowStore = create<DesktopWindowState>((set, get) => ({
  pinned: typeof window === "undefined" ? DEFAULT_PINNED : loadPinned(),
  windows: typeof window === "undefined" ? [] : loadWindows(),
  focusedId:
    typeof window === "undefined" ? null : loadValidFocus(loadWindows()),

  hydrate: () => {
    if (typeof window === "undefined") return;
    const windows = loadWindows();
    set({
      pinned: loadPinned(),
      windows,
      focusedId: loadValidFocus(windows),
    });
  },

  pin: (toolId) =>
    set((s) => {
      if (s.pinned.includes(toolId)) return s;
      const next = { ...s, pinned: [...s.pinned, toolId] };
      persist(next);
      return next;
    }),

  unpin: (toolId) =>
    set((s) => {
      if (!s.pinned.includes(toolId)) return s;
      const next = { ...s, pinned: s.pinned.filter((id) => id !== toolId) };
      persist(next);
      return next;
    }),

  openWindow: async (toolId, opts) => {
    if (isTauriDesktop() && DETACHED_TAURI_TOOL_WINDOWS) {
      // Real OS window path. Spawn or focus the Tauri webview.
      const label = tauriLabelForTool(toolId);
      const existing = get().windows.find((w) => w.toolId === toolId);
      if (existing) {
        await focusTauriWindow(label);
        const now = Date.now();
        set((s) => {
          const windows = s.windows.map((w) =>
            w.id === existing.id ? withUpdatedAt(w, now) : w,
          );
          const next = { ...s, windows, focusedId: existing.id };
          persist(next);
          return next;
        });
        return existing.id;
      }
      if (await focusTauriWindow(label)) {
        const id = makeId();
        const z = nextZ(get().windows);
        const now = Date.now();
        set((s) => {
          const next = {
            ...s,
            windows: [
              ...s.windows,
              {
                id,
                toolId,
                hostLabel: "main",
                monitorId: normalizeMonitorId(opts?.monitorId),
                createdAtMs: now,
                updatedAtMs: now,
                x: 0,
                y: 0,
                width: 0,
                height: 0,
                z,
                minimized: false,
                maximized: false,
                tauri: true,
              },
            ],
            focusedId: id,
          };
          persist(next);
          return next;
        });
        return id;
      }
      const geo = defaultGeometry(get().windows.length);
      const created = await createShellWindow({
        label,
        route: toolRoute(toolId),
        title: toolTitle(toolId),
        bounds: {
          x: opts?.x ?? geo.x,
          y: opts?.y ?? geo.y,
          width: opts?.width ?? geo.width,
          height: opts?.height ?? geo.height,
        },
      });
      if (!created) return null;
      // Reconciler will populate windows[] from listForgeWindows; meanwhile
      // optimistically add an entry so the taskbar updates instantly.
      const id = makeId();
      const z = nextZ(get().windows);
      const now = Date.now();
      const optimistic: DesktopWindow = {
        id,
        toolId,
        hostLabel: "main",
        monitorId: normalizeMonitorId(opts?.monitorId),
        createdAtMs: now,
        updatedAtMs: now,
        x: opts?.x ?? geo.x,
        y: opts?.y ?? geo.y,
        width: opts?.width ?? geo.width,
        height: opts?.height ?? geo.height,
        z,
        minimized: false,
        maximized: false,
        tauri: true,
      };
      set((s) => ({
        ...s,
        windows: [...s.windows, optimistic],
        focusedId: id,
      }));
      return id;
    }

    // Browser fallback: in-shell window manager (single-instance per tool).
    if (isTauriDesktop()) {
      await closeTauriWindow(tauriLabelForTool(toolId)).catch(() => undefined);
    }
    const state = get();
    const hostLabel = normalizeHostLabel(opts?.hostLabel);
    const monitorId = normalizeMonitorId(opts?.monitorId);
    const existing = state.windows.find(
      (w) => w.toolId === toolId && (w.hostLabel || "main") === hostLabel,
    );
    if (existing) {
      const z = nextZ(state.windows);
      const now = Date.now();
      const next: DesktopWindowState = {
        ...state,
        windows: state.windows.map((w) =>
          w.id === existing.id
            ? withUpdatedAt(
                {
                  ...w,
                  monitorId:
                    opts?.monitorId === undefined ? w.monitorId : monitorId,
                  minimized: false,
                  z,
                },
                now,
              )
            : w,
        ),
        focusedId: existing.id,
      };
      set(next);
      persist(next);
      return existing.id;
    }
    const id = makeId();
    const geo = defaultGeometry(state.windows.length);
    const z = nextZ(state.windows);
    const now = Date.now();
    const newWindow: DesktopWindow = {
      id,
      toolId,
      hostLabel,
      monitorId,
      createdAtMs: now,
      updatedAtMs: now,
      x: opts?.x ?? geo.x,
      y: opts?.y ?? geo.y,
      width: opts?.width ?? geo.width,
      height: opts?.height ?? geo.height,
      z,
      minimized: false,
      maximized: false,
      tauri: false,
    };
    const normalizedWindow = normalizeBrowserWindow(newWindow);
    const next: DesktopWindowState = {
      ...state,
      windows: [...state.windows, normalizedWindow],
      focusedId: id,
    };
    set(next);
    persist(next);
    return id;
  },

  closeWindow: async (id) => {
    const target = get().windows.find((w) => w.id === id);
    if (!target) return;
    if (target.tauri) {
      await closeTauriWindow(tauriLabelForTool(target.toolId));
      // Reconciler will remove from list. Optimistically remove now.
    }
    set((s) => {
      const remaining = s.windows.filter((w) => w.id !== id);
      const focusedId =
        s.focusedId === id
          ? topOf(remaining.filter((w) => !w.minimized))?.id ?? null
          : s.focusedId;
      const next = { ...s, windows: remaining, focusedId };
      persist(next);
      return next;
    });
  },

  closeByTool: async (toolId) => {
    const target = get().windows.find((w) => w.toolId === toolId);
    if (!target) return;
    await get().closeWindow(target.id);
  },

  minimize: async (id) => {
    const target = get().windows.find((w) => w.id === id);
    if (!target || target.minimized) return;
    if (target.tauri) {
      await minimizeTauriWindow(tauriLabelForTool(target.toolId));
    }
    set((s) => {
      const now = Date.now();
      const windows = s.windows.map((w) =>
        w.id === id ? withUpdatedAt({ ...w, minimized: true }, now) : w,
      );
      const focusedId =
        s.focusedId === id
          ? topOf(windows.filter((w) => !w.minimized))?.id ?? null
          : s.focusedId;
      const next = { ...s, windows, focusedId };
      persist(next);
      return next;
    });
  },

  restore: async (id) => {
    const target = get().windows.find((w) => w.id === id);
    if (!target) return;
    if (target.tauri) {
      await focusTauriWindow(tauriLabelForTool(target.toolId));
    }
    set((s) => {
      const z = nextZ(s.windows);
      const now = Date.now();
      const windows = s.windows.map((w) =>
        w.id === id ? withUpdatedAt({ ...w, minimized: false, z }, now) : w,
      );
      const next = { ...s, windows, focusedId: id };
      persist(next);
      return next;
    });
  },

  toggleMaximize: (id) =>
    set((s) => {
      const now = Date.now();
      const windows = s.windows.map((w) =>
        w.id === id
          ? withUpdatedAt({ ...w, maximized: !w.maximized }, now)
          : w,
      );
      const next = { ...s, windows };
      persist(next);
      return next;
    }),

  focus: async (id) => {
    const target = get().windows.find((w) => w.id === id);
    if (!target) return;
    if (target.tauri) {
      await focusTauriWindow(tauriLabelForTool(target.toolId));
    }
    set((s) => {
      const z = nextZ(s.windows);
      const now = Date.now();
      const windows = s.windows.map((w) =>
        w.id === id ? withUpdatedAt({ ...w, z, minimized: false }, now) : w,
      );
      const next = { ...s, windows, focusedId: id };
      persist(next);
      return next;
    });
  },

  move: (id, x, y) =>
    set((s) => {
      const now = Date.now();
      const windows = s.windows.map((w) => {
        if (w.id !== id) return w;
        const nextWindow = withUpdatedAt({ ...w, x, y }, now);
        return w.tauri ? nextWindow : normalizeBrowserWindow(nextWindow);
      });
      const next = { ...s, windows };
      persist(next);
      return next;
    }),

  moveToHost: (id, hostLabel, x, y, monitorId) =>
    set((s) => {
      const cleanHostLabel = normalizeHostLabel(hostLabel);
      const cleanMonitorId = normalizeMonitorId(monitorId);
      const now = Date.now();
      const windows = s.windows.map((w) => {
        if (w.id !== id) return w;
        return normalizeBrowserWindow({
          ...withUpdatedAt(
            {
              ...w,
              hostLabel: cleanHostLabel,
              monitorId: monitorId === undefined ? w.monitorId : cleanMonitorId,
              x,
              y,
            },
            now,
          ),
        });
      });
      const next = { ...s, windows, focusedId: id };
      persist(next);
      return next;
    }),

  resize: (id, width, height) =>
    set((s) => {
      const now = Date.now();
      const windows = s.windows.map((w) => {
        if (w.id !== id) return w;
        const nextWindow = withUpdatedAt({ ...w, width, height }, now);
        return w.tauri ? nextWindow : normalizeBrowserWindow(nextWindow);
      });
      const next = { ...s, windows };
      persist(next);
      return next;
    }),

  reconcileHostAvailability: (hosts) =>
    set((s) => {
      const normalizedHosts = hosts.map((host) => ({
        ...host,
        hostLabel: normalizeHostLabel(host.hostLabel),
        monitorId: normalizeMonitorId(host.monitorId),
      }));
      const activeHostsByLabel = new Map(
        normalizedHosts.map((host) => [host.hostLabel, host] as const),
      );
      const fallbackHost = primaryHostTarget(normalizedHosts);
      const fallbackHostLabel = normalizeHostLabel(fallbackHost.hostLabel);
      const fallbackMonitorId = normalizeMonitorId(fallbackHost.monitorId);
      const now = Date.now();
      let changed = false;
      const windows = s.windows.map((w) => {
        const ownerHostLabel = normalizeHostLabel(w.hostLabel);
        const activeHost = activeHostsByLabel.get(ownerHostLabel) ?? null;
        const windowMonitorId = normalizeMonitorId(w.monitorId);
        if (
          activeHost &&
          (!windowMonitorId || windowMonitorId === activeHost.monitorId)
        ) {
          if (windowMonitorId === activeHost.monitorId) return w;
          changed = true;
          return withUpdatedAt(
            {
              ...w,
              hostLabel: activeHost.hostLabel,
              monitorId: activeHost.monitorId,
            },
            now,
          );
        }
        changed = true;
        return normalizeBrowserWindow(
          withUpdatedAt(
            {
              ...w,
              hostLabel: fallbackHostLabel,
              monitorId: fallbackMonitorId,
              x: Math.max(0, w.x),
              y: Math.max(0, w.y),
            },
            now,
          ),
        );
      });
      if (!changed) return s;
      const next = {
        ...s,
        windows,
        focusedId: validFocusForWindows(windows, s.focusedId),
      };
      persist(next);
      return next;
    }),

  reconcileFromTauri: (snapshots) =>
    set((s) => {
      // Only reconcile in Tauri mode; otherwise leave the in-shell windows
      // untouched.
      if (!isTauriDesktop() || !DETACHED_TAURI_TOOL_WINDOWS) return s;

      // Map current Tauri snapshots (excluding the shell "main" window) by
      // toolId.
      const seenToolIds = new Set<ShellToolId>();
      const focusedLabels = new Set<string>();
      for (const snap of snapshots) {
        if (snap.label === "main") continue;
        const toolId = toolIdFromTauriLabel(snap.label);
        if (!toolId) continue;
        seenToolIds.add(toolId);
        if (snap.focused) focusedLabels.add(snap.label);
      }

      const minimizedByTool = new Map<ShellToolId, boolean>();
      for (const snap of snapshots) {
        const toolId = toolIdFromTauriLabel(snap.label);
        if (!toolId) continue;
        minimizedByTool.set(toolId, snap.minimized);
      }

      // 1) Drop any tauri-flagged window whose toolId is no longer present.
      // 2) Update minimized flags from snapshots.
      // 3) Add tauri entries for any new toolId we haven't tracked yet.
      const filtered: DesktopWindow[] = [];
      const knownTauriTools = new Set<ShellToolId>();
      for (const w of s.windows) {
        if (w.tauri) {
          if (!seenToolIds.has(w.toolId)) continue;
          knownTauriTools.add(w.toolId);
          const minimized = minimizedByTool.get(w.toolId) ?? false;
          filtered.push({ ...w, minimized });
        } else {
          filtered.push(w);
        }
      }
      let z = nextZ(filtered);
      const now = Date.now();
      for (const toolId of seenToolIds) {
        if (knownTauriTools.has(toolId)) continue;
        z += 1;
        filtered.push({
          id: makeId(),
          toolId,
          hostLabel: "main",
          monitorId: null,
          createdAtMs: now,
          updatedAtMs: now,
          x: 0,
          y: 0,
          width: 0,
          height: 0,
          z,
          minimized: minimizedByTool.get(toolId) ?? false,
          maximized: false,
          tauri: true,
        });
      }

      // Resolve focusedId from focused Tauri window if any.
      let focusedId = s.focusedId;
      if (focusedLabels.size > 0) {
        const matchedTool = Array.from(seenToolIds).find((toolId) =>
          focusedLabels.has(tauriLabelForTool(toolId)),
        );
        if (matchedTool) {
          const matchedWindow = filtered.find(
            (w) => w.tauri && w.toolId === matchedTool,
          );
          if (matchedWindow) focusedId = matchedWindow.id;
        }
      } else {
        // If the previously focused window is gone, pick the topmost.
        if (focusedId && !filtered.some((w) => w.id === focusedId)) {
          focusedId = topOf(filtered.filter((w) => !w.minimized))?.id ?? null;
        }
      }

      const next = { ...s, windows: filtered, focusedId };
      persist(next);
      return next;
    }),
}));
