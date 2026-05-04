package semantic

import (
	"sort"
	"strings"
)

func NewDefaultOperatorRegistry() *OperatorRegistry {
	registry := NewOperatorRegistry()
	for _, definition := range []OperatorDefinition{
		{OperationType: OperationMerge, Version: "v1", Deterministic: true, OutputType: ObjectTypeDerived, Handler: mergeOperator, Description: "merge compatible semantic objects"},
		{OperationType: OperationDiff, Version: "v1", Deterministic: true, OutputType: ObjectTypeDerived, Handler: diffOperator, Description: "compute deterministic difference"},
		{OperationType: OperationIntersect, Version: "v1", Deterministic: true, OutputType: ObjectTypeDerived, Handler: intersectOperator, Description: "compute deterministic overlap"},
		{OperationType: OperationContradict, Version: "v1", Deterministic: true, OutputType: ObjectTypeContradiction, RequiresSyscall: true, Handler: contradictOperator, Description: "record explicit contradiction transform"},
		{OperationType: OperationSupersede, Version: "v1", Deterministic: true, OutputType: ObjectTypeDerived, RequiresSyscall: true, Handler: supersedeOperator, Description: "record explicit supersession transform"},
		{OperationType: OperationCompress, Version: "v1", Deterministic: true, OutputType: ObjectTypeDerived, Handler: compressOperator, Description: "compress summaries without creating truth"},
		{OperationType: OperationDerive, Version: "v1", Deterministic: true, OutputType: ObjectTypeDerived, Handler: deriveOperator, Description: "derive cited semantic object"},
		{OperationType: OperationPromote, Version: "v1", Deterministic: true, OutputType: ObjectTypeDerived, RequiresSyscall: true, Handler: promoteOperator, Description: "produce explicit promotion request"},
		{OperationType: OperationDemote, Version: "v1", Deterministic: true, OutputType: ObjectTypeDerived, RequiresSyscall: true, Handler: demoteOperator, Description: "produce explicit demotion result"},
		{OperationType: OperationExpire, Version: "v1", Deterministic: true, OutputType: ObjectTypeDerived, RequiresSyscall: true, Handler: expireOperator, Description: "produce explicit expiration result"},
		{OperationType: OperationRetrieve, Version: "v1", Deterministic: true, OutputType: ObjectTypeDerived, RequiresSyscall: true, Handler: requestOnlyOperator("palace.route"), Description: "request retrieval"},
		{OperationType: OperationSubmit, Version: "v1", Deterministic: true, OutputType: ObjectTypeDerived, RequiresSyscall: true, Handler: requestOnlyOperator("court.submit"), Description: "request evidence submission"},
		{OperationType: OperationAdmit, Version: "v1", Deterministic: true, OutputType: ObjectTypeDerived, RequiresSyscall: true, Handler: requestOnlyOperator("court.admit"), Description: "request evidence admission"},
		{OperationType: OperationReject, Version: "v1", Deterministic: true, OutputType: ObjectTypeDerived, RequiresSyscall: true, Handler: requestOnlyOperator("court.reject"), Description: "request evidence rejection"},
	} {
		_ = registry.Register(definition)
	}
	return registry
}

func mergeOperator(ctx OperatorContext) (SemanticTransformResult, error) {
	if err := validateInputObjects(ctx, 2); err != nil {
		return SemanticTransformResult{}, err
	}
	if !boolParam(ctx.Parameters, "allow_contradictions") {
		for _, obj := range ctx.InputObjects {
			if len(obj.ContradictedBy) > 0 {
				return SemanticTransformResult{}, ErrContradictionMerge
			}
		}
	}
	output := derivedObject(ctx, "merged semantic object", strings.Join(summaries(ctx.InputObjects), " | "))
	output.Metadata["merged"] = true
	return resultWithOutput(ctx, output, nil), nil
}

func diffOperator(ctx OperatorContext) (SemanticTransformResult, error) {
	if err := validateInputObjects(ctx, 2); err != nil {
		return SemanticTransformResult{}, err
	}
	left := termSet(ctx.InputObjects[0].NormalizedContent + " " + ctx.InputObjects[0].ContentSummary)
	right := termSet(ctx.InputObjects[1].NormalizedContent + " " + ctx.InputObjects[1].ContentSummary)
	var diff []string
	for term := range left {
		if !right[term] {
			diff = append(diff, term)
		}
	}
	sort.Strings(diff)
	output := derivedObject(ctx, "semantic difference", strings.Join(diff, " "))
	output.Metadata["diff"] = true
	return resultWithOutput(ctx, output, nil), nil
}

