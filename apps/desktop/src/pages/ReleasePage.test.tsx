import { render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import { ReleasePage } from "./ReleasePage";

const apiMocks = vi.hoisted(() => ({
  artifacts: vi.fn(),
  firstRun: vi.fn(),
  readiness: vi.fn(),
  recordArtifact: vi.fn(),
}));

vi.mock("../lib/api", () => ({
  api: {
    release: {
      artifacts: apiMocks.artifacts,
      firstRun: apiMocks.firstRun,
      readiness: apiMocks.readiness,
      recordArtifact: apiMocks.recordArtifact,
    },
  },
}));

describe("ReleasePage", () => {
  it("renders release readiness, first-run data, and artifacts", async () => {
    apiMocks.readiness.mockResolvedValueOnce({
      checklist: {
        ready: true,
        items: [
          {
            id: "filesystem",
            title: "Filesystem ready",
            status: "ok",
            detail: "Paths initialized",
            category: "storage",
          },
        ],
      },
    });
    apiMocks.firstRun.mockResolvedValueOnce({
      firstRun: { completed: true, operator: "local" },
    });
    apiMocks.artifacts.mockResolvedValueOnce({
      artifacts: [{ kind: "desktop_bundle", versionTag: "local" }],
    });

    render(<ReleasePage />);

    expect(await screen.findByText("Filesystem ready")).toBeTruthy();
    expect(screen.getByText("READY (reported)")).toBeTruthy();
    expect(screen.getByText("desktop_bundle")).toBeTruthy();
    expect(apiMocks.artifacts).toHaveBeenCalledWith(40);
  });
});
