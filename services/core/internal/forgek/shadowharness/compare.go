package shadowharness

import (
	"fmt"
	"sort"
	"strings"
)

func ValidateNoEffect(policy ShadowHarnessPolicy, report ShadowComparisonReport) error {
	if err := ValidateShadowHarnessPolicy(policy); err != nil {
		return err
	}
	if !report.NoEffectVerified {
		return ErrNoEffectNotVerified
	}
	if !report.RAGShadow.NoExecutionVerified || !report.ConsensusShadow.DiagnosticOnlyVerified ||
		!report.ContextShadow.DiagnosticOnlyVerified || !report.RuntimeShadow.ProposalOnlyVerified ||
		!report.KVShadow.AccelerationNotMemoryVerified || !report.LymphaticShadow.NoSilentMutationVerified ||
		!report.LymphaticShadow.ProposalsDoNotExecute {
		return fmt.Errorf("%w: subreport", ErrNoEffectNotVerified)
	}
	return nil
}

func normalizeStrings(values []string) []string {
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

func (r RAGShadowReport) CanExecuteRetrieval() bool        { return false }
func (r RAGShadowReport) CanCallEmbeddings() bool          { return false }
func (r RAGShadowReport) CanWriteMemory() bool             { return false }
func (r RAGShadowReport) CanCompileContext() bool          { return false }
func (r RAGShadowReport) CanAffectUserVisibleOutput() bool { return false }

func (r ConsensusShadowReport) AcceptedClaimsBecomeTruth() bool { return false }
func (r ConsensusShadowReport) EmitsUserVisibleOutput() bool    { return false }

func (r ContextShadowReport) ModifiesContextBundle() bool    { return false }
func (r ContextShadowReport) AltersLiveCompileContext() bool { return false }

func (r RuntimeShadowReport) CanCallModelRuntime() bool { return false }

func (r KVShadowReport) CanReuseLiveKV() bool { return false }

func (r LymphaticShadowReport) CanExecuteProposals() bool { return false }
