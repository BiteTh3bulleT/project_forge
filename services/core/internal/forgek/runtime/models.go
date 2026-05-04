package runtime

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"sort"
	"strings"
	"time"
)

type RuntimeDriverManifest struct {
	DriverID                   string         `json:"driver_id"`
	DriverName                 string         `json:"driver_name"`
	DriverKind                 DriverKind     `json:"driver_kind"`
	Version                    string         `json:"version"`
	RuntimeBackend             string         `json:"runtime_backend"`
	RuntimeVersion             string         `json:"runtime_version"`
	SupportedModels            []string       `json:"supported_models,omitempty"`
	SupportedCapabilities      []string       `json:"supported_capabilities,omitempty"`
	SupportsStreaming          bool           `json:"supports_streaming"`
	SupportsToolCalling        bool           `json:"supports_tool_calling"`
	SupportsStructuredOutputs  bool           `json:"supports_structured_outputs"`
	SupportsPrefixCache        bool           `json:"supports_prefix_cache"`
	SupportsPagedKV            bool           `json:"supports_paged_kv"`
	SupportsKVQuantization     bool           `json:"supports_kv_quantization"`
	SupportsKVOffload          bool           `json:"supports_kv_offload"`
	SupportsPriorityEviction   bool           `json:"supports_priority_eviction"`
	SupportsCacheSalt          bool           `json:"supports_cache_salt"`
	SupportsNonPrefixReuse     bool           `json:"supports_non_prefix_reuse"`
	SupportsCrossInstanceReuse bool           `json:"supports_cross_instance_reuse"`
	DeterministicForTests      bool           `json:"deterministic_for_tests"`
	AuthorityLevel             string         `json:"authority_level"`
	CreatedAt                  time.Time      `json:"created_at"`
	Metadata                   map[string]any `json:"metadata,omitempty"`
}

type RuntimeCapabilityManifest struct {
	RuntimeID                  string         `json:"runtime_id"`
	RuntimeVersion             string         `json:"runtime_version"`
	ModelID                    string         `json:"model_id"`
	ModelRevision              string         `json:"model_revision"`
	TokenizerID                string         `json:"tokenizer_id"`
	TokenizerRevision          string         `json:"tokenizer_revision"`
	ChatTemplateHash           string         `json:"chat_template_hash"`
	SupportsPrefixCache        bool           `json:"supports_prefix_cache"`
	SupportsPagedKV            bool           `json:"supports_paged_kv"`
	SupportsKVQuantization     bool           `json:"supports_kv_quantization"`
	SupportsKVOffload          bool           `json:"supports_kv_offload"`
	SupportsPriorityEviction   bool           `json:"supports_priority_eviction"`
	SupportsCacheSalt          bool           `json:"supports_cache_salt"`
	SupportsNonPrefixReuse     bool           `json:"supports_non_prefix_reuse"`
	SupportsCrossInstanceReuse bool           `json:"supports_cross_instance_reuse"`
	SupportsStructuredOutputs  bool           `json:"supports_structured_outputs"`
	SupportsToolCalling        bool           `json:"supports_tool_calling"`
	MaxContextTokens           int            `json:"max_context_tokens"`
	MaxOutputTokens            int            `json:"max_output_tokens"`
	Metadata                   map[string]any `json:"metadata,omitempty"`
}

type RuntimeGenerateRequest struct {
	RequestID              string         `json:"request_id"`
	DriverID               string         `json:"driver_id"`
	WorkspaceID            string         `json:"workspace_id"`
	CaseID                 string         `json:"case_id,omitempty"`
	BundleID               string         `json:"bundle_id,omitempty"`
	ContextBlockRefs       []string       `json:"context_block_refs,omitempty"`
	CanonicalPromptText    string         `json:"canonical_prompt_text,omitempty"`
	ModelID                string         `json:"model_id"`
	ModelRevision          string         `json:"model_revision"`
	TokenizerID            string         `json:"tokenizer_id"`
	TokenizerRevision      string         `json:"tokenizer_revision"`
	ChatTemplateHash       string         `json:"chat_template_hash"`
	PromptLayoutHash       string         `json:"prompt_layout_hash"`
	PolicySchemaHash       string         `json:"policy_schema_hash"`
	SyscallSchemaHash      string         `json:"syscall_schema_hash"`
	TokenInputHash         string         `json:"token_input_hash"`
	KVLookupID             string         `json:"kv_lookup_id,omitempty"`
	KVCacheID              string         `json:"kv_cache_id,omitempty"`
	MaxOutputTokens        int            `json:"max_output_tokens"`
	Temperature            float64        `json:"temperature"`
	StructuredOutputSchema map[string]any `json:"structured_output_schema,omitempty"`
	RequestedBy            string         `json:"requested_by"`
	CreatedAt              time.Time      `json:"created_at"`
	Metadata               map[string]any `json:"metadata,omitempty"`
}

