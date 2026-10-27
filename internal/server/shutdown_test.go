package server

import (
	"context"
	"testing"
	"time"

	"github.com/guidiju-50/spyscreen/internal/capture"
	"github.com/guidiju-50/spyscreen/internal/capture/mock"
)

func TestShutdownStopsBroadcaster(t *testing.T) {
	cfg := testConfig()
	cap := mock.New(capture.Options{FPS: 10, Width: 32, Height: 32})
	srv, err := New(Options{
		Config:    cfg,
		Capturer:  cap,
		CapOpts:   capture.Options{FPS: 10, Width: 32, Height: 32},
		Lifetime:  context.Background(),
		AuthToken: "secret",
	})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		t.Fatal(err)
	}
}
