import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import { RetrievalRunsPage } from "./RetrievalRunsPage";

const apiMocks = vi.hoisted(() => ({
  listRuns: vi.fn(),
  createRun: vi.fn(),
  getRun: vi.fn(),
  getRunVSASignals: vi.fn(),
  markUsefulness: vi.fn(),
  retrievalSelection: vi.fn(),
  getObservation: vi.fn(),
}));

vi.mock("../lib/api", () => ({
  api: {
    retrieval: {
      listRuns: apiMocks.listRuns,
      createRun: apiMocks.createRun,
      getRun: apiMocks.getRun,
      getRunVSASignals: apiMocks.getRunVSASignals,
      markUsefulness: apiMocks.markUsefulness,
    },
    memory: {
      retrievalSelection: apiMocks.retrievalSelection,
      getObservation: apiMocks.getObservation,
    },
  },
}));

describe("RetrievalRunsPage selection", () => {
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
