package kv

import (
	"strings"
	"time"
)

type ManifestInput struct {
	CacheID            string
	CacheMode          CacheMode
	WorkspaceID        string
	CaseID             string
	BundleID           string
	BlockID            string
	SnapshotID         string
	RestoreSeedID      string
	BundleHash         string
	StablePrefixHash   string
	VolatileSuffixHash string
	ModelID            string
	ModelRevision      string
	TokenizerID        string
	TokenizerRevision  string
	ChatTemplateHash   string
	PromptLayoutHash   string
	PolicySchemaHash   string
	SyscallSchemaHash  string
	TokenInputHash     string
	FinalTokenIDsHash  string
	RuntimeBackend     string
	RuntimeVersion     string
	AttentionBackend   string
	RopeConfigHash     string
	KVPrecision        string
	MemoryTier         MemoryTier
	CacheSalt          string
	Status             ManifestStatus
	CreatedAt          time.Time
	JournalRefs        []string
	Metadata           map[string]any
}

type KVCacheManifest struct {
	CacheID            string         `json:"cache_id"`
	CacheMode          CacheMode      `json:"cache_mode"`
	WorkspaceID        string         `json:"workspace_id"`
	CaseID             string         `json:"case_id,omitempty"`
	BundleID           string         `json:"bundle_id"`
	BlockID            string         `json:"block_id,omitempty"`
	SnapshotID         string         `json:"snapshot_id,omitempty"`
	RestoreSeedID      string         `json:"restore_seed_id,omitempty"`
	BundleHash         string         `json:"bundle_hash,omitempty"`
	StablePrefixHash   string         `json:"stable_prefix_hash,omitempty"`
	VolatileSuffixHash string         `json:"volatile_suffix_hash,omitempty"`
	ModelID            string         `json:"model_id"`
	ModelRevision      string         `json:"model_revision"`
	TokenizerID        string         `json:"tokenizer_id"`
	TokenizerRevision  string         `json:"tokenizer_revision"`
	ChatTemplateHash   string         `json:"chat_template_hash"`
	PromptLayoutHash   string         `json:"prompt_layout_hash"`
	PolicySchemaHash   string         `json:"policy_schema_hash"`
	SyscallSchemaHash  string         `json:"syscall_schema_hash"`
	TokenInputHash     string         `json:"token_input_hash"`
	FinalTokenIDsHash  string         `json:"final_token_ids_hash,omitempty"`
	RuntimeBackend     string         `json:"runtime_backend"`
	RuntimeVersion     string         `json:"runtime_version"`
	AttentionBackend   string         `json:"attention_backend"`
	RopeConfigHash     string         `json:"rope_config_hash"`
	KVPrecision        string         `json:"kv_precision"`
	MemoryTier         MemoryTier     `json:"memory_tier"`
	CacheSalt          string         `json:"cache_salt"`
	Status             ManifestStatus `json:"status"`
	ReuseCount         int            `json:"reuse_count"`
	LastUsedAt         *time.Time     `json:"last_used_at,omitempty"`
	InvalidatedAt      *time.Time     `json:"invalidated_at,omitempty"`
	InvalidationReason string         `json:"invalidation_reason,omitempty"`
	CreatedAt          time.Time      `json:"created_at"`
	JournalRefs        []string       `json:"journal_refs,omitempty"`
	Metadata           map[string]any `json:"metadata,omitempty"`
}

type KVLookupRequest struct {
	RequestID          string    `json:"request_id"`
	WorkspaceID        string    `json:"workspace_id"`
	CaseID             string    `json:"case_id,omitempty"`
	BundleID           string    `json:"bundle_id"`
	BlockID            string    `json:"block_id,omitempty"`
	SnapshotID         string    `json:"snapshot_id,omitempty"`
	RestoreSeedID      string    `json:"restore_seed_id,omitempty"`
	BundleHash         string    `json:"bundle_hash,omitempty"`
	StablePrefixHash   string    `json:"stable_prefix_hash,omitempty"`
	VolatileSuffixHash string    `json:"volatile_suffix_hash,omitempty"`
	ModelID            string    `json:"model_id"`
	ModelRevision      string    `json:"model_revision"`
	TokenizerID        string    `json:"tokenizer_id"`
	TokenizerRevision  string    `json:"tokenizer_revision"`
	ChatTemplateHash   string    `json:"chat_template_hash"`
	PromptLayoutHash   string    `json:"prompt_layout_hash"`
	PolicySchemaHash   string    `json:"policy_schema_hash"`
	SyscallSchemaHash  string    `json:"syscall_schema_hash"`
	TokenInputHash     string    `json:"token_input_hash"`
	FinalTokenIDsHash  string    `json:"final_token_ids_hash,omitempty"`
	RuntimeBackend     string    `json:"runtime_backend"`
	RuntimeVersion     string    `json:"runtime_version"`
	AttentionBackend   string    `json:"attention_backend"`
	RopeConfigHash     string    `json:"rope_config_hash"`
	KVPrecision        string    `json:"kv_precision"`
	CacheSalt          string    `json:"cache_salt"`
	CacheMode          CacheMode `json:"cache_mode"`
	CreatedAt          time.Time `json:"created_at"`
}

