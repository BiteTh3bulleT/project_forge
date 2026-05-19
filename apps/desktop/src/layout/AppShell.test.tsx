import {
  act,
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
import type {
  LinuxWindowSnapshot,
  OperatorApp,
  OperatorAppLaunchResult,
} from "../lib/desktop";

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

const apiMocks = vi.hoisted(() => ({
  dashboardSummary: vi.fn<() => Promise<unknown>>(
    () => new Promise(() => {}),
  ),
  autonomyStatus: vi.fn<() => Promise<unknown>>(() => new Promise(() => {})),
  auditList: vi.fn<() => Promise<unknown>>(() => new Promise(() => {})),
  modelRuntimeQueue: vi.fn<() => Promise<unknown>>(
    () => new Promise(() => {}),
  ),
  modelRuntimeBackends: vi.fn<() => Promise<unknown>>(
    () => new Promise(() => {}),
  ),
  contextSnapshots: vi.fn<() => Promise<unknown>>(
    () => new Promise(() => {}),
  ),
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
      summary: apiMocks.dashboardSummary,
    },
    autonomy: {
      status: apiMocks.autonomyStatus,
    },
    audit: {
      list: apiMocks.auditList,
    },
    modelRuntime: {
      queue: apiMocks.modelRuntimeQueue,
      backends: apiMocks.modelRuntimeBackends,
    },
    contextInspector: {
      listSnapshots: apiMocks.contextSnapshots,
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
    apiMocks.dashboardSummary.mockReset();
    apiMocks.dashboardSummary.mockImplementation(
      () => new Promise(() => {}),
    );
    apiMocks.autonomyStatus.mockReset();
    apiMocks.autonomyStatus.mockImplementation(
      () => new Promise<never>(() => {}),
    );
    apiMocks.auditList.mockReset();
    apiMocks.auditList.mockImplementation(() => new Promise<never>(() => {}));
    apiMocks.modelRuntimeQueue.mockReset();
    apiMocks.modelRuntimeQueue.mockImplementation(
      () => new Promise<never>(() => {}),
    );
    apiMocks.modelRuntimeBackends.mockReset();
    apiMocks.modelRuntimeBackends.mockImplementation(
      () => new Promise<never>(() => {}),
    );
    apiMocks.contextSnapshots.mockReset();
    apiMocks.contextSnapshots.mockImplementation(
      () => new Promise<never>(() => {}),
    );
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
    vi.useRealTimers();
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

  it("renders routed detail pages in a focused shell window", () => {
    render(
      <MemoryRouter initialEntries={["/jobs/job-1"]}>
        <AppShell isMainWindow={true}>
          <div data-testid="routed-detail">Job detail route</div>
        </AppShell>
      </MemoryRouter>,
    );

    expect(screen.getByLabelText("Job Detail")).toBeTruthy();
    expect(screen.getByTestId("routed-detail").textContent).toBe(
      "Job detail route",
    );
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
    expect(screen.queryByRole("complementary", {
      name: "Shell context inspector",
    })).toBeNull();
  });

  it("summarizes model runtime, autonomy, latest audit, and workspace in the shell status bar", async () => {
    apiMocks.dashboardSummary.mockResolvedValue({
      activeJobs: [],
      approvalsPending: 1,
      reviewsPending: 0,
      recentFailures: [],
      recentImports: [],
      dossierHealth: [],
      automationActivity: [],
      routingRecommendations: [],
      systemStatus: {},
    });
    apiMocks.modelRuntimeQueue.mockResolvedValue({
      queue: { depth: 2, active: { chat: "running" }, scheduler: "weighted" },
    });
    apiMocks.modelRuntimeBackends.mockResolvedValue({
      backends: [
        { kind: "ollama", name: "local", healthy: true, loadedModel: "qwen" },
      ],
    });
    apiMocks.autonomyStatus.mockResolvedValue({
      enabled: true,
      maintenanceLoop: { active: true },
    });
    apiMocks.auditList.mockResolvedValue({
      records: [
        {
          id: 42,
          createdAtMs: 1700000000000,
          category: "gateway",
          action: "tool.executed",
          outcome: "ok",
          summary: "Shell probe completed",
        },
      ],
    });
    apiMocks.contextSnapshots.mockResolvedValue({ snapshots: [] });

    render(
      <MemoryRouter>
        <AppShell isMainWindow={true}>
          <div />
        </AppShell>
      </MemoryRouter>,
    );

    const status = await screen.findByRole("status", {
      name: "FORGE shell status",
    });

    await waitFor(
      () => {
        expect(status.textContent).toContain("Core:");
        expect(status.textContent).toContain("Runtime:");
        expect(status.textContent).toContain("Queue:");
      },
      { timeout: 3000 },
    );
    fireEvent.click(
      screen.getByRole("button", { name: "Open shell status details" }),
    );
    const inspector = screen.getByRole("complementary", {
      name: "Shell context inspector",
    });
    await waitFor(() => {
      expect(within(inspector).getByText("healthy · queue 2")).toBeTruthy();
      expect(within(inspector).getByText("active")).toBeTruthy();
      expect(
        within(inspector).getAllByText("tool.executed ok").length,
      ).toBeGreaterThan(0);
    });
    expect(apiMocks.auditList).toHaveBeenCalledWith({ limit: 20 });
  });

  it("shows a right-side context inspector with context, audit, loops, and approvals", async () => {
    apiMocks.dashboardSummary.mockResolvedValue({
      activeJobs: [
        {
          id: "job-context",
          title: "Compile operator context",
          status: "running",
          targetAdapter: "context",
          createdAtMs: 1700000000100,
        },
      ],
      approvalsPending: 2,
      reviewsPending: 0,
      recentFailures: [],
      recentImports: [],
      dossierHealth: [],
      automationActivity: [],
      routingRecommendations: [],
      systemStatus: {},
    });
    apiMocks.modelRuntimeQueue.mockResolvedValue({ queue: { depth: 0 } });
    apiMocks.modelRuntimeBackends.mockResolvedValue({ backends: [] });
    apiMocks.autonomyStatus.mockResolvedValue({
      enabled: true,
      maintenanceLoop: { active: true, mode: "dry_run" },
    });
    apiMocks.auditList.mockResolvedValue({
      records: [
        {
          id: 7,
          createdAtMs: 1700000000000,
          category: "context",
          action: "context.compiled",
          outcome: "ok",
          summary: "Compiled FORGE_PUNCHLIST context",
        },
      ],
    });
    apiMocks.contextSnapshots.mockResolvedValue({
      snapshots: [
        {
          id: "ctx-1",
          createdAtMs: 1700000000000,
          workspaceId: "default",
          laneId: "control",
          snapshotKind: "compile",
          label: "FORGE_PUNCHLIST Section 6",
          summary: "Shell UI compile package",
        },
      ],
    });

    render(
      <MemoryRouter>
        <AppShell isMainWindow={true}>
          <div />
        </AppShell>
      </MemoryRouter>,
    );

    fireEvent.click(
      screen.getByRole("button", { name: "Open shell status details" }),
    );
    const inspector = await screen.findByRole("complementary", {
      name: "Shell context inspector",
    });

    expect(within(inspector).getByText("Context Inspector")).toBeTruthy();
    await waitFor(() => {
      expect(
        within(inspector).getByText("FORGE_PUNCHLIST Section 6"),
      ).toBeTruthy();
      expect(
        within(inspector).getByText("Compiled FORGE_PUNCHLIST context"),
      ).toBeTruthy();
      expect(within(inspector).getByText("maintenance active")).toBeTruthy();
      expect(within(inspector).getByText("2 pending")).toBeTruthy();
    });
  });

  it("opens the activity log surface with the last 20 audit events", async () => {
    apiMocks.auditList.mockResolvedValue({
      records: [
        {
          id: 1,
          createdAtMs: 1700000000000,
          category: "gateway",
          action: "tool.executed",
          outcome: "ok",
          summary: "First event",
        },
        {
          id: 2,
          createdAtMs: 1700000001000,
          category: "modelruntime",
          action: "model.runtime.chat",
          outcome: "authorized",
          summary: "Second event",
        },
      ],
    });
    apiMocks.autonomyStatus.mockResolvedValue({
      available: true,
      dream: { active: false },
    });
    apiMocks.modelRuntimeQueue.mockResolvedValue({ queue: { depth: 0 } });
    apiMocks.modelRuntimeBackends.mockResolvedValue({ backends: [] });
    apiMocks.contextSnapshots.mockResolvedValue({ snapshots: [] });

    render(
      <MemoryRouter>
        <AppShell isMainWindow={true}>
          <div />
        </AppShell>
      </MemoryRouter>,
    );

    fireEvent.click(
      await screen.findByRole("button", { name: "Open activity log" }),
    );

    const log = screen.getByRole("dialog", { name: "Activity log" });
    await waitFor(() => {
      expect(within(log).getByText("First event")).toBeTruthy();
      expect(within(log).getByText("Second event")).toBeTruthy();
    });
    expect(within(log).getByText("model.runtime.chat")).toBeTruthy();
    expect(apiMocks.auditList).toHaveBeenCalledWith({ limit: 20 });
  });

  it("persists a CSS-variable shell theme and accent from the status controls", async () => {
    render(
      <MemoryRouter>
        <AppShell isMainWindow={true}>
          <div />
        </AppShell>
      </MemoryRouter>,
    );

    fireEvent.click(
      screen.getByRole("button", { name: "Open shell status details" }),
    );
    fireEvent.click(screen.getByRole("button", { name: "Switch shell theme" }));
    fireEvent.change(screen.getByLabelText("Shell accent"), {
      target: { value: "amber" },
    });

    await waitFor(() => {
      expect(document.documentElement.dataset.theme).toBe("light");
      expect(document.documentElement.dataset.forgeAccent).toBe("amber");
      expect(window.localStorage.getItem("forge.ui.theme")).toBe("light");
      expect(window.localStorage.getItem("forge.ui.accent")).toBe("amber");
    });
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

  it("does not keep routed children mounted behind an active shell window", () => {
    render(
      <MemoryRouter>
        <AppShell isMainWindow={true}>
          <div>Deep linked chat route payload</div>
        </AppShell>
      </MemoryRouter>,
    );

    expect(screen.getByTestId("mock-tool-content")).toBeTruthy();
    expect(screen.queryByText("Deep linked chat route payload")).toBeNull();
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

  it("does not create a native taskbar placeholder for refused launches", async () => {
    resolvedOperatorApps();
    desktopMocks.launchOperatorApp.mockResolvedValue({
      appId: "terminal",
      label: "Terminal",
      executable: "foot",
      launched: false,
      pid: null,
      message: "Terminal launch refused by policy",
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
      screen.queryByRole("button", { name: "Terminal native app" }),
    ).toBeNull();

    fireEvent.click(screen.getByRole("button", { name: "Open Start menu" }));
    expect(
      await screen.findByText("Terminal launch refused by policy"),
    ).toBeTruthy();
  });

  it("prevents duplicate native launch requests while launch is pending", async () => {
    resolvedOperatorApps();
    let resolveLaunch: (result: OperatorAppLaunchResult) => void = () => {};
    desktopMocks.launchOperatorApp.mockImplementation(
      () =>
        new Promise<OperatorAppLaunchResult>((resolve) => {
          resolveLaunch = resolve;
        }),
    );

    render(
      <MemoryRouter>
        <AppShell isMainWindow={true}>
          <div />
        </AppShell>
      </MemoryRouter>,
    );

    fireEvent.click(screen.getByRole("button", { name: "Open Start menu" }));
    fireEvent.click(await screen.findByRole("button", { name: /Terminal/ }));
    fireEvent.click(screen.getByRole("button", { name: "Open Start menu" }));
    fireEvent.click(await screen.findByRole("button", { name: /Terminal/ }));

    expect(desktopMocks.launchOperatorApp).toHaveBeenCalledTimes(1);

    resolveLaunch({
      appId: "terminal",
      label: "Terminal",
      executable: "foot",
      launched: true,
      pid: 4242,
      message: "Terminal launch requested",
    });

    expect(
      await screen.findByRole("button", { name: "Terminal native app" }),
    ).toBeTruthy();
  });

  it("expires stale native launch placeholders when no compositor window appears", async () => {
    vi.useFakeTimers();
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
    await act(async () => {
      await Promise.resolve();
    });
    fireEvent.click(screen.getByRole("button", { name: /Terminal/ }));
    await act(async () => {
      await Promise.resolve();
    });

    expect(
      screen.getByRole("button", { name: "Terminal native app" }),
    ).toBeTruthy();

    await act(async () => {
      await vi.advanceTimersByTimeAsync(31_000);
    });

    expect(
      screen.queryByRole("button", { name: "Terminal native app" }),
    ).toBeNull();
  });

  it("removes launched native placeholders after the Linux window appears", async () => {
    resolvedOperatorApps();
    desktopMocks.listLinuxWindows.mockResolvedValue([]);
    desktopMocks.launchOperatorApp.mockResolvedValue({
      appId: "files",
      label: "Files",
      executable: "pcmanfm",
      launched: true,
      pid: 4343,
      message: "Files launch requested",
    });

    render(
      <MemoryRouter>
        <AppShell isMainWindow={true}>
          <div />
        </AppShell>
      </MemoryRouter>,
    );

    fireEvent.click(screen.getByRole("button", { name: "Open Start menu" }));
    fireEvent.click(await screen.findByRole("button", { name: /Files/ }));

    await waitFor(() => {
      expect(desktopMocks.launchOperatorApp).toHaveBeenCalledWith("files");
    });
    expect(
      screen.getByRole("button", { name: "Files native app" }),
    ).toBeTruthy();

    desktopMocks.listLinuxWindows.mockResolvedValue([
      {
        id: "pcmanfm-window",
        title: "default - File Manager",
        appId: "pcmanfm-qt",
        iconName: "system-file-manager",
        iconPath: null,
        focused: false,
        minimized: false,
        native: true,
        firstSeenMs: Date.now() + 100,
        lastSeenMs: Date.now() + 100,
      },
    ]);

    await waitFor(
      () => {
        expect(
          screen.queryByRole("button", { name: "Files native app" }),
        ).toBeNull();
      },
      { timeout: 2500 },
    );
    expect(
      screen.getByRole("button", { name: "default - File Manager linux app" }),
    ).toBeTruthy();
  });

  it("does not resolve a new native launch against an already visible matching window", async () => {
    resolvedOperatorApps();
    const firstSeenMs = Date.now();
    desktopMocks.listLinuxWindows.mockResolvedValue([
      {
        id: "terminal-existing",
        title: "Terminal",
        appId: "foot",
        iconName: "foot",
        iconPath: null,
        focused: false,
        minimized: false,
        native: true,
        firstSeenMs,
        lastSeenMs: firstSeenMs,
      },
    ]);
    desktopMocks.launchOperatorApp.mockResolvedValue({
      appId: "terminal",
      label: "Terminal",
      executable: "foot",
      launched: true,
      pid: 5151,
      message: "Terminal launch requested",
    });

    render(
      <MemoryRouter>
        <AppShell isMainWindow={true}>
          <div />
        </AppShell>
      </MemoryRouter>,
    );

    expect(
      await screen.findByRole("button", { name: "Terminal linux app" }),
    ).toBeTruthy();

    fireEvent.click(screen.getByRole("button", { name: "Open Start menu" }));
    fireEvent.click(
      await within(
        screen.getByRole("dialog", { name: "FORGE Start" }),
      ).findByRole("button", { name: /Terminal/ }),
    );

    expect(
      await screen.findByRole("button", { name: "Terminal native app" }),
    ).toBeTruthy();
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

  it("keeps multiple compositor windows from the same native app separate", async () => {
    desktopMocks.listLinuxWindows.mockResolvedValue([
      {
        id: "terminal-one",
        title: "Terminal",
        appId: "foot",
        iconName: "foot",
        iconPath: null,
        focused: false,
        minimized: false,
        native: true,
      },
      {
        id: "terminal-two",
        title: "Terminal",
        appId: "foot",
        iconName: "foot",
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

    expect(
      await screen.findAllByRole("button", { name: "Terminal linux app" }),
    ).toHaveLength(2);
  });

  it("does not render compositor windows marked closed", async () => {
    desktopMocks.listLinuxWindows.mockResolvedValue([
      {
        id: "terminal-closed",
        title: "Terminal",
        appId: "foot",
        iconName: "foot",
        iconPath: null,
        focused: false,
        minimized: false,
        native: true,
        lifecycle: "closed",
      },
      {
        id: "firefox-window",
        title: "Mozilla Firefox",
        appId: "firefox",
        iconName: "firefox",
        iconPath: null,
        focused: false,
        minimized: false,
        native: true,
        lifecycle: "active",
      },
    ]);

    render(
      <MemoryRouter>
        <AppShell isMainWindow={true}>
          <div />
        </AppShell>
      </MemoryRouter>,
    );

    expect(
      await screen.findByRole("button", {
        name: "Mozilla Firefox linux app",
      }),
    ).toBeTruthy();
    expect(
      screen.queryByRole("button", { name: "Terminal linux app" }),
    ).toBeNull();
  });

  it("minimizes focused native Linux taskbar windows on left click", async () => {
    desktopMocks.listLinuxWindows.mockResolvedValue([
      {
        id: "firefox-window",
        title: "Mozilla Firefox",
        appId: "firefox",
        iconName: "firefox",
        iconPath: null,
        focused: true,
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

    fireEvent.click(firefox);

    await waitFor(() => {
      expect(desktopMocks.controlLinuxWindow).toHaveBeenCalledWith(
        "firefox-window",
        "minimize",
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

  it("middle-click closes compositor Linux taskbar windows", async () => {
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

    fireEvent(
      firefox,
      new MouseEvent("auxclick", { bubbles: true, button: 1 }),
    );

    await waitFor(() => {
      expect(desktopMocks.controlLinuxWindow).toHaveBeenCalledWith(
        "firefox-window",
        "close",
      );
    });
  });

  it("reports bounded failures for unsupported native Linux taskbar actions", async () => {
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
    desktopMocks.controlLinuxWindow.mockResolvedValue(false);

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
    fireEvent.click(screen.getByRole("button", { name: "Open Start menu" }));

    expect(
      await screen.findByText(
        "Mozilla Firefox did not accept the minimize request.",
      ),
    ).toBeTruthy();
  });
});
