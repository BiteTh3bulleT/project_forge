package contextcompiler

import (
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"forge/projectforge/services/core/internal/forgek/snapshots"
)

type BundleListFilter struct {
	WorkspaceID   string
	CaseID        string
	SnapshotID    string
	RestoreSeedID string
}

type BlockListFilter struct {
	WorkspaceID string
	CaseID      string
	BundleID    string
	BlockType   ContextBlockType
}

type Service struct {
	mu      sync.RWMutex
	bundles map[string]ContextBundle
	blocks  map[string]ContextBlock
}

func NewService() *Service {
	return &Service{
		bundles: make(map[string]ContextBundle),
		blocks:  make(map[string]ContextBlock),
	}
}

func (s *Service) Compile(request ContextCompileRequest) (ContextCompileResult, error) {
	request = NormalizeCompileRequest(request)
	if err := ValidateCompileRequest(request); err != nil {
		return ContextCompileResult{}, err
	}
	if request.RequestID == "" {
		request.RequestID = "context-request-" + SHA256Text(StableJSON(request))[:12]
	}
	if request.BundleID == "" {
		request.BundleID = "context-bundle-" + SHA256Text(request.RequestID + request.WorkspaceID)[:12]
	}
	blocks, warnings, err := s.BuildBlocks(request)
	if err != nil {
		return ContextCompileResult{}, err
	}
	layout := DefaultPromptLayout(request.WorkspaceID, request.LayoutVersion, request.PolicyVersion, request.SyscallSchemaVersion, request.CreatedAt, nil)
	bundle, err := s.AssembleBundle(request, blocks, layout)
	if err != nil {
		return ContextCompileResult{}, err
	}
	if request.TokenBudget > 0 && bundle.EstimatedTokenCount > request.TokenBudget {
		warnings = appendUniqueString(warnings, WarningTokenEstimateExceedsBudget)
	}
	if bundle.StablePrefixHash == "" {
		warnings = appendUniqueString(warnings, WarningNoCacheablePrefix)
	}
	result := ContextCompileResult{
		ResultID:       request.RequestID + "-result",
		RequestID:      request.RequestID,
		WorkspaceID:    request.WorkspaceID,
		CaseID:         request.CaseID,
		SnapshotID:     request.SnapshotID,
		RestoreSeedID:  request.RestoreSeedID,
		Bundle:         bundle.Clone(),
		Blocks:         cloneBlocks(bundle.Blocks),
		Warnings:       NormalizeRefs(warnings),
		ProvenanceRefs: bundle.SourceRefs,
		CreatedAt:      request.CreatedAt,
		Metadata:       CloneMap(request.Metadata),
	}
	s.StoreBundle(bundle)
	return cloneCompileResult(result), nil
}

func (s *Service) CompileFromSnapshot(snapshot snapshots.Snapshot, request ContextCompileRequest) (ContextCompileResult, error) {
	if snapshot.SnapshotID == "" {
		return ContextCompileResult{}, snapshots.ErrSnapshotNotFound
	}
	if request.WorkspaceID == "" {
		request.WorkspaceID = snapshot.WorkspaceID
	}
	if request.WorkspaceID != snapshot.WorkspaceID {
		return ContextCompileResult{}, ErrWorkspaceMismatch
	}
	request.CaseID = firstNonEmpty(request.CaseID, snapshot.CaseID)
	request.SnapshotID = snapshot.SnapshotID
	request.SourceObjectRefs = append(request.SourceObjectRefs, snapshot.SourceObjectRefs...)
	request.SourceRefs = append(request.SourceRefs, snapshot.SourceRefs...)
	request.AdmittedExhibitRefs = append(request.AdmittedExhibitRefs, snapshot.AdmittedObjectRefs...)
	request.RejectedExhibitRefs = append(request.RejectedExhibitRefs, snapshot.RejectedObjectRefs...)
	request.ContradictionRefs = append(request.ContradictionRefs, snapshot.ContradictionRefs...)
	request.SupersessionRefs = append(request.SupersessionRefs, snapshot.SupersessionRefs...)
	request.PalaceRouteRefs = append(request.PalaceRouteRefs, snapshot.PalaceRouteRefs...)
	request.SemanticOperationRefs = append(request.SemanticOperationRefs, snapshot.SemanticOperationRefs...)
	request.DerivedObjectRefs = append(request.DerivedObjectRefs, snapshot.DerivedObjectRefs...)
	if request.Metadata == nil {
		request.Metadata = map[string]any{}
	}
	request.Metadata["source_snapshot_shape_hash"] = snapshot.ShapeHash
	return s.Compile(request)
}

