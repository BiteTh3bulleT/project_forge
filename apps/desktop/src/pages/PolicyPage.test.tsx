import { render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import { PolicyPage } from "./PolicyPage";

const apiMocks = vi.hoisted(() => ({
  getGlobalPreset: vi.fn(),
  listDossiers: vi.fn(),
  listGuidance: vi.fn(),
  listPresets: vi.fn(),
  listRecommendations: vi.fn(),
  listStrategies: vi.fn(),
  recommend: vi.fn(),
  savePreset: vi.fn(),
  setGlobalPreset: vi.fn(),
}));

vi.mock("../lib/api", () => ({
  api: {
    dossiers: {
      list: apiMocks.listDossiers,
    },
    packetGuidance: {
      analyze: vi.fn(),
      list: apiMocks.listGuidance,
    },
    policy: {
      getGlobalPreset: apiMocks.getGlobalPreset,
      listPresets: apiMocks.listPresets,
      listRecommendations: apiMocks.listRecommendations,
      recommend: apiMocks.recommend,
      savePreset: apiMocks.savePreset,
      setGlobalPreset: apiMocks.setGlobalPreset,
    },
    strategies: {
      list: apiMocks.listStrategies,
    },
  },
}));

describe("PolicyPage", () => {
  it("renders empty policy board states", async () => {
    apiMocks.listPresets.mockResolvedValueOnce({ presets: [] });
    apiMocks.getGlobalPreset.mockResolvedValueOnce({ presetId: "" });
    apiMocks.listRecommendations.mockResolvedValueOnce({
      recommendations: [],
    });
    apiMocks.listStrategies.mockResolvedValueOnce({ strategies: [] });
    apiMocks.listDossiers.mockResolvedValueOnce({ dossiers: [] });
    apiMocks.listGuidance.mockResolvedValueOnce({ guidance: [] });

    render(<PolicyPage />);

    expect(await screen.findByText("No approval presets")).toBeTruthy();
    expect(screen.getByText("No recommendations recorded")).toBeTruthy();
    expect(screen.getByText("No packet guidance records")).toBeTruthy();
    expect(apiMocks.listPresets).toHaveBeenCalledWith(80);
    expect(apiMocks.getGlobalPreset).toHaveBeenCalledOnce();
    expect(apiMocks.listRecommendations).toHaveBeenCalledWith({ limit: 120 });
    expect(apiMocks.listStrategies).toHaveBeenCalledWith({ limit: 240 });
    expect(apiMocks.listDossiers).toHaveBeenCalledWith(180);
    expect(apiMocks.listGuidance).toHaveBeenCalledWith({ limit: 120 });
  });
});
