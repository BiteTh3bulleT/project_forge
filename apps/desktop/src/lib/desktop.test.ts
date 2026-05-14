import { beforeEach, describe, expect, it, vi } from "vitest";

const windowMocks = vi.hoisted(() => ({
  isMaximized: vi.fn(async () => false),
  unmaximize: vi.fn(async () => undefined),
  setPosition: vi.fn(async () => undefined),
  setSize: vi.fn(async () => undefined),
  setFocus: vi.fn(async () => undefined),
}));

const shellWindowMocks = vi.hoisted(() => ({
  show: vi.fn(async () => undefined),
  setFocus: vi.fn(async () => undefined),
  setPosition: vi.fn(async () => undefined),
  setSize: vi.fn(async () => undefined),
}));

vi.mock("@tauri-apps/api/event", () => ({
  emit: vi.fn(async () => undefined),
}));

vi.mock("@tauri-apps/api/webviewWindow", () => ({
  WebviewWindow: vi.fn(() => shellWindowMocks),
}));

vi.mock("@tauri-apps/api/core", () => ({
  convertFileSrc: vi.fn((path: string) => `asset://${path}`),
  invoke: vi.fn(),
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
  PhysicalPosition: class PhysicalPosition {
    constructor(
      public x: number,
      public y: number,
    ) {}
  },
  PhysicalSize: class PhysicalSize {
    constructor(
      public width: number,
      public height: number,
    ) {}
  },
  availableMonitors: vi.fn(async () => []),
  currentMonitor: vi.fn(async () => null),
  getAllWindows: vi.fn(async () => []),
  getCurrentWindow: () => windowMocks,
}));

describe("desktop monitor spanning", () => {
  beforeEach(() => {
    windowMocks.isMaximized.mockReset();
    windowMocks.unmaximize.mockClear();
    windowMocks.setPosition.mockClear();
    windowMocks.setSize.mockClear();
    windowMocks.setFocus.mockClear();
    windowMocks.isMaximized.mockResolvedValue(false);
    shellWindowMocks.show.mockClear();
    shellWindowMocks.setFocus.mockClear();
    shellWindowMocks.setPosition.mockClear();
    shellWindowMocks.setSize.mockClear();
    Object.defineProperty(window, "__TAURI_INTERNALS__", {
      configurable: true,
      value: {},
    });
  });

  it("unmaximizes before resizing the shell across multiple monitors", async () => {
    windowMocks.isMaximized.mockResolvedValue(true);
    const { spanCurrentWindowAcrossMonitors } = await import("./desktop");

    await expect(
      spanCurrentWindowAcrossMonitors([
        {
          id: "left",
          ordinal: 0,
          name: "Left",
          position: { x: -1920, y: 0 },
          size: { width: 1920, height: 1080 },
          workArea: { x: -1920, y: 0, width: 1920, height: 1040 },
          scaleFactor: 1,
        },
        {
          id: "main",
          ordinal: 1,
          name: "Main",
          position: { x: 0, y: 0 },
          size: { width: 2560, height: 1440 },
          workArea: { x: 0, y: 0, width: 2560, height: 1400 },
          scaleFactor: 1,
        },
      ]),
    ).resolves.toBe(true);

    expect(windowMocks.unmaximize.mock.invocationCallOrder[0]).toBeLessThan(
      windowMocks.setPosition.mock.invocationCallOrder[0] ?? 0,
    );
    expect(windowMocks.setPosition).toHaveBeenCalledWith(
      expect.objectContaining({ x: -1920, y: 0 }),
    );
    expect(windowMocks.setSize).toHaveBeenCalledWith(
      expect.objectContaining({ width: 4480, height: 1400 }),
    );
  });

  it("creates monitor host windows without overflow clamping", async () => {
    const { WebviewWindow } = await import("@tauri-apps/api/webviewWindow");
    const { createShellWindow } = await import("./desktop");

    await expect(
      createShellWindow({
        label: "forge-monitor-2",
        route: "/?host=forge-monitor-2",
        title: "FORGE Monitor 2",
        bounds: { x: 1920, y: 0, width: 1280, height: 720 },
      }),
    ).resolves.toBe(shellWindowMocks);

    expect(WebviewWindow).toHaveBeenCalledWith(
      "forge-monitor-2",
      expect.objectContaining({
        x: 1920,
        y: 0,
        width: 1280,
        height: 720,
        preventOverflow: false,
      }),
    );
    expect(shellWindowMocks.setPosition).toHaveBeenCalledWith(
      expect.objectContaining({ x: 1920, y: 0 }),
    );
    expect(shellWindowMocks.setSize).toHaveBeenCalledWith(
      expect.objectContaining({ width: 1280, height: 720 }),
    );
  });

  it("sends allowlisted native Linux window actions through Tauri", async () => {
    const { invoke } = await import("@tauri-apps/api/core");
    const { controlLinuxWindow } = await import("./desktop");
    vi.mocked(invoke).mockResolvedValueOnce(true);

    await expect(
      controlLinuxWindow("firefox-window", "minimize"),
    ).resolves.toBe(true);

    expect(invoke).toHaveBeenCalledWith("control_linux_window", {
      windowId: "firefox-window",
      action: "minimize",
    });
  });

  it("parses backend-owned native Linux window registry metadata", async () => {
    const { invoke } = await import("@tauri-apps/api/core");
    const { listLinuxWindows } = await import("./desktop");
    vi.mocked(invoke).mockResolvedValueOnce([
      {
        id: "firefox-window",
        title: "Mozilla Firefox",
        app_id: "firefox",
        focused: true,
        minimized: false,
        native: true,
        lifecycle: "active",
        first_seen_ms: 100,
        last_seen_ms: 250,
      },
    ]);

    await expect(listLinuxWindows()).resolves.toEqual([
      expect.objectContaining({
        id: "firefox-window",
        appId: "firefox",
        lifecycle: "active",
        firstSeenMs: 100,
        lastSeenMs: 250,
      }),
    ]);
  });
});
