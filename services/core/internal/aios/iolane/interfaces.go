package iolane

import (
	"context"

	"forge/projectforge/services/core/internal/adapters"
	"forge/projectforge/services/core/internal/artifacts"
	"forge/projectforge/services/core/internal/backup"
	"forge/projectforge/services/core/internal/gateway"
	"forge/projectforge/services/core/internal/imports"
)

// ToolGateway is the controlled I/O execution boundary.
type ToolGateway interface {
	Execute(ctx context.Context, req gateway.Request) (*gateway.Result, error)
}

// AdapterRuntime runs bounded adapter requests through existing adapter contracts.
type AdapterRuntime interface {
	Invoke(ctx context.Context, req adapters.InvokeRequest) (adapters.InvokeResult, error)
}

// ArtifactService persists execution evidence.
type ArtifactService interface {
	CreateTextArtifact(ctx context.Context, req artifacts.CreateTextArtifactRequest) (artifacts.Artifact, error)
}

// EventIngestionService pulls external/source events into FORGE evidence tables.
type EventIngestionService interface {
	IndexSource(ctx context.Context, sourceID int64, rootPath string) error
	IndexAllSources(ctx context.Context) error
}

// ImportExportService handles durable portable bundles.
type ImportExportService interface {
	CreateBundle(ctx context.Context, req backup.CreateBundleRequest) (*backup.Bundle, error)
	InspectBundle(ctx context.Context, req backup.RestoreBundleRequest) (*backup.RestoreResult, error)
}

// ExecutionImportService captures imported external execution outcomes as evidence.
type ExecutionImportService interface {
	Create(ctx context.Context, req imports.CreateRequest) (*imports.Record, error)
}
