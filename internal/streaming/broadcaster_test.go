package streaming

import (
	"context"
	"testing"
	"time"

	"github.com/guidiju-50/spyscreen/internal/capture"
	"github.com/guidiju-50/spyscreen/internal/capture/mock"
	"github.com/guidiju-50/spyscreen/internal/encoder"
	"github.com/guidiju-50/spyscreen/internal/metrics"
)

func TestBroadcasterPublishesFrames(t *testing.T) {
	m := metrics.New()
	bc := NewBroadcaster(m, true)

	cap := mock.New(capture.Options{FPS: 30, Width: 64, Height: 64})
	enc, err := encoder.New(encoder.Config{Type: "jpeg", JPEGQuality: 70})
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := bc.Start(ctx, cap, enc); err != nil {
		t.Fatal(err)
	}
	defer bc.Stop()

	ch, unsub := bc.Subscribe(1)
	defer unsub()

	select {
	case frame := <-ch:
		if len(frame.JPEG) == 0 {
			t.Fatal("empty jpeg")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for frame")
	}
}
