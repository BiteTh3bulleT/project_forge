import {
  cleanup,
  fireEvent,
  render,
  screen,
  waitFor,
} from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { useNavigate } from "react-router-dom";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { useDesktopWindowStore } from "../stores/desktopWindowStore";
import { useWorkspaceLayoutStore } from "../stores/workspaceLayoutStore";

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
  spanCurrentWindowAcrossMonitors: vi.fn(async () => false),
}));

vi.mock("../lib/desktop", () => ({
  DETACHED_TAURI_TOOL_WINDOWS: false,
  isTauriDesktop: desktopMocks.isTauriDesktop,
  listForgeWindows: desktopMocks.listForgeWindows,
  monitorSignature: (monitors: Array<{ id: string }>) =>
    monitors.map((monitor) => monitor.id).join(";"),
  spanCurrentWindowAcrossMonitors:
    desktopMocks.spanCurrentWindowAcrossMonitors,
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

describe("AppShell confined Tauri tool surfaces", () => {
  beforeEach(() => {
    window.localStorage.clear();
    desktopMocks.isTauriDesktop.mockReturnValue(true);
    desktopMocks.listForgeWindows.mockImplementation(
      () => new Promise<never>(() => {}),
    );
    desktopMocks.spanCurrentWindowAcrossMonitors.mockClear();
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
    useWorkspaceLayoutStore.setState({
      monitors: [],
      supported: true,
      currentWindowLabel: "main",
      fallbackNotice: null,
    });
  });

  afterEach(() => {
    cleanup();
  });

  it("renders Tauri tool surfaces as movable in-shell windows", async () => {
    render(
      <MemoryRouter>
        <AppShell isMainWindow={true}>
          <div />
        </AppShell>
      </MemoryRouter>,
    );

    expect(screen.getByTestId("mock-tool-content")).toBeTruthy();

    fireEvent.click(screen.getByRole("button", { name: "Minimize" }));

    await waitFor(() => {
      expect(screen.queryByTestId("mock-tool-content")).toBeNull();
      expect(useDesktopWindowStore.getState().focusedId).toBeNull();
      expect(useDesktopWindowStore.getState().windows[0]?.minimized).toBe(true);
    });
  });

  it("uses floating window chrome for Tauri shell surfaces", () => {
    const { container } = render(
      <MemoryRouter>
        <AppShell isMainWindow={true}>
          <div />
        </AppShell>
      </MemoryRouter>,
    );

    expect(container.querySelector(".forge-os-window__body")).toBeTruthy();
    expect(container.querySelector(".forge-os-window__body--docked")).toBeNull();
  });

  it("does not poll detached Tauri window state in confined shell mode", () => {
    render(
      <MemoryRouter>
        <AppShell isMainWindow={true}>
          <div />
        </AppShell>
      </MemoryRouter>,
    );

    expect(desktopMocks.listForgeWindows).not.toHaveBeenCalled();
  });

  it("spans the main shell across detected monitor work areas", async () => {
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
    useWorkspaceLayoutStore.setState({ monitors });

    render(
      <MemoryRouter>
        <AppShell isMainWindow={true}>
          <div />
        </AppShell>
      </MemoryRouter>,
    );

    await waitFor(() => {
      expect(
        desktopMocks.spanCurrentWindowAcrossMonitors,
      ).toHaveBeenCalledWith(monitors);
    });
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
