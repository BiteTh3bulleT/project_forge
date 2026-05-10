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
      name: "Approve",
    });
    const deny = screen.getByRole<HTMLButtonElement>("button", {
      name: "Deny",
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

    fireEvent.click(await screen.findByRole("button", { name: "Approve" }));

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

    fireEvent.click(await screen.findByRole("button", { name: "Deny" }));

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
      name: "Non-public approval",
    });
    expect(nonPublic.disabled).toBe(true);
    expect(
      screen.getByText(
        "This request requires a non-public approval authority.",
      ),
    ).toBeTruthy();
    expect(screen.queryByRole("button", { name: "Approve" })).toBeNull();

    const deny = screen.getByRole<HTMLButtonElement>("button", {
      name: "Deny",
    });
    expect(deny.disabled).toBe(false);
    expect(mocks.approve).not.toHaveBeenCalled();
  });
});
