import { cleanup, fireEvent, render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { useNavigate } from "react-router-dom";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { useDesktopWindowStore } from "../stores/desktopWindowStore";

import { AppShell } from "./AppShell";

function SearchChangeHarness() {
  const navigate = useNavigate();
  return (
    <>
      <button type="button" onClick={() => navigate("/?boardId=1")}>
        Change search
      </button>
      <AppShell isMainWindow={true}>
        <div />
      </AppShell>
    </>
  );
}

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

  it("does not duplicate detached Tauri windows inside the main shell", () => {
    useDesktopWindowStore.setState({
      pinned: ["chat"],
      windows: [
        {
          id: "chat-native-window",
          toolId: "chat",
          x: 0,
          y: 0,
          width: 0,
          height: 0,
          z: 1,
          minimized: false,
          maximized: false,
          tauri: true,
        },
      ],
      focusedId: "chat-native-window",
    });

    const { container } = render(
      <MemoryRouter>
        <AppShell isMainWindow={true}>
          <div />
        </AppShell>
      </MemoryRouter>,
    );

    expect(screen.queryByTestId("mock-tool-content")).toBeNull();
    expect(
      container.querySelector(".forge-os-window__body--docked"),
    ).toBeNull();
  });

  it("keeps Start open when a background surface only changes search params", () => {
    render(
      <MemoryRouter initialEntries={["/"]}>
        <SearchChangeHarness />
      </MemoryRouter>,
    );

    fireEvent.click(screen.getByRole("button", { name: "Open Start menu" }));
    expect(screen.getByRole("dialog", { name: "FORGE Start" })).toBeTruthy();

    fireEvent.click(screen.getByRole("button", { name: "Change search" }));

    expect(screen.getByRole("dialog", { name: "FORGE Start" })).toBeTruthy();
  });
});
