package api

import (
	"context"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"forge/projectforge/services/core/internal/aios/controllane"
	"forge/projectforge/services/core/internal/forgeh"
	"forge/projectforge/services/core/internal/hostbridge"
)

const shellSystemStatusProposalLimit = 12

type forgeSystemStatusResponse struct {
	GeneratedAt   time.Time                                   `json:"generated_at"`
	Core          forgeSystemCoreStatus                       `json:"core"`
	ShellSession  forgeSystemShellSession                     `json:"shell_session"`
	HostBridge    forgeSystemHostBridgeStatus                 `json:"hostbridge"`
	ForgeH        forgeSystemForgeHStatus                     `json:"forgeh"`
	Kernel        controllane.ForgeKActivationReadinessReport `json:"kernel_activation"`
	ModelRuntime  forgeSystemModelRuntime                     `json:"modelruntime"`
	Storage       forgeSystemStorageStatus                    `json:"storage"`
	ApprovalQueue forgeSystemApprovalQueue                    `json:"approval_queue"`
	Warnings      []string                                    `json:"warnings,omitempty"`
	Errors        []string                                    `json:"errors,omitempty"`
}

type forgeSystemCoreStatus struct {
	Reachable     bool      `json:"reachable"`
	Service       string    `json:"service"`
	HealthState   string    `json:"health_state"`
	CoreURL       string    `json:"core_url,omitempty"`
	LastRefreshAt time.Time `json:"last_refresh_at"`
}

type forgeSystemShellSession struct {
	ShellMode                     string `json:"shell_mode"`
	DisplayBackend                string `json:"display_backend"`
	CompositorSession             string `json:"compositor_session"`
	SafeMode                      bool   `json:"safe_mode"`
	HostMutationDisabled          bool   `json:"host_mutation_disabled"`
	ModelMutationDisabled         bool   `json:"model_mutation_disabled"`
	SemanticMemoryWriteDisabled   bool   `json:"semantic_memory_write_disabled"`
	ForgeKLiveAuthorityDisabled   bool   `json:"forge_k_live_authority_disabled"`
	ContextCompilerRequiredForLLM bool   `json:"context_compiler_required_for_llm"`
}

type forgeSystemHostBridgeStatus struct {
	Wired             bool      `json:"wired"`
	Reason            string    `json:"reason,omitempty"`
	SnapshotID        string    `json:"snapshot_id,omitempty"`
	CapturedAt        time.Time `json:"captured_at,omitempty"`
	HostIdentity      string    `json:"host_identity,omitempty"`
	Architecture      string    `json:"architecture,omitempty"`
	RAMPressure       string    `json:"ram_pressure"`
	DiskPressure      string    `json:"disk_pressure"`
	GPUAvailable      bool      `json:"gpu_available"`
	ThermalAvailable  bool      `json:"thermal_available"`
	SourceErrorsCount int       `json:"source_errors_count"`
	Degraded          bool      `json:"degraded"`
}

type forgeSystemForgeHStatus struct {
	Wired                   bool                            `json:"wired"`
	Policy                  forgeh.ResourcePolicySnapshot   `json:"policy"`
	Proposals               []forgeh.ResourceActionProposal `json:"proposals"`
	Executions              forgeSystemForgeHExecutions     `json:"executions"`
	AdvisoryOnly            bool                            `json:"advisory_only"`
	CanonicalWriteCommitted bool                            `json:"canonical_write_committed"`
}

type forgeSystemForgeHExecutions struct {
	Available bool                             `json:"available"`
	Reason    string                           `json:"reason,omitempty"`
	Items     []forgeh.ResourceActionExecution `json:"items"`
}

type forgeSystemModelRuntime struct {
	Available        bool     `json:"available"`
	State            string   `json:"state"`
	Backend          string   `json:"backend,omitempty"`
	RuntimeEnabled   bool     `json:"runtime_enabled,omitempty"`
	GPUAware         bool     `json:"gpu_aware,omitempty"`
	MutationDisabled bool     `json:"mutation_disabled"`
	Warnings         []string `json:"warnings,omitempty"`
	Errors           []string `json:"errors,omitempty"`
}

