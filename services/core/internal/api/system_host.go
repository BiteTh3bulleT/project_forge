package api

import (
	"net/http"
	"os"
	"time"

	"forge/projectforge/services/core/internal/hostbridge"
)

type forgeSystemHostResponse struct {
	GeneratedAt      time.Time                      `json:"generated_at"`
	LiveOwner        string                         `json:"live_owner"`
	ReadOnly         bool                           `json:"read_only"`
	MutationDisabled bool                           `json:"mutation_disabled"`
	Host             hostbridge.HostDiagnostics     `json:"host"`
	Kernel           hostbridge.KernelDiagnostics   `json:"kernel"`
	Boot             hostbridge.BootDiagnostics     `json:"boot"`
	CPU              hostbridge.CPUDiagnostics      `json:"cpu"`
	Memory           hostbridge.MemoryDiagnostics   `json:"memory"`
	Storage          forgeSystemHostStorage         `json:"storage"`
	GPU              hostbridge.GPUDiagnostics      `json:"gpu"`
	Thermal          hostbridge.ThermalDiagnostics  `json:"thermal"`
	Display          forgeSystemHostReadOnlySection `json:"display"`
	Audio            forgeSystemHostReadOnlySection `json:"audio"`
	Network          forgeSystemHostReadOnlySection `json:"network"`
	Power            forgeSystemHostReadOnlySection `json:"power"`
	Session          forgeSystemHostSession         `json:"session"`
	Config           forgeSystemHostConfig          `json:"config"`
	SourceErrors     []hostbridge.SourceError       `json:"source_errors,omitempty"`
	Warnings         []string                       `json:"warnings,omitempty"`
}

type forgeSystemHostStorage struct {
	Root          string `json:"root"`
	TotalBytes    uint64 `json:"total_bytes,omitempty"`
	UsedBytes     uint64 `json:"used_bytes,omitempty"`
	FreeBytes     uint64 `json:"free_bytes,omitempty"`
	PressureLevel string `json:"pressure_level"`
}

type forgeSystemHostReadOnlySection struct {
	Status           string `json:"status"`
	Reason           string `json:"reason"`
	ReadOnly         bool   `json:"read_only"`
	MutationDisabled bool   `json:"mutation_disabled"`
}

type forgeSystemHostSession struct {
	ShellMode                       string `json:"shell_mode"`
	DisplayBackend                  string `json:"display_backend"`
	CompositorSession               string `json:"compositor_session"`
	SafeMode                        bool   `json:"safe_mode"`
	HostMutationDisabled            bool   `json:"host_mutation_disabled"`
	ModelMutationDisabled           bool   `json:"model_mutation_disabled"`
	SemanticMemoryWriteDisabled     bool   `json:"semantic_memory_write_disabled"`
	ShellCannotClaimKernelAuthority bool   `json:"shell_cannot_claim_kernel_authority"`
	ContextCompilerRequiredForLLM   bool   `json:"context_compiler_required_for_llm"`
}

type forgeSystemHostConfig struct {
	DataDir                          string `json:"data_dir,omitempty"`
	WorkspaceDir                     string `json:"workspace_dir,omitempty"`
	StoreBackend                     string `json:"store_backend,omitempty"`
	EnableModelRuntime               bool   `json:"enable_model_runtime"`
	GPUEnabled                       bool   `json:"gpu_enabled"`
	SafeModeForceCPUOnly             bool   `json:"safe_mode_force_cpu_only"`
	ModelRuntimeAllowCloudModels     bool   `json:"modelruntime_allow_cloud_models"`
	ModelPolicyRequireExplicitLoad   bool   `json:"model_policy_require_explicit_load"`
	ModelPolicyAllowAutoLoad         bool   `json:"model_policy_allow_auto_load"`
	ModelPolicyRequireWorkspaceScope bool   `json:"model_policy_require_workspace_scope"`
}

