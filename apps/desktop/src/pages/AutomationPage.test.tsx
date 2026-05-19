import { fireEvent, render, screen } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { AutomationPage } from "./AutomationPage";

const apiMocks = vi.hoisted(() => ({
  history: vi.fn(),
  listRules: vi.fn(),
  runRule: vi.fn(),
  saveRule: vi.fn(),
}));

vi.mock("../lib/api", () => ({
  api: {
    automation: {
      history: apiMocks.history,
      listRules: apiMocks.listRules,
      runRule: apiMocks.runRule,
      saveRule: apiMocks.saveRule,
    },
  },
}));

const rule = {
  id: 5,
  createdAtMs: 1,
  updatedAtMs: 2,
  name: "Review import",
  trigger: "import.execution.created",
  condition: { always: true },
  action: { type: "create_review" },
  scope: {},
  enabled: true,
  dryRunDefault: true,
};

describe("AutomationPage", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("renders empty automation rule and history states", async () => {
    apiMocks.listRules.mockResolvedValueOnce({ rules: [] });
    apiMocks.history.mockResolvedValueOnce({ history: [] });

    render(<AutomationPage />);

    expect(await screen.findByText("No automation rules yet.")).toBeTruthy();
    expect(screen.getByText("No history rows yet.")).toBeTruthy();
    expect(apiMocks.listRules).toHaveBeenCalledWith({ limit: 200 });
    expect(apiMocks.history).toHaveBeenCalledWith(150);
  });

  it("renders selected rule run errors", async () => {
    apiMocks.listRules.mockResolvedValue({ rules: [rule] });
    apiMocks.history.mockResolvedValue({ history: [] });
    apiMocks.runRule.mockRejectedValueOnce(new Error("automation run denied"));

    render(<AutomationPage />);

    fireEvent.click(
      await screen.findByRole("button", { name: "Run Selected Rule" }),
    );

    expect(await screen.findByText("automation run denied")).toBeTruthy();
  });
});
