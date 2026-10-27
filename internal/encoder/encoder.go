package encoder

import (
	"errors"

	"github.com/guidiju-50/spyscreen/internal/capture"
)

var ErrNotImplemented = errors.New("encoder not implemented")

type Encoder interface {
	Encode(frame capture.Frame) ([]byte, error)
	Close() error
}

type Config struct {
	Type        string
	JPEGQuality int
}

func New(cfg Config) (Encoder, error) {
	switch cfg.Type {
	case "jpeg", "":
		return NewJPEG(cfg.JPEGQuality), nil
	case "h264":
		return nil, ErrNotImplemented
	default:
		return nil, errors.New("unknown encoder type")
	}
}
