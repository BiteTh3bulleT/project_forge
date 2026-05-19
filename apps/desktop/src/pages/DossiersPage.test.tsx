import { render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { DossiersPage } from "./DossiersPage";

const apiMocks = vi.hoisted(() => ({
  createDossier: vi.fn(),
  dossierDetail: vi.fn(),
  generateBrief: vi.fn(),
  getDossierProfile: vi.fn(),
  listAutomationRules: vi.fn(),
  listDossiers: vi.fn(),
  listPresets: vi.fn(),
  listReviews: vi.fn(),
  listSources: vi.fn(),
  listStrategies: vi.fn(),
  saveDossierProfile: vi.fn(),
  dossierView: vi.fn(),
  dossierVSASummary: vi.fn(),
}));

vi.mock("../lib/api", () => ({
  api: {
    automation: {
      listRules: apiMocks.listAutomationRules,
    },
    dossiers: {
      create: apiMocks.createDossier,
      detail: apiMocks.dossierDetail,
      generateBrief: apiMocks.generateBrief,
      list: apiMocks.listDossiers,
    },
    memory: {
      dossierView: apiMocks.dossierView,
      dossierVSASummary: apiMocks.dossierVSASummary,
    },
    policy: {
      getDossierProfile: apiMocks.getDossierProfile,
      listPresets: apiMocks.listPresets,
      saveDossierProfile: apiMocks.saveDossierProfile,
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

const dossier = {
  id: 7,
  createdAtMs: 1_800_000_000_000,
  updatedAtMs: 1_800_000_001_000,
  name: "Nullable dossier",
  description: "Live null collection shape",
  primaryPaths: [],
  relatedRepos: [],
  constraints: [],
  preferredAdapters: [],
  importantFiles: [],
  routingNotes: "",
};

describe("DossiersPage", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    apiMocks.createDossier.mockResolvedValue({ dossier });
    apiMocks.generateBrief.mockResolvedValue({});
    apiMocks.saveDossierProfile.mockResolvedValue({ profile: null });
    apiMocks.dossierVSASummary.mockResolvedValue({ summary: null });
  });

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

  it("renders when list endpoints serialize empty arrays as null", async () => {
    apiMocks.listDossiers.mockResolvedValue({ dossiers: null });
    apiMocks.listSources.mockResolvedValue({ sources: null });
    apiMocks.listPresets.mockResolvedValue({ presets: null });
    apiMocks.listStrategies.mockResolvedValue({ strategies: null });
    apiMocks.listAutomationRules.mockResolvedValue({ rules: null });
    apiMocks.listReviews.mockResolvedValue({ reviews: null });

    render(
      <MemoryRouter initialEntries={["/dossiers"]}>
        <DossiersPage />
      </MemoryRouter>,
    );

    expect(await screen.findByText("No dossiers yet.")).toBeTruthy();
    expect(screen.queryByText(/Known sources:/)).toBeNull();
  });

  it("renders dossier detail when profile and memory collections are null", async () => {
    apiMocks.listDossiers.mockResolvedValue({ dossiers: [dossier] });
    apiMocks.listSources.mockResolvedValue({ sources: null });
    apiMocks.listPresets.mockResolvedValue({ presets: null });
    apiMocks.listStrategies.mockResolvedValue({ strategies: null });
    apiMocks.listAutomationRules.mockResolvedValue({ rules: null });
    apiMocks.listReviews.mockResolvedValue({ reviews: null });
    apiMocks.dossierDetail.mockResolvedValue({
      detail: {
        dossier,
        sources: null,
        recentJobs: null,
        briefs: null,
        vsaSummary: null,
      },
    });
    apiMocks.getDossierProfile.mockResolvedValue({
      profile: {
        dossierId: dossier.id,
        updatedAtMs: 1_800_000_002_000,
        preferredStrategies: null,
        preferredAdapters: null,
        approvalPresetId: null,
        retrievalDefaults: null,
        highValueFiles: null,
        noisyFiles: null,
        routingNotes: "",
        automationBindings: null,
      },
    });
    apiMocks.dossierView.mockResolvedValue({
      view: {
        dossierId: dossier.id,
        observationCount: 0,
        staleObservationCount: 0,
        recentObservations: null,
        recentSignals: null,
        recentAlignmentNotes: null,
        vsaSummary: null,
      },
    });

    render(
      <MemoryRouter initialEntries={["/dossiers"]}>
        <DossiersPage />
      </MemoryRouter>,
    );

    expect(await screen.findByText("Nullable dossier")).toBeTruthy();
    expect(await screen.findByText("No source links.")).toBeTruthy();
    expect(screen.getByText("No linked jobs.")).toBeTruthy();
    expect(screen.getByText("No brief snapshots yet.")).toBeTruthy();
    expect(screen.getByText("No observations yet.")).toBeTruthy();
    expect(screen.getByText("No packet alignment notes yet.")).toBeTruthy();
  });
});
