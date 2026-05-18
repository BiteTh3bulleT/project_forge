import {
  cleanup,
  fireEvent,
  render,
  screen,
  within,
  waitFor,
} from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { useNavigate } from "react-router-dom";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { useDesktopWindowStore } from "../stores/desktopWindowStore";
import { useWorkspaceLayoutStore } from "../stores/workspaceLayoutStore";
import type { LinuxWindowSnapshot, OperatorApp } from "../lib/desktop";

import { AppShell } from "./AppShell";

function SearchChangeHarness() {
  const navigate = useNavigate();
  return (
    <>
      <button type="button" onClick={() => navigate("/?boardId=1")}>
        Change search
      </button>
      <AppShell isMainWindow={true}>
        <div />
      </AppShell>
    </>
  );
}

const desktopMocks = vi.hoisted(() => ({
  closeTauriWindow: vi.fn(() => Promise.resolve(true)),
  createShellWindow: vi.fn(() => Promise.resolve({ label: "forge-app-chat" })),
  focusTauriWindow: vi.fn(() => Promise.resolve(true)),
  isTauriDesktop: vi.fn(() => true),
  listForgeWindows: vi.fn<() => Promise<unknown[]>>(
    () => new Promise(() => {}),
  ),
  listLinuxWindows: vi.fn<() => Promise<LinuxWindowSnapshot[]>>(
    () => new Promise(() => {}),
  ),
  focusLinuxWindow: vi.fn(() => Promise.resolve(true)),
  controlLinuxWindow: vi.fn(() => Promise.resolve(true)),
  minimizeTauriWindow: vi.fn(() => Promise.resolve(true)),
  listOperatorApps: vi.fn<() => Promise<OperatorApp[]>>(
    () => new Promise(() => {}),
  ),
  launchOperatorApp: vi.fn(),
  requestHostPowerAction: vi.fn(),
  iconAssetUrl: vi.fn((path: string) => `asset://${path}`),
}));

function resolvedOperatorApps() {
  desktopMocks.listOperatorApps.mockResolvedValue([
    {
      id: "terminal",
      label: "Terminal",
      description: "Open a Foot terminal.",
      executable: "foot",
      category: "Workspace",
      iconName: "foot",
      iconPath:
        "/run/current-system/sw/share/icons/hicolor/48x48/apps/foot.png",
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
      iconPath:
        "/run/current-system/sw/share/icons/hicolor/48x48/apps/system-file-manager.png",
      desktopFile: "/run/current-system/sw/share/applications/pcmanfm.desktop",
      native: true,
    },
    {
      id: "browser",
      label: "Browser",
      description: "Open Firefox.",
      executable: "firefox",
      category: "Internet",
      iconName: "firefox",
      iconPath:
        "/run/current-system/sw/share/icons/hicolor/128x128/apps/firefox.png",
      desktopFile: "/run/current-system/sw/share/applications/firefox.desktop",
      native: true,
    },
  ]);
}

vi.mock("../lib/desktop", () => ({
  DETACHED_TAURI_TOOL_WINDOWS: false,
  closeTauriWindow: desktopMocks.closeTauriWindow,
  createShellWindow: desktopMocks.createShellWindow,
  focusTauriWindow: desktopMocks.focusTauriWindow,
  isShellHostWindowLabel: (label: string) =>
    label === "main" ||
    (label.startsWith("forge-") && !label.startsWith("forge-app-")),
  isTauriDesktop: desktopMocks.isTauriDesktop,
  listForgeWindows: desktopMocks.listForgeWindows,
  listLinuxWindows: desktopMocks.listLinuxWindows,
  focusLinuxWindow: desktopMocks.focusLinuxWindow,
  controlLinuxWindow: desktopMocks.controlLinuxWindow,
  listOperatorApps: desktopMocks.listOperatorApps,
  launchOperatorApp: desktopMocks.launchOperatorApp,
  requestHostPowerAction: desktopMocks.requestHostPowerAction,
  iconAssetUrl: desktopMocks.iconAssetUrl,
  minimizeTauriWindow: desktopMocks.minimizeTauriWindow,
  monitorSignature: (monitors: Array<{ id: string }>) =>
    monitors.map((monitor) => monitor.id).join(";"),
}));

vi.mock("../lib/api", () => ({
  api: {
    dashboard: {
      summary: vi.fn(() => new Promise<never>(() => {})),
    },
  },
}));

