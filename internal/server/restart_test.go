package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/guidiju-50/spyscreen/internal/capture"
	"github.com/guidiju-50/spyscreen/internal/capture/mock"
)

func TestStartStopStart(t *testing.T) {
	cfg := testConfig()
	cap := mock.New(capture.Options{FPS: 20, Width: 32, Height: 32})
	srv, err := New(Options{
		Config:    cfg,
		Capturer:  cap,
		CapOpts:   capture.Options{FPS: 20, Width: 32, Height: 32},
		Lifetime:  context.Background(),
		AuthToken: "secret",
	})
	if err != nil {
		t.Fatal(err)
	}
	h := srv.Handler()
	auth := func(method, path string) *httptest.ResponseRecorder {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(method, path, nil)
		req.Header.Set("Authorization", "Bearer secret")
		h.ServeHTTP(rec, req)
		return rec
	}

	if auth(http.MethodPost, "/api/start").Code != http.StatusOK {
		t.Fatal("first start failed")
	}
	if auth(http.MethodPost, "/api/stop").Code != http.StatusOK {
		t.Fatal("stop failed")
	}
	if auth(http.MethodPost, "/api/start").Code != http.StatusOK {
		t.Fatal("second start failed")
	}
}
