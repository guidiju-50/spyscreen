package streaming

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/guidiju-50/spyscreen/internal/capture"
	"github.com/guidiju-50/spyscreen/internal/encoder"
	"github.com/guidiju-50/spyscreen/internal/metrics"
)

var ErrAlreadyRunning = errors.New("stream already running")

type EncodedFrame struct {
	JPEG      []byte
	Timestamp time.Time
	Width     int
	Height    int
}

type Broadcaster struct {
	mu sync.RWMutex

	capturer capture.ScreenCapturer
	enc      encoder.Encoder
	metrics  *metrics.Collector
	dropOld  bool

	runCtx    context.Context
	runCancel context.CancelFunc
	running   bool
	lastErr   string
	width     int
	height    int

	subMu   sync.Mutex
	subs    map[chan EncodedFrame]struct{}
	latest  EncodedFrame
	hasLatest bool
}

func NewBroadcaster(m *metrics.Collector, dropOld bool) *Broadcaster {
	return &Broadcaster{
		metrics: m,
		dropOld: dropOld,
		subs:    make(map[chan EncodedFrame]struct{}),
	}
}

func (b *Broadcaster) Running() bool {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.running
}

func (b *Broadcaster) LastError() string {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.lastErr
}

func (b *Broadcaster) Dimensions() (int, int) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.width, b.height
}

func (b *Broadcaster) Start(ctx context.Context, cap capture.ScreenCapturer, enc encoder.Encoder) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.running {
		return ErrAlreadyRunning
	}
	b.capturer = cap
	b.enc = enc
	b.runCtx, b.runCancel = context.WithCancel(ctx)
	b.running = true
	b.lastErr = ""

	if err := cap.Start(b.runCtx); err != nil {
		b.running = false
		b.runCancel()
		b.lastErr = err.Error()
		return err
	}

	go b.pipeline()
	return nil
}

func (b *Broadcaster) Stop() error {
	b.mu.Lock()
	if !b.running {
		b.mu.Unlock()
		return nil
	}
	cancel := b.runCancel
	cap := b.capturer
	enc := b.enc
	b.running = false
	b.mu.Unlock()

	if cancel != nil {
		cancel()
	}
	if cap != nil {
		_ = cap.Stop()
	}
	if enc != nil {
		_ = enc.Close()
	}

	b.subMu.Lock()
	for ch := range b.subs {
		close(ch)
	}
	b.subs = make(map[chan EncodedFrame]struct{})
	b.hasLatest = false
	b.subMu.Unlock()
	return nil
}

func (b *Broadcaster) Subscribe(buffer int) (<-chan EncodedFrame, func()) {
	if buffer < 1 {
		buffer = 1
	}
	ch := make(chan EncodedFrame, buffer)
	b.subMu.Lock()
	b.subs[ch] = struct{}{}
	if b.hasLatest {
		select {
		case ch <- b.latest:
		default:
		}
	}
	b.subMu.Unlock()
	unsub := func() {
		b.subMu.Lock()
		defer b.subMu.Unlock()
		if _, ok := b.subs[ch]; !ok {
			return
		}
		delete(b.subs, ch)
		close(ch)
	}
	return ch, unsub
}

func (b *Broadcaster) pipeline() {
	frames := b.capturer.Frames()
	for {
		select {
		case <-b.runCtx.Done():
			return
		case frame, ok := <-frames:
			if !ok {
				return
			}
			if !frame.Timestamp.IsZero() {
				b.metrics.RecordCapture(time.Since(frame.Timestamp))
			} else {
				b.metrics.RecordCapture(0)
			}

			b.mu.Lock()
			b.width = frame.Width
			b.height = frame.Height
			b.mu.Unlock()

			encStart := time.Now()
			jpeg, err := b.enc.Encode(frame) // Encode releases the frame
			b.metrics.RecordEncode(time.Since(encStart))
			if err != nil {
				b.mu.Lock()
				b.lastErr = err.Error()
				b.mu.Unlock()
				continue
			}

			ef := EncodedFrame{
				JPEG:      jpeg,
				Timestamp: time.Now(),
				Width:     frame.Width,
				Height:    frame.Height,
			}
			b.publish(ef)
		}
	}
}

func (b *Broadcaster) publish(ef EncodedFrame) {
	b.subMu.Lock()
	defer b.subMu.Unlock()
	b.latest = ef
	b.hasLatest = true
	b.metrics.RecordStream()

	for ch := range b.subs {
		if b.dropOld {
			select {
			case <-ch:
				b.metrics.AddDropped()
			default:
			}
		}
		select {
		case ch <- ef:
		default:
			b.metrics.AddDropped()
		}
	}
}
