import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { MemoryPage } from "./MemoryPage";

const apiMocks = vi.hoisted(() => {
  let observations: Array<Record<string, unknown>> = [];
  let nextID = 1;

  function toObservation(input: Record<string, unknown>) {
    const now = 1_800_000_000_000 + nextID;
    return {
      id: nextID++,
      createdAtMs: now,
      updatedAtMs: now,
      observedAtMs: Number(input.observedAtMs ?? now),
      type: String(input.type ?? "note"),
      rawContent: String(input.rawContent ?? ""),
      summary: String(input.summary ?? ""),
      embeddingRef: "",
      dossierId: input.dossierId ?? null,
      projectKey: "",
      sourcePath: "",
      entities: [],
      tags: Array.isArray(input.tags) ? input.tags : [],
      relatedFiles: [],
      taskType: "",
      confidence: Number(input.confidence ?? 0.8),
      verificationState: String(input.verificationState ?? "unknown"),
      lineage: [],
      originKind: String(input.originKind ?? ""),
      originId: String(input.originId ?? ""),
      stale: false,
      lastVerifiedAtMs: null,
      usefulnessScore: 0,
      usefulnessCount: 0,
      noiseCount: 0,
      vsaPointerId: null,
    };
  }

  return {
    reset() {
      observations = [];
      nextID = 1;
    },
    search: vi.fn(async () => ({ hits: [] })),
    listObservations: vi.fn(async () => ({ observations })),
    createObservation: vi.fn(async (body: Record<string, unknown>) => {
      const observation = toObservation(body);
      observations = [observation, ...observations];
      return { observation };
    }),
    getObservation: vi.fn(async (id: number) => {
      const observation = observations.find((item) => item.id === id);
      return {
        observation: {
          observation,
          incomingLinks: [],
          outgoingLinks: [],
          signals: [],
          vsa: null,
        },
      };
    }),
    getObservationVSA: vi.fn(async () => ({ detail: null })),
    patchObservation: vi.fn(),
    markObservationUsefulness: vi.fn(),
    listRepairRuns: vi.fn(async () => ({ runs: [] })),
    getRepairRun: vi.fn(),
    listVSAReindexRuns: vi.fn(async () => ({ runs: [] })),
    getVSAReindexRun: vi.fn(),
    runRepair: vi.fn(),
    runVSAReindex: vi.fn(),
  };
});

vi.mock("../lib/api", () => ({
  api: {
    search: apiMocks.search,
    memory: {
      listObservations: apiMocks.listObservations,
      createObservation: apiMocks.createObservation,
      getObservation: apiMocks.getObservation,
      getObservationVSA: apiMocks.getObservationVSA,
      patchObservation: apiMocks.patchObservation,
      markObservationUsefulness: apiMocks.markObservationUsefulness,
      listRepairRuns: apiMocks.listRepairRuns,
      getRepairRun: apiMocks.getRepairRun,
      listVSAReindexRuns: apiMocks.listVSAReindexRuns,
      getVSAReindexRun: apiMocks.getVSAReindexRun,
      runRepair: apiMocks.runRepair,
      runVSAReindex: apiMocks.runVSAReindex,
    },
  },
}));

describe("MemoryPage note composer", () => {
  beforeEach(() => {
    apiMocks.reset();
    vi.clearAllMocks();
  });

  it("records an operator note through the memory observation API", async () => {
    render(
      <MemoryRouter initialEntries={["/memory"]}>
        <MemoryPage />
      </MemoryRouter>,
    );

    fireEvent.change(await screen.findByPlaceholderText(/decision/i), {
      target: { value: "Remember the basalt notebook" },
    });
    fireEvent.change(screen.getByPlaceholderText(/durable note/i), {
      target: { value: "The basalt notebook belongs in reopened chat memory." },
    });
    fireEvent.change(screen.getByPlaceholderText(/comma/i), {
      target: { value: "memory, section6" },
    });
    fireEvent.click(screen.getByRole("button", { name: "Record note" }));

    await waitFor(() => {
      expect(apiMocks.createObservation).toHaveBeenCalledWith(
        expect.objectContaining({
          type: "note",
          summary: "Remember the basalt notebook",
          rawContent:
            "The basalt notebook belongs in reopened chat memory.",
          tags: ["memory", "section6"],
          verificationState: "operator_recorded",
          originKind: "operator_note",
        }),
      );
    });
    expect(await screen.findByText(/Memory note recorded as observation #1/)).toBeTruthy();
    expect(await screen.findByText(/Remember the basalt notebook/)).toBeTruthy();
  });
});
