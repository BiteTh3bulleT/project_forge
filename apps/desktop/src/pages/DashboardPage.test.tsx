import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { DashboardPage } from "./DashboardPage";

const mocks = vi.hoisted(() => ({
  patchSettings: vi.fn(),
}));

vi.mock("../lib/api", () => ({
  api: {
    dashboard: {
      summary: vi.fn(async () => ({
        activeJobs: [],
        recentFailures: [],
        recentImports: [],
        automationActivity: [],
        routingRecommendations: [],
        systemStatus: {},
        approvalsPending: 0,
        reviewsPending: 0,
      })),
    },
    health: vi.fn(async () => ({ modelRuntime: { available: false } })),
    gateway: {
      capabilities: vi.fn(async () => ({ capabilities: [] })),
      invocations: vi.fn(async () => ({ invocations: [] })),
    },
    memory: {
      listObservations: vi.fn(async () => ({ observations: [] })),
    },
    modelRuntime: {
      health: vi.fn(async () => ({ health: null })),
      queue: vi.fn(async () => ({ queue: null })),
      usage: vi.fn(async () => ({ usage: null })),
    },
    settings: {
      get: vi.fn(async () => ({ shadowMode: { enabled: false } })),
      patch: mocks.patchSettings,
    },
  },
}));

describe("DashboardPage shadow mode toggle", () => {
  beforeEach(() => {
    mocks.patchSettings.mockReset();
    mocks.patchSettings.mockResolvedValue({ shadowMode: { enabled: true } });
  });

  it("renders shadow mode state and patches the setting when toggled", async () => {
    render(
      <MemoryRouter>
        <DashboardPage />
      </MemoryRouter>,
    );

    const toggle = await screen.findByRole("button", {
      name: /enable shadow mode/i,
    });
    expect(screen.getByText("Off")).toBeTruthy();

    fireEvent.click(toggle);

    await waitFor(() => {
      expect(mocks.patchSettings).toHaveBeenCalledWith({
        shadowMode: { enabled: true },
      });
    });
    expect(await screen.findByText("On")).toBeTruthy();
  });

  it("renders the cognition console cockpit sections", async () => {
    render(
      <MemoryRouter>
        <DashboardPage />
      </MemoryRouter>,
    );

    expect(await screen.findByRole("heading", { name: "Cognition Console" }))
      .toBeTruthy();
    expect(screen.getAllByText("Active Goals").length).toBeGreaterThan(0);
    expect(screen.getAllByText("Workspace").length).toBeGreaterThan(0);
    expect(screen.getAllByText("Decisions").length).toBeGreaterThan(0);
    expect(screen.getAllByText("Open Loops").length).toBeGreaterThan(0);
    expect(screen.getByText("Runtime Monitor")).toBeTruthy();
    expect(screen.getByText("Cognitive Surfaces")).toBeTruthy();
  });
});
