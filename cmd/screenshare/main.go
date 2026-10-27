package main

import (
	"context"
	"errors"
	"flag"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/guidiju-50/spyscreen/internal/capture"
	"github.com/guidiju-50/spyscreen/internal/config"
	"github.com/guidiju-50/spyscreen/internal/server"
)

func main() {
	configPath := flag.String("config", "configs/config.yaml", "path to config YAML")
	insecure := flag.Bool("insecure-allow-no-token", false, "allow empty token (local development only)")
	flag.Parse()

	cfg, err := config.Load(*configPath, *insecure)
	if err != nil {
		slog.Error("config", "err", err)
		os.Exit(1)
	}

	capOpts := capture.Options{
		DisplayID: cfg.Capture.DisplayID,
		Width:     cfg.Capture.Width,
		Height:    cfg.Capture.Height,
		FPS:       cfg.Capture.FPS,
	}
	capturer, err := newCapturer(capOpts)
	if err != nil {
		slog.Error("capturer", "err", err)
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	srv, err := server.New(server.Options{
		Config:   cfg,
		Capturer: capturer,
		CapOpts:  capOpts,
		Lifetime: ctx,
	})
	if err != nil {
		slog.Error("server init", "err", err)
		os.Exit(1)
	}

	go func() {
		slog.Info("listening", "addr", cfg.ListenAddr(), "tls", cfg.Security.TLSEnabled)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("http server", "err", err)
			stop()
		}
	}()

	<-ctx.Done()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		slog.Error("shutdown", "err", err)
	}
}