vi.mock("./toolRegistry", () => ({
  getToolComponent: () =>
    function MockToolComponent() {
      return <div data-testid="mock-tool-content">Mock tool content</div>;
    },
}));

describe("AppShell confined Tauri tool surfaces", () => {
  beforeEach(() => {
    window.localStorage.clear();
    desktopMocks.isTauriDesktop.mockReturnValue(true);
    desktopMocks.closeTauriWindow.mockClear();
    desktopMocks.createShellWindow.mockClear();
    desktopMocks.focusTauriWindow.mockClear();
    desktopMocks.listForgeWindows.mockImplementation(
      () => new Promise(() => {}),
    );
    desktopMocks.listLinuxWindows.mockImplementation(
      () => new Promise(() => {}),
    );
    desktopMocks.focusLinuxWindow.mockResolvedValue(true);
    desktopMocks.controlLinuxWindow.mockResolvedValue(true);
    desktopMocks.listOperatorApps.mockClear();
    desktopMocks.listOperatorApps.mockImplementation(
      () => new Promise(() => {}),
    );
    desktopMocks.launchOperatorApp.mockClear();
    desktopMocks.requestHostPowerAction.mockReset();
    desktopMocks.requestHostPowerAction.mockResolvedValue({
      action: "reboot",
      requested: true,
      message: "Host reboot requested",
    });
    desktopMocks.iconAssetUrl.mockClear();
    desktopMocks.minimizeTauriWindow.mockClear();
    desktopMocks.controlLinuxWindow.mockClear();
    useDesktopWindowStore.setState({
      pinned: ["chat", "jobs", "memory", "models", "approvals", "settings"],
      windows: [
        {
          id: "chat-window",
          toolId: "chat",
          hostLabel: "main",
          x: 120,
          y: 92,
          width: 960,
          height: 640,
          z: 1,
          minimized: false,
          maximized: false,
          tauri: false,
        },
      ],
      focusedId: "chat-window",
    });
    useWorkspaceLayoutStore.setState({
      monitors: [],
      supported: true,
      currentWindowLabel: "main",
      fallbackNotice: null,
      runtimeWindows: [],
    });
  });

  afterEach(() => {
    cleanup();
  });

  it("renders Tauri tool surfaces as movable in-shell windows", async () => {
    render(
      <MemoryRouter>
        <AppShell isMainWindow={true}>
          <div />
        </AppShell>
      </MemoryRouter>,
    );

    expect(screen.getByTestId("mock-tool-content")).toBeTruthy();

    fireEvent.click(screen.getByRole("button", { name: "Minimize" }));

    await waitFor(() => {
      expect(screen.queryByTestId("mock-tool-content")).toBeNull();
      expect(useDesktopWindowStore.getState().focusedId).toBeNull();
      expect(useDesktopWindowStore.getState().windows[0]?.minimized).toBe(true);
    });
  });

  it("uses floating window chrome for Tauri shell surfaces", () => {
    const { container } = render(
      <MemoryRouter>
        <AppShell isMainWindow={true}>
          <div />
        </AppShell>
      </MemoryRouter>,
    );

    expect(container.querySelector(".forge-os-window__body")).toBeTruthy();
    expect(
      container.querySelector(".forge-os-window__body--docked"),
    ).toBeNull();
  });

  it("does not poll detached Tauri window state in confined shell mode", () => {
    render(
      <MemoryRouter>
        <AppShell isMainWindow={true}>
          <div />
        </AppShell>
      </MemoryRouter>,
    );

    expect(desktopMocks.listForgeWindows).not.toHaveBeenCalled();
  });

  it("exposes the global shell status line as a polite live region", () => {
    render(
      <MemoryRouter>
        <AppShell isMainWindow={true}>
          <div />
        </AppShell>
      </MemoryRouter>,
    );

    const status = screen.getByRole("status", {
      name: "FORGE shell status",
    });
    expect(status.getAttribute("aria-live")).toBe("polite");
    expect(status.getAttribute("aria-atomic")).toBe("true");
    expect(status.textContent).toContain("Core:");
    expect(status.textContent).toContain("Runtime:");
    expect(status.textContent).toContain("Queue:");
  });

  it("renders each in-shell window only on its assigned desktop host", () => {
    useDesktopWindowStore.setState({
      windows: [
        {
          id: "chat-window",
          toolId: "chat",
          hostLabel: "forge-right",
          x: 100,
          y: 92,
          width: 960,
          height: 640,
          z: 1,
          minimized: false,
          maximized: false,
          tauri: false,
        },
      ],
      focusedId: "chat-window",
    });

    const mainShell = render(
      <MemoryRouter>
        <AppShell isMainWindow={true} hostLabel="main">
          <div />
        </AppShell>
      </MemoryRouter>,
    );

    expect(mainShell.queryByTestId("mock-tool-content")).toBeNull();
    mainShell.unmount();

    render(
      <MemoryRouter>
        <AppShell isMainWindow={true} hostLabel="forge-right">
          <div />
        </AppShell>
      </MemoryRouter>,
    );

    expect(screen.getByTestId("mock-tool-content")).toBeTruthy();
  });

  it("shows global taskbar windows on secondary shell hosts", () => {
    useDesktopWindowStore.setState({
      windows: [
        {
          id: "chat-window",
          toolId: "chat",
          hostLabel: "main",
          x: 100,
          y: 92,
          width: 960,
          height: 640,
          z: 1,
          minimized: false,
          maximized: false,
          tauri: false,
        },
      ],
      focusedId: null,
    });

    render(
      <MemoryRouter>
        <AppShell isMainWindow={false} hostLabel="forge-right">
          <div />
        </AppShell>
      </MemoryRouter>,
    );

    expect(screen.queryByTestId("mock-tool-content")).toBeNull();
    const remoteChatButton = screen.getByTitle("Chat (open)");
    expect(remoteChatButton).toBeTruthy();
  });

  it("opens routed surfaces inside secondary monitor hosts", async () => {
    useDesktopWindowStore.setState({
      windows: [],
      focusedId: null,
    });
    useWorkspaceLayoutStore.setState({
      runtimeWindows: [
        {
          runtimeLabel: "forge-right",
          layoutId: "layout",
          layoutWindowId: "window",
          role: "mixed",
          currentRoute: "/operator-apps",
          title: "FORGE Monitor 2",
          monitorId: "display-2",
          isFocused: true,
          bounds: { x: 1920, y: 0, width: 1200, height: 800 },
          lastSeenAtMs: 1,
        },
      ],
    });

    render(
      <MemoryRouter initialEntries={["/operator-apps"]}>
        <AppShell isMainWindow={false} hostLabel="forge-right">
          <div />
        </AppShell>
      </MemoryRouter>,
    );

    await waitFor(() => {
      expect(screen.getByTestId("mock-tool-content")).toBeTruthy();
    });
    expect(useDesktopWindowStore.getState().windows[0]).toMatchObject({
      toolId: "operator-apps",
      hostLabel: "forge-right",
      monitorId: "display-2",
    });
  });

  it("represents same-tool windows from multiple hosts on the taskbar", () => {
    useDesktopWindowStore.setState({
      windows: [
        {
          id: "chat-main",
          toolId: "chat",
          hostLabel: "main",
          x: 100,
          y: 92,
          width: 960,
          height: 640,
          z: 1,
          minimized: false,
          maximized: false,
          tauri: false,
        },
        {
          id: "chat-right",
          toolId: "chat",
          hostLabel: "forge-right",
          x: 100,
          y: 92,
          width: 960,
          height: 640,
          z: 2,
          minimized: false,
          maximized: false,
          tauri: false,
        },
      ],
      focusedId: null,
    });

    const { container } = render(
      <MemoryRouter>
        <AppShell isMainWindow={true} hostLabel="main">
          <div />
        </AppShell>
      </MemoryRouter>,
    );

    const chatTaskbarItems = Array.from(
      container.querySelectorAll(".forge-os-taskbar__item--open"),
    ).filter((item) => item.textContent?.includes("Chat"));
    expect(chatTaskbarItems).toHaveLength(2);
  });

  it("does not duplicate a window across simultaneously mounted shell hosts", () => {
    useDesktopWindowStore.setState({
      windows: [
        {
          id: "chat-window",
          toolId: "chat",
          hostLabel: "forge-right",
          x: 100,
          y: 92,
          width: 960,
          height: 640,
          z: 1,
          minimized: false,
          maximized: false,
          tauri: false,
        },
      ],
      focusedId: "chat-window",
    });

    const mainShell = render(
      <MemoryRouter>
        <AppShell isMainWindow={true} hostLabel="main">
          <div />
        </AppShell>
      </MemoryRouter>,
    );
    const rightShell = render(
      <MemoryRouter>
        <AppShell isMainWindow={true} hostLabel="forge-right">
          <div />
        </AppShell>
      </MemoryRouter>,
    );

    expect(
      within(mainShell.container).queryByTestId("mock-tool-content"),
    ).toBeNull();
    expect(
      within(rightShell.container).getByTestId("mock-tool-content"),
    ).toBeTruthy();
    expect(screen.getAllByTestId("mock-tool-content")).toHaveLength(1);
  });

  it("keeps Start open when a background surface only changes search params", () => {
    render(
      <MemoryRouter initialEntries={["/"]}>
        <SearchChangeHarness />
      </MemoryRouter>,
    );

    fireEvent.click(screen.getByRole("button", { name: "Open Start menu" }));
    expect(screen.getByRole("dialog", { name: "FORGE Start" })).toBeTruthy();

    fireEvent.click(screen.getByRole("button", { name: "Change search" }));

    expect(screen.getByRole("dialog", { name: "FORGE Start" })).toBeTruthy();
  });

  it("shows categorized native operator apps in Start with installed icons", async () => {
    resolvedOperatorApps();

    render(
      <MemoryRouter>
        <AppShell isMainWindow={true}>
          <div />
        </AppShell>
      </MemoryRouter>,
    );

    fireEvent.click(screen.getByRole("button", { name: "Open Start menu" }));

    expect(await screen.findByText("Native Apps")).toBeTruthy();
    expect(screen.getAllByText("Workspace").length).toBeGreaterThanOrEqual(1);
    expect(screen.getByRole("button", { name: /Terminal/ })).toBeTruthy();
    expect(screen.getByRole("button", { name: /Browser/ })).toBeTruthy();
    expect(
      screen
        .getByText("Native Apps")
        .closest(".forge-os-startmenu__panel")
        ?.classList.contains("forge-os-startmenu__panel--native"),
    ).toBe(true);
    expect(
      screen.getByRole("img", { name: "Terminal icon" }).getAttribute("src"),
    ).toBe(
      "asset:///run/current-system/sw/share/icons/hicolor/48x48/apps/foot.png",
    );
    expect(screen.queryByPlaceholderText(/command|path/i)).toBeNull();
  });

  it("organizes Start into launcher columns with counted native categories", async () => {
    resolvedOperatorApps();

    render(
      <MemoryRouter>
        <AppShell isMainWindow={true}>
          <div />
        </AppShell>
      </MemoryRouter>,
    );

    fireEvent.click(screen.getByRole("button", { name: "Open Start menu" }));

    expect(await screen.findByText("Native Apps")).toBeTruthy();
    expect(screen.getByText("FORGE Surfaces")).toBeTruthy();
    expect(screen.getByText("Knowledge")).toBeTruthy();
    expect(screen.getByText("Governance")).toBeTruthy();
    expect(
      screen.getByText("Native Apps").closest(".forge-os-startmenu__panel"),
    ).toBeTruthy();
    expect(
      screen.getByText("FORGE Surfaces").closest(".forge-os-startmenu__panel"),
    ).toBeTruthy();
    expect(screen.getAllByText(/\d+ apps?/).length).toBeGreaterThan(0);
    expect(screen.queryByText("Native")).toBeNull();
  });

  it("shows Start menu power controls and logs out through the shell callback", () => {
    const onForgeLogout = vi.fn();

    render(
      <MemoryRouter>
        <AppShell isMainWindow={true} onForgeLogout={onForgeLogout}>
          <div />
        </AppShell>
      </MemoryRouter>,
    );

    fireEvent.click(screen.getByRole("button", { name: "Open Start menu" }));

    expect(screen.getByRole("button", { name: /Logout/ })).toBeTruthy();
    expect(screen.getByRole("button", { name: /Reboot/ })).toBeTruthy();
    expect(screen.getByRole("button", { name: /Shutdown/ })).toBeTruthy();

    fireEvent.click(screen.getByRole("button", { name: /Logout/ }));

    expect(onForgeLogout).toHaveBeenCalledTimes(1);
  });

  it("confirms and requests host reboot from Start", async () => {
    const confirm = vi.spyOn(window, "confirm").mockReturnValue(true);
    desktopMocks.requestHostPowerAction.mockResolvedValue({
      action: "reboot",
      requested: true,
      message: "Host reboot requested",
    });

    render(
      <MemoryRouter>
        <AppShell isMainWindow={true}>
          <div />
        </AppShell>
      </MemoryRouter>,
    );

    fireEvent.click(screen.getByRole("button", { name: "Open Start menu" }));
    fireEvent.click(screen.getByRole("button", { name: /Reboot/ }));

    await waitFor(() => {
      expect(desktopMocks.requestHostPowerAction).toHaveBeenCalledWith(
        "reboot",
      );
    });
    expect(confirm).toHaveBeenCalledWith("Reboot the FORGE host now?");

    confirm.mockRestore();
  });

  it("loads native operator apps when the shell runtime probe is unavailable", async () => {
    resolvedOperatorApps();
    desktopMocks.isTauriDesktop.mockReturnValue(false);

    render(
      <MemoryRouter>
        <AppShell isMainWindow={true}>
          <div />
        </AppShell>
      </MemoryRouter>,
    );

    fireEvent.click(screen.getByRole("button", { name: "Open Start menu" }));

    expect(await screen.findByText("Native Apps")).toBeTruthy();
    expect(screen.getByRole("button", { name: /Terminal/ })).toBeTruthy();
    expect(desktopMocks.listOperatorApps).toHaveBeenCalled();
  });

  it("keeps launched native apps visible on the taskbar", async () => {
    resolvedOperatorApps();
    desktopMocks.launchOperatorApp.mockResolvedValue({
      appId: "terminal",
      label: "Terminal",
      executable: "foot",
      launched: true,
      pid: 4242,
      message: "Terminal launch requested",
    });

    render(
      <MemoryRouter>
        <AppShell isMainWindow={true}>
          <div />
        </AppShell>
      </MemoryRouter>,
    );

    fireEvent.click(screen.getByRole("button", { name: "Open Start menu" }));
    fireEvent.click(await screen.findByRole("button", { name: /Terminal/ }));

    await waitFor(() => {
      expect(desktopMocks.launchOperatorApp).toHaveBeenCalledWith("terminal");
    });

    expect(
      screen.getByRole("button", { name: "Terminal native app" }),
    ).toBeTruthy();
    expect(screen.getByText("PID 4242")).toBeTruthy();
  });

  it("shows compositor-reported Linux apps on the taskbar", async () => {
    desktopMocks.listLinuxWindows.mockResolvedValue([
      {
        id: "firefox-window",
        title: "Mozilla Firefox",
        appId: "firefox",
        iconName: "firefox",
        iconPath:
          "/run/current-system/sw/share/icons/hicolor/128x128/apps/firefox.png",
        focused: false,
        minimized: false,
        native: true,
      },
    ]);

    render(
      <MemoryRouter>
        <AppShell isMainWindow={true}>
          <div />
        </AppShell>
      </MemoryRouter>,
    );

    const firefox = await screen.findByRole("button", {
      name: "Mozilla Firefox linux app",
    });
    expect(firefox).toBeTruthy();

    fireEvent.click(firefox);

    await waitFor(() => {
      expect(desktopMocks.focusLinuxWindow).toHaveBeenCalledWith(
        "firefox-window",
      );
    });
  });

  it("offers compositor controls for native Linux taskbar windows", async () => {
    desktopMocks.listLinuxWindows.mockResolvedValue([
      {
        id: "firefox-window",
        title: "Mozilla Firefox",
        appId: "firefox",
        iconName: "firefox",
        iconPath: null,
        focused: false,
        minimized: false,
        native: true,
      },
    ]);

    render(
      <MemoryRouter>
        <AppShell isMainWindow={true}>
          <div />
        </AppShell>
      </MemoryRouter>,
    );

    const firefox = await screen.findByRole("button", {
      name: "Mozilla Firefox linux app",
    });

    fireEvent.contextMenu(firefox);
    fireEvent.click(
      await screen.findByRole("menuitem", { name: "Minimize window" }),
    );

    await waitFor(() => {
      expect(desktopMocks.controlLinuxWindow).toHaveBeenCalledWith(
        "firefox-window",
        "minimize",
      );
    });
  });
});
