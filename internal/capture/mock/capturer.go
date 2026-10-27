package mock

import (
	"context"
	"sync"
	"time"

	"github.com/guidiju-50/spyscreen/internal/capture"
)

type Capturer struct {
	opts capture.Options

	mu      sync.Mutex
	running bool
	stopCh  chan struct{}
	frames  chan capture.Frame
	wg      sync.WaitGroup
	seq     int
}

func New(opts capture.Options) *Capturer {
	if opts.FPS <= 0 {
		opts.FPS = 15
	}
	if opts.Width <= 0 {
		opts.Width = 640
	}
	if opts.Height <= 0 {
		opts.Height = 480
	}
	return &Capturer{opts: opts}
}

func (c *Capturer) ListDisplays(_ context.Context) ([]capture.DisplayInfo, error) {
	return []capture.DisplayInfo{{
		DisplayID: c.opts.DisplayID,
		Width:     c.opts.Width,
		Height:    c.opts.Height,
		Name:      "mock-display",
	}}, nil
}

func (c *Capturer) Start(ctx context.Context) error {
	c.mu.Lock()
	if c.running {
		c.mu.Unlock()
		return nil
	}
	c.running = true
	c.stopCh = make(chan struct{})
	c.frames = make(chan capture.Frame, 2)
	stopCh := c.stopCh
	c.wg.Add(1)
	c.mu.Unlock()

	go func() {
		defer c.wg.Done()
		c.loop(ctx, stopCh)
	}()
	return nil
}

func (c *Capturer) loop(ctx context.Context, stopCh <-chan struct{}) {
	interval := time.Second / time.Duration(c.opts.FPS)
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-stopCh:
			return
		case <-ticker.C:
			frame := c.syntheticFrame()
			c.mu.Lock()
			ch := c.frames
			c.mu.Unlock()
			if ch == nil {
				return
			}
			select {
			case ch <- frame:
			default:
				select {
				case old := <-ch:
					old.Release()
				default:
				}
				select {
				case ch <- frame:
				default:
					frame.Release()
				}
			}
		}
	}
}

func (c *Capturer) syntheticFrame() capture.Frame {
	c.seq++
	w, h := c.opts.Width, c.opts.Height
	data := make([]byte, w*h*4)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			i := (y*w + x) * 4
			data[i+0] = byte((x + c.seq) % 256)
			data[i+1] = byte((y + c.seq) % 256)
			data[i+2] = byte(c.seq % 256)
			data[i+3] = 255
		}
	}
	return capture.Frame{
		Timestamp: time.Now(),
		Width:     w,
		Height:    h,
		Format:    capture.PixelFormatBGRA,
		Data:      data,
	}
}

func (c *Capturer) Frames() <-chan capture.Frame {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.frames
}

func (c *Capturer) Stop() error {
	c.mu.Lock()
	if !c.running {
		c.mu.Unlock()
		return nil
	}
	close(c.stopCh)
	c.running = false
	ch := c.frames
	c.frames = nil
	c.mu.Unlock()

	c.wg.Wait()
	if ch != nil {
		close(ch)
	}
	return nil
}
