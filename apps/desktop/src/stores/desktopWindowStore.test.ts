import { beforeEach, describe, expect, it, vi } from "vitest";

const desktopMocks = vi.hoisted(() => ({
  closeTauriWindow: vi.fn(),
  createShellWindow: vi.fn(async () => ({ label: "forge-app-chat" })),
  focusTauriWindow: vi.fn(),
  isTauriDesktop: vi.fn(() => true),
  minimizeTauriWindow: vi.fn(),
}));

vi.mock("../lib/desktop", () => ({
  DETACHED_TAURI_TOOL_WINDOWS: false,
  closeTauriWindow: desktopMocks.closeTauriWindow,
  createShellWindow: desktopMocks.createShellWindow,
  focusTauriWindow: desktopMocks.focusTauriWindow,
  isTauriDesktop: desktopMocks.isTauriDesktop,
  minimizeTauriWindow: desktopMocks.minimizeTauriWindow,
}));

describe("desktop window store", () => {
  beforeEach(() => {
    vi.resetModules();
    window.localStorage.clear();
    desktopMocks.closeTauriWindow.mockClear();
    desktopMocks.createShellWindow.mockClear();
    desktopMocks.focusTauriWindow.mockClear();
    desktopMocks.minimizeTauriWindow.mockClear();
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
    const state = useDesktopWindowStore.getState();
    expect(state.windows).toHaveLength(1);
    expect(state.windows[0]?.toolId).toBe("chat");
    expect(state.windows[0]?.tauri).toBe(false);
    expect(state.focusedId).toBe(id);
  });

  it("drops stale persisted focus and selects a visible window during hydration", async () => {
    window.localStorage.setItem(
      "forge.os.windows.v1",
      JSON.stringify([
        {
          id: "alive-window",
          toolId: "chat",
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
