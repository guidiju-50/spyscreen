package streaming

import (
	"fmt"
	"net/http"
	"time"

	"github.com/guidiju-50/spyscreen/internal/metrics"
)

const mjpegBoundary = "screenshareframe"

func MJPEGHandler(b *Broadcaster, m *metrics.Collector) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "streaming unsupported", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", fmt.Sprintf("multipart/x-mixed-replace; boundary=%s", mjpegBoundary))
		w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
		w.Header().Set("Connection", "close")

		ch, unsub := b.Subscribe(2)
		defer unsub()

		for {
			select {
			case <-r.Context().Done():
				return
			case ef, ok := <-ch:
				if !ok {
					return
				}
				part := buildMJPEGPart(ef.JPEG)
				n, err := w.Write(part)
				if err != nil {
					return
				}
				if m != nil {
					m.AddBytes(int64(n))
				}
				flusher.Flush()
			}
		}
	})
}

func buildMJPEGPart(jpeg []byte) []byte {
	header := fmt.Sprintf("--%s\r\nContent-Type: image/jpeg\r\nContent-Length: %d\r\n\r\n", mjpegBoundary, len(jpeg))
	out := make([]byte, 0, len(header)+len(jpeg)+2)
	out = append(out, header...)
	out = append(out, jpeg...)
	out = append(out, '\r', '\n')
	return out
}

// WriteHeartbeat can be used in tests to detect stalled writers.
func WriteHeartbeat(w http.ResponseWriter, d time.Duration) error {
	deadline := time.Now().Add(d)
	for time.Now().Before(deadline) {
		if f, ok := w.(http.Flusher); ok {
			f.Flush()
		}
		time.Sleep(50 * time.Millisecond)
	}
	return nil
}