type KVLookupResult struct {
	Hit              bool             `json:"hit"`
	CacheID          string           `json:"cache_id,omitempty"`
	ValidationResult ValidationResult `json:"validation_result"`
	MissReason       string           `json:"miss_reason,omitempty"`
	FailedGates      []string         `json:"failed_gates,omitempty"`
	Manifest         *KVCacheManifest `json:"manifest,omitempty"`
	CreatedAt        time.Time        `json:"created_at"`
	Metadata         map[string]any   `json:"metadata,omitempty"`
}

type ManifestListFilter struct {
	WorkspaceID string
	CaseID      string
	BundleID    string
	BlockID     string
	CacheMode   CacheMode
	Status      ManifestStatus
	MemoryTier  MemoryTier
}

func NewManifest(input ManifestInput) (KVCacheManifest, error) {
	input = NormalizeManifestInput(input)
	if input.CacheID == "" || input.WorkspaceID == "" || input.BundleID == "" {
		return KVCacheManifest{}, ErrInvalidManifest
	}
	if input.BlockID == "" && input.BundleID == "" {
		return KVCacheManifest{}, ErrInvalidManifest
	}
	if !ValidCacheMode(input.CacheMode) {
		return KVCacheManifest{}, ErrInvalidCacheMode
	}
	if !ValidMemoryTier(input.MemoryTier) {
		return KVCacheManifest{}, ErrInvalidMemoryTier
	}
	if !ValidStatus(input.Status) {
		return KVCacheManifest{}, ErrInvalidStatus
	}
	if input.CacheMode == ModeSnapshotPrefix && input.SnapshotID == "" && input.RestoreSeedID == "" {
		return KVCacheManifest{}, ErrInvalidManifest
	}
	if input.ModelID == "" || input.ModelRevision == "" ||
		input.TokenizerID == "" || input.TokenizerRevision == "" ||
		input.ChatTemplateHash == "" || input.PromptLayoutHash == "" ||
		input.PolicySchemaHash == "" || input.SyscallSchemaHash == "" ||
		input.TokenInputHash == "" || input.RuntimeBackend == "" ||
		input.RuntimeVersion == "" || input.AttentionBackend == "" ||
		input.RopeConfigHash == "" || input.KVPrecision == "" ||
		input.CacheSalt == "" {
		return KVCacheManifest{}, ErrInvalidManifest
	}
	manifest := KVCacheManifest{
		CacheID:            input.CacheID,
		CacheMode:          input.CacheMode,
		WorkspaceID:        input.WorkspaceID,
		CaseID:             input.CaseID,
		BundleID:           input.BundleID,
		BlockID:            input.BlockID,
		SnapshotID:         input.SnapshotID,
		RestoreSeedID:      input.RestoreSeedID,
		BundleHash:         input.BundleHash,
		StablePrefixHash:   input.StablePrefixHash,
		VolatileSuffixHash: input.VolatileSuffixHash,
		ModelID:            input.ModelID,
		ModelRevision:      input.ModelRevision,
		TokenizerID:        input.TokenizerID,
		TokenizerRevision:  input.TokenizerRevision,
		ChatTemplateHash:   input.ChatTemplateHash,
		PromptLayoutHash:   input.PromptLayoutHash,
		PolicySchemaHash:   input.PolicySchemaHash,
		SyscallSchemaHash:  input.SyscallSchemaHash,
		TokenInputHash:     input.TokenInputHash,
		FinalTokenIDsHash:  input.FinalTokenIDsHash,
		RuntimeBackend:     input.RuntimeBackend,
		RuntimeVersion:     input.RuntimeVersion,
		AttentionBackend:   input.AttentionBackend,
		RopeConfigHash:     input.RopeConfigHash,
		KVPrecision:        input.KVPrecision,
		MemoryTier:         input.MemoryTier,
		CacheSalt:          input.CacheSalt,
		Status:             input.Status,
		CreatedAt:          input.CreatedAt,
		JournalRefs:        NormalizeRefs(input.JournalRefs),
		Metadata:           CloneMap(input.Metadata),
	}
	return manifest, nil
}

