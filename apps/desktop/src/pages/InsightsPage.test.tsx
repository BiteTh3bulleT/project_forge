import { render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import { InsightsPage } from "./InsightsPage";

const apiMocks = vi.hoisted(() => ({
  createImport: vi.fn(),
  embeddingStatus: vi.fn(),
  generateInsights: vi.fn(),
  listImports: vi.fn(),
  listInsights: vi.fn(),
  reembed: vi.fn(),
}));

vi.mock("../lib/api", () => ({
  api: {
    embeddings: {
      reembed: apiMocks.reembed,
      status: apiMocks.embeddingStatus,
    },
    imports: {
      create: apiMocks.createImport,
      list: apiMocks.listImports,
    },
    insights: {
      generate: apiMocks.generateInsights,
      list: apiMocks.listInsights,
    },
  },
}));

describe("InsightsPage", () => {
  it("renders empty insight, import, and embedding states", async () => {
    apiMocks.listInsights.mockResolvedValueOnce({ insights: [] });
    apiMocks.listImports.mockResolvedValueOnce({ imports: [] });
    apiMocks.embeddingStatus.mockResolvedValueOnce({ status: [] });

    render(<InsightsPage />);

    expect(await screen.findByText("No insights stored.")).toBeTruthy();
    expect(screen.getByText("No imports yet.")).toBeTruthy();
    expect(screen.getByText("No source embedding rows yet.")).toBeTruthy();
    expect(apiMocks.listInsights).toHaveBeenCalledWith(120, undefined);
    expect(apiMocks.listImports).toHaveBeenCalledWith(100, undefined);
    expect(apiMocks.embeddingStatus).toHaveBeenCalledOnce();
  });
});
