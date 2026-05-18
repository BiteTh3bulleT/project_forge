package main

import (
	"context"
	"errors"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"forge/projectforge/services/core/internal/api"
	"forge/projectforge/services/core/internal/config"
	"forge/projectforge/services/core/internal/store"
)

func main() {
	cfg := config.Load()
	if err := validateCoreListenConfig(cfg); err != nil {
		slog.Error("invalid configuration", slog.String("error", err.Error()))
		os.Exit(1)
	}

	st, err := store.Open(cfg.DataDir)
	if err != nil {
		slog.Error("store open failed", slog.String("error", err.Error()))
		os.Exit(1)
	}
	defer st.Close()

	srv := api.NewServer(st, cfg)
	defer srv.ShutdownWatch()

	addr := coreListenAddr(cfg)
	httpSrv := &http.Server{
		Addr:              addr,
		Handler:           srv.Handler(),
		ReadHeaderTimeout: 10 * time.Second,
	}

	go func() {
		slog.Info("forge-core listening", slog.String("addr", addr))
		if err := httpSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("http server failed", slog.String("error", err.Error()))
			os.Exit(1)
		}
	}()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	<-ctx.Done()
	stop()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	_ = httpSrv.Shutdown(shutdownCtx)
}

var (
	errWildcardBindRequiresOptIn = errors.New("wildcard bind host requires FORGE_ALLOW_WILDCARD_BIND=true")
	errWildcardBindRequiresAuth  = errors.New("wildcard bind host requires API auth token")
)

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
