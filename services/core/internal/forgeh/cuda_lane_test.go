package forgeh

import "testing"

func TestGpuWorkClassesMatchM5CudaLaneContract(t *testing.T) {
	expected := []GpuWorkClass{
		GpuWorkInference,
		GpuWorkEmbedding,
		GpuWorkReranking,
		GpuWorkVectorScoring,
		GpuWorkKVCacheAnalysis,
		GpuWorkSimulation,
		GpuWorkBatchDiagnostics,
		GpuWorkCompressionPrepass,
	}
	if got := AllGpuWorkClasses(); len(got) != len(expected) {
		t.Fatalf("expected %d gpu work classes, got %d: %v", len(expected), len(got), got)
	}
	for i, want := range expected {
		if got := AllGpuWorkClasses()[i]; got != want {
			t.Fatalf("gpu work class %d mismatch: got %q want %q", i, got, want)
		}
		if !want.IsKnown() {
			t.Fatalf("expected gpu work class %q to be known", want)
		}
	}
}

func TestGpuWorkClassParsingIsBounded(t *testing.T) {
	if got := ParseGpuWorkClass(" embedding "); got != GpuWorkEmbedding {
		t.Fatalf("expected trimmed embedding class, got %q", got)
	}
	if got := ParseGpuWorkClass("EMBEDDING"); got != GpuWorkUnknown {
		t.Fatalf("gpu work class parsing should be exact after trim, got %q", got)
	}
	if got := ParseGpuWorkClass("raw_pointer_scan"); got != GpuWorkUnknown {
		t.Fatalf("unknown gpu work class should stay unknown, got %q", got)
	}
}
