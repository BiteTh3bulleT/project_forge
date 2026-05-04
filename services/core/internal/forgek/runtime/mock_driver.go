package runtime

import (
	"context"
	"errors"
	"strings"
	"time"
)

type MockRuntimeDriver struct {
	manifest    RuntimeDriverManifest
	capability  RuntimeCapabilityManifest
	outputText  string
	outputJSON  map[string]any
	generateErr error
	health      RuntimeHealth
}

type MockRuntimeDriverOptions struct {
	Manifest    RuntimeDriverManifest
	Capability  RuntimeCapabilityManifest
	OutputText  string
	OutputJSON  map[string]any
	GenerateErr error
	Health      RuntimeHealth
}

func NewMockRuntimeDriver(options MockRuntimeDriverOptions) (*MockRuntimeDriver, error) {
	manifest := options.Manifest
	if manifest.DriverID == "" {
		manifest.DriverID = "mock-runtime"
	}
	if manifest.DriverName == "" {
		manifest.DriverName = "Mock Runtime Driver"
	}
	if manifest.DriverKind == "" {
		manifest.DriverKind = DriverKindMock
	}
	if manifest.Version == "" {
		manifest.Version = "v1"
	}
	if manifest.RuntimeBackend == "" {
		manifest.RuntimeBackend = "mock"
	}
	if manifest.RuntimeVersion == "" {
		manifest.RuntimeVersion = "v1"
	}
	manifest.DeterministicForTests = true
	manifest.AuthorityLevel = RuntimeAuthorityProposalOnly
	manifest, err := NewRuntimeDriverManifest(manifest)
	if err != nil {
		return nil, err
	}
	capability := options.Capability
	if capability.RuntimeID == "" {
		capability.RuntimeID = manifest.DriverID
	}
	if capability.RuntimeVersion == "" {
		capability.RuntimeVersion = manifest.RuntimeVersion
	}
	if capability.ModelID == "" {
		capability.ModelID = firstSupported(manifest.SupportedModels, "mock-model")
	}
	if capability.ModelRevision == "" {
		capability.ModelRevision = "mock-revision"
	}
	if capability.TokenizerID == "" {
		capability.TokenizerID = "mock-tokenizer"
	}
	if capability.TokenizerRevision == "" {
		capability.TokenizerRevision = "mock-tokenizer-revision"
	}
	if capability.ChatTemplateHash == "" {
		capability.ChatTemplateHash = SHA256Text("mock-chat-template")
	}
	capability.SupportsPrefixCache = manifest.SupportsPrefixCache
	capability.SupportsPagedKV = manifest.SupportsPagedKV
	capability.SupportsKVQuantization = manifest.SupportsKVQuantization
	capability.SupportsKVOffload = manifest.SupportsKVOffload
	capability.SupportsPriorityEviction = manifest.SupportsPriorityEviction
	capability.SupportsCacheSalt = manifest.SupportsCacheSalt
	capability.SupportsNonPrefixReuse = manifest.SupportsNonPrefixReuse
	capability.SupportsCrossInstanceReuse = manifest.SupportsCrossInstanceReuse
	capability.SupportsStructuredOutputs = manifest.SupportsStructuredOutputs
	capability.SupportsToolCalling = manifest.SupportsToolCalling
	if capability.MaxContextTokens == 0 {
		capability.MaxContextTokens = 4096
	}
	if capability.MaxOutputTokens == 0 {
		capability.MaxOutputTokens = 512
	}
	if err := ValidateCapabilityManifest(capability); err != nil {
		return nil, err
	}
	health := options.Health
	if health.DriverID == "" {
		health.DriverID = manifest.DriverID
	}
	if health.Status == "" {
		health.Status = HealthAvailable
	}
	return &MockRuntimeDriver{
		manifest:    manifest,
		capability:  NormalizeCapabilityManifest(capability),
		outputText:  options.OutputText,
		outputJSON:  CloneMap(options.OutputJSON),
		generateErr: options.GenerateErr,
		health:      health,
	}, nil
}

func (d *MockRuntimeDriver) Manifest() RuntimeDriverManifest {
	return d.manifest
}

func (d *MockRuntimeDriver) Capabilities(_ context.Context, modelID string) (RuntimeCapabilityManifest, error) {
	capability := d.capability
	if modelID != "" {
		capability.ModelID = modelID
	}
	return capability, nil
}

func (d *MockRuntimeDriver) Generate(_ context.Context, request RuntimeGenerateRequest) (RuntimeGenerateResult, error) {
	if d.generateErr != nil {
		return RuntimeGenerateResult{}, d.generateErr
	}
	if err := ValidateGenerateRequest(request); err != nil {
		return RuntimeGenerateResult{}, err
	}
	output := d.outputText
	if output == "" {
		output = "mock runtime proposal " + SHA256Text(strings.Join([]string{
			request.RequestID,
			request.BundleID,
			request.CanonicalPromptText,
			request.TokenInputHash,
		}, "|"))[:16]
	}
	return RuntimeGenerateResult{
		ResultID:            "runtime-result-" + SHA256Text(request.RequestID + "|" + output)[:12],
		RequestID:           request.RequestID,
		DriverID:            d.manifest.DriverID,
		WorkspaceID:         request.WorkspaceID,
		CaseID:              request.CaseID,
		BundleID:            request.BundleID,
		KVLookupID:          request.KVLookupID,
		KVCacheID:           request.KVCacheID,
		ModelID:             request.ModelID,
		OutputText:          output,
		OutputJSON:          CloneMap(d.outputJSON),
		FinishReason:        FinishStop,
		PromptTokenEstimate: EstimateTokens(request.CanonicalPromptText),
		OutputTokenEstimate: EstimateTokens(output),
		RuntimeMetadata: map[string]any{
			"driver_kind":  string(DriverKindMock),
			"network":      false,
			"real_runtime": false,
		},
		CreatedAt:      request.CreatedAt,
		ProvenanceRefs: []string{request.RequestID, request.BundleID, request.KVLookupID, request.KVCacheID},
	}, nil
}

func (d *MockRuntimeDriver) Health(_ context.Context) (RuntimeHealth, error) {
	health := d.health
	if health.Status == "" {
		health.Status = HealthAvailable
	}
	if health.DriverID == "" {
		health.DriverID = d.manifest.DriverID
	}
	if health.CheckedAt.IsZero() {
		health.CheckedAt = time.Now().UTC()
	}
	return health, nil
}

func NewMockRuntimeError(message string) error {
	return errors.New(message)
}

func firstSupported(values []string, fallback string) string {
	if len(values) > 0 {
		return values[0]
	}
	return fallback
}
