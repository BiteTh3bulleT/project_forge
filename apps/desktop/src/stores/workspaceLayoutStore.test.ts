import { beforeEach, describe, expect, it, vi } from "vitest";

type TestMonitor = {
  id: string;
  ordinal: number;
  name: string;
  position: { x: number; y: number };
  size: { width: number; height: number };
  workArea: { x: number; y: number; width: number; height: number };
  scaleFactor: number;
};

const desktopMocks = vi.hoisted(() => ({
  createShellWindow: vi.fn(async () => ({ label: "layout-chat" })),
  emitWorkspaceSync: vi.fn(async () => undefined),
  getCurrentWindowLabel: vi.fn(async () => "main"),
  getCurrentWindowSnapshot: vi.fn(async () => ({
    label: "main",
    title: "FORGE",
    isFocused: true,
    monitorId: null,
    bounds: null,
  })),
  getWindowByLabel: vi.fn(async () => null),
  isTauriDesktop: vi.fn(() => true),
  listAvailableMonitors: vi.fn(async (): Promise<TestMonitor[]> => []),
  listRuntimeWindows: vi.fn(async () => [{ label: "main" }]),
  monitorSignature: vi.fn(() => ""),
  spanCurrentWindowAcrossMonitors: vi.fn(async () => true),
  tauriWindow: {
    setTitle: vi.fn(async () => undefined),
    setPosition: vi.fn(async () => undefined),
    setSize: vi.fn(async () => undefined),
    show: vi.fn(async () => undefined),
    setFocus: vi.fn(async () => undefined),
  },
}));

vi.mock("@tauri-apps/api/window", () => ({
  LogicalPosition: class LogicalPosition {
    constructor(
      public x: number,
      public y: number,
    ) {}
  },
  LogicalSize: class LogicalSize {
    constructor(
      public width: number,
      public height: number,
    ) {}
  },
  getCurrentWindow: () => desktopMocks.tauriWindow,
}));

vi.mock("../lib/desktop", () => ({
  DETACHED_TAURI_TOOL_WINDOWS: false,
  WORKSPACE_NAVIGATE_EVENT: "forge://workspace-navigate",
  createShellWindow: desktopMocks.createShellWindow,
  emitWorkspaceSync: desktopMocks.emitWorkspaceSync,
  getCurrentWindowLabel: desktopMocks.getCurrentWindowLabel,
  getCurrentWindowSnapshot: desktopMocks.getCurrentWindowSnapshot,
  getWindowByLabel: desktopMocks.getWindowByLabel,
  isTauriDesktop: desktopMocks.isTauriDesktop,
  isForgeManagedWindowLabel: (label: string) =>
    label === "main" || label.startsWith("forge-"),
  isShellHostWindowLabel: (label: string) =>
    label === "main" ||
    (label.startsWith("forge-") && !label.startsWith("forge-app-")),
  listAvailableMonitors: desktopMocks.listAvailableMonitors,
  listRuntimeWindows: desktopMocks.listRuntimeWindows,
  monitorSignature: desktopMocks.monitorSignature,
  spanCurrentWindowAcrossMonitors: desktopMocks.spanCurrentWindowAcrossMonitors,
  virtualDesktopBounds: (monitors: TestMonitor[]) => {
    if (monitors.length === 0) return null;
    const minX = Math.min(...monitors.map((monitor) => monitor.workArea.x));
    const minY = Math.min(...monitors.map((monitor) => monitor.workArea.y));
    const maxX = Math.max(
      ...monitors.map((monitor) => monitor.workArea.x + monitor.workArea.width),
    );
    const maxY = Math.max(
      ...monitors.map(
        (monitor) => monitor.workArea.y + monitor.workArea.height,
      ),
    );
    return { x: minX, y: minY, width: maxX - minX, height: maxY - minY };
  },
}));

function activeLayoutDoc() {
  return {
    version: 2,
    monitorDesignations: { mainMonitorId: null, customLabels: {} },
    activeLayoutId: "layout-1",
    selectedLayoutId: "layout-1",
    layouts: [
      {
        id: "layout-1",
        name: "Old detached layout",
        windows: [
          {
            id: "window-1",
            runtimeLabel: "forge-build-workbench",
            title: "FORGE Chat",
            role: "chat",
            assignedRoutes: ["/chat"],
            activeRoute: "/chat",
            targetMonitorId: null,
            targetMonitorOrdinal: 0,
            targetMonitorRole: null,
            bounds: null as
              | { x: number; y: number; width: number; height: number }
              | null,
            fallbackReason: null,
          },
        ],
        createdAtMs: 1,
        updatedAtMs: 1,
        lastActivatedAtMs: 1,
      },
    ],
    runtimeWindows: [],
    lastKnownMonitors: [],
    lastMonitorSignature: "",
    fallbackNotice: null,
    lastRestoreAtMs: null,
  };
}

