package main

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"forge/projectforge/services/core/internal/config"
)

func TestCoreListenAddrUsesLoopbackBindHost(t *testing.T) {
	got := coreListenAddr(config.Config{BindHost: "127.0.0.1", Port: 18492})
	if got != "127.0.0.1:18492" {
		t.Fatalf("expected loopback listen address, got %q", got)
	}
}

func TestCoreListenAddrSupportsExplicitWildcardBindHost(t *testing.T) {
	got := coreListenAddr(config.Config{BindHost: "0.0.0.0", Port: 18492})
	if got != "0.0.0.0:18492" {
		t.Fatalf("expected explicit wildcard listen address, got %q", got)
	}
}

func TestCoreListenAddrFormatsIPv6BindHost(t *testing.T) {
	got := coreListenAddr(config.Config{BindHost: "::1", Port: 18492})
	if got != "[::1]:18492" {
		t.Fatalf("expected bracketed IPv6 listen address, got %q", got)
	}
}

func TestWildcardBindHostDetection(t *testing.T) {
	cases := []struct {
		host string
		want bool
	}{
		{"0.0.0.0", true},
		{"::", true},
		{"[::]", true},
		{"127.0.0.1", false},
		{"localhost", false},
		{"::1", false},
	}
	for _, tc := range cases {
		if got := isWildcardBindHost(tc.host); got != tc.want {
			t.Fatalf("isWildcardBindHost(%q) = %v, want %v", tc.host, got, tc.want)
		}
	}
}

func TestValidateCoreListenConfigRejectsWildcardWithoutOptIn(t *testing.T) {
	for _, host := range []string{"0.0.0.0", "::", "[::]", " 0.0.0.0 "} {
		err := validateCoreListenConfig(config.Config{BindHost: host})
		if !errors.Is(err, errWildcardBindRequiresOptIn) {
			t.Fatalf("host %q error = %v, want %v", host, err, errWildcardBindRequiresOptIn)
		}
	}
}

func TestValidateCoreListenConfigAllowsWildcardWithExplicitOptIn(t *testing.T) {
	for _, host := range []string{"0.0.0.0", "::", "[::]"} {
		err := validateCoreListenConfig(config.Config{BindHost: host, AllowWildcardBind: true, APIToken: "secret"})
		if err != nil {
			t.Fatalf("host %q unexpected error: %v", host, err)
		}
	}
}

func TestValidateCoreListenConfigRejectsWildcardWithoutAPIToken(t *testing.T) {
	err := validateCoreListenConfig(config.Config{BindHost: "0.0.0.0", AllowWildcardBind: true})
	if !errors.Is(err, errWildcardBindRequiresAuth) {
		t.Fatalf("error = %v, want %v", err, errWildcardBindRequiresAuth)
	}
}

func TestValidateCoreListenConfigAllowsLoopbackWithoutOptIn(t *testing.T) {
	for _, host := range []string{"127.0.0.1", "localhost", ""} {
		err := validateCoreListenConfig(config.Config{BindHost: host})
		if err != nil {
			t.Fatalf("host %q unexpected error: %v", host, err)
		}
	}
}

func TestValidateCoreConfigRejectsRootWorkspaceWithoutOptIn(t *testing.T) {
	err := validateCoreConfig(config.Config{
		BindHost:     "127.0.0.1",
		WorkspaceDir: filepath.Clean(string(filepath.Separator)),
	})
	if !errors.Is(err, errRootWorkspaceRequiresOptIn) {
		t.Fatalf("error = %v, want %v", err, errRootWorkspaceRequiresOptIn)
	}
}

func TestValidateCoreConfigAllowsRootWorkspaceWithExplicitOptIn(t *testing.T) {
	err := validateCoreConfig(config.Config{
		BindHost:           "127.0.0.1",
		WorkspaceDir:       filepath.Clean(string(filepath.Separator)),
		AllowRootWorkspace: true,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDockerComposeCoreDefaultsAreLoopbackAndWildcardOptInDisabled(t *testing.T) {
	body, err := os.ReadFile(filepath.Join("..", "..", "docker-compose.yml"))
	if err != nil {
		t.Fatalf("read docker-compose.yml: %v", err)
	}
	compose := string(body)
	if !strings.Contains(compose, `FORGE_CORE_BIND_HOST: "${FORGE_CORE_BIND_HOST:-127.0.0.1}"`) {
		t.Fatal("docker-compose.yml must default FORGE_CORE_BIND_HOST to 127.0.0.1")
	}
	if !strings.Contains(compose, `FORGE_ALLOW_WILDCARD_BIND: "${FORGE_ALLOW_WILDCARD_BIND:-false}"`) {
		t.Fatal("docker-compose.yml must keep wildcard bind opt-in disabled by default")
	}
	if !strings.Contains(compose, `FORGE_API_TOKEN: "${FORGE_API_TOKEN:-}"`) {
		t.Fatal("docker-compose.yml must keep API surfaces auth-gated by FORGE_API_TOKEN")
	}
	if !strings.Contains(compose, `${FORGE_DOCKER_BIND_HOST:-127.0.0.1}`) {
		t.Fatal("docker-compose.yml must keep host-published ports loopback-bound by default")
	}
}

func TestDockerHelperOptsIntoContainerWildcardWithGeneratedToken(t *testing.T) {
	for _, path := range []string{
		filepath.Join("..", "..", "scripts", "forge-docker-up.sh"),
		filepath.Join("..", "..", "scripts", "forge-docker-up.ps1"),
	} {
		body, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s: %v", path, err)
		}
		text := string(body)
		if !strings.Contains(text, "FORGE_CORE_BIND_HOST") || !strings.Contains(text, "0.0.0.0") {
			t.Fatalf("%s must explicitly set container-internal core bind host for published Docker ports", path)
		}
		if !strings.Contains(text, "FORGE_ALLOW_WILDCARD_BIND") || !strings.Contains(text, "true") {
			t.Fatalf("%s must explicitly opt into wildcard bind for the generated-token Docker helper path", path)
		}
		if !strings.Contains(text, "FORGE_API_TOKEN") || !strings.Contains(text, "docker-api-token") {
			t.Fatalf("%s must generate or pass a Docker API token before enabling wildcard bind", path)
		}
	}
}
