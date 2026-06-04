import { render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { describe, expect, it, vi } from "vitest";

import { LineagePage } from "./LineagePage";

const apiMocks = vi.hoisted(() => ({
  byJob: vi.fn(),
  listJobs: vi.fn(),
  replay: vi.fn(),
  retry: vi.fn(),
}));

vi.mock("../lib/api", () => ({
  api: {
    jobs: {
      list: apiMocks.listJobs,
      replay: apiMocks.replay,
      retry: apiMocks.retry,
    },
    lineage: {
      byJob: apiMocks.byJob,
    },
  },
}));

describe("LineagePage", () => {
  it("renders empty recent job and lineage states", async () => {
    apiMocks.listJobs.mockResolvedValue({ jobs: [] });

    render(
      <MemoryRouter>
        <LineagePage />
      </MemoryRouter>,
    );

    expect(await screen.findByText("No jobs yet.")).toBeTruthy();
    expect(screen.getByText("No lineage loaded.")).toBeTruthy();
    expect(apiMocks.listJobs).toHaveBeenCalledWith("", 80);
  });

  it("exposes Audit pivots for the selected lineage job and related jobs", async () => {
    apiMocks.listJobs.mockResolvedValue({
      jobs: [
        {
          id: "job-lineage-1",
          title: "Lineage origin",
          status: "succeeded",
          targetAdapter: "gateway",
          createdAtMs: Date.UTC(2026, 4, 24, 12, 0, 0),
        },
      ],
    });
    apiMocks.byJob.mockResolvedValue({
      parents: [],
      children: [],
      relatedJobs: [
        {
          id: "job-lineage-child",
          title: "Lineage child",
          status: "succeeded",
          targetAdapter: "gateway",
        },
      ],
    });

    render(
      <MemoryRouter initialEntries={["/lineage?jobId=job-lineage-1"]}>
        <LineagePage />
      </MemoryRouter>,
    );

    expect(
      (await screen.findByRole("link", { name: "Audit job-lineage-1" }))
        .getAttribute("href"),
    ).toBe("/audit?jobId=job-lineage-1");
    expect(
      (
        await screen.findByRole("link", {
          name: "Audit job-lineage-child",
        })
      ).getAttribute("href"),
    ).toBe("/audit?jobId=job-lineage-child");
  });
});
