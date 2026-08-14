package iolane

import (
	"context"
	"testing"

	"forge/projectforge/services/core/internal/adapters"
	"forge/projectforge/services/core/internal/artifacts"
	"forge/projectforge/services/core/internal/backup"
	"forge/projectforge/services/core/internal/gateway"
	"forge/projectforge/services/core/internal/imports"
)

type stubGateway struct{}

func (stubGateway) Execute(_ context.Context, _ gateway.Request) (*gateway.Result, error) {
	return &gateway.Result{Status: gateway.StatusOK, Allowed: true}, nil
}

type stubAdapterRuntime struct{}

func (stubAdapterRuntime) Invoke(_ context.Context, _ adapters.InvokeRequest) (adapters.InvokeResult, error) {
	return adapters.InvokeResult{OK: true, Message: "ok"}, nil
}

type stubArtifacts struct{}

func (stubArtifacts) CreateTextArtifact(_ context.Context, _ artifacts.CreateTextArtifactRequest) (artifacts.Artifact, error) {
	return artifacts.Artifact{ID: 1, Type: "job_result"}, nil
}

type stubIngest struct{}

func (stubIngest) IndexSource(_ context.Context, _ int64, _ string) error { return nil }
func (stubIngest) IndexAllSources(_ context.Context) error                { return nil }

type stubBundles struct{}

func (stubBundles) CreateBundle(_ context.Context, _ backup.CreateBundleRequest) (*backup.Bundle, error) {
	return &backup.Bundle{ID: 1, Kind: "portable_snapshot"}, nil
}
func (stubBundles) InspectBundle(_ context.Context, _ backup.RestoreBundleRequest) (*backup.RestoreResult, error) {
	return &backup.RestoreResult{Accepted: true}, nil
}

type stubImports struct{}

func (stubImports) Create(_ context.Context, _ imports.CreateRequest) (*imports.Record, error) {
	return &imports.Record{ID: 1}, nil
}

func TestIOLaneInterfacesCompile(t *testing.T) {
	var _ ToolGateway = stubGateway{}
	var _ AdapterRuntime = stubAdapterRuntime{}
	var _ ArtifactService = stubArtifacts{}
	var _ EventIngestionService = stubIngest{}
	var _ ImportExportService = stubBundles{}
	var _ ExecutionImportService = stubImports{}
}
