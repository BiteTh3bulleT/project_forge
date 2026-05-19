import { render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import { BackupPage } from "./BackupPage";

const apiMocks = vi.hoisted(() => ({
  bundles: vi.fn(),
  createBundle: vi.fn(),
  deleteBundle: vi.fn(),
  restore: vi.fn(),
}));

vi.mock("../lib/api", () => ({
  api: {
    backup: {
      bundles: apiMocks.bundles,
      createBundle: apiMocks.createBundle,
      deleteBundle: apiMocks.deleteBundle,
      restore: apiMocks.restore,
    },
  },
}));

describe("BackupPage", () => {
  it("renders backup directories and bundle kind options", async () => {
    apiMocks.bundles.mockResolvedValueOnce({
      bundles: [],
      backupDir: "E:/forge/backups",
      exportDir: "E:/forge/exports",
      knownKinds: ["portable_snapshot"],
    });

    render(<BackupPage />);

    expect(await screen.findByText("E:/forge/backups")).toBeTruthy();
    expect(screen.getByText("E:/forge/exports")).toBeTruthy();
    expect(screen.getByRole("option", { name: "portable_snapshot" })).toBeTruthy();
    expect(apiMocks.bundles).toHaveBeenCalledWith(80);
  });
});
