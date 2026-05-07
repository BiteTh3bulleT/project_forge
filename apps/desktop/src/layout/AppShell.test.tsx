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
}));

vi.mock("../lib/desktop", () => ({
  DETACHED_TAURI_TOOL_WINDOWS: false,
  isShellHostWindowLabel: (label: string) =>
    label === "main" ||
    (label.startsWith("forge-") && !label.startsWith("forge-app-")),
  isTauriDesktop: desktopMocks.isTauriDesktop,
  listForgeWindows: desktopMocks.listForgeWindows,
  monitorSignature: (monitors: Array<{ id: string }>) =>
    monitors.map((monitor) => monitor.id).join(";"),
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
    useDesktopWindowStore.setState({
      pinned: ["chat", "jobs", "memory", "models", "approvals", "settings"],
      windows: [
        {
          id: "chat-window",
          toolId: "chat",
          hostLabel: "main",
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
      runtimeWindows: [],
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

  it("renders each in-shell window only on its assigned desktop host", () => {
    useDesktopWindowStore.setState({
      windows: [
        {
          id: "chat-window",
          toolId: "chat",
          hostLabel: "forge-right",
          x: 100,
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

    const mainShell = render(
      <MemoryRouter>
        <AppShell isMainWindow={true} hostLabel="main">
          <div />
        </AppShell>
      </MemoryRouter>,
    );

    expect(mainShell.queryByTestId("mock-tool-content")).toBeNull();
    mainShell.unmount();

    render(
      <MemoryRouter>
        <AppShell isMainWindow={true} hostLabel="forge-right">
          <div />
        </AppShell>
      </MemoryRouter>,
    );

    expect(screen.getByTestId("mock-tool-content")).toBeTruthy();
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
