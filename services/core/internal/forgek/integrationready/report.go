package integrationready

import (
	"fmt"
	"strings"
)

func NewIntegrationReadinessReport(input ReportInput) (IntegrationReadinessReport, error) {
	if strings.TrimSpace(input.ReportID) == "" {
		return IntegrationReadinessReport{}, fmt.Errorf("%w: report_id", ErrMissingRequiredField)
	}
	if strings.TrimSpace(input.Phase) == "" {
		return IntegrationReadinessReport{}, fmt.Errorf("%w: phase", ErrMissingRequiredField)
	}
	if input.GeneratedAt.IsZero() {
		return IntegrationReadinessReport{}, fmt.Errorf("%w: generated_at", ErrMissingRequiredField)
	}
	report := IntegrationReadinessReport{
		ReportID:          input.ReportID,
		GeneratedAt:       input.GeneratedAt,
		Phase:             input.Phase,
		SubsystemStatuses: normalizeSubsystems(input.SubsystemStatuses),
		LivePathMappings:  normalizeMappings(input.LivePathMappings),
		AdapterContracts:  normalizeContracts(input.AdapterContracts),
		ShadowPolicy:      input.ShadowPolicy,
		MissingContracts:  NormalizeStrings(input.MissingContracts),
		MissingTests:      NormalizeStrings(input.MissingTests),
		Blockers:          NormalizeStrings(input.Blockers),
		Warnings:          NormalizeStrings(input.Warnings),
		Metadata:          input.Metadata,
	}
	if len(report.SubsystemStatuses) == 0 || len(report.LivePathMappings) == 0 || len(report.AdapterContracts) == 0 {
		return IntegrationReadinessReport{}, fmt.Errorf("%w: report readiness inputs", ErrMissingRequiredField)
	}
	for _, status := range report.SubsystemStatuses {
		if err := validateSubsystem(status); err != nil {
			return IntegrationReadinessReport{}, err
		}
	}
	if err := ValidateNoLiveMutation(report.AdapterContracts, report.ShadowPolicy, report.LivePathMappings); err != nil {
		return IntegrationReadinessReport{}, err
	}
	report.ReadinessScore = advisoryReadinessScore(report.SubsystemStatuses)
	return report, nil
}

func (r IntegrationReadinessReport) IsDiagnosticOnly() bool {
	return true
}

func (r IntegrationReadinessReport) AuthorizesLiveIntegration() bool {
	return false
}

func (r IntegrationReadinessReport) CanMutateLiveState() bool {
	return false
}

func advisoryReadinessScore(statuses []SubsystemReadiness) float64 {
	if len(statuses) == 0 {
		return 0
	}
	score := 0.0
	for _, status := range statuses {
		switch status.Status {
		case StatusReadyForShadow:
			score += 1
		case StatusNeedsTests, StatusNeedsAdapter:
			score += 0.5
		}
	}
	return score / float64(len(statuses)) * 100
}
