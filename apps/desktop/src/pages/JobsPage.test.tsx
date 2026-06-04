import { render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { describe, expect, it, vi } from "vitest";

import { JobsPage } from "./JobsPage";

const apiMocks = vi.hoisted(() => ({
  create: vi.fn(),
  list: vi.fn(),
}));

vi.mock("../lib/api", () => ({
  api: {
    jobs: {
      create: apiMocks.create,
      list: apiMocks.list,
    },
  },
}));

describe("JobsPage", () => {
  it("renders an empty jobs board", async () => {
    apiMocks.list.mockResolvedValue({ jobs: [] });

    render(
      <MemoryRouter>
        <JobsPage />
      </MemoryRouter>,
    );

    expect(await screen.findByText("No jobs yet.")).toBeTruthy();
    expect(screen.getByText("Jobs command board")).toBeTruthy();
    expect(apiMocks.list).toHaveBeenCalledWith("", 180);
  });

  it("exposes an Audit pivot for each recent job", async () => {
    apiMocks.list.mockResolvedValue({
      jobs: [
        {
          id: "job-audit-1",
          title: "Traceable job",
          requestedAction: "inspect",
          targetAdapter: "gateway",
          status: "succeeded",
          createdAtMs: Date.UTC(2026, 4, 24, 12, 0, 0),
          updatedAtMs: Date.UTC(2026, 4, 24, 12, 1, 0),
          queuedAtMs: null,
          startedAtMs: null,
          completedAtMs: null,
          initiatingSource: "test",
          executionBoundary: "bounded",
          riskClass: "read_only",
          approvalStatus: "not_required",
          writeIntent: false,
          cancelRequested: false,
          taskPacketId: null,
          resultSummary: null,
          failureInfo: null,
          lastFailureCode: null,
          lastError: null,
          metadata: {},
        },
      ],
    });

    render(
      <MemoryRouter>
        <JobsPage />
      </MemoryRouter>,
    );

    const audit = await screen.findByRole("link", {
      name: "Audit job-audit-1",
    });
    expect(audit.getAttribute("href")).toBe("/audit?jobId=job-audit-1");
  });
});
