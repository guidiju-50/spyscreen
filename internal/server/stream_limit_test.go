package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/guidiju-50/spyscreen/internal/capture"
	"github.com/guidiju-50/spyscreen/internal/capture/mock"
)

func TestStreamUnauthorizedNoSlot(t *testing.T) {
	cfg := testConfig()
	cfg.Server.MaxClients = 1
	cap := mock.New(capture.Options{FPS: 10, Width: 16, Height: 16})
	srv, err := New(Options{
		Config:    cfg,
		Capturer:  cap,
		CapOpts:   capture.Options{FPS: 10, Width: 16, Height: 16},
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

func TestStreamMaxClients(t *testing.T) {
	cfg := testConfig()
	cfg.Server.MaxClients = 1
	cap := mock.New(capture.Options{FPS: 30, Width: 16, Height: 16})
	srv, err := New(Options{
		Config:    cfg,
		Capturer:  cap,
		CapOpts:   capture.Options{FPS: 30, Width: 16, Height: 16},
		Lifetime:  context.Background(),
		AuthToken: "secret",
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = srv.bc.Stop() })
	h := srv.Handler()

	start := httptest.NewRecorder()
	startReq := httptest.NewRequest(http.MethodPost, "/api/start", nil)
	startReq.Header.Set("Authorization", "Bearer secret")
	h.ServeHTTP(start, startReq)
	if start.Code != http.StatusOK {
		t.Fatalf("start: %d", start.Code)
	}

	go func() {
		req := httptest.NewRequest(http.MethodGet, "/stream", nil)
		req.Header.Set("Authorization", "Bearer secret")
		h.ServeHTTP(httptest.NewRecorder(), req)
	}()

	time.Sleep(150 * time.Millisecond)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/stream", nil)
	req.Header.Set("Authorization", "Bearer secret")
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("expected 429, got %d", rec.Code)
	}
}
