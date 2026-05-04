package runtime

import "context"

type RuntimeDriver interface {
	Manifest() RuntimeDriverManifest
	Capabilities(ctx context.Context, modelID string) (RuntimeCapabilityManifest, error)
	Generate(ctx context.Context, request RuntimeGenerateRequest) (RuntimeGenerateResult, error)
	Health(ctx context.Context) (RuntimeHealth, error)
}
