package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/guidiju-50/spyscreen/internal/capture"
	"github.com/guidiju-50/spyscreen/internal/capture/mock"
	"github.com/guidiju-50/spyscreen/internal/metrics"
)

func TestStatusAuthenticated(t *testing.T) {
	cfg := testConfig()
	cap := mock.New(capture.Options{FPS: 10, Width: 32, Height: 32})
	srv, err := New(Options{
		Config:    cfg,
		Capturer:  cap,
		CapOpts:   capture.Options{Width: 32, Height: 32},
		AuthToken: "secret",
	})
	if err != nil {
		t.Fatal(err)
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/status", nil)
	req.Header.Set("Authorization", "Bearer secret")
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status code: %d", rec.Code)
	}

	var snap metrics.Snapshot
	if err := json.NewDecoder(rec.Body).Decode(&snap); err != nil {
		t.Fatal(err)
	}
	if snap.Running {
		t.Fatal("expected not running before start")
	}
}
