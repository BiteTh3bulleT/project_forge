package integrationready

import (
	"encoding/json"
	"reflect"
	"testing"
	"time"
)

func TestIntegrationReadinessReportValidatesAndSerializesDeterministically(t *testing.T) {
	generatedAt := time.Unix(100, 0).UTC()
	report, err := NewIntegrationReadinessReport(ReportInput{
		ReportID:          "report-phase-11f",
		GeneratedAt:       generatedAt,
		Phase:             Phase11F,
		SubsystemStatuses: DefaultSubsystemStatuses(),
		LivePathMappings:  DefaultLivePathMappings(),
		AdapterContracts:  DefaultAdapterContracts(),
		ShadowPolicy:      DefaultShadowModePolicy(),
		MissingContracts:  []string{"rollback_plan", "shadow_comparison_tests", "rollback_plan"},
		MissingTests:      []string{"route_inventory_tests", "shadow_comparison_tests"},
		Blockers:          []string{"phase_12_design_not_started"},
		Warnings:          []string{"readiness_score_is_advisory"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if report.ReportID != "report-phase-11f" || report.GeneratedAt != generatedAt || report.Phase != Phase11F {
		t.Fatalf("unexpected report identity: %#v", report)
	}
	if !reflect.DeepEqual(report.MissingContracts, []string{"rollback_plan", "shadow_comparison_tests"}) {
		t.Fatalf("missing contracts were not normalized: %#v", report.MissingContracts)
	}
	if report.ReadinessScore <= 0 || report.ReadinessScore > 100 {
		t.Fatalf("unexpected readiness score: %f", report.ReadinessScore)
	}
	if report.AuthorizesLiveIntegration() {
		t.Fatal("readiness report must not authorize live integration")
	}
	if !report.IsDiagnosticOnly() || report.CanMutateLiveState() {
		t.Fatal("readiness report must remain diagnostic only")
	}
	first, err := json.Marshal(report)
	if err != nil {
		t.Fatal(err)
	}
	again, err := NewIntegrationReadinessReport(ReportInput{
		ReportID:          "report-phase-11f",
		GeneratedAt:       generatedAt,
		Phase:             Phase11F,
		SubsystemStatuses: reversedSubsystems(DefaultSubsystemStatuses()),
		LivePathMappings:  reversedMappings(DefaultLivePathMappings()),
		AdapterContracts:  reversedContracts(DefaultAdapterContracts()),
		ShadowPolicy:      DefaultShadowModePolicy(),
		MissingContracts:  []string{"shadow_comparison_tests", "rollback_plan"},
		MissingTests:      []string{"shadow_comparison_tests", "route_inventory_tests"},
		Blockers:          []string{"phase_12_design_not_started"},
		Warnings:          []string{"readiness_score_is_advisory"},
	})
	if err != nil {
		t.Fatal(err)
	}
	second, err := json.Marshal(again)
	if err != nil {
		t.Fatal(err)
	}
	if string(first) != string(second) {
		t.Fatalf("report serialization should be deterministic\nfirst:  %s\nsecond: %s", first, second)
	}
}

func TestIntegrationReadinessReportRejectsMissingRequiredFields(t *testing.T) {
	if _, err := NewIntegrationReadinessReport(ReportInput{
		GeneratedAt:       time.Unix(100, 0).UTC(),
		Phase:             Phase11F,
		SubsystemStatuses: DefaultSubsystemStatuses(),
		LivePathMappings:  DefaultLivePathMappings(),
		AdapterContracts:  DefaultAdapterContracts(),
		ShadowPolicy:      DefaultShadowModePolicy(),
	}); err == nil {
		t.Fatal("expected missing report_id to be rejected")
	}
	if _, err := NewIntegrationReadinessReport(ReportInput{
		ReportID:          "report-phase-11f",
		GeneratedAt:       time.Unix(100, 0).UTC(),
		SubsystemStatuses: DefaultSubsystemStatuses(),
		LivePathMappings:  DefaultLivePathMappings(),
		AdapterContracts:  DefaultAdapterContracts(),
		ShadowPolicy:      DefaultShadowModePolicy(),
	}); err == nil {
		t.Fatal("expected missing phase to be rejected")
	}
}

func reversedSubsystems(values []SubsystemReadiness) []SubsystemReadiness {
	out := append([]SubsystemReadiness(nil), values...)
	for i, j := 0, len(out)-1; i < j; i, j = i+1, j-1 {
		out[i], out[j] = out[j], out[i]
	}
	return out
}

func reversedMappings(values []LivePathMapping) []LivePathMapping {
	out := append([]LivePathMapping(nil), values...)
	for i, j := 0, len(out)-1; i < j; i, j = i+1, j-1 {
		out[i], out[j] = out[j], out[i]
	}
	return out
}

func reversedContracts(values []AdapterContract) []AdapterContract {
	out := append([]AdapterContract(nil), values...)
	for i, j := 0, len(out)-1; i < j; i, j = i+1, j-1 {
		out[i], out[j] = out[j], out[i]
	}
	return out
}