func NormalizeManifestInput(input ManifestInput) ManifestInput {
	input.CacheID = strings.TrimSpace(input.CacheID)
	input.WorkspaceID = strings.TrimSpace(input.WorkspaceID)
	input.CaseID = strings.TrimSpace(input.CaseID)
	input.BundleID = strings.TrimSpace(input.BundleID)
	input.BlockID = strings.TrimSpace(input.BlockID)
	input.SnapshotID = strings.TrimSpace(input.SnapshotID)
	input.RestoreSeedID = strings.TrimSpace(input.RestoreSeedID)
	input.BundleHash = strings.TrimSpace(input.BundleHash)
	input.StablePrefixHash = strings.TrimSpace(input.StablePrefixHash)
	input.VolatileSuffixHash = strings.TrimSpace(input.VolatileSuffixHash)
	input.ModelID = strings.TrimSpace(input.ModelID)
	input.ModelRevision = strings.TrimSpace(input.ModelRevision)
	input.TokenizerID = strings.TrimSpace(input.TokenizerID)
	input.TokenizerRevision = strings.TrimSpace(input.TokenizerRevision)
	input.ChatTemplateHash = strings.TrimSpace(input.ChatTemplateHash)
	input.PromptLayoutHash = strings.TrimSpace(input.PromptLayoutHash)
	input.PolicySchemaHash = strings.TrimSpace(input.PolicySchemaHash)
	input.SyscallSchemaHash = strings.TrimSpace(input.SyscallSchemaHash)
	input.TokenInputHash = strings.TrimSpace(input.TokenInputHash)
	input.FinalTokenIDsHash = strings.TrimSpace(input.FinalTokenIDsHash)
	input.RuntimeBackend = strings.TrimSpace(input.RuntimeBackend)
	input.RuntimeVersion = strings.TrimSpace(input.RuntimeVersion)
	input.AttentionBackend = strings.TrimSpace(input.AttentionBackend)
	input.RopeConfigHash = strings.TrimSpace(input.RopeConfigHash)
	input.KVPrecision = strings.TrimSpace(input.KVPrecision)
	input.CacheSalt = strings.TrimSpace(input.CacheSalt)
	if input.CacheMode == "" {
		input.CacheMode = DefaultCacheMode
	}
	if input.MemoryTier == "" {
		input.MemoryTier = DefaultTier
	}
	if input.Status == "" {
		input.Status = StatusAvailable
	}
	input.Metadata = CloneMap(input.Metadata)
	return input
}

func NormalizeLookupRequest(request KVLookupRequest) KVLookupRequest {
	request.WorkspaceID = strings.TrimSpace(request.WorkspaceID)
	request.CaseID = strings.TrimSpace(request.CaseID)
	request.BundleID = strings.TrimSpace(request.BundleID)
	request.BlockID = strings.TrimSpace(request.BlockID)
	request.SnapshotID = strings.TrimSpace(request.SnapshotID)
	request.RestoreSeedID = strings.TrimSpace(request.RestoreSeedID)
	request.BundleHash = strings.TrimSpace(request.BundleHash)
	request.StablePrefixHash = strings.TrimSpace(request.StablePrefixHash)
	request.VolatileSuffixHash = strings.TrimSpace(request.VolatileSuffixHash)
	request.ModelID = strings.TrimSpace(request.ModelID)
	request.ModelRevision = strings.TrimSpace(request.ModelRevision)
	request.TokenizerID = strings.TrimSpace(request.TokenizerID)
	request.TokenizerRevision = strings.TrimSpace(request.TokenizerRevision)
	request.ChatTemplateHash = strings.TrimSpace(request.ChatTemplateHash)
	request.PromptLayoutHash = strings.TrimSpace(request.PromptLayoutHash)
	request.PolicySchemaHash = strings.TrimSpace(request.PolicySchemaHash)
	request.SyscallSchemaHash = strings.TrimSpace(request.SyscallSchemaHash)
	request.TokenInputHash = strings.TrimSpace(request.TokenInputHash)
	request.FinalTokenIDsHash = strings.TrimSpace(request.FinalTokenIDsHash)
	request.RuntimeBackend = strings.TrimSpace(request.RuntimeBackend)
	request.RuntimeVersion = strings.TrimSpace(request.RuntimeVersion)
	request.AttentionBackend = strings.TrimSpace(request.AttentionBackend)
	request.RopeConfigHash = strings.TrimSpace(request.RopeConfigHash)
	request.KVPrecision = strings.TrimSpace(request.KVPrecision)
	request.CacheSalt = strings.TrimSpace(request.CacheSalt)
	if request.CacheMode == "" {
		request.CacheMode = DefaultCacheMode
	}
	return request
}

func (m KVCacheManifest) Clone() KVCacheManifest {
	m.JournalRefs = CloneStrings(m.JournalRefs)
	m.Metadata = CloneMap(m.Metadata)
	if m.LastUsedAt != nil {
		last := *m.LastUsedAt
		m.LastUsedAt = &last
	}
	if m.InvalidatedAt != nil {
		invalidated := *m.InvalidatedAt
		m.InvalidatedAt = &invalidated
	}
	return m
}

func (m KVCacheManifest) AllRefs() []string {
	return NormalizeRefs([]string{
		m.BundleID,
		m.BlockID,
		m.SnapshotID,
		m.RestoreSeedID,
		m.BundleHash,
		m.StablePrefixHash,
		m.VolatileSuffixHash,
		m.TokenInputHash,
		m.FinalTokenIDsHash,
	})
}

func (m KVCacheManifest) IsCanonicalTruth() bool   { return false }
func (m KVCacheManifest) IsSemanticEvidence() bool { return false }
func (m KVCacheManifest) IsMemory() bool           { return false }
