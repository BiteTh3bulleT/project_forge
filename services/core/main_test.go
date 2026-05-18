package main

import (
	"errors"
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