func (s *Service) CompileFromRestoreSeed(seed snapshots.RestoreSeed, request ContextCompileRequest) (ContextCompileResult, error) {
	if seed.RestoreSeedID == "" {
		return ContextCompileResult{}, snapshots.ErrRestoreSeedNotFound
	}
	if request.WorkspaceID == "" {
		request.WorkspaceID = seed.WorkspaceID
	}
	if request.WorkspaceID != seed.WorkspaceID {
		return ContextCompileResult{}, ErrWorkspaceMismatch
	}
	request.CaseID = firstNonEmpty(request.CaseID, seed.CaseID)
	request.SnapshotID = firstNonEmpty(request.SnapshotID, seed.SnapshotID)
	request.RestoreSeedID = seed.RestoreSeedID
	request.SourceRefs = append(request.SourceRefs, seed.RecommendedSourceRefs...)
	request.SemanticOperationRefs = append(request.SemanticOperationRefs, seed.RecommendedOperationRefs...)
	request.IncludeRestoreSeed = true
	if request.Metadata == nil {
		request.Metadata = map[string]any{}
	}
	request.Metadata["source_shape_hash"] = seed.SourceShapeHash
	return s.Compile(request)
}

func (s *Service) BuildBlocks(request ContextCompileRequest) ([]ContextBlock, []string, error) {
	warnings := make([]string, 0)
	blocks := make([]ContextBlock, 0)
	add := func(blockType ContextBlockType, summary string, configure func(*ContextBlockInput)) error {
		input := ContextBlockInput{
			BlockID:              blockID(request.BundleID, blockType, len(blocks)+1),
			BlockType:            blockType,
			WorkspaceID:          request.WorkspaceID,
			CaseID:               request.CaseID,
			SnapshotID:           request.SnapshotID,
			RestoreSeedID:        request.RestoreSeedID,
			ContentSummary:       summary,
			PolicyVersion:        request.PolicyVersion,
			SyscallSchemaVersion: request.SyscallSchemaVersion,
			CreatedBy:            request.CreatedBy,
			CreatedAt:            request.CreatedAt,
			Metadata:             request.Metadata,
		}
		if configure != nil {
			configure(&input)
		}
		block, err := NewContextBlock(input)
		if err != nil {
			return err
		}
		blocks = append(blocks, block)
		return nil
	}
	if request.CaseID != "" || len(request.SourceObjectRefs)+len(request.SourceRefs)+len(request.RulingRefs)+len(request.SupersessionRefs) > 0 {
		if err := add(BlockCaseSummary, "Case semantic shape and governing refs.", func(input *ContextBlockInput) {
			input.SourceObjectRefs = request.SourceObjectRefs
			input.SourceRefs = request.SourceRefs
			input.RulingRefs = request.RulingRefs
			input.SupersessionRefs = request.SupersessionRefs
		}); err != nil {
			return nil, nil, err
		}
	}
	if len(request.PalaceRouteRefs) > 0 {
		if err := add(BlockPalaceRouteSummary, "Memory Palace retrieval route shape.", func(input *ContextBlockInput) {
			input.PalaceRouteRefs = request.PalaceRouteRefs
			input.SourceObjectRefs = request.SourceObjectRefs
		}); err != nil {
			return nil, nil, err
		}
	}
	if len(request.AdmittedExhibitRefs) > 0 {
		if err := add(BlockAdmittedEvidence, "Admitted evidence refs for deterministic context.", func(input *ContextBlockInput) {
			input.AdmittedExhibitRefs = request.AdmittedExhibitRefs
			input.SourceRefs = request.SourceRefs
		}); err != nil {
			return nil, nil, err
		}
	} else {
		warnings = appendUniqueString(warnings, WarningMissingAdmittedEvidence)
	}
	if request.IncludeRejectedEvidenceSummary && len(request.RejectedExhibitRefs) > 0 {
		warnings = appendUniqueString(warnings, WarningRejectedEvidenceSummary)
		if err := add(BlockRejectedEvidenceSummary, "Rejected evidence refs included as rejected summary only.", func(input *ContextBlockInput) {
			input.RejectedExhibitRefs = request.RejectedExhibitRefs
		}); err != nil {
			return nil, nil, err
		}
	}
	if request.IncludeContradictions && len(request.ContradictionRefs) > 0 {
		warnings = appendUniqueString(warnings, WarningContradictionsPresent)
		if err := add(BlockContradictionSummary, "Contradiction refs preserved for context inspection.", func(input *ContextBlockInput) {
			input.ContradictionRefs = request.ContradictionRefs
		}); err != nil {
			return nil, nil, err
		}
	}
	if len(request.SemanticOperationRefs)+len(request.DerivedObjectRefs) > 0 {
		if err := add(BlockSemanticOperationSummary, "Semantic operation provenance and derived refs.", func(input *ContextBlockInput) {
			input.SemanticOperationRefs = request.SemanticOperationRefs
			input.DerivedObjectRefs = request.DerivedObjectRefs
		}); err != nil {
			return nil, nil, err
		}
	}
	if request.IncludeRestoreSeed && (request.RestoreSeedID != "" || request.SnapshotID != "") {
		warnings = appendUniqueString(warnings, WarningRestoreSeedIncluded)
		if err := add(BlockSnapshotRestoreSeed, "Restore seed proposal refs for future context restoration.", func(input *ContextBlockInput) {
			input.SourceRefs = request.SourceRefs
			input.SemanticOperationRefs = request.SemanticOperationRefs
		}); err != nil {
			return nil, nil, err
		}
	}
	if len(request.ActiveConstraints) > 0 {
		if err := add(BlockActiveConstraints, strings.Join(request.ActiveConstraints, "; "), nil); err != nil {
			return nil, nil, err
		}
	}
	if request.CurrentTaskSummary != "" {
		if err := add(BlockCurrentTask, request.CurrentTaskSummary, nil); err != nil {
			return nil, nil, err
		}
	}
	if request.UserMessage != "" {
		warnings = appendUniqueString(warnings, WarningVolatileUserMessagePresent)
		if err := add(BlockUserMessage, request.UserMessage, nil); err != nil {
			return nil, nil, err
		}
	}
	if len(blocks) == 0 {
		return nil, nil, ErrInvalidCompileRequest
	}
	return blocks, warnings, nil
}

