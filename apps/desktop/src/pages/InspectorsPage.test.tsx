import { render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { describe, expect, it, vi } from "vitest";

import { InspectorsPage } from "./InspectorsPage";

const apiMocks = vi.hoisted(() => ({
  auditLookup: vi.fn(),
  getDreamReport: vi.fn(),
  getPacket: vi.fn(),
  getSnapshot: vi.fn(),
  listDreamReports: vi.fn(),
  listGuidance: vi.fn(),
  listSnapshots: vi.fn(),
  packetAlignment: vi.fn(),
  processHealth: vi.fn(),
}));

vi.mock("../lib/api", () => ({
  api: {
    audit: {
      lookup: apiMocks.auditLookup,
    },
    contextInspector: {
      getSnapshot: apiMocks.getSnapshot,
      listSnapshots: apiMocks.listSnapshots,
    },
    dreamReports: {
      get: apiMocks.getDreamReport,
      list: apiMocks.listDreamReports,
    },
    memory: {
      packetAlignment: apiMocks.packetAlignment,
    },
    packetGuidance: {
      list: apiMocks.listGuidance,
    },
    packets: {
      get: apiMocks.getPacket,
    },
    processHealth: apiMocks.processHealth,
  },
}));

describe("InspectorsPage", () => {
  it("renders empty inspector surfaces after snapshot load", async () => {
    apiMocks.listSnapshots.mockResolvedValueOnce({ snapshots: [] });

    render(
      <MemoryRouter>
        <InspectorsPage />
      </MemoryRouter>,
    );

    expect(await screen.findByText("No snapshots matched")).toBeTruthy();
    expect(screen.getByText("Select a snapshot")).toBeTruthy();
    expect(screen.getByText("No Dream reports matched")).toBeTruthy();
    expect(screen.getByText("No packet loaded")).toBeTruthy();
    expect(screen.getByText("No trace loaded")).toBeTruthy();
    expect(screen.getByText("No process trace loaded")).toBeTruthy();
    expect(apiMocks.listSnapshots).toHaveBeenCalledWith({
      correlationId: undefined,
      kind: undefined,
      laneId: undefined,
      limit: 60,
      query: undefined,
      workspaceId: undefined,
    });
  });
});
