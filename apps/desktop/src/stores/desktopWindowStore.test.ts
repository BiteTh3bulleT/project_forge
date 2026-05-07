import { beforeEach, describe, expect, it, vi } from "vitest";

const desktopMocks = vi.hoisted(() => ({
  closeTauriWindow: vi.fn(),
  createShellWindow: vi.fn(async () => ({ label: "forge-app-chat" })),
  focusTauriWindow: vi.fn(),
  isTauriDesktop: vi.fn(() => true),
  minimizeTauriWindow: vi.fn(),
}));

vi.mock("../lib/desktop", () => ({
  closeTauriWindow: desktopMocks.closeTauriWindow,
  createShellWindow: desktopMocks.createShellWindow,
  focusTauriWindow: desktopMocks.focusTauriWindow,
  isTauriDesktop: desktopMocks.isTauriDesktop,
  minimizeTauriWindow: desktopMocks.minimizeTauriWindow,
}));

describe("desktop window store", () => {
  beforeEach(() => {
    window.localStorage.clear();
    desktopMocks.closeTauriWindow.mockClear();
    desktopMocks.createShellWindow.mockClear();
    desktopMocks.focusTauriWindow.mockClear();
    desktopMocks.minimizeTauriWindow.mockClear();
  });

  it("opens tool surfaces in the main shell instead of detachable Tauri windows", async () => {
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
});