type forgeSystemStorageStatus struct {
	Root           string                   `json:"root"`
	DataDir        string                   `json:"data_dir,omitempty"`
	DBPath         string                   `json:"db_path,omitempty"`
	TruthAuthority string                   `json:"truth_authority"`
	PingOK         bool                     `json:"ping_ok"`
	TotalBytes     uint64                   `json:"total_bytes,omitempty"`
	UsedBytes      uint64                   `json:"used_bytes,omitempty"`
	FreeBytes      uint64                   `json:"free_bytes,omitempty"`
	PressureLevel  string                   `json:"pressure_level"`
	Redis          forgeSystemExternalStore `json:"redis"`
	Qdrant         forgeSystemExternalStore `json:"qdrant"`
}

type forgeSystemExternalStore struct {
	Enabled        bool   `json:"enabled"`
	TruthAuthority bool   `json:"truth_authority"`
	Role           string `json:"role"`
}

type forgeSystemApprovalQueue struct {
	Wired  bool   `json:"wired"`
	Reason string `json:"reason,omitempty"`
}

type shellSystemStatusCommandRunner struct{}

func (shellSystemStatusCommandRunner) LookPath(name string) (string, error) {
	return "", errors.New(name + " disabled for read-only shell system status")
}

func (shellSystemStatusCommandRunner) Run(ctx context.Context, name string, args ...string) (hostbridge.CommandResult, error) {
	return hostbridge.CommandResult{}, errors.New(name + " disabled for read-only shell system status")
}

func (s *Server) handleForgeSystemStatus(w http.ResponseWriter, r *http.Request) {
	now := time.Now().UTC()
	storageRoot := s.shellSystemStorageRoot()
	hostSvc := hostbridge.New(hostbridge.Options{
		StorageRoot: storageRoot,
		Runner:      shellSystemStatusCommandRunner{},
		Now:         func() time.Time { return now },
	})
	snapshot := hostSvc.Snapshot(r.Context())
	policy := forgeh.New(forgeh.Options{Now: func() time.Time { return now }}).Evaluate(snapshot)
	proposals := forgeh.GenerateResourceActionProposals(policy, forgeh.ProposalOptions{
		Now: func() time.Time { return now },
	})
	if len(proposals) > shellSystemStatusProposalLimit {
		proposals = proposals[:shellSystemStatusProposalLimit]
	}

	payload := forgeSystemStatusResponse{
		GeneratedAt: now,
		Core: forgeSystemCoreStatus{
			Reachable:     true,
			Service:       "forge-core",
			HealthState:   "ok",
			CoreURL:       shellSystemCoreURL(r),
			LastRefreshAt: now,
		},
		ShellSession: forgeSystemShellSession{
			ShellMode:                     firstNonEmpty(os.Getenv("FORGE_SHELL_MODE"), "manual"),
			DisplayBackend:                firstNonEmpty(os.Getenv("XDG_SESSION_TYPE"), os.Getenv("WAYLAND_DISPLAY"), "unknown"),
			CompositorSession:             firstNonEmpty(os.Getenv("FORGE_SHELL_COMPOSITOR"), "not reported"),
			SafeMode:                      s.cfg.SafeModeForceCPUOnly,
			HostMutationDisabled:          true,
			ModelMutationDisabled:         true,
			SemanticMemoryWriteDisabled:   true,
			ForgeKLiveAuthorityDisabled:   true,
			ContextCompilerRequiredForLLM: true,
		},
		HostBridge: forgeSystemHostBridgeStatus{
			Wired:             true,
			Reason:            "bounded read-only diagnostics; command-backed probes disabled for shell surface",
			SnapshotID:        snapshot.SnapshotID,
			CapturedAt:        snapshot.CapturedAt,
			HostIdentity:      firstNonEmpty(snapshot.Host.Hostname, snapshot.Host.OSRelease, "unknown"),
			Architecture:      snapshot.Host.Architecture,
			RAMPressure:       snapshot.Memory.PressureLevel,
			DiskPressure:      snapshot.Disk.PressureLevel,
			GPUAvailable:      snapshot.GPU.Available,
			ThermalAvailable:  snapshot.Thermal.Available,
			SourceErrorsCount: len(snapshot.SourceErrors),
			Degraded:          snapshot.Degraded,
		},
		ForgeH: forgeSystemForgeHStatus{
			Wired:        true,
			Policy:       policy,
			Proposals:    proposals,
			AdvisoryOnly: true,
			Executions: forgeSystemForgeHExecutions{
				Available: false,
				Reason:    "bounded execution ledger is not exposed as a live shell authority surface in Phase G6",
				Items:     []forgeh.ResourceActionExecution{},
			},
			CanonicalWriteCommitted: false,
		},
		Kernel:        s.forgeKActivationReadiness(now),
		ModelRuntime:  s.shellSystemModelRuntime(r),
		Storage:       s.shellSystemStorage(snapshot),
		ApprovalQueue: forgeSystemApprovalQueue{Wired: true, Reason: "use governed approvals surface for decisions; G6 status is read-only"},
		Warnings: []string{
			"shell system surface is read-only",
			"host command-backed probes are disabled for this endpoint",
			"FORGE-K remains simulator authority only",
		},
	}
	writeJSON(w, http.StatusOK, payload)
}

