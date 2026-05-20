import { beforeEach, describe, expect, it, vi } from "vitest";

const desktopMocks = vi.hoisted(() => ({
  createShellWindow: vi.fn(async () => ({ label: "legacy-window" })),
  getWindowByLabel: vi.fn(async (label: string) => ({ label })),
  isTauriDesktop: vi.fn(() => true),
  isShellHostWindowLabel: vi.fn(
    (label: string) =>
      label === "main" ||
      (label.startsWith("forge-") && !label.startsWith("forge-app-")),
  ),
  listRuntimeWindows: vi.fn(async () => []),
  focusTauriWindow: vi.fn(async () => true),
  closeTauriWindow: vi.fn(async () => true),
  minimizeTauriWindow: vi.fn(async () => true),
  listForgeWindows: vi.fn(async () => []),
}));

const tauriMocks = vi.hoisted(() => ({
  invoke: vi.fn(),
  listen: vi.fn(async () => vi.fn()),
}));

vi.mock("@tauri-apps/api/core", () => ({
  invoke: tauriMocks.invoke,
}));

vi.mock("@tauri-apps/api/event", () => ({
  listen: tauriMocks.listen,
}));

vi.mock("./desktop", () => ({
  createShellWindow: desktopMocks.createShellWindow,
  getWindowByLabel: desktopMocks.getWindowByLabel,
  isTauriDesktop: desktopMocks.isTauriDesktop,
  isShellHostWindowLabel: desktopMocks.isShellHostWindowLabel,
  listRuntimeWindows: desktopMocks.listRuntimeWindows,
  focusTauriWindow: desktopMocks.focusTauriWindow,
  closeTauriWindow: desktopMocks.closeTauriWindow,
  minimizeTauriWindow: desktopMocks.minimizeTauriWindow,
  listForgeWindows: desktopMocks.listForgeWindows,
}));

describe("FORGE frontend window manager bridge", () => {
  beforeEach(() => {
    vi.resetModules();
    tauriMocks.invoke.mockReset();
    tauriMocks.listen.mockClear();
    desktopMocks.createShellWindow.mockClear();
    desktopMocks.getWindowByLabel.mockClear();
    desktopMocks.isTauriDesktop.mockReturnValue(true);
    desktopMocks.isShellHostWindowLabel.mockClear();
    desktopMocks.focusTauriWindow.mockClear();
    desktopMocks.closeTauriWindow.mockClear();
    desktopMocks.minimizeTauriWindow.mockClear();
  });

  it("opens supported native windows through the backend window manager", async () => {
    tauriMocks.invoke.mockResolvedValueOnce({
      label: "settings",
      kind: "settings",
      route: "/settings",
      title: "FORGE Settings",
      visible: true,
      focused: true,
      minimized: false,
      singleton: true,
      createdAtMs: 100,
      updatedAtMs: 100,
    });
    const { createShellWindow } = await import("./windowManager");

    await expect(
      createShellWindow({
        label: "settings",
        route: "/settings",
        title: "FORGE Settings",
        bounds: { x: 120, y: 80, width: 900, height: 640 },
      }),
    ).resolves.toEqual({ label: "settings" });

    expect(tauriMocks.invoke).toHaveBeenCalledWith("forge_window_open", {
      request: expect.objectContaining({
        kind: "settings",
        route: "/settings",
        title: "FORGE Settings",
      }),
    });
    expect(desktopMocks.createShellWindow).not.toHaveBeenCalled();
  });

  it("opens shell-host labels through the backend window manager", async () => {
    tauriMocks.invoke.mockResolvedValueOnce({
      label: "forge-monitor-2",
      kind: "shell_host",
      route: "/?host=forge-monitor-2",
      title: "FORGE Monitor 2",
      visible: true,
      focused: true,
      minimized: false,
      singleton: true,
      hostId: "forge-monitor-2",
      createdAtMs: 100,
      updatedAtMs: 100,
    });
    const { createShellWindow } = await import("./windowManager");

    await createShellWindow({
      label: "forge-monitor-2",
      route: "/?host=forge-monitor-2",
      title: "FORGE Monitor 2",
      bounds: { x: 1920, y: 0, width: 1280, height: 720 },
    });

    expect(tauriMocks.invoke).toHaveBeenCalledWith("forge_window_open", {
      request: expect.objectContaining({
        kind: "shell_host",
        hostId: "forge-monitor-2",
        route: "/?host=forge-monitor-2",
        title: "FORGE Monitor 2",
      }),
    });
    expect(desktopMocks.createShellWindow).not.toHaveBeenCalled();
  });

  it("does not open the reserved debug-console surface by default", async () => {
    const { createShellWindow } = await import("./windowManager");

    await expect(
      createShellWindow({
        label: "debug-console",
        route: "/system?surface=debug-console",
        title: "FORGE Debug Console",
        bounds: { x: 120, y: 80, width: 900, height: 640 },
      }),
    ).resolves.toBeNull();

    expect(tauriMocks.invoke).not.toHaveBeenCalled();
    expect(desktopMocks.createShellWindow).not.toHaveBeenCalled();
  });

  it("falls back to legacy focus when backend focus rejects a stale label", async () => {
    tauriMocks.invoke.mockRejectedValueOnce(new Error("not registered"));
    const { focusForgeWindow } = await import("./windowManager");

    await expect(focusForgeWindow("forge-app-chat")).resolves.toBe(true);

    expect(desktopMocks.focusTauriWindow).toHaveBeenCalledWith(
      "forge-app-chat",
    );
  });

  it("parses backend registry snapshots", async () => {
    tauriMocks.invoke.mockResolvedValueOnce({
      windows: [
        {
          label: "artifact-report-1",
          kind: "artifact_viewer",
          route: "/artifacts/report-1",
          title: "Report",
          visible: true,
          focused: false,
          minimized: false,
          singleton: false,
          created_at_ms: 10,
          updated_at_ms: 20,
        },
      ],
      timestamp_ms: 30,
    });
    const { snapshotForgeWindows } = await import("./windowManager");

    await expect(snapshotForgeWindows()).resolves.toEqual({
      windows: [
        expect.objectContaining({
          label: "artifact-report-1",
          kind: "artifact_viewer",
          createdAtMs: 10,
          updatedAtMs: 20,
        }),
      ],
      timestampMs: 30,
    });
  });
});
