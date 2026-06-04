import type { ApprovalRequest } from "@forge/shared";
import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { ApprovalsPage } from "./ApprovalsPage";

const mocks = vi.hoisted(() => ({
  list: vi.fn(),
  approve: vi.fn(),
  deny: vi.fn(),
}));

vi.mock("../lib/api", () => ({
  api: {
    approvals: {
      list: mocks.list,
      approve: mocks.approve,
      deny: mocks.deny,
    },
  },
}));

const pendingApproval: ApprovalRequest = {
  id: 7,
  jobId: "job-approval-7",
  createdAtMs: Date.UTC(2026, 4, 10, 12, 0, 0),
  status: "pending",
  requestedAction: "filesystem.write",
  riskClass: "write_files",
  requestedAdapter: "gateway",
  writeIntent: true,
  scopeSnapshot: { workspace: "/workspace" },
  taskPacketId: null,
  requestSummary: "Write a governed file",
};

const approvedApproval: ApprovalRequest = {
  ...pendingApproval,
  id: 9,
  jobId: "job-approval-9",
  status: "resolved",
  requestSummary: "Approved governed file write",
  decision: {
    id: 91,
    requestId: 9,
    createdAtMs: Date.UTC(2026, 4, 10, 12, 5, 0),
    actor: "operator-a",
    decision: "approved",
    note: "approved after review",
  },
};

const deniedApproval: ApprovalRequest = {
  ...pendingApproval,
  id: 10,
  jobId: "job-approval-10",
  status: "resolved",
  requestSummary: "Denied governed file write",
  decision: {
    id: 101,
    requestId: 10,
    createdAtMs: Date.UTC(2026, 4, 10, 12, 6, 0),
    actor: "operator-b",
    decision: "denied",
    note: "denied after review",
  },
};

describe("ApprovalsPage decision controls", () => {
  beforeEach(() => {
    mocks.list.mockReset();
    mocks.approve.mockReset();
    mocks.deny.mockReset();
    mocks.list.mockResolvedValue({ approvals: [pendingApproval] });
  });

  it("prevents duplicate approval submissions while a row decision is in flight", async () => {
    let resolveApprove: (value: unknown) => void = () => {};
    mocks.approve.mockReturnValue(
      new Promise((resolve) => {
        resolveApprove = resolve;
      }),
    );

    render(
      <MemoryRouter>
        <ApprovalsPage />
      </MemoryRouter>,
    );

    const approve = await screen.findByRole<HTMLButtonElement>("button", {
      name: "Approve request 7",
    });
    expect(
      screen.getByRole("article", { name: "Request #7" }),
    ).toBeTruthy();
    const deny = screen.getByRole<HTMLButtonElement>("button", {
      name: "Deny request 7",
    });

    fireEvent.click(approve);
    fireEvent.click(approve);

    await waitFor(() => {
      expect(approve.disabled).toBe(true);
      expect(deny.disabled).toBe(true);
    });
    expect(mocks.approve).toHaveBeenCalledTimes(1);

    resolveApprove({ decision: { id: 1 } });
    await waitFor(() => expect(approve.disabled).toBe(false));
  });

  it("refreshes and explains stale already-resolved approvals", async () => {
    mocks.approve.mockRejectedValueOnce(new Error("approval request not pending"));

    render(
      <MemoryRouter>
        <ApprovalsPage />
      </MemoryRouter>,
    );

    fireEvent.click(
      await screen.findByRole("button", { name: "Approve request 7" }),
    );

    expect(
      await screen.findByText("Request 7 was already resolved."),
    ).toBeTruthy();
    expect(mocks.list).toHaveBeenCalledTimes(2);
  });

  it("renders decision API errors instead of leaving an unhandled failure", async () => {
    mocks.deny.mockRejectedValueOnce(new Error("gateway denied decision"));

    render(
      <MemoryRouter>
        <ApprovalsPage />
      </MemoryRouter>,
    );

    fireEvent.click(
      await screen.findByRole("button", { name: "Deny request 7" }),
    );

    expect(await screen.findByText("gateway denied decision")).toBeTruthy();
  });

  it("does not expose public approve controls for non-public approval requests", async () => {
    mocks.list.mockResolvedValue({
      approvals: [
        {
          ...pendingApproval,
          id: 8,
          requestedAction: "gateway.capability.status.update",
          riskClass: "high",
          scopeSnapshot: {
            capabilityId: "process.spawn_process",
            publicDecisionAllowed: false,
          },
          requestSummary: "Activate process.spawn_process capability",
        },
      ],
    });

    render(
      <MemoryRouter>
        <ApprovalsPage />
      </MemoryRouter>,
    );

    const nonPublic = await screen.findByRole<HTMLButtonElement>("button", {
      name: "Non-public approval request 8",
    });
    expect(nonPublic.disabled).toBe(true);
    expect(
      screen.getByText(
        "This request requires a non-public approval authority.",
      ),
    ).toBeTruthy();
    expect(
      screen.queryByRole("button", { name: "Approve request 8" }),
    ).toBeNull();

    const deny = screen.getByRole<HTMLButtonElement>("button", {
      name: "Deny request 8",
    });
    expect(deny.disabled).toBe(false);
    expect(mocks.approve).not.toHaveBeenCalled();
  });

  it("separates pending, recent resolved, and denied approval views", async () => {
    // Desktop-side evidence for the approval flow: a pending public request
    // exposes one-click controls, then resolved decision records remain visible.
    mocks.list.mockImplementation((status: string) => {
      if (status === "resolved") {
        return Promise.resolve({
          approvals: [approvedApproval, deniedApproval],
        });
      }
      return Promise.resolve({ approvals: [pendingApproval] });
    });

    render(
      <MemoryRouter>
        <ApprovalsPage />
      </MemoryRouter>,
    );

    expect(await screen.findByText("Write a governed file")).toBeTruthy();
    expect(
      screen.getByText("Public one-click decision allowed for this request."),
    ).toBeTruthy();
    expect(
      screen.getByRole("button", { name: "Approve request 7" }),
    ).toBeTruthy();
    expect(
      screen.getByRole("button", { name: "Deny request 7" }),
    ).toBeTruthy();

    fireEvent.click(screen.getByRole("button", { name: "Recent / Resolved" }));

    expect(await screen.findByText("Approved governed file write")).toBeTruthy();
    expect(screen.getByText("Denied governed file write")).toBeTruthy();
    expect(
      screen.queryByRole("button", { name: /Approve request/ }),
    ).toBeNull();

    fireEvent.click(screen.getByRole("button", { name: "Denied" }));

    expect(await screen.findByText("Denied governed file write")).toBeTruthy();
    expect(screen.queryByText("Approved governed file write")).toBeNull();
    expect(screen.getByText("denied after review")).toBeTruthy();
    expect(mocks.list).toHaveBeenLastCalledWith("resolved", 120);
  });

  it("exposes an Audit pivot for approval job context", async () => {
    render(
      <MemoryRouter>
        <ApprovalsPage />
      </MemoryRouter>,
    );

    const audit = await screen.findByRole("link", {
      name: "Audit job-approval-7",
    });
    expect(audit.getAttribute("href")).toBe(
      "/audit?jobId=job-approval-7",
    );
  });
});
