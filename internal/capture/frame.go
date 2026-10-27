package capture

import "time"

type PixelFormat string

const (
	PixelFormatBGRA PixelFormat = "bgra"
	PixelFormatRGBA PixelFormat = "rgba"
)

// Frame holds one captured screen buffer. Call Release when done if a release hook was set.
type Frame struct {
	Timestamp time.Time
	Width     int
	Height    int
	Format    PixelFormat
	Data      []byte
	releaseFn func()
}

func (f Frame) Release() {
	if f.releaseFn != nil {
		f.releaseFn()
	}
}
