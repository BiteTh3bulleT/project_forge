import { render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import { AdaptersPage } from "./AdaptersPage";

const apiMocks = vi.hoisted(() => ({
  invoke: vi.fn(),
  list: vi.fn(),
}));

vi.mock("../lib/api", () => ({
  api: {
    adapters: {
      invoke: apiMocks.invoke,
      list: apiMocks.list,
    },
  },
}));

describe("AdaptersPage", () => {
  it("renders adapter rows returned by the API", async () => {
    apiMocks.list.mockResolvedValueOnce({
      adapters: [
        {
          id: "ollama",
          displayName: "Ollama",
          status: "ready",
          detail: "Local model adapter",
          capabilities: ["status"],
          config: { endpoint: "local" },
        },
      ],
    });

    render(<AdaptersPage />);

    expect(await screen.findByText("Ollama")).toBeTruthy();
    expect(screen.getByText("Local model adapter")).toBeTruthy();
    expect(apiMocks.list).toHaveBeenCalledOnce();
  });
});
