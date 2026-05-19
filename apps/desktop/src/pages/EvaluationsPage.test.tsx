import { render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import { EvaluationsPage } from "./EvaluationsPage";

const apiMocks = vi.hoisted(() => ({
  createEvaluation: vi.fn(),
  listEvaluations: vi.fn(),
  listJobs: vi.fn(),
  metrics: vi.fn(),
}));

vi.mock("../lib/api", () => ({
  api: {
    evaluations: {
      create: apiMocks.createEvaluation,
      list: apiMocks.listEvaluations,
      metrics: apiMocks.metrics,
    },
    jobs: {
      list: apiMocks.listJobs,
    },
  },
}));

describe("EvaluationsPage", () => {
  it("renders empty evaluation and adapter metric states", async () => {
    apiMocks.listEvaluations.mockResolvedValueOnce({ evaluations: [] });
    apiMocks.metrics.mockResolvedValueOnce({ metrics: [] });
    apiMocks.listJobs.mockResolvedValueOnce({ jobs: [] });

    render(<EvaluationsPage />);

    expect(await screen.findByText("No adapter metrics yet")).toBeTruthy();
    expect(screen.getByText("No evaluations recorded")).toBeTruthy();
    expect(apiMocks.listEvaluations).toHaveBeenCalledWith(120, undefined);
    expect(apiMocks.metrics).toHaveBeenCalledWith(undefined);
    expect(apiMocks.listJobs).toHaveBeenCalledWith("", 60);
  });
});
