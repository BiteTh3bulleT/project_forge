package main

import (
	"context"
	"errors"
	"log"
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
		log.Fatalf("config: %v", err)
	}

	st, err := store.Open(cfg.DataDir)
	if err != nil {
		log.Fatalf("store: %v", err)
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
		log.Printf("forge-core listening on http://%s", addr)
		if err := httpSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("http: %v", err)
		}
	}()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	<-ctx.Done()
	stop()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	_ = httpSrv.Shutdown(shutdownCtx)
}

var errWildcardBindRequiresOptIn = errors.New("wildcard bind host requires FORGE_ALLOW_WILDCARD_BIND=true")

func validateCoreListenConfig(cfg config.Config) error {
	if isWildcardBindHost(cfg.BindHost) && !cfg.AllowWildcardBind {
		return errWildcardBindRequiresOptIn
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
