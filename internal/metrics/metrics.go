package metrics

import (
	"sync/atomic"
	"time"
)

type Snapshot struct {
	CaptureFPS        float64 `json:"capture_fps"`
	EncodeFPS         float64 `json:"encode_fps"`
	StreamFPS         float64 `json:"stream_fps"`
	CaptureDurationMS float64 `json:"capture_duration_ms"`
	EncodeDurationMS  float64 `json:"encode_duration_ms"`
	ConnectedClients  int64   `json:"connected_clients"`
	BytesSent         int64   `json:"bytes_sent"`
	FramesDropped     int64   `json:"frames_dropped"`
	Running           bool    `json:"running"`
	Width             int     `json:"width"`
	Height            int     `json:"height"`
	LastError         string  `json:"last_error,omitempty"`
}

type Collector struct {
	captureCount atomic.Uint64
	encodeCount  atomic.Uint64
	streamCount  atomic.Uint64
	bytesSent    atomic.Int64
	framesDrop   atomic.Int64
	clients      atomic.Int64

	captureDurNS atomic.Uint64
	encodeDurNS  atomic.Uint64

	lastCapture atomic.Int64
	lastEncode  atomic.Int64
	lastStream  atomic.Int64
}

func New() *Collector {
	return &Collector{}
}

func (c *Collector) SetClients(n int64) {
	c.clients.Store(n)
}

func (c *Collector) AddClient()  { c.clients.Add(1) }
func (c *Collector) SubClient()  { c.clients.Add(-1) }
func (c *Collector) AddDropped() { c.framesDrop.Add(1) }
func (c *Collector) AddBytes(n int64) {
	c.bytesSent.Add(n)
}

func (c *Collector) RecordCapture(d time.Duration) {
	c.captureCount.Add(1)
	c.captureDurNS.Add(uint64(d.Nanoseconds()))
	c.lastCapture.Store(time.Now().UnixNano())
}

func (c *Collector) RecordEncode(d time.Duration) {
	c.encodeCount.Add(1)
	c.encodeDurNS.Add(uint64(d.Nanoseconds()))
	c.lastEncode.Store(time.Now().UnixNano())
}

func (c *Collector) RecordStream() {
	c.streamCount.Add(1)
	c.lastStream.Store(time.Now().UnixNano())
}

func fps(count uint64, since time.Time) float64 {
	if count == 0 {
		return 0
	}
	elapsed := time.Since(since).Seconds()
	if elapsed <= 0 {
		return 0
	}
	return float64(count) / elapsed
}

func (c *Collector) Snapshot(running bool, width, height int, lastErr string) Snapshot {
	capN := c.captureCount.Load()
	encN := c.encodeCount.Load()
	strN := c.streamCount.Load()

	capSince := time.Unix(0, c.lastCapture.Load())
	encSince := time.Unix(0, c.lastEncode.Load())
	strSince := time.Unix(0, c.lastStream.Load())

	var capDur, encDur float64
	if capN > 0 {
		capDur = float64(c.captureDurNS.Load()) / float64(capN) / 1e6
	}
	if encN > 0 {
		encDur = float64(c.encodeDurNS.Load()) / float64(encN) / 1e6
	}

	return Snapshot{
		CaptureFPS:        fps(capN, capSince),
		EncodeFPS:         fps(encN, encSince),
		StreamFPS:         fps(strN, strSince),
		CaptureDurationMS: capDur,
		EncodeDurationMS:  encDur,
		ConnectedClients:  c.clients.Load(),
		BytesSent:         c.bytesSent.Load(),
		FramesDropped:     c.framesDrop.Load(),
		Running:           running,
		Width:             width,
		Height:            height,
		LastError:         lastErr,
	}
}
