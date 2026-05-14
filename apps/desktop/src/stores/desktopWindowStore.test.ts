import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

const desktopMocks = vi.hoisted(() => ({
  closeTauriWindow: vi.fn(async () => true),
  createShellWindow: vi.fn(async () => ({ label: "forge-app-chat" })),
  focusTauriWindow: vi.fn(),
  isTauriDesktop: vi.fn(() => true),
  minimizeTauriWindow: vi.fn(),
}));

vi.mock("../lib/desktop", () => ({
  DETACHED_TAURI_TOOL_WINDOWS: false,
  isTauriDesktop: desktopMocks.isTauriDesktop,
}));

vi.mock("../lib/windowManager", () => ({
  closeTauriWindow: desktopMocks.closeTauriWindow,
  createShellWindow: desktopMocks.createShellWindow,
  focusTauriWindow: desktopMocks.focusTauriWindow,
  minimizeTauriWindow: desktopMocks.minimizeTauriWindow,
}));

describe("desktop window store", () => {
  beforeEach(() => {
    vi.resetModules();
    vi.useRealTimers();
    window.localStorage.clear();
    desktopMocks.closeTauriWindow.mockClear();
    desktopMocks.createShellWindow.mockClear();
    desktopMocks.focusTauriWindow.mockClear();
    desktopMocks.minimizeTauriWindow.mockClear();
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  it("opens tool surfaces as confined in-shell windows in the desktop shell", async () => {
    const { useDesktopWindowStore } = await import("./desktopWindowStore");
    useDesktopWindowStore.setState({
      windows: [],
      focusedId: null,
    });

    const id = await useDesktopWindowStore.getState().openWindow("chat");

    expect(id).toBeTruthy();
    expect(desktopMocks.createShellWindow).not.toHaveBeenCalled();
    expect(desktopMocks.closeTauriWindow).toHaveBeenCalledWith(
      "forge-app-chat",
    );
    const state = useDesktopWindowStore.getState();
    expect(state.windows).toHaveLength(1);
    expect(state.windows[0]?.toolId).toBe("chat");
    expect(state.windows[0]?.tauri).toBe(false);
    expect(state.focusedId).toBe(id);
  });

  it("hydrates secondary host labels and fills Phase G7 metadata for legacy in-shell windows", async () => {
    vi.useFakeTimers();
    vi.setSystemTime(new Date("2026-05-12T16:00:00.000Z"));
    const { useDesktopWindowStore } = await import("./desktopWindowStore");
    window.localStorage.setItem(
      "forge.os.windows.v1",
      JSON.stringify([
        {
          id: "legacy-window",
          toolId: "chat",
          hostLabel: "forge-right",
          x: 120,
          y: 80,
          width: 960,
          height: 640,
          z: 2,
          minimized: false,
          maximized: false,
          tauri: false,
        },
      ]),
    );

    useDesktopWindowStore.getState().hydrate();

    const state = useDesktopWindowStore.getState();
    expect(state.windows).toHaveLength(1);
    expect(state.windows[0]).toMatchObject({
      id: "legacy-window",
      toolId: "chat",
      hostLabel: "forge-right",
      monitorId: null,
      createdAtMs: 1778601600000,
      updatedAtMs: 1778601600000,
      x: 40,
      y: 0,
    });
  });

  it("openWindow stores monitorId and Phase G7 timestamps for in-shell windows", async () => {
    vi.useFakeTimers();
    vi.setSystemTime(new Date("2026-05-12T16:01:00.000Z"));
    const { useDesktopWindowStore } = await import("./desktopWindowStore");
    useDesktopWindowStore.setState({
      windows: [],
      focusedId: null,
    });

    const id = await useDesktopWindowStore
      .getState()
      .openWindow("jobs", {
        hostLabel: "forge-right",
        monitorId: "display-2",
        x: 10,
        y: 12,
        width: 500,
        height: 320,
      });

    expect(id).toBeTruthy();
    const state = useDesktopWindowStore.getState();
    expect(state.windows).toHaveLength(1);
    expect(state.windows[0]).toMatchObject({
      id,
      toolId: "jobs",
      hostLabel: "forge-right",
      monitorId: "display-2",
      createdAtMs: 1778601660000,
      updatedAtMs: 1778601660000,
      x: 10,
      y: 12,
      tauri: false,
    });
  });

  it("moveToHost persists host, monitor, and updatedAtMs for in-shell windows", async () => {
    vi.useFakeTimers();
    vi.setSystemTime(new Date("2026-05-12T16:02:00.000Z"));
    const { useDesktopWindowStore } = await import("./desktopWindowStore");
    useDesktopWindowStore.setState({
      windows: [],
      focusedId: null,
    });

    const id = await useDesktopWindowStore.getState().openWindow("models");
    useDesktopWindowStore.getState().resize(id!, 500, 320);
    expect(id).toBeTruthy();
    vi.setSystemTime(new Date("2026-05-12T16:03:00.000Z"));

    useDesktopWindowStore
      .getState()
      .moveToHost(id!, "forge-right", 24, 32, "display-2");

    const state = useDesktopWindowStore.getState();
    expect(state.windows).toHaveLength(1);
    expect(state.windows[0]).toMatchObject({
      id,
      toolId: "models",
      hostLabel: "forge-right",
      monitorId: "display-2",
      createdAtMs: 1778601720000,
      updatedAtMs: 1778601780000,
      x: 24,
      y: 32,
      tauri: false,
    });
    expect(
      JSON.parse(window.localStorage.getItem("forge.os.windows.v1")!),
    ).toEqual([
      expect.objectContaining({
        id,
        toolId: "models",
        hostLabel: "forge-right",
        monitorId: "display-2",
        updatedAtMs: 1778601780000,
      }),
    ]);
  });

  it("keeps existing same-tool windows scoped per host", async () => {
    const { useDesktopWindowStore } = await import("./desktopWindowStore");
    useDesktopWindowStore.setState({
      windows: [],
      focusedId: null,
    });

    const mainId = await useDesktopWindowStore
      .getState()
      .openWindow("chat", { hostLabel: "main" });
    const rightId = await useDesktopWindowStore
      .getState()
      .openWindow("chat", { hostLabel: "forge-right" });
    const rightAgainId = await useDesktopWindowStore
      .getState()
      .openWindow("chat", { hostLabel: "forge-right" });

    expect(mainId).toBeTruthy();
    expect(rightId).toBeTruthy();
    expect(rightAgainId).toBe(rightId);
    expect(rightId).not.toBe(mainId);
    expect(useDesktopWindowStore.getState().windows).toEqual([
      expect.objectContaining({
        id: mainId,
        toolId: "chat",
        hostLabel: "main",
      }),
      expect.objectContaining({
        id: rightId,
        toolId: "chat",
        hostLabel: "forge-right",
      }),
    ]);
  });

  it("moves windows from unavailable hosts back to the primary host", async () => {
    vi.useFakeTimers();
    vi.setSystemTime(new Date("2026-05-12T16:04:00.000Z"));
    const { useDesktopWindowStore } = await import("./desktopWindowStore");
    useDesktopWindowStore.setState({
      windows: [
        {
          id: "right-window",
          toolId: "chat",
          hostLabel: "forge-monitor-2",
          monitorId: "missing-display",
          x: 140,
          y: 90,
          width: 700,
          height: 440,
          z: 2,
          minimized: false,
          maximized: false,
          tauri: false,
          createdAtMs: 1778601600000,
          updatedAtMs: 1778601600000,
        },
      ],
      focusedId: "right-window",
    });

    useDesktopWindowStore.getState().reconcileHostAvailability([
      {
        hostLabel: "main",
        monitorId: "main-display",
        primary: true,
      },
    ]);

    expect(useDesktopWindowStore.getState().windows).toEqual([
      expect.objectContaining({
        id: "right-window",
        toolId: "chat",
        hostLabel: "main",
        monitorId: "main-display",
        createdAtMs: 1778601600000,
        updatedAtMs: 1778601840000,
      }),
    ]);
    expect(useDesktopWindowStore.getState().focusedId).toBe("right-window");
    expect(
      JSON.parse(window.localStorage.getItem("forge.os.windows.v1")!),
    ).toEqual([
      expect.objectContaining({
        id: "right-window",
        hostLabel: "main",
        monitorId: "main-display",
      }),
    ]);
  });

  it("moves windows with stale monitor mappings back to the primary host", async () => {
    vi.useFakeTimers();
    vi.setSystemTime(new Date("2026-05-12T16:05:00.000Z"));
    const { useDesktopWindowStore } = await import("./desktopWindowStore");
    useDesktopWindowStore.setState({
      windows: [
        {
          id: "stale-right-window",
          toolId: "jobs",
          hostLabel: "forge-monitor-2",
          monitorId: "old-display",
          x: 64,
          y: 72,
          width: 700,
          height: 440,
          z: 3,
          minimized: false,
          maximized: false,
          tauri: false,
          createdAtMs: 1778601600000,
          updatedAtMs: 1778601600000,
        },
      ],
      focusedId: "stale-right-window",
    });

    useDesktopWindowStore.getState().reconcileHostAvailability([
      { hostLabel: "main", monitorId: "main-display", primary: true },
      { hostLabel: "forge-monitor-2", monitorId: "new-display" },
    ]);

    expect(useDesktopWindowStore.getState().windows[0]).toMatchObject({
      id: "stale-right-window",
      toolId: "jobs",
      hostLabel: "main",
      monitorId: "main-display",
      createdAtMs: 1778601600000,
      updatedAtMs: 1778601900000,
    });
  });

  it("refreshes missing monitor metadata for an available host", async () => {
    vi.useFakeTimers();
    vi.setSystemTime(new Date("2026-05-12T16:06:00.000Z"));
    const { useDesktopWindowStore } = await import("./desktopWindowStore");
    useDesktopWindowStore.setState({
      windows: [
        {
          id: "legacy-main-window",
          toolId: "memory",
          hostLabel: "main",
          monitorId: null,
          x: 64,
          y: 72,
          width: 700,
          height: 440,
          z: 3,
          minimized: false,
          maximized: false,
          tauri: false,
          createdAtMs: 1778601600000,
          updatedAtMs: 1778601600000,
        },
      ],
      focusedId: "legacy-main-window",
    });

    useDesktopWindowStore.getState().reconcileHostAvailability([
      { hostLabel: "main", monitorId: "main-display", primary: true },
    ]);

    expect(useDesktopWindowStore.getState().windows[0]).toMatchObject({
      id: "legacy-main-window",
      hostLabel: "main",
      monitorId: "main-display",
      updatedAtMs: 1778601960000,
    });
  });

  it("leaves valid host and monitor pairs untouched", async () => {
    const { useDesktopWindowStore } = await import("./desktopWindowStore");
    const windowRecord = {
      id: "valid-right-window",
      toolId: "models" as const,
      hostLabel: "forge-monitor-2",
      monitorId: "right-display",
      x: 64,
      y: 72,
      width: 700,
      height: 440,
      z: 3,
      minimized: false,
      maximized: false,
      tauri: false,
      createdAtMs: 1778601600000,
      updatedAtMs: 1778601600000,
    };
    useDesktopWindowStore.setState({
      windows: [windowRecord],
      focusedId: "valid-right-window",
    });

    useDesktopWindowStore.getState().reconcileHostAvailability([
      { hostLabel: "main", monitorId: "main-display", primary: true },
      { hostLabel: "forge-monitor-2", monitorId: "right-display" },
    ]);

    expect(useDesktopWindowStore.getState().windows).toEqual([windowRecord]);
    expect(useDesktopWindowStore.getState().focusedId).toBe(
      "valid-right-window",
    );
  });

  it("drops stale persisted focus and selects a visible window during hydration", async () => {
    window.localStorage.setItem(
      "forge.os.windows.v1",
      JSON.stringify([
        {
          id: "alive-window",
          toolId: "chat",
          hostLabel: "main",
          x: 120,
          y: 92,
          width: 960,
          height: 640,
          z: 3,
          minimized: false,
          maximized: false,
          tauri: false,
        },
      ]),
    );
    window.localStorage.setItem("forge.os.focus.v1", "missing-window");

    const { useDesktopWindowStore } = await import("./desktopWindowStore");
    useDesktopWindowStore.getState().hydrate();

    expect(useDesktopWindowStore.getState().focusedId).toBe("alive-window");
  });
});
