//go:build darwin

package darwin

/*
#cgo CFLAGS: -x objective-c -Wno-deprecated-declarations -I${SRCDIR}/native
#cgo LDFLAGS: -framework ScreenCaptureKit -framework CoreMedia -framework CoreVideo -framework AVFoundation -framework CoreGraphics -framework Foundation -framework AppKit

#include "native/bridge.h"
#include <stdlib.h>
*/
import "C"

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"
	"unsafe"

	"github.com/guidiju-50/spyscreen/internal/capture"
)

var (
	activeMu       sync.Mutex
	activeCapturer *Capturer
)

type Capturer struct {
	opts capture.Options

	mu      sync.Mutex
	running bool
	frames  chan capture.Frame
}

func New(opts capture.Options) *Capturer {
	return &Capturer{opts: opts}
}

func (c *Capturer) ListDisplays(_ context.Context) ([]capture.DisplayInfo, error) {
	const max = 16
	cDisplays := make([]C.sc_display_info_t, max)
	errBuf := make([]byte, 512)
	n := C.sc_list_displays(&cDisplays[0], C.int(max), (*C.char)(unsafe.Pointer(&errBuf[0])), C.size_t(len(errBuf)))
	if n < 0 {
		return nil, mapNativeError(errBuf)
	}
	out := make([]capture.DisplayInfo, 0, int(n))
	for i := 0; i < int(n); i++ {
		d := cDisplays[i]
		out = append(out, capture.DisplayInfo{
			DisplayID: uint32(d.display_id),
			Width:     int(d.width),
			Height:    int(d.height),
			Name:      C.GoString(&d.name[0]),
		})
	}
	return out, nil
}

func (c *Capturer) Start(ctx context.Context) error {
	c.mu.Lock()
	if c.running {
		c.mu.Unlock()
		return nil
	}
	c.frames = make(chan capture.Frame, 2)
	c.running = true
	c.mu.Unlock()

	activeMu.Lock()
	activeCapturer = c
	activeMu.Unlock()

	displayID := C.uint32_t(c.opts.DisplayID)
	width := C.int(c.opts.Width)
	height := C.int(c.opts.Height)
	fps := C.int(c.opts.FPS)
	if fps <= 0 {
		fps = 15
	}

	errBuf := make([]byte, 512)
	rc := C.sc_stream_start(
		displayID,
		width,
		height,
		fps,
		C.sc_get_frame_callback(),
		nil,
		(*C.char)(unsafe.Pointer(&errBuf[0])),
		C.size_t(len(errBuf)),
	)
	if rc != 0 {
		activeMu.Lock()
		if activeCapturer == c {
			activeCapturer = nil
		}
		activeMu.Unlock()

		c.mu.Lock()
		c.running = false
		close(c.frames)
		c.frames = nil
		c.mu.Unlock()
		return mapNativeError(errBuf)
	}

	go func() {
		<-ctx.Done()
		_ = c.Stop()
	}()
	return nil
}

func (c *Capturer) deliverFrame(data []byte, w, h int, ts time.Time) {
	c.mu.Lock()
	if !c.running || c.frames == nil {
		c.mu.Unlock()
		return
	}
	ch := c.frames
	c.mu.Unlock()

	frame := capture.Frame{
		Timestamp: ts,
		Width:     w,
		Height:    h,
		Format:    capture.PixelFormatBGRA,
		Data:      data,
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
	c.running = false
	ch := c.frames
	c.frames = nil
	c.mu.Unlock()

	activeMu.Lock()
	if activeCapturer == c {
		activeCapturer = nil
	}
	activeMu.Unlock()

	C.sc_stream_stop()
	if ch != nil {
		close(ch)
	}
	return nil
}

func mapNativeError(errBuf []byte) error {
	msg := strings.TrimSpace(C.GoString((*C.char)(unsafe.Pointer(&errBuf[0]))))
	if msg == "" {
		msg = "screen capture failed"
	}
	lower := strings.ToLower(msg)
	if strings.Contains(lower, "permission") || strings.Contains(lower, "declined") || strings.Contains(lower, "not authorized") || strings.Contains(lower, "screen recording") {
		return fmt.Errorf("%w: %s", capture.ErrPermissionDenied, msg)
	}
	if strings.Contains(lower, "no displays") || strings.Contains(lower, "display not found") {
		return fmt.Errorf("%w: %s", capture.ErrDisplayNotFound, msg)
	}
	return fmt.Errorf("%s", msg)
}
