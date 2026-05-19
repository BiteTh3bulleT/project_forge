import { render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

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

describe("AutomationPage", () => {
  it("renders empty automation rule and history states", async () => {
    apiMocks.listRules.mockResolvedValueOnce({ rules: [] });
    apiMocks.history.mockResolvedValueOnce({ history: [] });

    render(<AutomationPage />);

    expect(await screen.findByText("No automation rules yet.")).toBeTruthy();
    expect(screen.getByText("No history rows yet.")).toBeTruthy();
    expect(apiMocks.listRules).toHaveBeenCalledWith({ limit: 200 });
    expect(apiMocks.history).toHaveBeenCalledWith(150);
  });
});
