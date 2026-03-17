package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/ricki/codexsess/internal/config"
	icrypto "github.com/ricki/codexsess/internal/crypto"
	"github.com/ricki/codexsess/internal/httpapi"
	"github.com/ricki/codexsess/internal/service"
	"github.com/ricki/codexsess/internal/store"
	"github.com/ricki/codexsess/internal/trafficlog"
	"github.com/ricki/codexsess/internal/util"
)

func main() {
	if err := run(); err != nil {
		log.Fatalf("codexsess: %v", err)
	}
}

func run() error {
	if len(os.Args) > 1 {
		return fmt.Errorf("no command-line arguments are supported")
	}

	cfg, err := config.LoadOrInit()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	if err := os.MkdirAll(cfg.DataDir, 0o700); err != nil {
		return fmt.Errorf("prepare data dir: %w", err)
	}
	if err := os.MkdirAll(cfg.AuthStoreDir, 0o700); err != nil {
		return fmt.Errorf("prepare auth store dir: %w", err)
	}
	if err := os.MkdirAll(cfg.CodexHome, 0o700); err != nil {
		return fmt.Errorf("prepare codex home dir: %w", err)
	}

	key, err := util.LoadOrCreateMasterKey(cfg.MasterKeyPath)
	if err != nil {
		return fmt.Errorf("load master key: %w", err)
	}
	cry, err := icrypto.New(key)
	if err != nil {
		return fmt.Errorf("init crypto: %w", err)
	}

	st, err := store.Open(filepath.Join(cfg.DataDir, "data.db"))
	if err != nil {
		return fmt.Errorf("open store: %w", err)
	}
	defer st.Close()

	traffic, err := trafficlog.New(filepath.Join(cfg.DataDir, "traffic.log"), 2*1024*1024)
	if err != nil {
		return fmt.Errorf("init traffic logger: %w", err)
	}

	svc := service.New(cfg, st, cry)
	srv := httpapi.New(svc, cfg.BindAddr, cfg.ProxyAPIKey, traffic)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	log.Printf("codexsess listening on http://%s", cfg.BindAddr)
	err = srv.ListenAndServe(ctx)
	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}

