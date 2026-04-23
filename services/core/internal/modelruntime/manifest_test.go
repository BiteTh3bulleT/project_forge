package modelruntime

import (
	"errors"
	"strings"
	"testing"
)

func TestParseManifestValid(t *testing.T) {
	t.Parallel()

	manifestJSON := `{
	  "schemaVersion": "1",
	  "id": "qwen2.5-7b-instruct-q4",
	  "displayName": "Qwen 2.5 7B Instruct Q4",
	  "family": "qwen2.5",
	  "format": "gguf",
	  "backend": "llama_cpp",
	  "file": "model.gguf",
	  "sha256": "sha256:0123456789abcdef",
	  "sizeBytes": 123456,
	  "quantization": "Q4_K_M",
	  "contextLength": 32768,
	  "capabilities": ["chat", "completion", "chat"],
	  "defaultRuntime": {
	    "temperature": 0.2,
	    "maxTokens": 1024,
	    "timeoutMs": 30000
	  },
	  "license": {
	    "name": "Apache-2.0",
	    "url": "https://example.test/license"
	  },
	  "metadata": {
	    "source": "local"
	  }
	}`

	manifest, err := ParseManifest([]byte(manifestJSON))
	if err != nil {
		t.Fatalf("ParseManifest error: %v", err)
	}

	if manifest.Format != ModelFormatGGUF {
		t.Fatalf("expected format gguf, got %q", manifest.Format)
	}
	if manifest.Backend != ModelBackendLlamaCPP {
		t.Fatalf("expected backend llama_cpp, got %q", manifest.Backend)
	}
	if got := ManifestModelPath(manifest); got != "model.gguf" {
		t.Fatalf("expected model path model.gguf, got %q", got)
	}
	if len(manifest.Capabilities) != 2 {
		t.Fatalf("expected deduped capabilities length 2, got %d", len(manifest.Capabilities))
	}
}

func TestParseManifestUnknownFormatDeterministic(t *testing.T) {
	t.Parallel()

	manifestJSON := `{
	  "schemaVersion": "1",
	  "id": "bad-format-model",
	  "displayName": "Bad Format",
	  "family": "unknown",
	  "format": "foobar",
	  "backend": "llama_cpp",
	  "file": "model.bin",
	  "sha256": "",
	  "sizeBytes": 0,
	  "quantization": "Q4_0",
	  "contextLength": 1024,
	  "capabilities": ["chat"],
	  "defaultRuntime": {},
	  "license": {"name": "MIT"},
	  "metadata": {}
	}`

	_, err := ParseManifest([]byte(manifestJSON))
	if err == nil {
		t.Fatal("expected ParseManifest to fail")
	}
	if !errors.Is(err, ErrUnsupportedModelFormat) {
		t.Fatalf("expected unsupported format error, got %v", err)
	}
	if !strings.Contains(err.Error(), `"unknown"`) {
		t.Fatalf("expected deterministic unknown format marker, got %v", err)
	}
}

func TestParseManifestMissingRequiredFieldRejected(t *testing.T) {
	t.Parallel()

	manifestJSON := `{
	  "schemaVersion": "1",
	  "displayName": "Missing ID",
	  "family": "qwen",
	  "format": "gguf",
	  "backend": "llama_cpp",
	  "file": "model.gguf",
	  "sha256": "",
	  "sizeBytes": 0,
	  "quantization": "Q4_0",
	  "contextLength": 1024,
	  "capabilities": ["chat"],
	  "defaultRuntime": {},
	  "license": {"name": "MIT"},
	  "metadata": {}
	}`

	_, err := ParseManifest([]byte(manifestJSON))
	if err == nil {
		t.Fatal("expected ParseManifest to fail")
	}
	if !errors.Is(err, ErrManifestMissingRequired) {
		t.Fatalf("expected missing required field error, got %v", err)
	}
	if !strings.Contains(err.Error(), "id") {
		t.Fatalf("expected missing id error detail, got %v", err)
	}
}
