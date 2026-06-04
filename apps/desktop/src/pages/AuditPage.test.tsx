import { render, screen, waitFor } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { AuditPage } from "./AuditPage";

const mocks = vi.hoisted(() => ({
  list: vi.fn(),
  trace: vi.fn(),
  lookup: vi.fn(),
}));

vi.mock("../lib/api", () => ({
  api: {
    audit: {
      list: mocks.list,
      trace: mocks.trace,
      lookup: mocks.lookup,
    },
  },
}));

const record = {
  id: 1,
  createdAtMs: 1_700_000_000_000,
  correlationId: "corr-from-url",
  category: "gateway",
  action: "invoke",
  actor: "operator",
  subjectType: "tool",
  subjectId: "filesystem.read_file",
  riskClass: "read_only",
  outcome: "success",
  summary: "Read file",
  payload: {},
};

describe("AuditPage correlation trace routing", () => {
  beforeEach(() => {
    mocks.list.mockReset();
    mocks.trace.mockReset();
    mocks.lookup.mockReset();
    mocks.list.mockResolvedValue({ records: [record] });
    mocks.trace.mockResolvedValue({ records: [record] });
    mocks.lookup.mockResolvedValue({
      mode: "correlation",
      correlationId: "corr-from-url",
      records: [record],
      report: {
        correlationId: "corr-from-url",
        gatewayInvocations: [
          {
            id: 42,
            toolId: "filesystem.read_file",
            laneId: "filesystem.read_file",
            status: "success",
          },
        ],
        auditRecords: [record],
        artifactRecords: [{ id: 7, path: "artifact://readme" }],
        gatewayArtifactRefs: [{ gatewayInvocationId: 42, path: "README.md" }],
        provenanceRecords: [{ id: "prov-1", auditId: "1" }],
        journalEvents: [{ id: "journal-1", provenanceId: "prov-1" }],
        artifactRefs: [{ id: "artifact-ref-1", provenanceId: "prov-1" }],
        links: {
          auditToGateway: [{ auditRecordId: 1, gatewayInvocationId: 42 }],
          auditToArtifact: [{ auditRecordId: 1, artifactId: 7 }],
          provenanceToAudit: [{ provenanceId: "prov-1", auditRecordId: 1 }],
          journalToProvenance: [
            { journalEventId: "journal-1", provenanceId: "prov-1" },
          ],
          artifactRefToProvenance: [
            { artifactRefId: "artifact-ref-1", provenanceId: "prov-1" },
          ],
          gatewayToArtifact: [
            { gatewayInvocationId: 42, artifactId: 7, path: "README.md" },
          ],
        },
      },
      reports: [],
    });
  });

  it("loads the authority trace report from the correlation id URL parameter", async () => {
    render(
      <MemoryRouter
        initialEntries={["/audit?correlationId=corr-from-url"]}
        future={{ v7_relativeSplatPath: true, v7_startTransition: true }}
      >
        <AuditPage />
      </MemoryRouter>,
    );

    await waitFor(() => {
      expect(mocks.lookup).toHaveBeenCalledWith({
        correlationId: "corr-from-url",
      });
    });
    expect(await screen.findByText("1 events")).toBeTruthy();
    expect(await screen.findByText("Authority Chain")).toBeTruthy();
    expect(screen.getByText("Gateway Invocations")).toBeTruthy();
    expect(screen.getByText("Journal Events")).toBeTruthy();
    expect(screen.getByText("6 linked edges")).toBeTruthy();
  });
});
