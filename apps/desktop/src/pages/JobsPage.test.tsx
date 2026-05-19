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
});
