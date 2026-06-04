import { render, screen } from "@testing-library/react";
import { MemoryRouter, Route, Routes } from "react-router-dom";
import { describe, expect, it, vi } from "vitest";

import { JobDetailPage } from "./JobDetailPage";

const apiMocks = vi.hoisted(() => ({
  approve: vi.fn(),
  cancel: vi.fn(),
  deny: vi.fn(),
  detail: vi.fn(),
  listReviews: vi.fn(),
  packetAlignment: vi.fn(),
  replay: vi.fn(),
  retry: vi.fn(),
}));

vi.mock("../lib/api", () => ({
  api: {
    approvals: {
      approve: apiMocks.approve,
      deny: apiMocks.deny,
    },
    jobs: {
      cancel: apiMocks.cancel,
      detail: apiMocks.detail,
      replay: apiMocks.replay,
      retry: apiMocks.retry,
    },
    memory: {
      packetAlignment: apiMocks.packetAlignment,
    },
    reviews: {
      list: apiMocks.listReviews,
    },
  },
}));

describe("JobDetailPage", () => {
  it("renders a loaded job projection with empty evidence states", async () => {
    apiMocks.detail.mockResolvedValueOnce({
      job: {
        id: "job-123",
        title: "Inspect repository state",
        status: "succeeded",
        requestedAction: "inspect",
        targetAdapter: "gateway",
        executionBoundary: "bounded",
        initiatingSource: "test",
        taskPacketId: null,
        cancelRequested: false,
        createdAtMs: 1_800_000_000_000,
        updatedAtMs: 1_800_000_001_000,
        approvalStatus: "not_required",
        riskClass: "low",
        writeIntent: false,
        resultSummary: "",
        lastError: "",
        lastFailureCode: null,
      },
      approvalRequest: null,
      artifacts: [],
      events: [],
      packet: null,
    });
    apiMocks.listReviews.mockResolvedValueOnce({ reviews: [] });

    render(
      <MemoryRouter initialEntries={["/jobs/job-123"]}>
        <Routes>
          <Route path="/jobs/:id" element={<JobDetailPage />} />
        </Routes>
      </MemoryRouter>,
    );

    expect(await screen.findByText("Inspect repository state")).toBeTruthy();
    expect(screen.getByText(/inspect through gateway/)).toBeTruthy();
    expect(screen.getByText("No review records linked to this job.")).toBeTruthy();
    expect(screen.getByText("No artifacts persisted for this job.")).toBeTruthy();
    expect(screen.getByText("No events yet.")).toBeTruthy();
    expect(screen.getAllByText("No packet attached.").length).toBeGreaterThan(1);
    expect(apiMocks.detail).toHaveBeenCalledWith("job-123", 0);
    expect(apiMocks.listReviews).toHaveBeenCalledWith({ limit: 260 });
    expect(apiMocks.packetAlignment).not.toHaveBeenCalled();
  });

  it("exposes Audit pivots for job artifacts and event trace payloads", async () => {
    apiMocks.detail.mockResolvedValueOnce({
      job: {
        id: "job-trace-9",
        title: "Traceable job projection",
        status: "succeeded",
        requestedAction: "inspect",
        targetAdapter: "gateway",
        executionBoundary: "bounded",
        initiatingSource: "test",
        taskPacketId: null,
        cancelRequested: false,
        createdAtMs: 1_800_000_000_000,
        updatedAtMs: 1_800_000_001_000,
        approvalStatus: "not_required",
        riskClass: "low",
        writeIntent: false,
        resultSummary: "",
        lastError: "",
        lastFailureCode: null,
      },
      approvalRequest: null,
      artifacts: [
        {
          id: 42,
          createdAtMs: 1_800_000_000_500,
          jobId: "job-trace-9",
          packetId: null,
          type: "job_result",
          title: "Trace artifact",
          filePath: "/tmp/trace-artifact.json",
          mimeType: "application/json",
          metadata: {},
        },
      ],
      events: [
        {
          id: 3,
          jobId: "job-trace-9",
          createdAtMs: 1_800_000_000_750,
          type: "gateway.completed",
          message: "Gateway completed",
          payload: { correlationId: "corr-job-event-9" },
        },
      ],
      packet: null,
    });
    apiMocks.listReviews.mockResolvedValueOnce({ reviews: [] });

    render(
      <MemoryRouter initialEntries={["/jobs/job-trace-9"]}>
        <Routes>
          <Route path="/jobs/:id" element={<JobDetailPage />} />
        </Routes>
      </MemoryRouter>,
    );

    expect(await screen.findByText("Traceable job projection")).toBeTruthy();
    expect(
      screen
        .getByRole("link", { name: "Audit artifact 42" })
        .getAttribute("href"),
    ).toBe("/audit?jobId=job-trace-9");
    expect(
      screen
        .getByRole("link", { name: "Audit corr-job-event-9" })
        .getAttribute("href"),
    ).toBe("/audit?correlationId=corr-job-event-9");
  });
});
