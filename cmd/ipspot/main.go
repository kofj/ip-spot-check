package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/kofj/ip-spot-check/internal/config"
	"github.com/kofj/ip-spot-check/internal/httpapi"
	"github.com/kofj/ip-spot-check/internal/query"
	"github.com/kofj/ip-spot-check/internal/store"
)

func main() {
	if len(os.Args) > 1 && os.Args[1] == "migrate" {
		if err := migrateCmd(os.Args[2:]); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return
	}
	if err := serve(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func loadConfig() (config.Config, error) {
	path := os.Getenv("IPSPOT_CONFIG")
	if path == "" {
		path = "configs/config.yaml"
		if _, err := os.Stat(path); err != nil {
			path = ""
		}
	}
	return config.Load(path)
}

func openStore(cfg config.Config) (*store.Store, error) {
	return store.Open(cfg.Database.Driver, cfg.Database.DSN, cfg.Database.MaxOpenConns)
}

func migrateCmd(args []string) error {
	fs := flag.NewFlagSet("migrate", flag.ContinueOnError)
	steps := fs.Int("steps", 1, "回滚步数（仅 down）")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() == 0 {
		return errors.New("用法: ipspot migrate <up|down|version> [-steps N]")
	}

	cfg, err := loadConfig()
	if err != nil {
		return err
	}
	switch fs.Arg(0) {
	case "up":
		return store.MigrateUp(cfg.Database.DSN)
	case "down":
		return store.MigrateDown(cfg.Database.DSN, *steps)
	case "version":
		v, dirty, err := store.MigrateVersion(cfg.Database.DSN)
		if err != nil {
			return err
		}
		fmt.Printf("version=%d dirty=%v\n", v, dirty)
		return nil
	default:
		return fmt.Errorf("未知子命令: %s", fs.Arg(0))
	}
}

func serve() error {
	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	level := slog.LevelInfo
	if cfg.Log.Level == "debug" {
		level = slog.LevelDebug
	}
	var handler slog.Handler = slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: level})
	if cfg.Log.Format == "json" {
		handler = slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: level})
	}
	slog.SetDefault(slog.New(handler))

	st, err := openStore(cfg)
	if err != nil {
		return err
	}
	defer st.Close()

	gin.SetMode(gin.ReleaseMode)
	srv, err := httpapi.New(query.New(), st)
	if err != nil {
		return err
	}

	httpSrv := &http.Server{
		Addr:         cfg.Server.Addr,
		Handler:      srv.Router(),
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
	}

	errCh := make(chan error, 1)
	go func() {
		slog.Info("listening", "addr", cfg.Server.Addr)
		if err := httpSrv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	select {
	case err := <-errCh:
		return err
	case sig := <-sigCh:
		slog.Info("shutting down", "signal", sig.String())
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return httpSrv.Shutdown(ctx)
}