func (s *Service) AssembleBundle(request ContextCompileRequest, blocks []ContextBlock, layout PromptLayout) (ContextBundle, error) {
	if err := ValidateLayout(layout); err != nil {
		return ContextBundle{}, err
	}
	ordered := SortBlocksForLayout(blocks, layout)
	sourceRefs := make([]string, 0)
	for _, block := range ordered {
		sourceRefs = append(sourceRefs, block.AllRefs()...)
	}
	bundle := ContextBundle{
		BundleID:      request.BundleID,
		WorkspaceID:   request.WorkspaceID,
		CaseID:        request.CaseID,
		SnapshotID:    request.SnapshotID,
		RestoreSeedID: request.RestoreSeedID,
		LayoutID:      layout.LayoutID,
		LayoutVersion: layout.LayoutVersion,
		Blocks:        ordered,
		SourceRefs:    NormalizeRefs(sourceRefs),
		CreatedBy:     request.CreatedBy,
		CreatedAt:     request.CreatedAt,
		Metadata:      CloneMap(request.Metadata),
	}
	bundle = FinalizeBundle(bundle)
	return bundle, nil
}

func (s *Service) StoreBundle(bundle ContextBundle) {
	s.mu.Lock()
	defer s.mu.Unlock()
	stored := bundle.Clone()
	s.bundles[stored.BundleID] = stored
	for _, block := range stored.Blocks {
		s.blocks[block.BlockID] = block.Clone()
	}
}

