package config

import (
	"os"
	"path/filepath"
	"strconv"
)

type Config struct {
	DataDir      string
	Port         int
	WorkspaceDir string
}

func Load() Config {
	dataDir := os.Getenv("FORGE_DATA_DIR")
	if dataDir == "" {
		base, err := os.UserConfigDir()
		if err != nil {
			base = "."
		}
		dataDir = filepath.Join(base, "forge")
	}
	port := 18492
	if v := os.Getenv("FORGE_CORE_PORT"); v != "" {
		if p, err := strconv.Atoi(v); err == nil && p > 0 {
			port = p
		}
	}
	workspace := os.Getenv("FORGE_WORKSPACE_DIR")
	if workspace == "" {
		cwd, err := os.Getwd()
		if err == nil {
			workspace = cwd
		} else {
			workspace = "."
		}
	}
	if abs, err := filepath.Abs(workspace); err == nil {
		workspace = abs
	}
	return Config{DataDir: dataDir, Port: port, WorkspaceDir: workspace}
}