func intersectOperator(ctx OperatorContext) (SemanticTransformResult, error) {
	if err := validateInputObjects(ctx, 2); err != nil {
		return SemanticTransformResult{}, err
	}
	refs := stringSet(ctx.InputObjects[0].SourceObjectRefs)
	var overlapRefs []string
	for _, ref := range ctx.InputObjects[1].SourceObjectRefs {
		if refs[ref] {
			overlapRefs = appendUnique(overlapRefs, ref)
		}
	}
	left := termSet(ctx.InputObjects[0].NormalizedContent + " " + ctx.InputObjects[0].ContentSummary)
	right := termSet(ctx.InputObjects[1].NormalizedContent + " " + ctx.InputObjects[1].ContentSummary)
	var overlapTerms []string
	for term := range left {
		if right[term] {
			overlapTerms = append(overlapTerms, term)
		}
	}
	sort.Strings(overlapTerms)
	output := derivedObject(ctx, "semantic intersection", strings.Join(overlapTerms, " "))
	output.SourceObjectRefs = unionStrings(output.SourceObjectRefs, overlapRefs)
	output.Metadata["intersection"] = true
	return resultWithOutput(ctx, output, nil), nil
}

func contradictOperator(ctx OperatorContext) (SemanticTransformResult, error) {
	if err := validateInputObjects(ctx, 2); err != nil {
		return SemanticTransformResult{}, err
	}
	output := derivedObject(ctx, "semantic contradiction", "explicit contradiction transform")
	output.ObjectType = ObjectTypeContradiction
	output.Metadata["contradiction"] = true
	request := SemanticSyscallRequest{
		RequestID:          ctx.OperationID + "-court-contradiction",
		SyscallName:        "court.register_contradiction",
		WorkspaceID:        ctx.WorkspaceID,
		CaseID:             ctx.CaseID,
		Payload:            cloneMap(ctx.Parameters),
		Reason:             "semantic contradiction requires Courthouse registration",
		RequiredCapability: "court.register_contradiction",
		CreatedAt:          ctx.CreatedAt,
	}
	return resultWithOutput(ctx, output, []SemanticSyscallRequest{request}), nil
}

func supersedeOperator(ctx OperatorContext) (SemanticTransformResult, error) {
	if err := validateInputObjects(ctx, 2); err != nil {
		return SemanticTransformResult{}, err
	}
	output := derivedObject(ctx, "semantic supersession", "explicit supersession transform")
	output.Metadata["supersession"] = true
	request := SemanticSyscallRequest{
		RequestID:          ctx.OperationID + "-court-supersession",
		SyscallName:        "court.register_supersession",
		WorkspaceID:        ctx.WorkspaceID,
		CaseID:             ctx.CaseID,
		Payload:            cloneMap(ctx.Parameters),
		Reason:             "semantic supersession requires Courthouse registration",
		RequiredCapability: "court.register_supersession",
		CreatedAt:          ctx.CreatedAt,
	}
	return resultWithOutput(ctx, output, []SemanticSyscallRequest{request}), nil
}

func compressOperator(ctx OperatorContext) (SemanticTransformResult, error) {
	if err := validateInputObjects(ctx, 1); err != nil {
		return SemanticTransformResult{}, err
	}
	summary := strings.Join(summaries(ctx.InputObjects), " ")
	if len(summary) > 160 {
		summary = summary[:160]
	}
	output := derivedObject(ctx, "compressed semantic summary", summary)
	output.Metadata["derived"] = true
	output.Metadata["compressed"] = true
	return resultWithOutput(ctx, output, []SemanticSyscallRequest{}).withWarnings(WarningCompressionCannotCreateTruth), nil
}

func deriveOperator(ctx OperatorContext) (SemanticTransformResult, error) {
	if err := validateInputObjects(ctx, 1); err != nil {
		return SemanticTransformResult{}, err
	}
	output := derivedObject(ctx, "derived semantic object", strings.Join(summaries(ctx.InputObjects), " | "))
	output.Metadata["derived"] = true
	return resultWithOutput(ctx, output, nil), nil
}

func promoteOperator(ctx OperatorContext) (SemanticTransformResult, error) {
	return authorityOperator(ctx, "promoted", AuthorityValidated)
}

func demoteOperator(ctx OperatorContext) (SemanticTransformResult, error) {
	return authorityOperator(ctx, "demoted", AuthorityProposal)
}

func expireOperator(ctx OperatorContext) (SemanticTransformResult, error) {
	return authorityOperator(ctx, "expired", AuthorityExpired)
}

func authorityOperator(ctx OperatorContext, marker, authority string) (SemanticTransformResult, error) {
	if err := validateInputObjects(ctx, 1); err != nil {
		return SemanticTransformResult{}, err
	}
	output := ctx.InputObjects[0].Clone()
	output.SemanticObjectID = nextObjectID(ctx)
	output.AuthorityLevel = authority
	output.Metadata = cloneMap(output.Metadata)
	if output.Metadata == nil {
		output.Metadata = make(map[string]any)
	}
	output.Metadata[marker] = true
	output.ProvenanceRefs = unionStrings(output.ProvenanceRefs, collectProvenance(ctx.InputObjects))
	return resultWithOutput(ctx, output, nil), nil
}

