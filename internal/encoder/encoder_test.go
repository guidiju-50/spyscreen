package encoder

import (
	"errors"
	"testing"
)

func TestH264NotImplemented(t *testing.T) {
	_, err := New(Config{Type: "h264"})
	if !errors.Is(err, ErrNotImplemented) {
		t.Fatalf("expected ErrNotImplemented, got %v", err)
	}
}
