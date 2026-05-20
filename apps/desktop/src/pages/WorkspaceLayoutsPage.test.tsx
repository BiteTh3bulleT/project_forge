import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import { WorkspaceLayoutsPage } from "./WorkspaceLayoutsPage";

const storeMocks = vi.hoisted(() => {
  const actions = {
    activateLayout: vi.fn(),
    addLayoutWindow: vi.fn(),
    captureRuntimeIntoLayout: vi.fn(),
    clearFallbackNotice: vi.fn(),
    createLayout: vi.fn(),
    deleteLayout: vi.fn(),
    duplicateLayout: vi.fn(),
    removeLayoutWindow: vi.fn(),
    renameLayout: vi.fn(),
    selectLayout: vi.fn(),
    setMainMonitor: vi.fn(),
    setDisplayArrangementMode: vi.fn(),
    setMonitorRoleLabel: vi.fn(),
    updateLayoutWindow: vi.fn(),
  };
  const state = {
    activeLayoutId: "focus",
    selectedLayoutId: "focus",
    supported: true,
    fallbackNotice: null,
    monitorDesignations: {
      mainMonitorId: "display-main",
      customLabels: { "display-main": "Desk" },
    },
    monitorRoleMap: { "display-main": "main" },
    displayIntent: {
      arrangementMode: "preserve",
      primaryMonitorId: "display-main",
      preferredOrder: ["display-main"],
      applyDeferred: true,
      updatedAtMs: null,
    },
    monitors: [
      {
        id: "display-main",
        ordinal: 0,
        name: "Internal Display",
        position: { x: 0, y: 0 },
        size: { width: 1920, height: 1080 },
        workArea: { x: 0, y: 0, width: 1920, height: 1040 },
        scaleFactor: 1,
      },
    ],
    runtimeWindows: [
      {
        runtimeLabel: "main",
        title: "FORGE Runtime",
        currentRoute: "/chat",
        monitorId: "display-main",
        bounds: { x: 10, y: 20, width: 1200, height: 800 },
        isFocused: true,
      },
    ],
    layouts: [
      {
        id: "focus",
        name: "Focus Layout",
        createdAtMs: 1_800_000_000_000,
        updatedAtMs: 1_800_000_000_000,
        lastActivatedAtMs: 1_800_000_000_000,
        windows: [
          {
            id: "main-window",
            runtimeLabel: "main",
            title: "FORGE Main",
            role: "chat",
            assignedRoutes: ["/chat", "/jobs"],
            activeRoute: "/chat",
            targetMonitorId: "display-main",
            targetMonitorOrdinal: 0,
            targetMonitorRole: "main",
            bounds: null,
            fallbackReason: null,
          },
        ],
      },
    ],
    ...actions,
  };

  return {
    actions,
    state,
    useWorkspaceLayoutStore: vi.fn(
      (selector: (state: Record<string, unknown>) => unknown) =>
        selector(state),
    ),
  };
});

vi.mock("../stores/workspaceLayoutStore", () => ({
  useWorkspaceLayoutStore: storeMocks.useWorkspaceLayoutStore,
}));

describe("WorkspaceLayoutsPage", () => {
  it("renders the selected layout, monitor role, and runtime window state", () => {
    render(<WorkspaceLayoutsPage />);

    expect(screen.getByText("Layout command board")).toBeTruthy();
    expect(screen.getByText("Desktop runtime")).toBeTruthy();
    expect(screen.getByText("Focus Layout")).toBeTruthy();
    expect(screen.getAllByText("Internal Display").length).toBeGreaterThan(1);
    expect(screen.getByText(/Role:/).textContent).toContain("Main");
    expect(screen.getByText("FORGE Runtime")).toBeTruthy();
    expect(screen.getByText("focused")).toBeTruthy();
    expect(storeMocks.useWorkspaceLayoutStore).toHaveBeenCalled();
  });

  it("wires the display layout intent selector to the workspace layout store", () => {
    render(<WorkspaceLayoutsPage />);

    fireEvent.change(screen.getByLabelText("Display arrangement intent"), {
      target: { value: "extend" },
    });

    expect(storeMocks.actions.setDisplayArrangementMode).toHaveBeenCalledWith(
      "extend",
    );
    expect(screen.getByText("Apply deferred")).toBeTruthy();
  });
});
