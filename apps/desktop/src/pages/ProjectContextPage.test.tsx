import { render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import { ProjectContextPage } from "./ProjectContextPage";

const apiMocks = vi.hoisted(() => ({
  get: vi.fn(),
  importContext: vi.fn(),
  regenerate: vi.fn(),
}));

vi.mock("../lib/api", () => ({
  api: {
    projectContext: {
      get: apiMocks.get,
      import: apiMocks.importContext,
      regenerate: apiMocks.regenerate,
    },
  },
}));

describe("ProjectContextPage", () => {
  it("renders the empty project context state", async () => {
    apiMocks.get.mockResolvedValueOnce({ record: null });

    render(<ProjectContextPage />);

    expect(await screen.findByText("No normalized context")).toBeTruthy();
    expect(screen.getByText("No project context record exists yet.")).toBeTruthy();
    expect(apiMocks.get).toHaveBeenCalledOnce();
  });
});
