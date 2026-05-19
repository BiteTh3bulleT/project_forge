import { fireEvent, render, screen, waitFor } from "@testing-library/react";
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

  it("clears stale selected run detail when a refreshed filter returns no runs", async () => {
    const run = {
      id: 1,
      query: "old run",
      mode: "hybrid",
      createdAtMs: 1_800_000_000_000,
      packetId: null,
      jobId: null,
      dossierId: null,
      results: [],
    };
    apiMocks.listRuns
      .mockResolvedValueOnce({ runs: [run] })
      .mockResolvedValueOnce({ runs: [] });
    apiMocks.retrievalSelection.mockResolvedValue({ selection: [] });
    apiMocks.getRunVSASignals.mockResolvedValue({ signals: [] });

    render(<RetrievalRunsPage />);

    expect(await screen.findByText(/run #1 \| mode hybrid/i)).toBeTruthy();

    fireEvent.change(screen.getByPlaceholderText("e.g. 1"), {
      target: { value: "999" },
    });
    fireEvent.click(screen.getByRole("button", { name: "Refresh" }));

    await waitFor(() => {
      expect(screen.queryByText(/run #1 \| mode hybrid/i)).toBeNull();
    });
    expect(await screen.findByText("Select a run")).toBeTruthy();
  });
});
