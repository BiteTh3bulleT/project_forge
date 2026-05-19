import { render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { describe, expect, it, vi } from "vitest";

import { StartPage } from "./StartPage";

const apiMocks = vi.hoisted(() => ({
  adapters: vi.fn(),
  commandExecute: vi.fn(),
  contextGet: vi.fn(),
  dashboardSummary: vi.fn(),
  sourcesList: vi.fn(),
}));

vi.mock("../lib/api", () => ({
  api: {
    adapters: {
      list: apiMocks.adapters,
    },
    commands: {
      execute: apiMocks.commandExecute,
    },
    dashboard: {
      summary: apiMocks.dashboardSummary,
    },
    projectContext: {
      get: apiMocks.contextGet,
    },
    sources: {
      list: apiMocks.sourcesList,
    },
  },
}));

describe("StartPage", () => {
  it("renders readiness and operator queue state", async () => {
    apiMocks.dashboardSummary.mockResolvedValue({
      activeJobs: [],
      approvalsPending: 0,
      recentFailures: [],
      reviewsPending: 0,
    });
    apiMocks.sourcesList.mockResolvedValue({ sources: [] });
    apiMocks.contextGet.mockResolvedValue({ record: null });
    apiMocks.adapters.mockResolvedValue({ adapters: [] });

    render(
      <MemoryRouter>
        <StartPage />
      </MemoryRouter>,
    );

    expect(await screen.findByText("Start Here")).toBeTruthy();
    expect(screen.getByText("No sources configured yet")).toBeTruthy();
    expect(screen.getByText("Open Approvals Queue")).toBeTruthy();
    expect(apiMocks.dashboardSummary).toHaveBeenCalledOnce();
  });
});
