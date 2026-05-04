package runtime

import (
	"context"
	"errors"
	"testing"
)

func TestMockRuntimeDriverIsDeterministicAndProposalOnly(t *testing.T) {
	driver, err := NewMockRuntimeDriver(MockRuntimeDriverOptions{Manifest: testManifest()})
	if err != nil {
		t.Fatalf("new mock driver: %v", err)
	}
	first, err := driver.Generate(context.Background(), testGenerateRequest())
	if err != nil {
		t.Fatalf("first generate: %v", err)
	}
	second, err := driver.Generate(context.Background(), testGenerateRequest())
	if err != nil {
		t.Fatalf("second generate: %v", err)
	}
	if first.ResultID != second.ResultID || first.OutputText != second.OutputText {
		t.Fatalf("mock generate was not deterministic: first=%#v second=%#v", first, second)
	}
	result := NormalizeGenerateResult(first, testGenerateRequest())
	if result.IsTruth() || result.IsEvidenceAdmitted() || !result.IsModelDriverProposal || result.AuthorityLevel != RuntimeAuthorityProposalOnly {
		t.Fatalf("runtime result crossed authority boundary: %#v", result)
	}
	if result.RuntimeMetadata["real_runtime"] != false || result.RuntimeMetadata["network"] != false {
		t.Fatalf("mock driver claimed live runtime behavior: %#v", result.RuntimeMetadata)
	}
}

func TestRuntimeRegistryRegistersListsAndRejectsDuplicates(t *testing.T) {
	registry := NewRegistry()
	driver, err := NewMockRuntimeDriver(MockRuntimeDriverOptions{Manifest: testManifest()})
	if err != nil {
		t.Fatalf("new mock driver: %v", err)
	}
	if _, err := registry.RegisterDriver(driver); err != nil {
		t.Fatalf("register driver: %v", err)
	}
	if _, err := registry.RegisterDriver(driver); !errors.Is(err, ErrDriverAlreadyRegistered) {
		t.Fatalf("expected duplicate rejection, got %v", err)
	}
	if _, ok := registry.GetManifest("driver-a"); !ok {
		t.Fatal("registered manifest not found")
	}
	if list := registry.ListManifests(); len(list) != 1 || list[0].DriverID != "driver-a" {
		t.Fatalf("unexpected manifest list: %#v", list)
	}
	if _, err := registry.Capabilities(context.Background(), "missing", "model-a"); !errors.Is(err, ErrDriverNotFound) {
		t.Fatalf("expected missing driver error, got %v", err)
	}
}

func TestRuntimeServiceGeneratesAndStoresResults(t *testing.T) {
	service := NewService()
	driver, err := NewMockRuntimeDriver(MockRuntimeDriverOptions{Manifest: testManifest(), OutputText: "bounded proposal"})
	if err != nil {
		t.Fatalf("new mock driver: %v", err)
	}
	if _, err := service.RegisterDriver(driver); err != nil {
		t.Fatalf("register driver: %v", err)
	}
	result, err := service.Generate(context.Background(), testGenerateRequest())
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	if result.OutputText != "bounded proposal" || result.IsCanonicalTruth || result.IsAdmittedEvidence {
		t.Fatalf("unexpected generate result: %#v", result)
	}
	stored, ok := service.GetResult(result.ResultID)
	if !ok || stored.ResultID != result.ResultID {
		t.Fatalf("result not stored: %#v ok=%v", stored, ok)
	}
	if listed := service.ListResults("workspace-a"); len(listed) != 1 || listed[0].ResultID != result.ResultID {
		t.Fatalf("unexpected result list: %#v", listed)
	}

	errorDriver, err := NewMockRuntimeDriver(MockRuntimeDriverOptions{
		Manifest:    RuntimeDriverManifest{DriverID: "driver-b", DriverName: "Error Driver", DriverKind: DriverKindMock, Version: "v1", RuntimeBackend: "mock", RuntimeVersion: "v1"},
		GenerateErr: NewMockRuntimeError("simulated failure"),
	})
	if err != nil {
		t.Fatalf("new error driver: %v", err)
	}
	if _, err := service.RegisterDriver(errorDriver); err != nil {
		t.Fatalf("register error driver: %v", err)
	}
	request := testGenerateRequest()
	request.DriverID = "driver-b"
	if _, err := service.Generate(context.Background(), request); err == nil {
		t.Fatal("expected simulated generation failure")
	}
}
