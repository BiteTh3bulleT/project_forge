import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { SettingsPage } from "./SettingsPage";

const mocks = vi.hoisted(() => ({
  settingsGet: vi.fn(),
  settingsPatch: vi.fn(),
  ollamaModels: vi.fn(),
  adaptersList: vi.fn(),
  meta: vi.fn(),
  telegramStatus: vi.fn(),
  discordStatus: vi.fn(),
  remoteTelegram: vi.fn(),
  remoteDiscord: vi.fn(),
  systemHost: vi.fn(),
  desktopIsTauri: vi.fn(),
  launchOperatorApp: vi.fn(),
}));

vi.mock("../lib/api", () => ({
  api: {
    settings: {
      get: mocks.settingsGet,
      patch: mocks.settingsPatch,
      ollamaModels: mocks.ollamaModels,
    },
    adapters: {
      list: mocks.adaptersList,
    },
    meta: mocks.meta,
    telegram: {
      status: mocks.telegramStatus,
    },
    discord: {
      status: mocks.discordStatus,
    },
    remote: {
      telegram: mocks.remoteTelegram,
      discord: mocks.remoteDiscord,
    },
    system: {
      host: mocks.systemHost,
    },
  },
}));

vi.mock("../lib/desktop", () => ({
  isTauriDesktop: mocks.desktopIsTauri,
  getDesktopSystemDiagnostics: vi.fn(),
  launchOperatorApp: mocks.launchOperatorApp,
}));

const baseSettings = {
  extensionsCsv: ".ts,.tsx,.go,.md",
  theme: "dark",
  ollamaBaseUrl: "http://127.0.0.1:11434",
  ollamaModel: "",
  embeddingProvider: "local_hash",
  embeddingModel: "",
  embeddingDims: "128",
  retrievalWeightKeyword: "0.45",
  retrievalWeightSemantic: "0.55",
  retrievalVSAMode: "off",
  retrievalVSADims: "128",
  retrievalVSASeed: "17",
  retrievalVSAWeightAssociative: "0.06",
  retrievalVSAWeightRoleMatch: "0.04",
  retrievalVSAWeightRelational: "0.03",
  retrievalVSAWeightFeedback: "0.03",
  retrievalVSAMaxAdditive: "0.12",
  chatPersonalityPrompt: "",
  chatPromptDefault: "",
  remoteAccessEnabled: true,
  remoteAccessToken: "[redacted]",
  remoteAccessTokenConfigured: true,
  remoteCrossChatContext: false,
  remoteDefaultThreadId: "thread-1",
  telegramBotToken: "[redacted]",
  telegramBotTokenConfigured: true,
  telegramDefaultChatId: "42",
  discordBotToken: "[redacted]",
  discordBotTokenConfigured: true,
  discordDefaultChannelId: "channel-1",
  discordWebhookUrl: "[redacted]",
  discordWebhookUrlConfigured: true,
  discordCrossChatContext: true,
  runtimeControls: {
    gpuEnabled: false,
    nvidiaDcgmEnabled: false,
    intelLevelZeroEnabled: false,
    allowOllamaCloudModels: false,
    safeModeForceCpuOnly: true,
    effectiveGpuEnabled: false,
  },
};

function renderSettingsPage() {
  return render(
    <MemoryRouter>
      <SettingsPage />
    </MemoryRouter>,
  );
}

