import { render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { SystemPage } from "./SystemPage";

const mocks = vi.hoisted(() => ({
  status: vi.fn(),
  approvals: vi.fn(),
}));

vi.mock("../lib/api", () => ({
  api: {
    system: {
      status: mocks.status,
    },
    approvals: {
      list: mocks.approvals,
    },
  },
}));

describe("SystemPage", () => {
  beforeEach(() => {
    mocks.status.mockReset();
    mocks.approvals.mockReset();
    mocks.approvals.mockResolvedValue({
      approvals: [{ id: 7, status: "pending", risk: "medium" }],
    });
    mocks.status.mockResolvedValue({
      generated_at: "2026-05-09T12:00:00Z",
      core: {
        reachable: true,
        service: "forge-core",
        health_state: "ok",
        core_url: "http://127.0.0.1:18492",
        last_refresh_at: "2026-05-09T12:00:00Z",
      },
      shell_session: {
        shell_mode: "manual",
        display_backend: "wayland",
        compositor_session: "cage",
        safe_mode: false,
        host_mutation_disabled: true,
        model_mutation_disabled: true,
        semantic_memory_write_disabled: true,
        forge_k_live_authority_disabled: true,
        context_compiler_required_for_llm: true,
      },
      hostbridge: {
        wired: true,
        reason: "bounded read-only diagnostics",
        snapshot_id: "hostdiag_123",
        host_identity: "forge-vm",
        architecture: "x86_64",
        ram_pressure: "normal",
        disk_pressure: "normal",
        gpu_available: false,
        thermal_available: false,
        source_errors_count: 0,
        degraded: false,
      },
      forgeh: {
        wired: true,
        policy: {
          policy_id: "policy_1",
          overall_posture: "normal",
          ram_pressure: "normal",
          swap_pressure: "normal",
          disk_pressure: "normal",
          vram_pressure: "unavailable",
          thermal_pressure: "unavailable",
          model_load_recommendation: "cpu_safe_mode_preferred",
          background_work_recommendation: "allow",
          warnings: ["gpu diagnostics unavailable"],
          advisory_only: true,
        },
        proposals: [
          {
            proposal_id: "proposal_1",
            action_type: "warn_operator",
            target_lane: "model_load",
            risk_level: "low",
            status: "proposed",
            expires_at: "2026-05-10T12:00:00Z",
            advisory_only: true,
          },
        ],
        executions: {
          available: true,
          reason: "governed bounded execution ledger",
          items: [
            {
              execution_id: "execution_1",
              proposal_id: "proposal_1",
              action_type: "warn_operator",
              status: "completed",
              result: "reported",
              bounded: true,
              host_mutation: false,
              semantic_memory_write: false,
              modelruntime_mutation: false,
              side_effects: ["operator_notification"],
            },
          ],
        },
        advisory_only: true,
        canonical_write_committed: false,
      },
      modelruntime: {
        available: false,
        state: "unavailable",
        mutation_disabled: true,
      },
      storage: {
        root: "/forge",
        data_dir: "/forge/data",
        db_path: "/forge/data/forge.sqlite",
        truth_authority: "sqlite",
        ping_ok: true,
        pressure_level: "normal",
        redis: {
          enabled: false,
          truth_authority: false,
          role: "optional cache",
        },
        qdrant: {
          enabled: false,
          truth_authority: false,
          role: "optional vector index",
        },
      },
      approval_queue: {
        wired: true,
        reason: "use governed approvals surface",
      },
      warnings: ["shell system surface is read-only"],
    });
  });

  it("renders read-only shell system surfaces", async () => {
    render(
      <MemoryRouter>
        <SystemPage />
      </MemoryRouter>,
    );

    expect(await screen.findByRole("heading", { name: "System Surfaces" }))
      .toBeTruthy();
    expect(screen.getByText("forge-core")).toBeTruthy();
    expect(screen.getByText("http://127.0.0.1:18492")).toBeTruthy();
    expect(screen.getByText("Core health")).toBeTruthy();
    expect(screen.getByText("Last core refresh")).toBeTruthy();
    expect(screen.getByText("Safe mode")).toBeTruthy();
    expect(screen.getByText("Host mutation disabled")).toBeTruthy();
    expect(screen.getByText("FORGE-K live authority disabled")).toBeTruthy();
    expect(screen.getByText("Degraded")).toBeTruthy();
    expect(screen.getByText("FORGE-H Resource Posture")).toBeTruthy();
    expect(screen.getByText("Disk")).toBeTruthy();
    expect(screen.getByText("Warnings")).toBeTruthy();
    expect(screen.getByText("proposal_1")).toBeTruthy();
    expect(screen.getByText("Advisory only")).toBeTruthy();
    expect(screen.getByText("execution_1")).toBeTruthy();
    expect(screen.getByText("Proposal: proposal_1")).toBeTruthy();
    expect(screen.getByText("Action: warn_operator")).toBeTruthy();
    expect(screen.getByText("Semantic memory write: no")).toBeTruthy();
    expect(screen.getByText("Modelruntime mutation: no")).toBeTruthy();
    expect(screen.getByText("Side effects: operator_notification")).toBeTruthy();
    expect(screen.getByText("Used")).toBeTruthy();
    expect(screen.getByText("Pending approvals")).toBeTruthy();
    expect(screen.getAllByText("1").length).toBeGreaterThanOrEqual(1);
    expect(screen.getAllByRole("button").map((button) => button.textContent))
      .toEqual(["Refresh"]);
    expect(screen.queryByRole("button", { name: /approve/i })).toBeNull();
    expect(screen.queryByRole("button", { name: /load model/i })).toBeNull();
    expect(screen.queryByRole("button", { name: /unload/i })).toBeNull();
    expect(screen.queryByRole("button", { name: /restart/i })).toBeNull();
    expect(screen.queryByRole("button", { name: /rebuild/i })).toBeNull();
    expect(screen.queryByRole("button", { name: /delete/i })).toBeNull();
  });

  it("renders core unreachable state when the status endpoint is unavailable", async () => {
    mocks.status.mockRejectedValueOnce(new Error("network down"));

    render(
      <MemoryRouter>
        <SystemPage />
      </MemoryRouter>,
    );

    expect(await screen.findByText("Core unreachable")).toBeTruthy();
    expect(screen.getByText(/network down/i)).toBeTruthy();
  });
});
