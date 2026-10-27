//go:build darwin

package main

import (
	"github.com/guidiju-50/spyscreen/internal/capture"
	"github.com/guidiju-50/spyscreen/internal/capture/darwin"
)

func newCapturer(opts capture.Options) (capture.ScreenCapturer, error) {
	return darwin.New(opts), nil
}