describe("SettingsPage remote secrets", () => {
  beforeEach(() => {
    for (const mock of Object.values(mocks)) {
      mock.mockReset();
    }
    mocks.settingsGet.mockResolvedValue(baseSettings);
    mocks.desktopIsTauri.mockReturnValue(false);
    mocks.launchOperatorApp.mockResolvedValue({
      appId: "network-settings",
      label: "Network Connections",
      executable: "nm-connection-editor",
      launched: true,
      pid: 123,
      message: "Network Connections launch requested",
    });
    mocks.settingsPatch.mockResolvedValue(baseSettings);
    mocks.ollamaModels.mockResolvedValue({
      models: [],
      baseUrl: "http://127.0.0.1:11434",
      status: "ok",
    });
    mocks.adaptersList.mockResolvedValue({ adapters: [] });
    mocks.meta.mockResolvedValue({
      dataDir: "/forge/data",
      dbPath: "/forge/data/forge.db",
      workspaceDir: "/workspace",
    });
    mocks.telegramStatus.mockResolvedValue({
      remoteAccessEnabled: true,
      tokenConfigured: true,
      defaultChatId: "42",
      crossChatContext: false,
      ready: false,
      reason: "not connected in test",
    });
    mocks.discordStatus.mockResolvedValue({
      enabled: true,
      status: "disabled",
      reason: "not connected in test",
    });
    mocks.systemHost.mockResolvedValue({
      read_only: true,
      mutation_disabled: true,
      host: {
        hostname: "forge-vm",
        architecture: "x86_64",
        os_release: "FORGE OS",
      },
      cpu: { count: 8 },
      memory: {
        total_bytes: 34_359_738_368,
        available_bytes: 17_179_869_184,
        pressure_level: "normal",
      },
      storage: {
        root: "/forge",
        total_bytes: 1_000_000_000,
        free_bytes: 700_000_000,
        used_bytes: 300_000_000,
        pressure_level: "normal",
      },
      gpu: {
        available: true,
        vendor: "nvidia",
        devices: [{ name: "RTX 3050", memory_total_mib: 8192 }],
      },
      display: {
        status: "read_only",
        reason: "display control deferred",
        read_only: true,
        mutation_disabled: true,
      },
      audio: {
        status: "not_wired",
        reason: "audio control not live",
        read_only: true,
        mutation_disabled: true,
      },
      network: {
        status: "not_wired",
        reason: "network mutation not exposed",
        read_only: true,
        mutation_disabled: true,
      },
      power: {
        status: "policy_gated",
        reason: "power actions are policy gated",
        read_only: true,
        mutation_disabled: true,
      },
      session: {
        shell_mode: "manual",
        display_backend: "wayland",
        compositor_session: "labwc",
        host_mutation_disabled: true,
      },
      config: {
        safe_mode_force_cpu_only: false,
        gpu_enabled: true,
      },
    });
  });

  it("does not send redacted remote secret placeholders back to core", async () => {
    renderSettingsPage();

    fireEvent.click(
      await screen.findByRole("button", { name: "Save remote access" }),
    );

    await waitFor(() => expect(mocks.settingsPatch).toHaveBeenCalledTimes(1));
    const patch = mocks.settingsPatch.mock.calls[0][0];

    expect(patch).toMatchObject({
      remoteAccessEnabled: true,
      remoteCrossChatContext: false,
      remoteDefaultThreadId: "thread-1",
      telegramDefaultChatId: "42",
      discordDefaultChannelId: "channel-1",
      discordCrossChatContext: true,
    });
    expect(patch).not.toHaveProperty("remoteAccessToken");
    expect(patch).not.toHaveProperty("telegramBotToken");
    expect(patch).not.toHaveProperty("discordBotToken");
    expect(patch).not.toHaveProperty("discordWebhookUrl");
  });

  it("sends a changed remote token value to core", async () => {
    renderSettingsPage();

    const remoteToken = await screen.findByPlaceholderText(
      "Share with Telegram/Discord webhook callers",
    );
    fireEvent.change(remoteToken, { target: { value: "new-remote-token" } });
    fireEvent.click(screen.getByRole("button", { name: "Save remote access" }));

    await waitFor(() => expect(mocks.settingsPatch).toHaveBeenCalledTimes(1));
    expect(mocks.settingsPatch.mock.calls[0][0]).toMatchObject({
      remoteAccessToken: "new-remote-token",
    });
  });

  it("separates GPU acceleration from optional telemetry controls", async () => {
    renderSettingsPage();

    expect(
      await screen.findByText("GPU acceleration and model visibility"),
    ).toBeTruthy();
    expect(
      screen.getByText(
        "Enable this for host GPU/model acceleration. This does not enable vendor telemetry.",
      ),
    ).toBeTruthy();
    expect(screen.getByText("Optional telemetry probes")).toBeTruthy();
    expect(
      screen.getByText(
        "Leave these off unless NVIDIA DCGM or Intel Level Zero telemetry is installed and reachable.",
      ),
    ).toBeTruthy();
  });

  it("links model lifecycle work to the Models surface", async () => {
    renderSettingsPage();

    expect(await screen.findByText("Model Lifecycle")).toBeTruthy();
    expect(screen.getByRole("button", { name: "Open Models" })).toBeTruthy();
  });

  it("renders read-only host and hardware settings from core", async () => {
    renderSettingsPage();

    expect(await screen.findByText("Host and Hardware")).toBeTruthy();
    expect(screen.getByText("forge-vm · x86_64")).toBeTruthy();
    expect(screen.getByText("8 logical cores")).toBeTruthy();
    expect(screen.getByText("16.0 GB available / 32.0 GB total")).toBeTruthy();
    expect(screen.getByText("RTX 3050")).toBeTruthy();
    expect(screen.getByText("Display: read_only")).toBeTruthy();
    expect(screen.getByText("Audio: not_wired")).toBeTruthy();
    expect(screen.getByText("Network: not_wired")).toBeTruthy();
    expect(screen.getByText("Power: policy_gated")).toBeTruthy();
    expect(screen.getByText("Host mutation disabled")).toBeTruthy();
    expect(mocks.systemHost).toHaveBeenCalledTimes(1);
  });

  it("launches native network settings from the host settings surface", async () => {
    mocks.desktopIsTauri.mockReturnValue(true);
    renderSettingsPage();

    expect(await screen.findByText("Native System Controls")).toBeTruthy();
    fireEvent.click(
      screen.getByRole("button", { name: "Open Network Connections" }),
    );

    await waitFor(() =>
      expect(mocks.launchOperatorApp).toHaveBeenCalledWith("network-settings"),
    );
  });

  it("keeps primary settings usable when secondary status loads fail", async () => {
    mocks.adaptersList.mockRejectedValueOnce(
      new Error("adapter discovery down"),
    );
    mocks.meta.mockRejectedValueOnce(new Error("meta down"));
    mocks.telegramStatus.mockRejectedValueOnce(new Error("telegram down"));
    mocks.discordStatus.mockRejectedValueOnce(new Error("discord down"));

    renderSettingsPage();

    expect(await screen.findByText("Settings operations board")).toBeTruthy();
    expect(
      screen.getByRole("button", { name: "Save extensions" }),
    ).toBeTruthy();
    expect(
      await screen.findByText(
        "Ollama model list unavailable: adapter discovery down",
      ),
    ).toBeTruthy();
    expect(await screen.findByText("telegram down")).toBeTruthy();
    expect(await screen.findByText("discord down")).toBeTruthy();
    expect(screen.queryByText("meta down")).toBeNull();
    expect(mocks.settingsPatch).not.toHaveBeenCalled();
  });
});
