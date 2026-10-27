//go:build darwin && integration

package integration

import (
	"context"
	"testing"
	"time"

	"github.com/guidiju-50/spyscreen/internal/capture"
	"github.com/guidiju-50/spyscreen/internal/capture/darwin"
)

func TestDarwinCaptureOneFrame(t *testing.T) {
	cap := darwin.New(capture.Options{FPS: 5, Width: 640, Height: 480})
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	if err := cap.Start(ctx); err != nil {
		t.Skipf("capture unavailable (permissions or display): %v", err)
	}
	defer cap.Stop()

	select {
	case frame := <-cap.Frames():
		if len(frame.Data) == 0 {
			t.Fatal("empty frame")
		}
	case <-ctx.Done():
		t.Fatal("timeout waiting for frame — grant Screen Recording to Terminal/Cursor, then retry; if permission is already granted, rebuild after updating capture code")
	}
}
