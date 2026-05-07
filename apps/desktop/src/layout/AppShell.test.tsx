import { cleanup, fireEvent, render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { useDesktopWindowStore } from "../stores/desktopWindowStore";

import { AppShell } from "./AppShell";

const desktopMocks = vi.hoisted(() => ({
  isTauriDesktop: vi.fn(() => true),
  listForgeWindows: vi.fn(() => new Promise<never>(() => {})),
}));

vi.mock("../lib/desktop", () => ({
  isTauriDesktop: desktopMocks.isTauriDesktop,
  listForgeWindows: desktopMocks.listForgeWindows,
}));

vi.mock("../lib/api", () => ({
  api: {
    dashboard: {
      summary: vi.fn(() => new Promise<never>(() => {})),
    },
  },
}));

vi.mock("./toolRegistry", () => ({
  getToolComponent: () => function MockToolComponent() {
    return <div data-testid="mock-tool-content">Mock tool content</div>;
  },
}));

describe("AppShell docked Tauri tool surfaces", () => {
  beforeEach(() => {
    window.localStorage.clear();
    desktopMocks.isTauriDesktop.mockReturnValue(true);
    desktopMocks.listForgeWindows.mockImplementation(
      () => new Promise<never>(() => {}),
    );
    useDesktopWindowStore.setState({
      pinned: ["chat", "jobs", "memory", "models", "approvals", "settings"],
      windows: [
        {
          id: "chat-window",
          toolId: "chat",
          x: 120,
          y: 92,
          width: 960,
          height: 640,
          z: 1,
          minimized: false,
          maximized: false,
          tauri: false,
        },
      ],
      focusedId: "chat-window",
    });
  });

  afterEach(() => {
    cleanup();
  });

  it("keeps the active dock tab visible when clicked in Tauri shell mode", () => {
    render(
      <MemoryRouter>
        <AppShell isMainWindow={true}>
          <div />
        </AppShell>
      </MemoryRouter>,
    );

    expect(screen.getByTestId("mock-tool-content")).toBeTruthy();

    fireEvent.click(screen.getByTitle("Chat (active)"));

    expect(screen.getByTestId("mock-tool-content")).toBeTruthy();
    expect(useDesktopWindowStore.getState().focusedId).toBe("chat-window");
    expect(useDesktopWindowStore.getState().windows[0]?.minimized).toBe(false);
  });

  it("uses the docked scroll body for Tauri shell surfaces", () => {
    const { container } = render(
      <MemoryRouter>
        <AppShell isMainWindow={true}>
          <div />
        </AppShell>
      </MemoryRouter>,
    );

    expect(
      container.querySelector(".forge-os-window__body--docked"),
    ).toBeTruthy();
  });
});
