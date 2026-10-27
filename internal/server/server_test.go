package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/guidiju-50/spyscreen/internal/capture"
	"github.com/guidiju-50/spyscreen/internal/capture/mock"
	"github.com/guidiju-50/spyscreen/internal/config"
)

func testConfig() *config.Config {
	return &config.Config{
		Server: config.ServerConfig{
			Host: "127.0.0.1", Port: 8080, MaxClients: 2,
			ReadTimeoutSeconds: 5, WriteTimeoutSeconds: 5,
		},
		Capture:  config.CaptureConfig{FPS: 15, Width: 64, Height: 64},
		Encoding: config.EncodingConfig{Type: "jpeg", JPEGQuality: 70, DropOldFrames: true},
		Stream:   config.StreamConfig{Protocol: "mjpeg", BufferSize: 2},
	}
}

func TestStreamRequiresAuth(t *testing.T) {
	cfg := testConfig()
	cap := mock.New(capture.Options{FPS: 15, Width: 64, Height: 64})
	srv, err := New(Options{
		Config: cfg, Capturer: cap, CapOpts: capture.Options{Width: 64, Height: 64},
		AuthToken: "secret",
	})
	if err != nil {
		t.Fatal(err)
	}

	h := srv.Handler()
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/stream", nil))
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}
}

func TestStartStopAuthenticated(t *testing.T) {
	cfg := testConfig()
	cap := mock.New(capture.Options{FPS: 15, Width: 64, Height: 64})
	srv, err := New(Options{
		Config:    cfg,
		Capturer:  cap,
		CapOpts:   capture.Options{Width: 64, Height: 64, FPS: 15},
		Lifetime:  context.Background(),
		AuthToken: "secret",
	})
	if err != nil {
		t.Fatal(err)
	}
	h := srv.Handler()

	start := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/start", nil)
	h.ServeHTTP(start, req)
	if start.Code != http.StatusUnauthorized {
		t.Fatalf("start without auth: %d", start.Code)
	}
}
