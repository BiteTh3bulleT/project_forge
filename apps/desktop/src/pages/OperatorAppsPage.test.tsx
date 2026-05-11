import { render, screen } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { OperatorAppsPage } from "./OperatorAppsPage";

const desktop = vi.hoisted(() => ({
  isTauriDesktop: vi.fn(),
  listOperatorApps: vi.fn(),
  launchOperatorApp: vi.fn(),
  iconAssetUrl: vi.fn((path: string) => `asset://${path}`),
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
        category: "Workspace",
        iconName: "foot",
        iconPath: "/run/current-system/sw/share/icons/hicolor/48x48/apps/foot.png",
        desktopFile: "/run/current-system/sw/share/applications/foot.desktop",
        native: true,
      },
      {
        id: "files",
        label: "Files",
        description: "Open the file manager.",
        executable: "pcmanfm",
        category: "Workspace",
        iconName: "system-file-manager",
        iconPath: "/run/current-system/sw/share/icons/hicolor/48x48/apps/system-file-manager.png",
        desktopFile: "/run/current-system/sw/share/applications/pcmanfm.desktop",
        native: true,
      },
      {
        id: "browser",
        label: "Browser",
        description: "Open docs and web consoles.",
        executable: "firefox",
        category: "Internet",
        iconName: "firefox",
        iconPath: null,
        desktopFile: "/run/current-system/sw/share/applications/firefox.desktop",
        native: true,
      },
      {
        id: "ollama-status",
        label: "Ollama Status",
        description: "Show local Ollama status.",
        executable: "foot",
        category: "AI Runtime",
        iconName: "utilities-terminal",
        iconPath: null,
        desktopFile: null,
        native: false,
      },
      {
        id: "system-monitor",
        label: "System Monitor",
        description: "Open btop.",
        executable: "foot",
        category: "System",
        iconName: "utilities-system-monitor",
        iconPath: null,
        desktopFile: null,
        native: false,
      },
      {
        id: "lazygit",
        label: "Git UI",
        description: "Open lazygit.",
        executable: "foot",
        category: "Developer",
        iconName: "git",
        iconPath: null,
        desktopFile: null,
        native: false,
      },
      {
        id: "core-logs",
        label: "Core Logs",
        description: "Show forge-core logs.",
        executable: "foot",
        category: "FORGE",
        iconName: "text-x-log",
        iconPath: null,
        desktopFile: null,
        native: false,
      },
    ]);
    desktop.launchOperatorApp.mockReset();
    desktop.iconAssetUrl.mockClear();
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

  it("renders native desktop metadata and categories for operator apps", async () => {
    render(<OperatorAppsPage />);

    expect(await screen.findByText("Workspace")).toBeTruthy();
    expect(screen.getByText("Internet")).toBeTruthy();
    expect(screen.getByText("AI Runtime")).toBeTruthy();
    expect(screen.getByText("System")).toBeTruthy();
    expect(screen.getByText("Developer")).toBeTruthy();
    expect(screen.getByText("FORGE")).toBeTruthy();
    expect(screen.getAllByText("foot").length).toBeGreaterThan(0);
    expect(screen.getAllByText("Native")).toHaveLength(3);
    expect(screen.getByRole("img", { name: "Terminal icon" })).toBeTruthy();
  });
});
