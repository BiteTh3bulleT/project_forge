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
        cache: {
          available: true,
          cache_hit: true,
          stale: false,
          age_ms: 700,
          read_only: true,
          advisory_only: true,
        },
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
        cache: {
          available: true,
          cache_hit: true,
          stale: false,
          age_ms: 900,
          read_only: true,
          advisory_only: true,
        },
      },
      kernel_activation: {
        phase: "19",
        status: "forge_k_durable_orchestration_live",
        summary:
          "FORGE-K owns live semantic syscall ingress; the existing Control Lane SQLite transaction path is the temporary durable commit adapter.",
        mode: "live_authority_migration",
        live_owner: "forge_k.kernel",
        policy_version: "phase-14f-control-lane-enforcement-v1",
        kernel_runtime_state: "forge_k_orchestration_live_control_lane_sqlite_port",
        closed_validation_lanes: 7,
        total_validation_lanes: 7,
        validation_actions: [
          {
            action: "VALIDATE_KV_IDENTITY",
            capability: "kv.identity.validate",
            registered: true,
            mutating: false,
            approval_possible: false,
            supports_dry_run: true,
            closed: true,
            live_owner: "aios.controllane",
            simulator_authority: false,
            live_kernel_authority: false,
          },
          {
            action: "VALIDATE_REF_SHAPE",
            capability: "ref.shape.validate",
            registered: true,
            mutating: false,
            approval_possible: false,
            supports_dry_run: true,
            closed: true,
            live_owner: "aios.controllane",
            simulator_authority: false,
            live_kernel_authority: false,
          },
          {
            action: "VALIDATE_SOURCE_OBJECT_AUTHORITY",
            capability: "source.object.authority.validate",
            registered: true,
            mutating: false,
            approval_possible: false,
            supports_dry_run: true,
            closed: true,
            live_owner: "aios.controllane",
            simulator_authority: false,
            live_kernel_authority: false,
          },
          {
            action: "VALIDATE_CONTEXT_ATTRIBUTION",
            capability: "context.attribution.validate",
            registered: true,
            mutating: false,
            approval_possible: false,
            supports_dry_run: true,
            closed: true,
            live_owner: "aios.controllane",
            simulator_authority: false,
            live_kernel_authority: false,
          },
        ],
        gates: [
          {
            name: "live_owner_explicit",
            passed: true,
            reason: "live owner remains aios.controllane",
          },
        ],
        authority_ready_gates: 4,
        authority_blocked_gates: 3,
        authority_gates: [
          {
            name: "control_lane_validation_enforcement",
            status: "ready",
            live_owner: "aios.controllane",
            required_for_live_authority: true,
            mutation_authority: false,
            reason: "validation-only Control Lane enforcement is connected",
            next_step: "keep validation-only",
          },
          {
            name: "source_object_authority_lookup",
            status: "ready",
            live_owner: "aios.controllane",
            required_for_live_authority: true,
            mutation_authority: false,
            reason: "source object authority lookup is connected through the live Control Lane read store and fails closed",
            next_step: "keep source-object authority lookup read-only while evidence admission and mutation routing gates are designed",
          },
        ],
        authority_matrix: [
          {
            subsystem: "Courthouse",
            current_status: "ADMISSION_CANDIDATE_ONLY",
            live_owner: "aios.controllane",
            target_owner: "forgek.court",
            feature_flag: "n/a; admission candidate validation only",
            rollback_path: "remove admission candidate validation",
            tests_required: ["admission candidate validation tests"],
            tests_passing: ["Control Lane validation action registry tests"],
            blockers: ["live evidence admission and ruling authority remain disabled"],
            operator_visible: true,
          },
          {
            subsystem: "Context Compiler",
            current_status: "CONTEXT_ATTRIBUTION_VALIDATION_ONLY",
            live_owner: "aios.controllane plus services/core/internal/forgekshadow and legacy COMPILE_CONTEXT paths",
            target_owner: "forgek.contextcompiler",
            feature_flag: "n/a; VALIDATE_CONTEXT_ATTRIBUTION is validation-only",
            rollback_path: "remove context attribution validation",
            tests_required: ["context attribution validation tests"],
            tests_passing: ["context attribution validation tests"],
            blockers: ["live prompt/context assembly remains outside FORGE-K"],
            operator_visible: true,
          },
          {
            subsystem: "Lymphatic Lane",
            current_status: "LYMPHATIC_PROPOSAL_ONLY_ONLINE",
            live_owner: "existing dream/autonomy/maintenance paths",
            target_owner: "forgek.lymphatic",
            feature_flag: "n/a; dry-run metadata only",
            rollback_path: "remove proposal-only lymphatic metadata",
            tests_required: ["cleanup proposal no-execution tests"],
            tests_passing: ["autonomy maintenance dry-run proposal-only tests"],
            blockers: ["FORGE-K Lymphatic Lane does not run live cleanup"],
            operator_visible: true,
          },
        ],
        no_effect: {
          memoryMutation: false,
          runtimeMutation: false,
          modelRuntimeCall: false,
          evidenceAdmission: false,
          contextCompilation: false,
          gatewayExecution: false,
          retrievalExecution: false,
          kernelIngressAuthority: true,
          durableOrchestrationAuthority: true,
          liveAuthorityMigration: true,
        },
        simulator_authority: false,
        live_kernel_ingress_authority: true,
        live_durable_orchestration: true,
        live_kernel_authority: false,
        live_authority_migration: true,
        shadow_authoritative: false,
        mutation_controls_available: false,
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
        cutover_readiness: {
          status: "blocked",
          selected_domain: "none",
          canonical_default: "sqlite",
          requested_backend: "sqlite",
          live_owner: "services/core/internal/store.Open with SQLite at ${FORGE_DATA_DIR}/forge.sqlite",
          target_owner: "future storagebackend Postgres repository adapters",
          ready_for_dual_write: false,
          ready_for_read_compare: false,
          ready_for_cutover_proposal: false,
          postgres_canonical_ready: false,
          redis_truth_authority: false,
          qdrant_truth_authority: false,
          tests_required: ["SQLite baseline tests"],
          tests_passing: [],
          blockers: ["postgres backend not explicitly selected for a cutover proposal"],
          rollback_path: "leave FORGE_STORE_BACKEND unset or set to sqlite",
          no_effect: {
            canonicalDefaultChanged: false,
            dualWriteEnabled: false,
            readSwitchEnabled: false,
          },
        },
      },
      operator_cockpit: {
        available: true,
        live_owner: "forge.system.status",
        target_forge_k_owner: "future FORGE-K operator cockpit",
        mutation_controls_available: false,
        rows: [
          {
            id: "authority_gates",
            label: "Authority gates",
            live: true,
            status: "4 ready / 3 blocked",
            live_owner: "aios.controllane",
            target_owner: "FORGE-K Kernel",
            source: "kernel_activation.authority_gates",
            mutation_allowed: false,
          },
          {
            id: "cases",
            label: "Cases",
            live: false,
            status: "unavailable",
            live_owner: "not live-wired",
            target_owner: "FORGE-K case packets",
            source: "planned Courthouse case surface",
            mutation_allowed: false,
          },
          {
            id: "context_bundles",
            label: "Context bundles",
            live: false,
            status: "unavailable",
            live_owner: "context snapshots/restore inspector",
            target_owner: "FORGE-K Context Compiler",
            source: "/inspectors",
            mutation_allowed: false,
          },
          {
            id: "proposals",
            label: "Proposals",
            live: true,
            status: "1 resource proposals",
            live_owner: "forgeh plus autonomy dry-run reports",
            target_owner: "proposal lanes",
            source: "forgeh.proposals",
            mutation_allowed: false,
          },
          {
            id: "journal_replay",
            label: "Journal/replay",
            live: false,
            status: "unavailable",
            live_owner: "audit/journal trace APIs",
            target_owner: "FORGE-K Kernel replay",
            source: "/inspectors",
            mutation_allowed: false,
          },
          {
            id: "lymphatic_reports",
            label: "Lymphatic reports",
            live: false,
            status: "unavailable",
            live_owner: "autonomy maintenance dry-run reports",
            target_owner: "FORGE-K Lymphatic Lane",
            source: "autonomy maintenance dry-run",
            mutation_allowed: false,
          },
        ],
      },
      authority: {
        matrix_available: true,
        matrix_rows: 18,
        live_authority_rows: 12,
        partial_validation_rows: 7,
        forge_k_live_authority_rows: 0,
        host_mutation_rows: 0,
        semantic_memory_write_rows: 4,
        modelruntime_gateway_alignment: "aligned_modelruntime_owned",
        model_delete_file_status: "real",
        model_chat_owner: "modelruntime",
        model_generate_owner: "modelruntime",
        rows: [
          {
            id: "model.delete_file",
            surface: "api.modelruntime",
            method: "POST",
            route: "/forge/models/{id}/delete-file",
            action: "model.delete_file",
            authorityOwner: "modelruntime",
            capabilityId: "model.delete_file",
            gatewayCapabilityStatus: "approval_only",
            mutating: true,
            destructive: true,
            requiresApproval: true,
            approvalMechanism: "modelruntime_management",
            liveAuthority: true,
            forgeKAuthority: false,
            hostMutation: false,
            modelruntimeMutation: true,
            semanticMemoryWrite: false,
            status: "real",
            notes: "Deletes a managed model artifact through approval-governed modelruntime management.",
          },
          {
            id: "system.status",
            surface: "api.system_status",
            method: "GET",
            route: "/forge/system/status",
            action: "system.status.read",
            authorityOwner: "system_status",
            capabilityId: "observability.get_metrics",
            gatewayCapabilityStatus: "not_applicable",
            mutating: false,
            destructive: false,
            requiresApproval: false,
            approvalMechanism: "none",
            liveAuthority: true,
            forgeKAuthority: false,
            hostMutation: false,
            modelruntimeMutation: false,
            semanticMemoryWrite: false,
            status: "read_only",
            notes: "System cockpit/status aggregation is read-only.",
          },
        ],
        blockers: [
          {
            row_id: "memory.observations.write",
            status: "legacy_gate",
            reason: "Legacy memory mutation remains gated.",
            next_step: "route canonical semantic writes through Control Lane",
          },
        ],
        notes: [
          "Gateway owns tool execution; modelruntime owns model inference and lifecycle governance",
        ],
      },
      control_lane: {
        approval_fingerprint: {
          available: true,
          version: "control_lane_approval_fingerprint.v1",
          enforcement_wired: false,
          deterministic_helper: true,
          reason: "approval fingerprint seam is deterministic",
        },
      },
      validation: {
        available: true,
        status: "passing",
        source: "desktop",
        latency_measured: false,
        reason: "static validation evidence",
        commands: [
          {
            command: "npm -w @forge/desktop run test -- src/pages/SystemPage.test.tsx",
            result: "PASS",
          },
        ],
      },
      approval_queue: {
        wired: true,
        available: true,
        pending_count: 1,
        read_only: true,
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
    expect(screen.getByText("FORGE-K Activation Readiness")).toBeTruthy();
    expect(screen.getByText("Authority Matrix")).toBeTruthy();
    expect(screen.getAllByText("model.delete_file").length).toBeGreaterThanOrEqual(1);
    expect(screen.getAllByText("real").length).toBeGreaterThanOrEqual(1);
    expect(screen.getByText("Authority Blockers")).toBeTruthy();
    expect(screen.getByText("Authority Rows")).toBeTruthy();
    expect(screen.getByText("/forge/models/{id}/delete-file")).toBeTruthy();
    expect(screen.getByText("modelruntime_management")).toBeTruthy();
    expect(screen.getByText("mutating, destructive, approval, modelruntime")).toBeTruthy();
    expect(screen.getByText("/forge/system/status")).toBeTruthy();
    expect(screen.getAllByText("read-only").length).toBeGreaterThanOrEqual(1);
    expect(screen.getByText("Next: route canonical semantic writes through Control Lane")).toBeTruthy();
    expect(screen.getByText("model.chat owner")).toBeTruthy();
    expect(screen.getByText("model.generate owner")).toBeTruthy();
    expect(screen.getAllByText("modelruntime").length).toBeGreaterThanOrEqual(2);
    expect(screen.getByText("aligned_modelruntime_owned")).toBeTruthy();
    expect(screen.getByText("Gateway owns tool execution; modelruntime owns model inference and lifecycle governance")).toBeTruthy();
    expect(screen.getByText("Control Lane Fingerprint Seam")).toBeTruthy();
    expect(screen.getByText("control_lane_approval_fingerprint.v1")).toBeTruthy();
    expect(screen.getByText("approval fingerprint seam is deterministic")).toBeTruthy();
    expect(screen.getByText("Enforcement wired")).toBeTruthy();
    expect(screen.getByText("Operator Cockpit Index")).toBeTruthy();
    expect(screen.getByText("Authority gates")).toBeTruthy();
    expect(screen.getByText("4 ready / 3 blocked")).toBeTruthy();
    expect(screen.getByText("Cases")).toBeTruthy();
    expect(screen.getByText("FORGE-K case packets")).toBeTruthy();
    expect(screen.getByText("Context bundles")).toBeTruthy();
    expect(screen.getByText("context snapshots/restore inspector")).toBeTruthy();
    expect(screen.getByText("Proposals")).toBeTruthy();
    expect(screen.getByText("1 resource proposals")).toBeTruthy();
    expect(screen.getByText("Journal/replay")).toBeTruthy();
    expect(screen.getByText("audit/journal trace APIs")).toBeTruthy();
    expect(screen.getByText("Lymphatic reports")).toBeTruthy();
    expect(screen.getByText("autonomy maintenance dry-run reports")).toBeTruthy();
    expect(screen.getAllByText("forge_k_durable_orchestration_live").length).toBeGreaterThanOrEqual(1);
    expect(screen.getByText("7/7")).toBeTruthy();
    expect(screen.getByText("Simulator authority disabled")).toBeTruthy();
    expect(screen.getByText("FORGE-K syscall ingress live")).toBeTruthy();
    expect(screen.getByText("FORGE-K durable orchestration live")).toBeTruthy();
    expect(screen.getByText("Full FORGE-K authority still gated")).toBeTruthy();
    expect(screen.getByText("Live authority migration active")).toBeTruthy();
    expect(screen.getByText("Mutation controls absent")).toBeTruthy();
    expect(screen.getByText("Kernel Authority Gates")).toBeTruthy();
    expect(screen.getByText("Ready: 4")).toBeTruthy();
    expect(screen.getByText("Blocked: 3")).toBeTruthy();
    expect(screen.getByText("source_object_authority_lookup")).toBeTruthy();
    expect(screen.getByText("keep source-object authority lookup read-only while evidence admission and mutation routing gates are designed")).toBeTruthy();
    expect(screen.getByText("FORGE-K Subsystem Cockpit")).toBeTruthy();
    expect(screen.getByText("Courthouse")).toBeTruthy();
    expect(screen.getByText("ADMISSION_CANDIDATE_ONLY")).toBeTruthy();
    expect(screen.getByText("Context Compiler")).toBeTruthy();
    expect(screen.getByText("CONTEXT_ATTRIBUTION_VALIDATION_ONLY")).toBeTruthy();
    expect(screen.getByText("Lymphatic Lane")).toBeTruthy();
    expect(screen.getByText("LYMPHATIC_PROPOSAL_ONLY_ONLINE")).toBeTruthy();
    expect(screen.getByText("FORGE-K Lymphatic Lane does not run live cleanup")).toBeTruthy();
    expect(screen.getByText("VALIDATE_KV_IDENTITY")).toBeTruthy();
    expect(screen.getByText("VALIDATE_SOURCE_OBJECT_AUTHORITY")).toBeTruthy();
    expect(screen.getByText("VALIDATE_CONTEXT_ATTRIBUTION")).toBeTruthy();
    expect(screen.getByText("ref.shape.validate")).toBeTruthy();
    expect(screen.queryByRole("button", { name: /approve/i })).toBeNull();
    expect(screen.queryByRole("button", { name: /reject/i })).toBeNull();
    expect(screen.queryByRole("button", { name: /execute/i })).toBeNull();
    expect(screen.getByText("Degraded")).toBeTruthy();
    expect(screen.getAllByText("Cache state").length).toBeGreaterThanOrEqual(2);
    expect(screen.getAllByText("fresh").length).toBeGreaterThanOrEqual(2);
    expect(screen.getByText("700 ms")).toBeTruthy();
    expect(screen.getByText("900 ms")).toBeTruthy();
    expect(screen.getByText("FORGE-H Resource Posture")).toBeTruthy();
    expect(screen.getByText("Disk")).toBeTruthy();
    expect(screen.getByText("Warnings")).toBeTruthy();
    expect(screen.getByText("proposal_1")).toBeTruthy();
    expect(screen.getAllByText("Advisory only").length).toBeGreaterThanOrEqual(1);
    expect(screen.getByText("execution_1")).toBeTruthy();
    expect(screen.getByText("Proposal: proposal_1")).toBeTruthy();
    expect(screen.getByText("Action: warn_operator")).toBeTruthy();
    expect(screen.getByText("Semantic memory write: no")).toBeTruthy();
    expect(screen.getByText("Modelruntime mutation: no")).toBeTruthy();
    expect(screen.getByText("Side effects: operator_notification")).toBeTruthy();
    expect(screen.getByText("Used")).toBeTruthy();
    expect(screen.getByText("Storage Cutover Readiness")).toBeTruthy();
    expect(screen.getByText("Canonical default")).toBeTruthy();
    expect(screen.getByText("postgres backend not explicitly selected for a cutover proposal")).toBeTruthy();
    expect(screen.getByText("Pending approvals")).toBeTruthy();
    expect(screen.getAllByText("1").length).toBeGreaterThanOrEqual(1);
    expect(screen.getByText("Latest Validation Evidence")).toBeTruthy();
    expect(screen.getByText("passing")).toBeTruthy();
    expect(screen.getByText("npm -w @forge/desktop run test -- src/pages/SystemPage.test.tsx")).toBeTruthy();
    expect(screen.getByText("PASS")).toBeTruthy();
    expect(screen.getAllByRole("button").map((button) => button.textContent))
      .toEqual(["Refresh"]);
    for (const label of [
      /restart/i,
      /shutdown/i,
      /rebuild/i,
      /systemctl/i,
      /nixos-rebuild/i,
      /^load$/i,
      /unload/i,
      /approve/i,
      /reject/i,
      /^delete$/i,
      /cleanup/i,
    ]) {
      expect(screen.queryByRole("button", { name: label })).toBeNull();
    }
  });

  it("renders stale cache and missing optional cockpit data honestly", async () => {
    mocks.approvals.mockRejectedValueOnce(new Error("approval route absent"));
    mocks.status.mockResolvedValueOnce({
      generated_at: "2026-05-09T12:00:00Z",
      core: {
        reachable: true,
        service: "forge-core",
        health_state: "ok",
      },
      shell_session: {
        host_mutation_disabled: true,
        model_mutation_disabled: true,
        semantic_memory_write_disabled: true,
        forge_k_live_authority_disabled: true,
      },
      hostbridge: {
        wired: true,
        reason: "bounded read-only diagnostics",
        ram_pressure: "unknown",
        disk_pressure: "unknown",
        cache: {
          available: true,
          cache_hit: true,
          stale: true,
          age_ms: 125000,
          read_only: true,
          advisory_only: true,
        },
      },
      forgeh: {
        wired: true,
        advisory_only: true,
        canonical_write_committed: false,
        cache: {
          available: true,
          cache_hit: false,
          stale: true,
          age_ms: 3660000,
          read_only: true,
          advisory_only: true,
        },
      },
      kernel_activation: {
        status: "unknown",
        validation_actions: [],
        authority_gates: [],
      },
      modelruntime: {
        available: false,
        state: "unavailable",
        mutation_disabled: true,
      },
      storage: {
        truth_authority: "sqlite",
        ping_ok: false,
        pressure_level: "unknown",
        redis: { enabled: false, truth_authority: false, role: "optional cache" },
        qdrant: { enabled: false, truth_authority: false, role: "optional vector index" },
      },
    });

    render(
      <MemoryRouter>
        <SystemPage />
      </MemoryRouter>,
    );

    expect(await screen.findByRole("heading", { name: "System Surfaces" }))
      .toBeTruthy();
    expect(screen.getAllByText("stale").length).toBeGreaterThanOrEqual(2);
    expect(screen.getByText("2m")).toBeTruthy();
    expect(screen.getByText("1h")).toBeTruthy();
    expect(screen.getByText("fingerprint seam status unavailable")).toBeTruthy();
    expect(screen.getByText("validation evidence not wired")).toBeTruthy();
    expect(screen.getByText("Approval queue surface not wired yet")).toBeTruthy();
    expect(screen.getByText("approval route absent")).toBeTruthy();
    expect(screen.getAllByRole("button").map((button) => button.textContent))
      .toEqual(["Refresh"]);
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
