import { render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { describe, expect, it, vi } from "vitest";

import { DossiersPage } from "./DossiersPage";

const apiMocks = vi.hoisted(() => ({
  createDossier: vi.fn(),
  listAutomationRules: vi.fn(),
  listDossiers: vi.fn(),
  listPresets: vi.fn(),
  listReviews: vi.fn(),
  listSources: vi.fn(),
  listStrategies: vi.fn(),
}));

vi.mock("../lib/api", () => ({
  api: {
    automation: {
      listRules: apiMocks.listAutomationRules,
    },
    dossiers: {
      create: apiMocks.createDossier,
      list: apiMocks.listDossiers,
    },
    policy: {
      listPresets: apiMocks.listPresets,
    },
    reviews: {
      list: apiMocks.listReviews,
    },
    sources: {
      list: apiMocks.listSources,
    },
    strategies: {
      list: apiMocks.listStrategies,
    },
  },
}));

describe("DossiersPage", () => {
  it("renders empty dossier list and detail states", async () => {
    apiMocks.listDossiers.mockResolvedValueOnce({ dossiers: [] });
    apiMocks.listSources.mockResolvedValueOnce({ sources: [] });
    apiMocks.listPresets.mockResolvedValueOnce({ presets: [] });
    apiMocks.listStrategies.mockResolvedValueOnce({ strategies: [] });
    apiMocks.listAutomationRules.mockResolvedValueOnce({ rules: [] });
    apiMocks.listReviews.mockResolvedValueOnce({ reviews: [] });

    render(
      <MemoryRouter>
        <DossiersPage />
      </MemoryRouter>,
    );

    expect(await screen.findByText("No dossiers yet.")).toBeTruthy();
    expect(screen.getByText("Select a dossier to inspect details.")).toBeTruthy();
    expect(screen.getByText("Select a dossier first.")).toBeTruthy();
    expect(apiMocks.listDossiers).toHaveBeenCalledWith(180);
    expect(apiMocks.listSources).toHaveBeenCalledOnce();
    expect(apiMocks.listPresets).toHaveBeenCalledWith(80);
    expect(apiMocks.listStrategies).toHaveBeenCalledWith({ limit: 220 });
    expect(apiMocks.listAutomationRules).toHaveBeenCalledWith({ limit: 220 });
    expect(apiMocks.listReviews).toHaveBeenCalledWith({ limit: 260 });
  });
});
