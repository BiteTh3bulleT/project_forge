import type { MonitorSnapshot } from "../../lib/desktop";

export type WindowRole =
  | "chat"
  | "workbench"
  | "canvas"
  | "dossier"
  | "ops"
  | "review"
  | "settings"
  | "mixed";

export type MonitorDesignation = {
  mainMonitorId: string | null;
  customLabels: Record<string, string>;
};

export type MonitorRoleMap = Record<string, string>;

export type DisplayArrangementMode = "preserve" | "extend" | "mirror";

export type DisplayLayoutIntent = {
  arrangementMode: DisplayArrangementMode;
  primaryMonitorId: string | null;
  preferredOrder: string[];
  applyDeferred: true;
  updatedAtMs: number | null;
};

export type LayoutWindowRecord = {
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

export type LayoutPreset = {
  id: string;
  name: string;
  windows: LayoutWindowRecord[];
  createdAtMs: number;
  updatedAtMs: number;
  lastActivatedAtMs: number | null;
};

export type RuntimeWindowRecord = {
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

export type LayoutDoc = {
  version: 2;
  monitorDesignations: MonitorDesignation;
  activeLayoutId: string | null;
  selectedLayoutId: string | null;
  layouts: LayoutPreset[];
  runtimeWindows: RuntimeWindowRecord[];
  lastKnownMonitors: MonitorSnapshot[];
  lastMonitorSignature: string;
  displayIntent: DisplayLayoutIntent;
  fallbackNotice: string | null;
  lastRestoreAtMs: number | null;
};

export type WorkspaceLayoutState = {
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
  displayIntent: DisplayLayoutIntent;
  fallbackNotice: string | null;
  hydrate: (pathname: string) => Promise<void>;
  refreshEnvironment: () => Promise<void>;
  syncCurrentRoute: (pathname: string) => Promise<void>;
  setMainMonitor: (monitorId: string) => void;
  setDisplayArrangementMode: (mode: DisplayArrangementMode) => void;
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
