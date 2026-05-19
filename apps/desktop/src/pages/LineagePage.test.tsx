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
});
