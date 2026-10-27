//go:build !darwin

package main

import (
	"errors"

	"github.com/guidiju-50/spyscreen/internal/capture"
)

func newCapturer(_ capture.Options) (capture.ScreenCapturer, error) {
	return nil, errors.New("screen capture is supported on macOS only")
}
