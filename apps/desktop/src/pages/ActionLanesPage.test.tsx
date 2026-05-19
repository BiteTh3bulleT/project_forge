import { render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import { ActionLanesPage } from "./ActionLanesPage";

const apiMocks = vi.hoisted(() => ({
  deleteLane: vi.fn(),
  list: vi.fn(),
  save: vi.fn(),
}));

vi.mock("../lib/api", () => ({
  api: {
    actionLanes: {
      delete: apiMocks.deleteLane,
      list: apiMocks.list,
      save: apiMocks.save,
    },
  },
}));

describe("ActionLanesPage", () => {
  it("renders lane rows returned by the API", async () => {
    apiMocks.list.mockResolvedValueOnce({
      lanes: [
        {
          id: "fs.read",
          createdAtMs: Date.UTC(2026, 4, 19, 12, 0, 0),
          updatedAtMs: Date.UTC(2026, 4, 19, 12, 0, 0),
          name: "Read files",
          description: "Read-only file access",
          actionType: "fs.read",
          allowedPaths: ["workspace"],
          forbiddenPaths: [],
          writeIntent: false,
          requiresApproval: false,
          riskClass: "read_only",
          maxBytes: 4096,
          expectedArtifacts: [],
          builtin: true,
          enabled: true,
        },
      ],
    });

    render(<ActionLanesPage />);

    expect(await screen.findByText("Read files")).toBeTruthy();
    expect(screen.getByText("Read-only file access")).toBeTruthy();
    expect(apiMocks.list).toHaveBeenCalledOnce();
  });
});
