import { render, screen } from "@testing-library/react";
import { MemoryRouter, Route, Routes } from "react-router-dom";
import { describe, expect, it, vi } from "vitest";

import { MemoryDetailPage } from "./MemoryDetailPage";

const apiMocks = vi.hoisted(() => ({
  chunk: vi.fn(),
}));

vi.mock("../lib/api", () => ({
  api: {
    chunk: apiMocks.chunk,
  },
}));

describe("MemoryDetailPage", () => {
  it("renders chunk metadata and content loaded from the API", async () => {
    apiMocks.chunk.mockResolvedValueOnce({
      chunkId: 42,
      chunkIndex: 3,
      relPath: "docs/memory.md",
      absPath: "E:\\dev\\imrobman-dev\\project_forge\\docs\\memory.md",
      mtimeNs: 1_800_000_000_000_000_000,
      contentLength: 31,
      content: "Durable memory evidence sample.",
    });

    render(
      <MemoryRouter initialEntries={["/memory/chunks/42"]}>
        <Routes>
          <Route path="/memory/chunks/:id" element={<MemoryDetailPage />} />
        </Routes>
      </MemoryRouter>,
    );

    expect(await screen.findByText("docs/memory.md")).toBeTruthy();
    expect(screen.getByText("#3 / id 42")).toBeTruthy();
    expect(screen.getByText("Durable memory evidence sample.")).toBeTruthy();
    expect(apiMocks.chunk).toHaveBeenCalledWith(42);
  });
});
