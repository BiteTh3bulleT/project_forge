import { fireEvent, render, screen } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { AutomationPage } from "./AutomationPage";

const mocks = vi.hoisted(() => ({
  listRules: vi.fn(),
  history: vi.fn(),
  saveRule: vi.fn(),
  runRule: vi.fn(),
}));

vi.mock("../lib/api", () => ({
  api: {
    automation: {
      listRules: mocks.listRules,
      history: mocks.history,
      saveRule: mocks.saveRule,
      runRule: mocks.runRule,
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

describe("AutomationPage mutation errors", () => {
  beforeEach(() => {
    for (const mock of Object.values(mocks)) {
      mock.mockReset();
    }
    mocks.listRules.mockResolvedValue({ rules: [rule] });
    mocks.history.mockResolvedValue({ history: [] });
  });

  it("renders selected rule run errors", async () => {
    mocks.runRule.mockRejectedValueOnce(new Error("automation run denied"));

    render(<AutomationPage />);

    fireEvent.click(
      await screen.findByRole("button", { name: "Run Selected Rule" }),
    );

    expect(await screen.findByText("automation run denied")).toBeTruthy();
  });
});
