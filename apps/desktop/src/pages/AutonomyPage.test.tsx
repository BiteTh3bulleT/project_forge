import { render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import { AutonomyPage } from "./AutonomyPage";

const apiMocks = vi.hoisted(() => ({
  budgets: vi.fn(),
  charters: vi.fn(),
  decisions: vi.fn(),
  events: vi.fn(),
  explainIntent: vi.fn(),
  intents: vi.fn(),
  status: vi.fn(),
}));

vi.mock("../lib/api", () => ({
  api: {
    autonomy: {
      budgets: apiMocks.budgets,
      charters: apiMocks.charters,
      decisions: apiMocks.decisions,
      events: apiMocks.events,
      explainIntent: apiMocks.explainIntent,
      intents: apiMocks.intents,
      status: apiMocks.status,
    },
  },
}));

describe("AutonomyPage", () => {
  it("renders inactive autonomy and empty evidence states", async () => {
    apiMocks.status.mockResolvedValueOnce({
      available: false,
      reason: "disabled by configuration",
    });
    apiMocks.intents.mockResolvedValueOnce({ intents: [] });
    apiMocks.decisions.mockResolvedValueOnce({ decisions: [] });
    apiMocks.budgets.mockResolvedValueOnce({ budgets: [] });
    apiMocks.charters.mockResolvedValueOnce({ charters: [] });
    apiMocks.events.mockResolvedValueOnce({ events: [] });

    render(<AutonomyPage />);

    expect(await screen.findByText(/Autonomy loop is not active/)).toBeTruthy();
    expect(screen.getByText("No intents recorded.")).toBeTruthy();
    expect(screen.getByText("No decisions recorded.")).toBeTruthy();
    expect(screen.getByText("No autonomy events yet.")).toBeTruthy();
    expect(screen.getByText("No budgets loaded.")).toBeTruthy();
    expect(screen.getByText("No charters loaded.")).toBeTruthy();
    expect(apiMocks.status).toHaveBeenCalledOnce();
    expect(apiMocks.intents).toHaveBeenCalledWith({ limit: 40 });
    expect(apiMocks.decisions).toHaveBeenCalledWith(40);
    expect(apiMocks.budgets).toHaveBeenCalledOnce();
    expect(apiMocks.charters).toHaveBeenCalledWith(false);
    expect(apiMocks.events).toHaveBeenCalledWith(60);
  });
});
