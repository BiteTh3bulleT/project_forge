import { render, screen, waitFor } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { describe, expect, it, vi } from "vitest";

import { WorkbenchPage } from "./WorkbenchPage";

const apiMocks = vi.hoisted(() => ({
  listArtifacts: vi.fn(),
  getArtifact: vi.fn(),
  getArtifactContent: vi.fn(),
  getJobDetail: vi.fn(),
}));

vi.mock("../lib/api", () => ({
  api: {
    artifacts: {
      list: apiMocks.listArtifacts,
      get: apiMocks.getArtifact,
      content: apiMocks.getArtifactContent,
    },
    jobs: {
      detail: apiMocks.getJobDetail,
    },
  },
}));

describe("WorkbenchPage", () => {
  it("renders an empty artifact index when the API returns a nullable list", async () => {
    apiMocks.listArtifacts.mockResolvedValueOnce({ artifacts: null });

    render(
      <MemoryRouter>
        <WorkbenchPage />
      </MemoryRouter>,
    );

    expect(await screen.findByText("No artifacts found")).toBeTruthy();
    expect(screen.getByText("0 artifacts")).toBeTruthy();
  });

  it("handles nullable job event lists without crashing", async () => {
    apiMocks.listArtifacts.mockResolvedValueOnce({ artifacts: [] });
    apiMocks.getJobDetail.mockResolvedValueOnce({
      job: {
        id: "job-1",
        title: "Nullable events job",
        status: "running",
        targetAdapter: "test",
        taskPacketId: null,
      },
      events: null,
    });

    render(
      <MemoryRouter initialEntries={["/workbench?jobId=job-1"]}>
        <WorkbenchPage />
      </MemoryRouter>,
    );

    expect(await screen.findByText("Nullable events job")).toBeTruthy();
    await waitFor(() => {
      expect(screen.queryByText("Recent job events")).toBeNull();
    });
  });
});
