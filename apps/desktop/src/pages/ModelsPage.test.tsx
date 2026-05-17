import { fireEvent, render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { ModelsPage } from "./ModelsPage";

const mocks = vi.hoisted(() => ({
  health: vi.fn(),
  settingsGet: vi.fn(),
  settingsPatch: vi.fn(),
  modelList: vi.fn(),
  modelHealth: vi.fn(),
  modelQueue: vi.fn(),
  modelLoaded: vi.fn(),
  modelUsage: vi.fn(),
  modelBackends: vi.fn(),
  modelScan: vi.fn(),
  modelImport: vi.fn(),
}));

vi.mock("../lib/api", () => ({
  api: {
    health: mocks.health,
    settings: {
      get: mocks.settingsGet,
      patch: mocks.settingsPatch,
    },
    modelRuntime: {
      list: mocks.modelList,
      health: mocks.modelHealth,
      queue: mocks.modelQueue,
      loaded: mocks.modelLoaded,
      usage: mocks.modelUsage,
      backends: mocks.modelBackends,
      scan: mocks.modelScan,
      import: mocks.modelImport,
    },
  },
}));

describe("ModelsPage", () => {
  beforeEach(() => {
    for (const mock of Object.values(mocks)) {
      mock.mockReset();
    }
    mocks.settingsGet.mockResolvedValue({
      runtimeControls: {
        gpuEnabled: true,
        allowOllamaCloudModels: true,
        effectiveGpuEnabled: false,
      },
    });
    mocks.health.mockResolvedValue({
      ok: true,
      service: "forge-core",
      modelRuntime: {
        available: false,
        status: "unavailable",
      },
    });
    mocks.modelList.mockResolvedValue({ models: [] });
    mocks.modelHealth.mockResolvedValue({
      health: { ok: true, status: "ok", backend: "none" },
    });
    mocks.modelQueue.mockResolvedValue({
      queue: { depth: 0, scheduler: "idle" },
    });
    mocks.modelLoaded.mockResolvedValue({ loaded: { count: 0, models: [] } });
    mocks.modelUsage.mockResolvedValue({
      usage: {
        totalRequests: 0,
        completedRequests: 0,
        failedRequests: 0,
        activeRequests: 0,
        totalTokensIn: 0,
        totalTokensOut: 0,
      },
    });
    mocks.modelBackends.mockResolvedValue({ backends: [] });
  });

  it("keeps model runtime controls read-only when runtime is unavailable", async () => {
    render(
      <MemoryRouter initialEntries={["/models?view=registry"]}>
        <ModelsPage />
      </MemoryRouter>,
    );

    expect((await screen.findAllByText("unavailable")).length)
      .toBeGreaterThan(0);

    expect(
      (screen.getByRole("button", {
        name: /scan model home/i,
      }) as HTMLButtonElement).disabled,
    ).toBe(true);
    expect(
      (screen.getByRole("button", {
        name: /import model/i,
      }) as HTMLButtonElement).disabled,
    ).toBe(true);
    expect(
      (screen.getByRole("button", {
        name: /reconcile registry/i,
      }) as HTMLButtonElement).disabled,
    ).toBe(true);
    expect(
      (screen.getByLabelText(/gpu acceleration/i) as HTMLInputElement)
        .disabled,
    ).toBe(true);
    expect(
      (screen.getByLabelText(/ollama cloud models/i) as HTMLInputElement)
        .disabled,
    ).toBe(true);

    fireEvent.click(screen.getByRole("button", { name: /scan model home/i }));
    fireEvent.click(screen.getByRole("button", { name: /import model/i }));
    fireEvent.click(screen.getByLabelText(/gpu acceleration/i));

    expect(mocks.modelList).not.toHaveBeenCalled();
    expect(mocks.modelScan).not.toHaveBeenCalled();
    expect(mocks.modelImport).not.toHaveBeenCalled();
    expect(mocks.settingsPatch).not.toHaveBeenCalled();
  });

  it("describes GPU acceleration without implying telemetry is enabled", async () => {
    mocks.health.mockResolvedValue({
      ok: true,
      service: "forge-core",
      modelRuntime: {
        available: true,
        status: "ok",
      },
    });

    render(
      <MemoryRouter initialEntries={["/models?view=registry"]}>
        <ModelsPage />
      </MemoryRouter>,
    );

    expect(
      await screen.findByText(
        "GPU acceleration uses the model runtime policy only; DCGM and Intel telemetry stay separate in Settings.",
      ),
    ).toBeTruthy();
  });
});