type RuntimeGenerateResult struct {
	ResultID              string         `json:"result_id"`
	RequestID             string         `json:"request_id"`
	DriverID              string         `json:"driver_id"`
	WorkspaceID           string         `json:"workspace_id"`
	CaseID                string         `json:"case_id,omitempty"`
	BundleID              string         `json:"bundle_id,omitempty"`
	KVLookupID            string         `json:"kv_lookup_id,omitempty"`
	KVCacheID             string         `json:"kv_cache_id,omitempty"`
	ModelID               string         `json:"model_id"`
	OutputText            string         `json:"output_text,omitempty"`
	OutputJSON            map[string]any `json:"output_json,omitempty"`
	FinishReason          FinishReason   `json:"finish_reason"`
	PromptTokenEstimate   int            `json:"prompt_token_estimate"`
	OutputTokenEstimate   int            `json:"output_token_estimate"`
	RuntimeMetadata       map[string]any `json:"runtime_metadata,omitempty"`
	Warnings              []string       `json:"warnings,omitempty"`
	Error                 string         `json:"error,omitempty"`
	CreatedAt             time.Time      `json:"created_at"`
	ProvenanceRefs        []string       `json:"provenance_refs,omitempty"`
	ProposalObjectRef     string         `json:"proposal_object_ref,omitempty"`
	JournalRefs           []string       `json:"journal_refs,omitempty"`
	AuthorityLevel        string         `json:"authority_level"`
	IsCanonicalTruth      bool           `json:"is_canonical_truth"`
	IsAdmittedEvidence    bool           `json:"is_admitted_evidence"`
	IsModelDriverProposal bool           `json:"is_model_driver_proposal"`
}

type RuntimeHealth struct {
	DriverID  string              `json:"driver_id"`
	Status    RuntimeHealthStatus `json:"status"`
	Message   string              `json:"message,omitempty"`
	CheckedAt time.Time           `json:"checked_at"`
	Metadata  map[string]any      `json:"metadata,omitempty"`
}

func NewRuntimeDriverManifest(manifest RuntimeDriverManifest) (RuntimeDriverManifest, error) {
	manifest = NormalizeRuntimeDriverManifest(manifest)
	if manifest.DriverID == "" || manifest.DriverName == "" || manifest.Version == "" ||
		manifest.RuntimeBackend == "" || manifest.RuntimeVersion == "" {
		return RuntimeDriverManifest{}, ErrInvalidDriverManifest
	}
	if !ValidDriverKind(manifest.DriverKind) {
		return RuntimeDriverManifest{}, ErrInvalidDriverKind
	}
	if manifest.AuthorityLevel == "" {
		manifest.AuthorityLevel = RuntimeAuthorityProposalOnly
	}
	if manifest.AuthorityLevel != RuntimeAuthorityProposalOnly {
		return RuntimeDriverManifest{}, ErrInvalidDriverManifest
	}
	if containsSecretKey(manifest.Metadata) {
		return RuntimeDriverManifest{}, ErrSecretInManifest
	}
	return manifest, nil
}

func ValidateCapabilityManifest(capability RuntimeCapabilityManifest) error {
	capability = NormalizeCapabilityManifest(capability)
	if capability.RuntimeID == "" || capability.RuntimeVersion == "" || capability.ModelID == "" ||
		capability.ModelRevision == "" || capability.TokenizerID == "" ||
		capability.TokenizerRevision == "" || capability.ChatTemplateHash == "" {
		return ErrInvalidCapabilityManifest
	}
	return nil
}

func ValidateGenerateRequest(request RuntimeGenerateRequest) error {
	request = NormalizeGenerateRequest(request)
	if request.RequestID == "" || request.DriverID == "" || request.WorkspaceID == "" {
		return ErrInvalidGenerateRequest
	}
	if request.BundleID == "" && request.CanonicalPromptText == "" {
		return ErrInvalidGenerateRequest
	}
	if request.ModelID == "" || request.ModelRevision == "" || request.TokenizerID == "" ||
		request.TokenizerRevision == "" || request.ChatTemplateHash == "" ||
		request.PromptLayoutHash == "" || request.PolicySchemaHash == "" ||
		request.SyscallSchemaHash == "" || request.TokenInputHash == "" {
		return ErrInvalidGenerateRequest
	}
	return nil
}

