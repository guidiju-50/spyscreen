package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadValidExample(t *testing.T) {
	path := filepath.Join("..", "..", "configs", "config.example.yaml")
	os.Setenv("SCREENSHARE_TOKEN", "test-token")
	t.Cleanup(func() { os.Unsetenv("SCREENSHARE_TOKEN") })

	cfg, err := Load(path, false)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Capture.FPS != 15 {
		t.Fatalf("fps: got %d", cfg.Capture.FPS)
	}
	if cfg.ResolveToken() != "test-token" {
		t.Fatalf("token mismatch")
	}
}

func TestValidateRejectsBadFPS(t *testing.T) {
	cfg := &Config{
		Server: ServerConfig{
			Host: "127.0.0.1", Port: 8080, MaxClients: 1,
			ReadTimeoutSeconds: 5, WriteTimeoutSeconds: 5,
		},
		Capture:  CaptureConfig{FPS: 7, Width: 640, Height: 480},
		Encoding: EncodingConfig{Type: "jpeg", JPEGQuality: 70},
		Stream:   StreamConfig{Protocol: "mjpeg", BufferSize: 2},
	}
	if err := cfg.Validate(true); err == nil {
		t.Fatal("expected fps error")
	}
}

func TestValidateRequiresTokenOnLAN(t *testing.T) {
	cfg := &Config{
		Server: ServerConfig{
			Host: "0.0.0.0", Port: 8080, MaxClients: 1,
			ReadTimeoutSeconds: 5, WriteTimeoutSeconds: 5,
		},
		Capture:  CaptureConfig{FPS: 15, Width: 640, Height: 480},
		Encoding: EncodingConfig{Type: "jpeg", JPEGQuality: 70},
		Stream:   StreamConfig{Protocol: "mjpeg", BufferSize: 2},
	}
	if err := cfg.Validate(false); err == nil {
		t.Fatal("expected token error")
	}
}
