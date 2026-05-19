import { render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { describe, expect, it, vi } from "vitest";

import { SourcesPage } from "./SourcesPage";

const apiMocks = vi.hoisted(() => ({
  addSource: vi.fn(),
  commandExecute: vi.fn(),
  createJob: vi.fn(),
  deleteSource: vi.fn(),
  listSources: vi.fn(),
  meta: vi.fn(),
}));

vi.mock("../lib/api", () => ({
  api: {
    commands: {
      execute: apiMocks.commandExecute,
    },
    jobs: {
      create: apiMocks.createJob,
    },
    meta: apiMocks.meta,
    sources: {
      add: apiMocks.addSource,
      delete: apiMocks.deleteSource,
      list: apiMocks.listSources,
    },
  },
}));

describe("SourcesPage", () => {
  it("renders empty source state and workspace hint", async () => {
    apiMocks.listSources.mockResolvedValueOnce({ sources: [] });
    apiMocks.meta.mockResolvedValueOnce({
      workspaceDir: "E:\\dev\\imrobman-dev\\project_forge",
    });

    render(
      <MemoryRouter>
        <SourcesPage />
      </MemoryRouter>,
    );

    expect(
      await screen.findByText("No sources yet. Add a folder to begin ingestion."),
    ).toBeTruthy();
    expect(screen.getByText(/Workspace root:/).textContent).toContain(
      "project_forge",
    );
    expect(apiMocks.listSources).toHaveBeenCalledOnce();
    expect(apiMocks.meta).toHaveBeenCalledOnce();
  });
});
