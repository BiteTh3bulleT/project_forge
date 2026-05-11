import { render, screen } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { OperatorAppsPage } from "./OperatorAppsPage";

const desktop = vi.hoisted(() => ({
  isTauriDesktop: vi.fn(),
  listOperatorApps: vi.fn(),
  launchOperatorApp: vi.fn(),
}));

vi.mock("../lib/desktop", () => desktop);

describe("OperatorAppsPage", () => {
  beforeEach(() => {
    desktop.isTauriDesktop.mockReturnValue(false);
    desktop.listOperatorApps.mockResolvedValue([
      {
        id: "terminal",
        label: "Terminal",
        description: "Open a Foot terminal.",
        executable: "foot",
      },
      {
        id: "files",
        label: "Files",
        description: "Open the file manager.",
        executable: "pcmanfm",
      },
    ]);
    desktop.launchOperatorApp.mockReset();
  });

  it("renders allowlisted apps without a free-form command input", async () => {
    const { container } = render(<OperatorAppsPage />);

    expect(await screen.findByText("Terminal")).toBeTruthy();
    expect(await screen.findByText("Files")).toBeTruthy();
    expect(
      screen.getByText(/operator apps require the tauri desktop runtime/i),
    ).toBeTruthy();
    expect(container.querySelector("input")).toBeNull();
    expect(container.querySelector("textarea")).toBeNull();
    expect(screen.queryByLabelText(/command/i)).toBeNull();
    expect(screen.queryByPlaceholderText(/command|path/i)).toBeNull();
  });
});