func shellSystemCoreURL(r *http.Request) string {
	if r == nil || strings.TrimSpace(r.Host) == "" {
		return ""
	}
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	if forwarded := strings.ToLower(strings.TrimSpace(r.Header.Get("X-Forwarded-Proto"))); forwarded == "http" || forwarded == "https" {
		scheme = forwarded
	}
	return scheme + "://" + r.Host
}

func (s *Server) shellSystemStorageRoot() string {
	if _, err := os.Stat("/forge"); err == nil {
		return "/forge"
	}
	if s != nil && s.cfg.DataDir != "" {
		return s.cfg.DataDir
	}
	return "/forge"
}

func (s *Server) shellSystemStorage(snapshot hostbridge.Snapshot) forgeSystemStorageStatus {
	dataDir := ""
	if s != nil {
		dataDir = s.cfg.DataDir
	}
	dbPath := ""
	if dataDir != "" {
		dbPath = filepath.Join(dataDir, "forge.sqlite")
	}
	pingOK := false
	if s != nil && s.st != nil && s.st.DB != nil {
		pingOK = s.st.DB.Ping() == nil
	}
	return forgeSystemStorageStatus{
		Root:           snapshot.Disk.Path,
		DataDir:        dataDir,
		DBPath:         dbPath,
		TruthAuthority: "sqlite",
		PingOK:         pingOK,
		TotalBytes:     snapshot.Disk.TotalBytes,
		UsedBytes:      snapshot.Disk.UsedBytes,
		FreeBytes:      snapshot.Disk.FreeBytes,
		PressureLevel:  snapshot.Disk.PressureLevel,
		Redis: forgeSystemExternalStore{
			Enabled:        false,
			TruthAuthority: false,
			Role:           "optional cache/coordination only; not live truth authority in G6",
		},
		Qdrant: forgeSystemExternalStore{
			Enabled:        false,
			TruthAuthority: false,
			Role:           "optional vector index only; not live truth authority in G6",
		},
	}
}

func (s *Server) shellSystemModelRuntime(r *http.Request) forgeSystemModelRuntime {
	out := forgeSystemModelRuntime{
		Available:        s != nil && s.modelRuntime != nil,
		State:            "unavailable",
		MutationDisabled: true,
	}
	if s == nil || s.modelRuntime == nil {
		out.Warnings = []string{"modelruntime service is not configured"}
		return out
	}
	meta := modelRuntimeMetaFromRequestAudit(requestAuditMetaForBackup(r, "", "", "", "forge.system.status"))
	health, err := s.modelRuntime.Health(r.Context(), meta)
	if err != nil {
		out.State = "degraded"
		out.Errors = []string{err.Error()}
		return out
	}
	out.State = firstNonEmpty(health.Status, "unknown")
	out.Backend = health.Backend
	out.RuntimeEnabled = health.RuntimeEnabled
	out.GPUAware = health.GPUAware
	out.Warnings = append(out.Warnings, health.DegradedReasons...)
	out.Warnings = append(out.Warnings, health.PolicyWarnings...)
	return out
}