func (s *Service) GetBundle(bundleID string) (ContextBundle, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	bundle, ok := s.bundles[bundleID]
	if !ok {
		return ContextBundle{}, false
	}
	return bundle.Clone(), true
}

func (s *Service) ListBundles(filter BundleListFilter) []ContextBundle {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]ContextBundle, 0)
	for _, bundle := range s.bundles {
		if filter.WorkspaceID != "" && bundle.WorkspaceID != filter.WorkspaceID {
			continue
		}
		if filter.CaseID != "" && bundle.CaseID != filter.CaseID {
			continue
		}
		if filter.SnapshotID != "" && bundle.SnapshotID != filter.SnapshotID {
			continue
		}
		if filter.RestoreSeedID != "" && bundle.RestoreSeedID != filter.RestoreSeedID {
			continue
		}
		out = append(out, bundle.Clone())
	}
	sort.Slice(out, func(i, j int) bool { return out[i].BundleID < out[j].BundleID })
	return out
}

func (s *Service) GetBlock(blockID string) (ContextBlock, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	block, ok := s.blocks[blockID]
	if !ok {
		return ContextBlock{}, false
	}
	return block.Clone(), true
}

func (s *Service) ListBlocks(filter BlockListFilter) []ContextBlock {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]ContextBlock, 0)
	if filter.BundleID != "" {
		if bundle, ok := s.bundles[filter.BundleID]; ok {
			for _, block := range bundle.Blocks {
				if blockMatchesFilter(block, filter) {
					out = append(out, block.Clone())
				}
			}
		}
		return out
	}
	for _, block := range s.blocks {
		if blockMatchesFilter(block, filter) {
			out = append(out, block.Clone())
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].LayoutPosition != out[j].LayoutPosition {
			return out[i].LayoutPosition < out[j].LayoutPosition
		}
		return out[i].BlockID < out[j].BlockID
	})
	return out
}

func blockMatchesFilter(block ContextBlock, filter BlockListFilter) bool {
	if filter.WorkspaceID != "" && block.WorkspaceID != filter.WorkspaceID {
		return false
	}
	if filter.CaseID != "" && block.CaseID != filter.CaseID {
		return false
	}
	if filter.BlockType != "" && block.BlockType != filter.BlockType {
		return false
	}
	return true
}

func blockID(bundleID string, blockType ContextBlockType, index int) string {
	slug := strings.ToLower(strings.ReplaceAll(string(blockType), "_", "-"))
	if bundleID == "" {
		bundleID = "context-bundle"
	}
	return fmt.Sprintf("%s-%02d-%s", bundleID, index, slug)
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

func appendUniqueString(values []string, value string) []string {
	if value == "" {
		return values
	}
	for _, existing := range values {
		if existing == value {
			return values
		}
	}
	return append(values, value)
}

func cloneCompileResult(result ContextCompileResult) ContextCompileResult {
	result.Bundle = result.Bundle.Clone()
	result.Blocks = cloneBlocks(result.Blocks)
	result.Warnings = CloneStrings(result.Warnings)
	result.Errors = CloneStrings(result.Errors)
	result.ProvenanceRefs = CloneStrings(result.ProvenanceRefs)
	result.Metadata = CloneMap(result.Metadata)
	return result
}

func cloneBlocks(blocks []ContextBlock) []ContextBlock {
	out := make([]ContextBlock, len(blocks))
	for i, block := range blocks {
		out[i] = block.Clone()
	}
	return out
}

func NowUTC() time.Time {
	return time.Now().UTC()
}
