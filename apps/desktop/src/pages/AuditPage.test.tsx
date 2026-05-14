import { render, screen, waitFor } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { AuditPage } from "./AuditPage";

const mocks = vi.hoisted(() => ({
  list: vi.fn(),
  trace: vi.fn(),
}));

vi.mock("../lib/api", () => ({
  api: {
    audit: {
      list: mocks.list,
      trace: mocks.trace,
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
    mocks.list.mockResolvedValue({ records: [record] });
    mocks.trace.mockResolvedValue({ records: [record] });
  });

  it("loads the correlation trace from the correlation id URL parameter", async () => {
    render(
      <MemoryRouter
        initialEntries={["/audit?correlationId=corr-from-url"]}
        future={{ v7_relativeSplatPath: true, v7_startTransition: true }}
      >
        <AuditPage />
      </MemoryRouter>,
    );

    await waitFor(() => {
      expect(mocks.trace).toHaveBeenCalledWith("corr-from-url");
    });
    expect(await screen.findByText("1 events")).toBeTruthy();
  });
});
