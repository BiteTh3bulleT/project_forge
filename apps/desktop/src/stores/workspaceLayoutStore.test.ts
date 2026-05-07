import { beforeEach, describe, expect, it, vi } from "vitest";

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
  listAvailableMonitors: vi.fn(async () => []),
  listRuntimeWindows: vi.fn(async () => [{ label: "main" }]),
  monitorSignature: vi.fn(() => ""),
}));

vi.mock("../lib/desktop", () => ({
  WORKSPACE_NAVIGATE_EVENT: "forge://workspace-navigate",
  createShellWindow: desktopMocks.createShellWindow,
  emitWorkspaceSync: desktopMocks.emitWorkspaceSync,
  getCurrentWindowLabel: desktopMocks.getCurrentWindowLabel,
  getCurrentWindowSnapshot: desktopMocks.getCurrentWindowSnapshot,
  getWindowByLabel: desktopMocks.getWindowByLabel,
  isTauriDesktop: desktopMocks.isTauriDesktop,
  listAvailableMonitors: desktopMocks.listAvailableMonitors,
  listRuntimeWindows: desktopMocks.listRuntimeWindows,
  monitorSignature: desktopMocks.monitorSignature,
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
            runtimeLabel: "layout-chat",
            title: "FORGE Chat",
            role: "chat",
            assignedRoutes: ["/chat"],
            activeRoute: "/chat",
            targetMonitorId: null,
            targetMonitorOrdinal: 0,
            targetMonitorRole: null,
            bounds: null,
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

describe("workspace layout hydration", () => {
  beforeEach(() => {
    vi.resetModules();
    localStorage.clear();
    desktopMocks.createShellWindow.mockClear();
    desktopMocks.emitWorkspaceSync.mockClear();
    localStorage.setItem(
      "forge.workspace.layouts.v2",
      JSON.stringify(activeLayoutDoc()),
    );
  });

  it("restores saved detachable layout windows during Tauri shell startup", async () => {
    const { useWorkspaceLayoutStore } = await import("./workspaceLayoutStore");

    await useWorkspaceLayoutStore.getState().hydrate("/");

    expect(desktopMocks.createShellWindow).toHaveBeenCalledWith(
      expect.objectContaining({
        label: "layout-chat",
        route: "/chat",
        title: "FORGE Chat",
      }),
    );
    expect(useWorkspaceLayoutStore.getState().ready).toBe(true);
    expect(useWorkspaceLayoutStore.getState().currentWindowLabel).toBe("main");
  });
});