func (s *Server) handleForgeSystemHost(w http.ResponseWriter, r *http.Request) {
	now := time.Now().UTC()
	hostSvc := hostbridge.New(hostbridge.Options{
		StorageRoot: s.shellSystemStorageRoot(),
		Runner:      shellSystemStatusCommandRunner{},
		Now:         func() time.Time { return now },
	})
	snapshot := hostSvc.Snapshot(r.Context())
	payload := forgeSystemHostResponse{
		GeneratedAt:      now,
		LiveOwner:        "forge.system.host",
		ReadOnly:         true,
		MutationDisabled: true,
		Host:             snapshot.Host,
		Kernel:           snapshot.Kernel,
		Boot:             snapshot.Boot,
		CPU:              snapshot.CPU,
		Memory:           snapshot.Memory,
		Storage: forgeSystemHostStorage{
			Root:          snapshot.Disk.Path,
			TotalBytes:    snapshot.Disk.TotalBytes,
			UsedBytes:     snapshot.Disk.UsedBytes,
			FreeBytes:     snapshot.Disk.FreeBytes,
			PressureLevel: snapshot.Disk.PressureLevel,
		},
		GPU:     snapshot.GPU,
		Thermal: snapshot.Thermal,
		Display: forgeSystemHostReadOnlySection{
			Status:           "read_only",
			Reason:           "display topology is visible through shell diagnostics; apply/arrange controls wait for compositor output-management support",
			ReadOnly:         true,
			MutationDisabled: true,
		},
		Audio: forgeSystemHostReadOnlySection{
			Status:           "not_wired",
			Reason:           "audio control is not a live shell authority surface yet",
			ReadOnly:         true,
			MutationDisabled: true,
		},
		Network: forgeSystemHostReadOnlySection{
			Status:           "not_wired",
			Reason:           "network mutation and service control remain outside this read-only settings endpoint",
			ReadOnly:         true,
			MutationDisabled: true,
		},
		Power: forgeSystemHostReadOnlySection{
			Status:           "policy_gated",
			Reason:           "host power actions are handled only by the explicit desktop shell policy gate",
			ReadOnly:         true,
			MutationDisabled: true,
		},
		Session: forgeSystemHostSession{
			ShellMode:                       firstNonEmpty(os.Getenv("FORGE_SHELL_MODE"), "manual"),
			DisplayBackend:                  firstNonEmpty(os.Getenv("XDG_SESSION_TYPE"), os.Getenv("WAYLAND_DISPLAY"), "unknown"),
			CompositorSession:               firstNonEmpty(os.Getenv("FORGE_SHELL_COMPOSITOR"), "not reported"),
			SafeMode:                        s != nil && s.cfg.SafeModeForceCPUOnly,
			HostMutationDisabled:            true,
			ModelMutationDisabled:           true,
			SemanticMemoryWriteDisabled:     true,
			ShellCannotClaimKernelAuthority: true,
			ContextCompilerRequiredForLLM:   true,
		},
		Config:       s.forgeSystemHostConfig(),
		SourceErrors: snapshot.SourceErrors,
		Warnings: []string{
			"host settings surface is read-only",
			"display apply, audio control, network mutation, and power mutation are not exposed by this endpoint",
		},
	}
	writeJSON(w, http.StatusOK, payload)
}

func (s *Server) forgeSystemHostConfig() forgeSystemHostConfig {
	if s == nil {
		return forgeSystemHostConfig{}
	}
	return forgeSystemHostConfig{
		DataDir:                          s.cfg.DataDir,
		WorkspaceDir:                     s.cfg.WorkspaceDir,
		StoreBackend:                     s.cfg.StoreBackend,
		EnableModelRuntime:               s.cfg.EnableModelRuntime,
		GPUEnabled:                       s.cfg.GPUEnabled,
		SafeModeForceCPUOnly:             s.cfg.SafeModeForceCPUOnly,
		ModelRuntimeAllowCloudModels:     s.cfg.ModelRuntimeAllowOllamaCloudModels,
		ModelPolicyRequireExplicitLoad:   s.cfg.ModelPolicyRequireExplicitLoad,
		ModelPolicyAllowAutoLoad:         s.cfg.ModelPolicyAllowAutoLoad,
		ModelPolicyRequireWorkspaceScope: s.cfg.ModelPolicyRequireWorkspaceScope,
	}
}
