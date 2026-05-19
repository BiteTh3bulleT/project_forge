import { render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { DossiersPage } from "./DossiersPage";

const mocks = vi.hoisted(() => ({
  dossiersList: vi.fn(),
  dossierDetail: vi.fn(),
  dossierCreate: vi.fn(),
  dossierGenerateBrief: vi.fn(),
  sourcesList: vi.fn(),
  listPresets: vi.fn(),
  getDossierProfile: vi.fn(),
  saveDossierProfile: vi.fn(),
  strategiesList: vi.fn(),
  listRules: vi.fn(),
  reviewsList: vi.fn(),
  dossierView: vi.fn(),
  dossierVSASummary: vi.fn(),
}));

vi.mock("../lib/api", () => ({
  api: {
    dossiers: {
      list: mocks.dossiersList,
      detail: mocks.dossierDetail,
      create: mocks.dossierCreate,
      generateBrief: mocks.dossierGenerateBrief,
    },
    sources: {
      list: mocks.sourcesList,
    },
    policy: {
      listPresets: mocks.listPresets,
      getDossierProfile: mocks.getDossierProfile,
      saveDossierProfile: mocks.saveDossierProfile,
    },
    strategies: {
      list: mocks.strategiesList,
    },
    automation: {
      listRules: mocks.listRules,
    },
    reviews: {
      list: mocks.reviewsList,
    },
    memory: {
      dossierView: mocks.dossierView,
      dossierVSASummary: mocks.dossierVSASummary,
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
    mocks.dossierCreate.mockResolvedValue({ dossier });
    mocks.dossierGenerateBrief.mockResolvedValue({});
    mocks.saveDossierProfile.mockResolvedValue({ profile: null });
    mocks.dossierVSASummary.mockResolvedValue({ summary: null });
  });

  it("renders when list endpoints serialize empty arrays as null", async () => {
    mocks.dossiersList.mockResolvedValue({ dossiers: null });
    mocks.sourcesList.mockResolvedValue({ sources: null });
    mocks.listPresets.mockResolvedValue({ presets: null });
    mocks.strategiesList.mockResolvedValue({ strategies: null });
    mocks.listRules.mockResolvedValue({ rules: null });
    mocks.reviewsList.mockResolvedValue({ reviews: null });

    render(
      <MemoryRouter initialEntries={["/dossiers"]}>
        <DossiersPage />
      </MemoryRouter>,
    );

    expect(await screen.findByText("No dossiers yet.")).toBeTruthy();
    expect(screen.queryByText(/Known sources:/)).toBeNull();
  });

  it("renders dossier detail when profile and memory collections are null", async () => {
    mocks.dossiersList.mockResolvedValue({ dossiers: [dossier] });
    mocks.sourcesList.mockResolvedValue({ sources: null });
    mocks.listPresets.mockResolvedValue({ presets: null });
    mocks.strategiesList.mockResolvedValue({ strategies: null });
    mocks.listRules.mockResolvedValue({ rules: null });
    mocks.reviewsList.mockResolvedValue({ reviews: null });
    mocks.dossierDetail.mockResolvedValue({
      detail: {
        dossier,
        sources: null,
        recentJobs: null,
        briefs: null,
        vsaSummary: null,
      },
    });
    mocks.getDossierProfile.mockResolvedValue({
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
    mocks.dossierView.mockResolvedValue({
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
