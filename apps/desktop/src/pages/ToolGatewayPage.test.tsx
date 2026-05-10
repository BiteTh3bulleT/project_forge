import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { ToolGatewayPage } from "./ToolGatewayPage";

const mocks = vi.hoisted(() => ({
  tools: vi.fn(),
  capabilities: vi.fn(),
  invocations: vi.fn(),
  invoke: vi.fn(),
  updateCapabilityStatus: vi.fn(),
  lanes: vi.fn(),
  meta: vi.fn(),
}));

vi.mock("../lib/api", () => ({
  api: {
    gateway: {
      tools: mocks.tools,
      capabilities: mocks.capabilities,
      invocations: mocks.invocations,
      invoke: mocks.invoke,
      updateCapabilityStatus: mocks.updateCapabilityStatus,
    },
    actionLanes: {
      list: mocks.lanes,
    },
    meta: mocks.meta,
  },
}));

const capability = {
  id: "cap.files.read",
  domain: "filesystem",
  name: "read",
  description: "Read files through governed gateway.",
  status: "active",
  lane: "io",
  effect: ["filesystem_read"],
  risk: "read_only",
  requiresWorkspace: true,
  requiresIntent: false,
  requiresApprovalByDefault: false,
  autonomyEligible: false,
  allowedInDryRun: true,
};

const tool = {
  id: "filesystem.read_file",
  domain: "filesystem",
  action: "read_file",
  description: "Read a file.",
  riskClass: "read_only",
  executionLevel: "L0",
  executes: false,
  usesNetwork: false,
  writeIntent: false,
  capabilityId: "cap.files.read",
  capabilityStatus: "active",
  capabilityRisk: "low",
};

const lane = {
  id: "fs.read",
  name: "Read files",
  enabled: true,
  actionType: "read_file",
  riskClass: "read_only",
  writeIntent: false,
  requiresApproval: false,
};

describe("ToolGatewayPage capability status drafts", () => {
  beforeEach(() => {
    for (const mock of Object.values(mocks)) {
      mock.mockReset();
    }
    mocks.tools.mockResolvedValue({ tools: [] });
    mocks.capabilities.mockResolvedValue({ capabilities: [capability] });
    mocks.lanes.mockResolvedValue({ lanes: [] });
    mocks.meta.mockResolvedValue({ workspaceDir: "/workspace" });
    mocks.invocations.mockResolvedValue({ invocations: [] });
  });

  it("resets stale capability status drafts after refresh and blocks no-op submissions", async () => {
    render(
      <MemoryRouter>
        <ToolGatewayPage />
      </MemoryRouter>,
    );

    const statusSelect = (await screen.findByDisplayValue(
      "active",
    )) as HTMLSelectElement;
    fireEvent.change(statusSelect, { target: { value: "disabled" } });
    expect(statusSelect.value).toBe("disabled");

    fireEvent.click(screen.getByRole("button", { name: "Refresh" }));

    await waitFor(() => expect(mocks.capabilities).toHaveBeenCalledTimes(2));
    await waitFor(() => {
      expect((screen.getByDisplayValue("active") as HTMLSelectElement).value)
        .toBe("active");
    });

    const apply = screen.getByRole<HTMLButtonElement>("button", {
      name: "Apply status",
    });
    expect(apply.disabled).toBe(true);
    fireEvent.click(apply);
    expect(mocks.updateCapabilityStatus).not.toHaveBeenCalled();
  });

  it("blocks gateway execution when refreshed authority data is stale", async () => {
    mocks.tools
      .mockResolvedValueOnce({ tools: [tool] })
      .mockRejectedValueOnce(new Error("gateway down"));
    mocks.capabilities.mockResolvedValue({ capabilities: [capability] });
    mocks.lanes.mockResolvedValue({ lanes: [lane] });

    render(
      <MemoryRouter>
        <ToolGatewayPage />
      </MemoryRouter>,
    );

    const execute = await screen.findByRole<HTMLButtonElement>("button", {
      name: "Execute",
    });
    expect(execute.disabled).toBe(false);

    fireEvent.click(screen.getByRole("button", { name: "Refresh" }));
    await screen.findByText(/gateway authority stale/i);

    const staleExecute = screen.getByRole<HTMLButtonElement>("button", {
      name: "Blocked by preflight",
    });
    expect(staleExecute.disabled).toBe(true);
    fireEvent.click(staleExecute);
    expect(mocks.invoke).not.toHaveBeenCalled();
  });
});
