import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { MemoryRouter, useLocation } from "react-router-dom";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { ModelsPage } from "./ModelsPage";

function LocationProbe() {
  const location = useLocation();
  return (
    <div data-testid="location-probe">
      {location.pathname}
      {location.search}
    </div>
  );
}

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
  modelGet: vi.fn(),
  modelCompatibility: vi.fn(),
  modelLoad: vi.fn(),
  modelUnload: vi.fn(),
  approvalApprove: vi.fn(),
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
      get: mocks.modelGet,
      compatibility: mocks.modelCompatibility,
      load: mocks.modelLoad,
      unload: mocks.modelUnload,
      scan: mocks.modelScan,
      import: mocks.modelImport,
    },
    approvals: {
      approve: mocks.approvalApprove,
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
    mocks.modelGet.mockResolvedValue({
      model: {
        id: "llama3.2:3b",
        displayName: "llama3.2:3b",
        family: "llama",
        backend: "ollama_compat",
        format: "gguf",
        status: "available",
        capabilities: ["chat", "completion"],
      },
    });
    mocks.modelCompatibility.mockResolvedValue({
      compatibility: {
        modelId: "llama3.2:3b",
        backend: "ollama_compat",
        backendHealthy: true,
        backendConfigured: true,
        supportedByBackend: true,
        canGenerate: true,
        preferred: false,
        warnings: [],
        details: {},
      },
    });
    mocks.modelLoad.mockResolvedValue({
      result: { modelId: "llama3.2:3b", loaded: true, status: "loaded" },
    });
    mocks.modelUnload.mockResolvedValue({
      result: { modelId: "llama3.2:3b", loaded: false, status: "unloaded" },
    });
    mocks.approvalApprove.mockResolvedValue({
      decision: { decision: "approved" },
    });
  });

  it("keeps model runtime controls read-only when runtime is unavailable", async () => {
    render(
      <MemoryRouter initialEntries={["/models?view=registry"]}>
        <ModelsPage />
      </MemoryRouter>,
    );

    expect((await screen.findAllByText("unavailable")).length).toBeGreaterThan(
      0,
    );

    expect(
      (
        screen.getByRole("button", {
          name: /scan model home/i,
        }) as HTMLButtonElement
      ).disabled,
    ).toBe(true);
    expect(
      (
        screen.getByRole("button", {
          name: /import model/i,
        }) as HTMLButtonElement
      ).disabled,
    ).toBe(true);
    expect(
      (
        screen.getByRole("button", {
          name: /reconcile registry/i,
        }) as HTMLButtonElement
      ).disabled,
    ).toBe(true);
    expect(
      (screen.getByLabelText(/gpu acceleration/i) as HTMLInputElement).disabled,
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

  it("shows compact lifecycle controls for the selected model", async () => {
    mocks.health.mockResolvedValue({
      ok: true,
      service: "forge-core",
      modelRuntime: {
        available: true,
        status: "ok",
      },
    });
    mocks.modelList.mockResolvedValue({
      models: [
        {
          id: "llama3.2:3b",
          displayName: "llama3.2:3b",
          family: "llama",
          backend: "ollama_compat",
          format: "gguf",
          status: "available",
          capabilities: ["chat", "completion"],
        },
      ],
    });

    render(
      <MemoryRouter initialEntries={["/models"]}>
        <ModelsPage />
      </MemoryRouter>,
    );

    expect(await screen.findByRole("button", { name: "Load" })).toBeTruthy();
  });

  it("opens the registry view from a shell window without mutating the shell route", async () => {
    mocks.health.mockResolvedValue({
      ok: true,
      service: "forge-core",
      modelRuntime: {
        available: true,
        status: "ok",
      },
    });

    render(
      <MemoryRouter initialEntries={["/"]}>
        <ModelsPage />
        <LocationProbe />
      </MemoryRouter>,
    );

    fireEvent.click(
      await screen.findByRole("button", { name: "Open Registry" }),
    );

    expect(
      await screen.findByRole("heading", { name: "Model runtime board" }),
    ).toBeTruthy();
    expect(screen.getByTestId("location-probe").textContent).toBe("/");
  });

  it("reuses pending load approval ids on the next load attempt", async () => {
    mocks.health.mockResolvedValue({
      ok: true,
      service: "forge-core",
      modelRuntime: {
        available: true,
        status: "ok",
      },
    });
    mocks.modelList.mockResolvedValue({
      models: [
        {
          id: "llama3.2:3b",
          displayName: "llama3.2:3b",
          family: "llama",
          backend: "ollama_compat",
          format: "gguf",
          status: "available",
          capabilities: ["chat", "completion"],
        },
      ],
    });
    mocks.modelLoad
      .mockResolvedValueOnce({
        governance: {
          operation: "load",
          modelId: "llama3.2:3b",
          requiresApproval: true,
          approved: false,
          approvalRequestId: 7,
        },
      })
      .mockResolvedValueOnce({
        result: { modelId: "llama3.2:3b", loaded: true, status: "loaded" },
      });

    render(
      <MemoryRouter initialEntries={["/models"]}>
        <ModelsPage />
      </MemoryRouter>,
    );

    fireEvent.click(await screen.findByRole("button", { name: "Load" }));

    expect(await screen.findByText("Approval required #7")).toBeTruthy();

    fireEvent.click(screen.getByRole("button", { name: "Load" }));

    await waitFor(() => expect(mocks.modelLoad).toHaveBeenCalledTimes(2));
    expect(mocks.modelLoad.mock.calls[1][1]).toMatchObject({
      approvalId: "7",
    });
  });

  it("can approve and retry pending model loads from the models surface", async () => {
    mocks.health.mockResolvedValue({
      ok: true,
      service: "forge-core",
      modelRuntime: {
        available: true,
        status: "ok",
      },
    });
    mocks.modelList.mockResolvedValue({
      models: [
        {
          id: "llama3.2:3b",
          displayName: "llama3.2:3b",
          family: "llama",
          backend: "ollama_compat",
          format: "gguf",
          status: "available",
          capabilities: ["chat", "completion"],
        },
      ],
    });
    mocks.modelLoad
      .mockResolvedValueOnce({
        governance: {
          operation: "load",
          modelId: "llama3.2:3b",
          requiresApproval: true,
          approved: false,
          approvalRequestId: 7,
        },
      })
      .mockResolvedValueOnce({
        result: { modelId: "llama3.2:3b", loaded: true, status: "loaded" },
      });

    render(
      <MemoryRouter initialEntries={["/models"]}>
        <ModelsPage />
      </MemoryRouter>,
    );

    fireEvent.click(await screen.findByRole("button", { name: "Load" }));
    fireEvent.click(
      await screen.findByRole("button", { name: "Approve and load" }),
    );

    await waitFor(() =>
      expect(mocks.approvalApprove).toHaveBeenCalledWith(
        7,
        "Approved model load from Models page",
      ),
    );
    await waitFor(() => expect(mocks.modelLoad).toHaveBeenCalledTimes(2));
    expect(mocks.modelLoad.mock.calls[1][1]).toMatchObject({
      approvalId: "7",
    });
  });
});
