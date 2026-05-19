import { render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import { StrategiesPage } from "./StrategiesPage";

const apiMocks = vi.hoisted(() => ({
  listPresets: vi.fn(),
  listStrategies: vi.fn(),
  saveStrategy: vi.fn(),
}));

vi.mock("../lib/api", () => ({
  api: {
    policy: {
      listPresets: apiMocks.listPresets,
    },
    strategies: {
      list: apiMocks.listStrategies,
      save: apiMocks.saveStrategy,
    },
  },
}));

describe("StrategiesPage", () => {
  it("renders an empty strategy inventory", async () => {
    apiMocks.listStrategies.mockResolvedValueOnce({ strategies: [] });
    apiMocks.listPresets.mockResolvedValueOnce({ presets: [] });

    render(<StrategiesPage />);

    expect(await screen.findByText("No strategies found.")).toBeTruthy();
    expect(apiMocks.listStrategies).toHaveBeenCalledWith({ limit: 240 });
    expect(apiMocks.listPresets).toHaveBeenCalledWith(80);
  });
});