function currentMainLayoutDoc() {
  const doc = activeLayoutDoc();
  doc.layouts[0]!.windows = [
    {
      id: "main-window",
      runtimeLabel: "main",
      title: "FORGE Main",
      role: "mixed",
      assignedRoutes: ["/chat"],
      activeRoute: "/chat",
      targetMonitorId: null,
      targetMonitorOrdinal: 0,
      targetMonitorRole: null,
      bounds: { x: 40, y: 40, width: 1200, height: 800 },
      fallbackReason: null,
    },
  ];
  return doc;
}

describe("workspace layout hydration", () => {
  beforeEach(() => {
    vi.resetModules();
    localStorage.clear();
    desktopMocks.createShellWindow.mockClear();
    desktopMocks.emitWorkspaceSync.mockClear();
    desktopMocks.spanCurrentWindowAcrossMonitors.mockClear();
    desktopMocks.tauriWindow.setPosition.mockClear();
    desktopMocks.tauriWindow.setSize.mockClear();
    localStorage.setItem(
      "forge.workspace.layouts.v2",
      JSON.stringify(activeLayoutDoc()),
    );
  });

  it("spans the main desktop shell instead of creating duplicate monitor desktops", async () => {
    const secondaryWindow = {
      label: "forge-build-workbench",
      close: vi.fn(async () => undefined),
    };
    const staleDetachedToolWindow = {
      label: "forge-app-chat",
      close: vi.fn(async () => undefined),
    };
    desktopMocks.listAvailableMonitors.mockResolvedValueOnce([
      {
        id: "main-display",
        ordinal: 0,
        name: "Main",
        position: { x: 0, y: 0 },
        size: { width: 1920, height: 1080 },
        workArea: { x: 0, y: 0, width: 1920, height: 1040 },
        scaleFactor: 1,
      },
      {
        id: "second-display",
        ordinal: 1,
        name: "Second",
        position: { x: 1920, y: 0 },
        size: { width: 1920, height: 1080 },
        workArea: { x: 1920, y: 0, width: 1920, height: 1040 },
        scaleFactor: 1,
      },
    ]);
    desktopMocks.listRuntimeWindows.mockResolvedValue([
      { label: "main" },
      secondaryWindow,
      staleDetachedToolWindow,
    ]);
    const { useWorkspaceLayoutStore } = await import("./workspaceLayoutStore");

    await useWorkspaceLayoutStore.getState().hydrate("/");

    expect(desktopMocks.createShellWindow).not.toHaveBeenCalled();
    expect(desktopMocks.spanCurrentWindowAcrossMonitors).toHaveBeenCalled();
    expect(secondaryWindow.close).toHaveBeenCalled();
    expect(staleDetachedToolWindow.close).toHaveBeenCalled();
    expect(useWorkspaceLayoutStore.getState().runtimeWindows).toEqual([
      expect.objectContaining({
        runtimeLabel: "main",
        currentRoute: "/chat",
        bounds: { x: 0, y: 0, width: 3840, height: 1040 },
      }),
    ]);
    expect(useWorkspaceLayoutStore.getState().fallbackNotice).toBeNull();
    expect(useWorkspaceLayoutStore.getState().ready).toBe(true);
    expect(useWorkspaceLayoutStore.getState().currentWindowLabel).toBe("main");
  });

  it("restores the main desktop host to its assigned monitor", async () => {
    const monitors = [
      {
        id: "left",
        ordinal: 0,
        name: "Left",
        position: { x: 0, y: 0 },
        size: { width: 1920, height: 1080 },
        workArea: { x: 0, y: 0, width: 1920, height: 1040 },
        scaleFactor: 1,
      },
      {
        id: "right",
        ordinal: 1,
        name: "Right",
        position: { x: 1920, y: 0 },
        size: { width: 2560, height: 1440 },
        workArea: { x: 1920, y: 0, width: 2560, height: 1400 },
        scaleFactor: 1,
      },
    ];
    desktopMocks.listAvailableMonitors.mockResolvedValueOnce(monitors);
    localStorage.setItem(
      "forge.workspace.layouts.v2",
      JSON.stringify(currentMainLayoutDoc()),
    );
    const { useWorkspaceLayoutStore } = await import("./workspaceLayoutStore");

    await useWorkspaceLayoutStore.getState().hydrate("/");

    expect(desktopMocks.spanCurrentWindowAcrossMonitors).toHaveBeenCalled();
    expect(desktopMocks.tauriWindow.setPosition).not.toHaveBeenCalled();
    expect(desktopMocks.tauriWindow.setSize).not.toHaveBeenCalled();
  });
});
