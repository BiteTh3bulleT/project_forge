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

  it("exposes Audit pivots from selected artifact job context and event traces", async () => {
    apiMocks.listArtifacts.mockResolvedValueOnce({
      artifacts: [
        {
          id: 12,
          createdAtMs: Date.UTC(2026, 4, 24, 12, 0, 0),
          jobId: "job-workbench-12",
          type: "job_result",
          title: "Trace output",
          filePath: "/tmp/trace-output.json",
          mimeType: "application/json",
          metadata: {},
        },
      ],
    });
    apiMocks.getJobDetail.mockResolvedValueOnce({
      job: {
        id: "job-workbench-12",
        title: "Workbench trace job",
        status: "succeeded",
        targetAdapter: "gateway",
        taskPacketId: null,
      },
      events: [
        {
          id: 4,
          createdAtMs: Date.UTC(2026, 4, 24, 12, 1, 0),
          type: "gateway.completed",
          message: "Gateway completed",
          payload: { traceId: "trace-workbench-12" },
        },
      ],
    });
    apiMocks.getArtifact.mockResolvedValueOnce({
      id: 12,
      createdAtMs: Date.UTC(2026, 4, 24, 12, 0, 0),
      jobId: "job-workbench-12",
      type: "job_result",
      title: "Trace output",
      filePath: "/tmp/trace-output.json",
      mimeType: "application/json",
      metadata: {},
    });
    apiMocks.getArtifactContent.mockResolvedValueOnce({
      content: "{}",
      textual: true,
      previewLimited: false,
    });

    render(
      <MemoryRouter
        initialEntries={[
          "/workbench?jobId=job-workbench-12&artifactId=12",
        ]}
      >
        <WorkbenchPage />
      </MemoryRouter>,
    );

    expect(await screen.findByText("Workbench trace job")).toBeTruthy();
    expect(
      screen
        .getByRole("link", { name: "Audit job-workbench-12" })
        .getAttribute("href"),
    ).toBe("/audit?jobId=job-workbench-12");
    expect(
      screen
        .getByRole("link", { name: "Audit trace-workbench-12" })
        .getAttribute("href"),
    ).toBe("/audit?traceId=trace-workbench-12");
  });
});