func requestOnlyOperator(syscallName string) OperatorHandler {
	return func(ctx OperatorContext) (SemanticTransformResult, error) {
		if err := validateInputObjects(ctx, 1); err != nil {
			return SemanticTransformResult{}, err
		}
		request := SemanticSyscallRequest{
			RequestID:          ctx.OperationID + "-" + strings.ReplaceAll(syscallName, ".", "-"),
			SyscallName:        syscallName,
			WorkspaceID:        ctx.WorkspaceID,
			CaseID:             ctx.CaseID,
			Payload:            cloneMap(ctx.Parameters),
			Reason:             "semantic operation requested governed syscall",
			RequiredCapability: syscallName,
			CreatedAt:          ctx.CreatedAt,
		}
		return SemanticTransformResult{
			ResultID:          ctx.ResultID,
			OperationID:       ctx.OperationID,
			OperationType:     ctx.OperationType,
			WorkspaceID:       ctx.WorkspaceID,
			CaseID:            ctx.CaseID,
			RequestedSyscalls: []SemanticSyscallRequest{request},
			ProvenanceRefs:    collectProvenance(ctx.InputObjects),
			CreatedAt:         ctx.CreatedAt,
		}, nil
	}
}

func validateInputObjects(ctx OperatorContext, min int) error {
	if ctx.WorkspaceID == "" || ctx.CreatedBy == "" || len(ctx.InputObjects) < min {
		return ErrInvalidOperation
	}
	for _, obj := range ctx.InputObjects {
		if obj.WorkspaceID != ctx.WorkspaceID {
			return ErrInvalidOperation
		}
		if obj.AdmissibilityStatus == AdmissibilityRejected {
			return ErrRejectedEvidenceInput
		}
	}
	return nil
}

func derivedObject(ctx OperatorContext, summaryPrefix, content string) SemanticObject {
	metadata := map[string]any{"derived": true}
	object := SemanticObject{
		SemanticObjectID:  nextObjectID(ctx),
		WorkspaceID:       ctx.WorkspaceID,
		ObjectType:        ObjectTypeDerived,
		SourceObjectRefs:  collectSourceObjectRefs(ctx.InputObjects),
		SourceRefs:        collectSourceRefs(ctx.InputObjects),
		ContentSummary:    strings.TrimSpace(summaryPrefix + ": " + content),
		NormalizedContent: strings.TrimSpace(strings.ToLower(content)),
		AuthorityLevel:    AuthorityProposal,
		ProvenanceRefs:    collectProvenance(ctx.InputObjects),
		CreatedAt:         ctx.CreatedAt,
		UpdatedAt:         ctx.CreatedAt,
		Metadata:          metadata,
	}
	return object
}

func resultWithOutput(ctx OperatorContext, output SemanticObject, requests []SemanticSyscallRequest) SemanticTransformResult {
	return SemanticTransformResult{
		ResultID:          ctx.ResultID,
		OperationID:       ctx.OperationID,
		OperationType:     ctx.OperationType,
		WorkspaceID:       ctx.WorkspaceID,
		CaseID:            ctx.CaseID,
		OutputObjects:     []SemanticObject{output},
		OutputRefs:        []string{output.SemanticObjectID},
		RequestedSyscalls: cloneSyscallRequests(requests),
		ProvenanceRefs:    output.ProvenanceRefs,
		CreatedAt:         ctx.CreatedAt,
	}
}

func (r SemanticTransformResult) withWarnings(warnings ...string) SemanticTransformResult {
	r.Warnings = append(r.Warnings, warnings...)
	return r
}

func nextObjectID(ctx OperatorContext) string {
	if ctx.NextObjectID != nil {
		return ctx.NextObjectID()
	}
	return ctx.OperationID + "-object"
}

func summaries(objects []SemanticObject) []string {
	out := make([]string, 0, len(objects))
	for _, obj := range objects {
		if obj.ContentSummary != "" {
			out = append(out, obj.ContentSummary)
		}
	}
	sort.Strings(out)
	return out
}

func collectSourceObjectRefs(objects []SemanticObject) []string {
	var refs []string
	for _, obj := range objects {
		refs = appendUnique(refs, obj.SemanticObjectID)
		refs = unionStrings(refs, obj.SourceObjectRefs)
	}
	return refs
}

func collectSourceRefs(objects []SemanticObject) []string {
	var refs []string
	for _, obj := range objects {
		refs = unionStrings(refs, obj.SourceRefs)
	}
	return refs
}

func collectProvenance(objects []SemanticObject) []string {
	var refs []string
	for _, obj := range objects {
		refs = appendUnique(refs, obj.SemanticObjectID)
		refs = unionStrings(refs, obj.ProvenanceRefs)
	}
	return refs
}

func unionStrings(base []string, values []string) []string {
	for _, value := range values {
		base = appendUnique(base, value)
	}
	return base
}

func appendUnique(values []string, value string) []string {
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

func stringSet(values []string) map[string]bool {
	out := make(map[string]bool, len(values))
	for _, value := range values {
		out[value] = true
	}
	return out
}

func termSet(text string) map[string]bool {
	out := make(map[string]bool)
	for _, term := range strings.Fields(strings.ToLower(text)) {
		trimmed := strings.Trim(term, ".,:;!?()[]{}")
		if trimmed != "" {
			out[trimmed] = true
		}
	}
	return out
}

func boolParam(parameters map[string]any, key string) bool {
	value, _ := parameters[key].(bool)
	return value
}
