package capture

import (
	"context"
	"errors"
)

var (
	ErrPermissionDenied = errors.New("screen capture permission denied")
	ErrDisplayNotFound  = errors.New("display not found")
	ErrNotRunning       = errors.New("capture not running")
)

type DisplayInfo struct {
	DisplayID uint32 `json:"display_id"`
	Width     int    `json:"width"`
	Height    int    `json:"height"`
	Name      string `json:"name,omitempty"`
}

type Options struct {
	DisplayID uint32
	Width     int
	Height    int
	FPS       int
}

type ScreenCapturer interface {
	Start(ctx context.Context) error
	Frames() <-chan Frame
	Stop() error
	ListDisplays(ctx context.Context) ([]DisplayInfo, error)
}
