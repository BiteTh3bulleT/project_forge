package main

import (
	"context"
	"errors"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"forge/projectforge/services/core/internal/api"
	"forge/projectforge/services/core/internal/config"
	"forge/projectforge/services/core/internal/store"
)

func main() {
	os.Exit(run())
}

func run() int {
	cfg := config.Load()
	if err := validateCoreConfig(cfg); err != nil {
		slog.Error("invalid configuration", slog.String("error", err.Error()))
		return 1
	}

	st, err := store.Open(cfg.DataDir)
	if err != nil {
		slog.Error("store open failed", slog.String("error", err.Error()))
		return 1
	}
	defer st.Close()

	if strings.TrimSpace(cfg.APIToken) == "" {
		slog.Warn("FORGE_API_TOKEN is empty — protected routes are unauthenticated; only safe on loopback bind", slog.String("bind", cfg.BindHost))
	}

	srv := api.NewServer(st, cfg)
	defer srv.ShutdownWatch()

	addr := coreListenAddr(cfg)
	httpSrv := &http.Server{
		Addr:              addr,
		Handler:           srv.Handler(),
		ReadHeaderTimeout: 10 * time.Second,
	}

	serverErr := make(chan error, 1)
	go func() {
		slog.Info("forge-core listening", slog.String("addr", addr))
		if err := httpSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			serverErr <- err
			return
		}
		serverErr <- nil
	}()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	exitCode := 0
	select {
	case <-ctx.Done():
	case err := <-serverErr:
		if err != nil {
			slog.Error("http server failed", slog.String("error", err.Error()))
			exitCode = 1
		}
	}
	stop()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	_ = httpSrv.Shutdown(shutdownCtx)

	return exitCode
}

var (
	errWildcardBindRequiresOptIn  = errors.New("wildcard bind host requires FORGE_ALLOW_WILDCARD_BIND=true")
	errWildcardBindRequiresAuth   = errors.New("wildcard bind host requires API auth token")
	errRootWorkspaceRequiresOptIn = errors.New("root workspace requires FORGE_ALLOW_ROOT_WORKSPACE=true")
)

func validateCoreConfig(cfg config.Config) error {
	if err := validateCoreListenConfig(cfg); err != nil {
		return err
	}
	if isRootWorkspaceDir(cfg.WorkspaceDir) && !cfg.AllowRootWorkspace {
		return errRootWorkspaceRequiresOptIn
	}
	return nil
}

func validateCoreListenConfig(cfg config.Config) error {
	if isWildcardBindHost(cfg.BindHost) {
		if !cfg.AllowWildcardBind {
			return errWildcardBindRequiresOptIn
		}
		if strings.TrimSpace(cfg.APIToken) == "" {
			return errWildcardBindRequiresAuth
		}
	}
	return nil
}

func coreListenAddr(cfg config.Config) string {
	host := strings.TrimSpace(cfg.BindHost)
	if host == "" {
		host = "127.0.0.1"
	}
	return net.JoinHostPort(host, strconv.Itoa(cfg.Port))
}

func isWildcardBindHost(host string) bool {
	switch strings.Trim(strings.TrimSpace(host), "[]") {
	case "0.0.0.0", "::":
		return true
	default:
		return false
	}
}

func isRootWorkspaceDir(path string) bool {
	clean := filepath.Clean(strings.TrimSpace(path))
	if clean == "" || clean == "." {
		return false
	}
	if clean == string(filepath.Separator) {
		return true
	}
	volume := filepath.VolumeName(clean)
	if volume == "" {
		return false
	}
	return clean == volume+string(filepath.Separator)
}
