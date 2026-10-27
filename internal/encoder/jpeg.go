package encoder

import (
	"bytes"
	"image"
	"image/color"
	"image/jpeg"
	"sync"

	"github.com/guidiju-50/spyscreen/internal/capture"
)

type JPEGEncoder struct {
	quality int
	pool    sync.Pool
}

func NewJPEG(quality int) *JPEGEncoder {
	if quality < 1 {
		quality = 70
	}
	if quality > 100 {
		quality = 100
	}
	return &JPEGEncoder{quality: quality}
}

func (e *JPEGEncoder) Close() error { return nil }

func (e *JPEGEncoder) Encode(frame capture.Frame) ([]byte, error) {
	defer frame.Release()

	w, h := frame.Width, frame.Height
	if w <= 0 || h <= 0 || len(frame.Data) < w*h*4 {
		return nil, image.ErrFormat
	}

	img := e.pool.Get()
	var rgba *image.RGBA
	if img != nil {
		rgba = img.(*image.RGBA)
		if rgba.Bounds().Dx() != w || rgba.Bounds().Dy() != h {
			rgba = image.NewRGBA(image.Rect(0, 0, w, h))
		}
	} else {
		rgba = image.NewRGBA(image.Rect(0, 0, w, h))
	}
	defer e.pool.Put(rgba)

	src := frame.Data
	for y := 0; y < h; y++ {
		row := rgba.Pix[y*rgba.Stride : y*rgba.Stride+w*4]
		off := y * w * 4
		for x := 0; x < w; x++ {
			i := off + x*4
			row[x*4+0] = src[i+2]
			row[x*4+1] = src[i+1]
			row[x*4+2] = src[i+0]
			row[x*4+3] = 255
		}
	}

	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, rgba, &jpeg.Options{Quality: e.quality}); err != nil {
		return nil, err
	}
	out := make([]byte, buf.Len())
	copy(out, buf.Bytes())
	return out, nil
}

// FillSolidRGBA is a test helper.
func FillSolidRGBA(w, h int, c color.RGBA) capture.Frame {
	data := make([]byte, w*h*4)
	for i := 0; i < w*h; i++ {
		o := i * 4
		data[o+0] = c.B
		data[o+1] = c.G
		data[o+2] = c.R
		data[o+3] = c.A
	}
	return capture.Frame{
		Width:  w,
		Height: h,
		Format: capture.PixelFormatBGRA,
		Data:   data,
	}
}