func NormalizeRuntimeDriverManifest(manifest RuntimeDriverManifest) RuntimeDriverManifest {
	manifest.DriverID = strings.TrimSpace(manifest.DriverID)
	manifest.DriverName = strings.TrimSpace(manifest.DriverName)
	manifest.Version = strings.TrimSpace(manifest.Version)
	manifest.RuntimeBackend = strings.TrimSpace(manifest.RuntimeBackend)
	manifest.RuntimeVersion = strings.TrimSpace(manifest.RuntimeVersion)
	manifest.SupportedModels = NormalizeRefs(manifest.SupportedModels)
	manifest.SupportedCapabilities = NormalizeRefs(manifest.SupportedCapabilities)
	manifest.Metadata = CloneMap(manifest.Metadata)
	return manifest
}

func NormalizeCapabilityManifest(capability RuntimeCapabilityManifest) RuntimeCapabilityManifest {
	capability.RuntimeID = strings.TrimSpace(capability.RuntimeID)
	capability.RuntimeVersion = strings.TrimSpace(capability.RuntimeVersion)
	capability.ModelID = strings.TrimSpace(capability.ModelID)
	capability.ModelRevision = strings.TrimSpace(capability.ModelRevision)
	capability.TokenizerID = strings.TrimSpace(capability.TokenizerID)
	capability.TokenizerRevision = strings.TrimSpace(capability.TokenizerRevision)
	capability.ChatTemplateHash = strings.TrimSpace(capability.ChatTemplateHash)
	capability.Metadata = CloneMap(capability.Metadata)
	return capability
}

func NormalizeGenerateRequest(request RuntimeGenerateRequest) RuntimeGenerateRequest {
	request.RequestID = strings.TrimSpace(request.RequestID)
	request.DriverID = strings.TrimSpace(request.DriverID)
	request.WorkspaceID = strings.TrimSpace(request.WorkspaceID)
	request.CaseID = strings.TrimSpace(request.CaseID)
	request.BundleID = strings.TrimSpace(request.BundleID)
	request.ContextBlockRefs = NormalizeRefs(request.ContextBlockRefs)
	request.ModelID = strings.TrimSpace(request.ModelID)
	request.ModelRevision = strings.TrimSpace(request.ModelRevision)
	request.TokenizerID = strings.TrimSpace(request.TokenizerID)
	request.TokenizerRevision = strings.TrimSpace(request.TokenizerRevision)
	request.ChatTemplateHash = strings.TrimSpace(request.ChatTemplateHash)
	request.PromptLayoutHash = strings.TrimSpace(request.PromptLayoutHash)
	request.PolicySchemaHash = strings.TrimSpace(request.PolicySchemaHash)
	request.SyscallSchemaHash = strings.TrimSpace(request.SyscallSchemaHash)
	request.TokenInputHash = strings.TrimSpace(request.TokenInputHash)
	request.KVLookupID = strings.TrimSpace(request.KVLookupID)
	request.KVCacheID = strings.TrimSpace(request.KVCacheID)
	request.RequestedBy = strings.TrimSpace(request.RequestedBy)
	request.Metadata = CloneMap(request.Metadata)
	request.StructuredOutputSchema = CloneMap(request.StructuredOutputSchema)
	return request
}

func (r RuntimeGenerateResult) Clone() RuntimeGenerateResult {
	r.OutputJSON = CloneMap(r.OutputJSON)
	r.RuntimeMetadata = CloneMap(r.RuntimeMetadata)
	r.Warnings = CloneStrings(r.Warnings)
	r.ProvenanceRefs = CloneStrings(r.ProvenanceRefs)
	r.JournalRefs = CloneStrings(r.JournalRefs)
	return r
}

func (r RuntimeGenerateResult) IsTruth() bool {
	return r.IsCanonicalTruth
}

func (r RuntimeGenerateResult) IsEvidenceAdmitted() bool {
	return r.IsAdmittedEvidence
}

func NormalizeRefs(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	sort.Strings(out)
	return out
}

func CloneStrings(values []string) []string {
	return append([]string(nil), values...)
}

func CloneMap(values map[string]any) map[string]any {
	if values == nil {
		return nil
	}
	out := make(map[string]any, len(values))
	for key, value := range values {
		out[key] = value
	}
	return out
}

func SHA256Text(text string) string {
	sum := sha256.Sum256([]byte(text))
	return hex.EncodeToString(sum[:])
}

func StableJSON(value any) string {
	encoded, err := json.Marshal(value)
	if err != nil {
		return ""
	}
	return string(encoded)
}

func EstimateTokens(text string) int {
	text = strings.TrimSpace(text)
	if text == "" {
		return 0
	}
	return (len(text) + 3) / 4
}

func containsSecretKey(metadata map[string]any) bool {
	for key := range metadata {
		lower := strings.ToLower(key)
		if strings.Contains(lower, "secret") ||
			strings.Contains(lower, "password") ||
			strings.Contains(lower, "token") ||
			strings.Contains(lower, "api_key") ||
			strings.Contains(lower, "apikey") {
			return true
		}
	}
	return false
}
