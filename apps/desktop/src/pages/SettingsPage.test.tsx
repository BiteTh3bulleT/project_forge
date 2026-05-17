import { fireEvent, render, screen, waitFor } from "@testing-library/react";
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
  },
}));

vi.mock("../lib/desktop", () => ({
  isTauriDesktop: () => false,
  getDesktopSystemDiagnostics: vi.fn(),
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

describe("SettingsPage remote secrets", () => {
  beforeEach(() => {
    for (const mock of Object.values(mocks)) {
      mock.mockReset();
    }
    mocks.settingsGet.mockResolvedValue(baseSettings);
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
  });

  it("does not send redacted remote secret placeholders back to core", async () => {
    render(<SettingsPage />);

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
    render(<SettingsPage />);

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
    render(<SettingsPage />);

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
});
