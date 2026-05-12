package forgeh

import "strings"

// GpuWorkClass names future governed GPU acceleration classes. These labels are
// advisory resource-policy contracts only; they do not allocate memory, launch
// kernels, or change runtime behavior.
type GpuWorkClass string

const (
	GpuWorkUnknown            GpuWorkClass = ""
	GpuWorkInference          GpuWorkClass = "inference"
	GpuWorkEmbedding          GpuWorkClass = "embedding"
	GpuWorkReranking          GpuWorkClass = "reranking"
	GpuWorkVectorScoring      GpuWorkClass = "vector_scoring"
	GpuWorkKVCacheAnalysis    GpuWorkClass = "kv_cache_analysis"
	GpuWorkSimulation         GpuWorkClass = "simulation"
	GpuWorkBatchDiagnostics   GpuWorkClass = "batch_diagnostics"
	GpuWorkCompressionPrepass GpuWorkClass = "compression_prepass"
)

var gpuWorkClasses = []GpuWorkClass{
	GpuWorkInference,
	GpuWorkEmbedding,
	GpuWorkReranking,
	GpuWorkVectorScoring,
	GpuWorkKVCacheAnalysis,
	GpuWorkSimulation,
	GpuWorkBatchDiagnostics,
	GpuWorkCompressionPrepass,
}

func AllGpuWorkClasses() []GpuWorkClass {
	out := make([]GpuWorkClass, len(gpuWorkClasses))
	copy(out, gpuWorkClasses)
	return out
}

func ParseGpuWorkClass(raw string) GpuWorkClass {
	class := GpuWorkClass(strings.TrimSpace(raw))
	if class.IsKnown() {
		return class
	}
	return GpuWorkUnknown
}

func (c GpuWorkClass) IsKnown() bool {
	for _, known := range gpuWorkClasses {
		if c == known {
			return true
		}
	}
	return false
}
