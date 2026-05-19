import { fireEvent, render, screen } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { PolicyPage } from "./PolicyPage";

const mocks = vi.hoisted(() => ({
  listPresets: vi.fn(),
  getGlobalPreset: vi.fn(),
  setGlobalPreset: vi.fn(),
  listRecommendations: vi.fn(),
  recommend: vi.fn(),
  savePreset: vi.fn(),
  strategiesList: vi.fn(),
  dossiersList: vi.fn(),
  packetGuidanceList: vi.fn(),
  packetGuidanceAnalyze: vi.fn(),
}));

vi.mock("../lib/api", () => ({
  api: {
    policy: {
      listPresets: mocks.listPresets,
      getGlobalPreset: mocks.getGlobalPreset,
      setGlobalPreset: mocks.setGlobalPreset,
      listRecommendations: mocks.listRecommendations,
      recommend: mocks.recommend,
      savePreset: mocks.savePreset,
    },
    strategies: {
      list: mocks.strategiesList,
    },
    dossiers: {
      list: mocks.dossiersList,
    },
    packetGuidance: {
      list: mocks.packetGuidanceList,
      analyze: mocks.packetGuidanceAnalyze,
    },
  },
}));

describe("PolicyPage mutation errors", () => {
  beforeEach(() => {
    for (const mock of Object.values(mocks)) {
      mock.mockReset();
    }
    mocks.listPresets.mockResolvedValue({ presets: [] });
    mocks.getGlobalPreset.mockResolvedValue({ presetId: "" });
    mocks.listRecommendations.mockResolvedValue({ recommendations: [] });
    mocks.strategiesList.mockResolvedValue({ strategies: [] });
    mocks.dossiersList.mockResolvedValue({ dossiers: [] });
    mocks.packetGuidanceList.mockResolvedValue({ guidance: [] });
  });

  it("renders global preset save errors", async () => {
    mocks.setGlobalPreset.mockRejectedValueOnce(
      new Error("global preset write denied"),
    );

    render(<PolicyPage />);

    fireEvent.click(
      await screen.findByRole("button", { name: "Apply Global Preset" }),
    );

    expect(await screen.findByText("global preset write denied")).toBeTruthy();
  });

  it("renders recommendation errors", async () => {
    mocks.recommend.mockRejectedValueOnce(new Error("recommendation failed"));

    render(<PolicyPage />);

    fireEvent.click(
      await screen.findByRole("button", { name: "Generate Recommendation" }),
    );

    expect(await screen.findByText("recommendation failed")).toBeTruthy();
  });
});
