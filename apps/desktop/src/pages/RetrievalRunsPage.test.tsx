import { render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import { RetrievalRunsPage } from "./RetrievalRunsPage";

const apiMocks = vi.hoisted(() => ({
  createRun: vi.fn(),
  getRun: vi.fn(),
  getRunVSASignals: vi.fn(),
  getObservation: vi.fn(),
  listRuns: vi.fn(),
  markUsefulness: vi.fn(),
  retrievalSelection: vi.fn(),
}));

vi.mock("../lib/api", () => ({
  api: {
    memory: {
      getObservation: apiMocks.getObservation,
      retrievalSelection: apiMocks.retrievalSelection,
    },
    retrieval: {
      createRun: apiMocks.createRun,
      getRun: apiMocks.getRun,
      getRunVSASignals: apiMocks.getRunVSASignals,
      listRuns: apiMocks.listRuns,
      markUsefulness: apiMocks.markUsefulness,
    },
  },
}));

describe("RetrievalRunsPage", () => {
  it("renders empty retrieval run states", async () => {
    apiMocks.listRuns.mockResolvedValueOnce({ runs: [] });

    render(<RetrievalRunsPage />);

    expect(await screen.findByText("No retrieval runs yet")).toBeTruthy();
    expect(screen.getByText("Select a run")).toBeTruthy();
    expect(apiMocks.listRuns).toHaveBeenCalledWith({
      dossierId: undefined,
      limit: 80,
    });
  });
});
