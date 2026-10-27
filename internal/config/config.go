package config

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

var allowedFPS = map[int]struct{}{
	5: {}, 10: {}, 15: {}, 24: {}, 30: {}, 60: {},
}

type Config struct {
	Server   ServerConfig   `yaml:"server"`
	Capture  CaptureConfig  `yaml:"capture"`
	Encoding EncodingConfig `yaml:"encoding"`
	Stream   StreamConfig   `yaml:"stream"`
	Security SecurityConfig `yaml:"security"`
	Logging  LoggingConfig  `yaml:"logging"`
}

type ServerConfig struct {
	Host                string `yaml:"host"`
	Port                int    `yaml:"port"`
	MaxClients          int    `yaml:"max_clients"`
	ReadTimeoutSeconds  int    `yaml:"read_timeout_seconds"`
	WriteTimeoutSeconds int    `yaml:"write_timeout_seconds"`
}

type CaptureConfig struct {
	DisplayID uint32 `yaml:"display_id"`
	FPS       int    `yaml:"fps"`
	Width     int    `yaml:"width"`
	Height    int    `yaml:"height"`
}

type EncodingConfig struct {
	Type           string `yaml:"type"`
	JPEGQuality    int    `yaml:"jpeg_quality"`
	DropOldFrames  bool   `yaml:"drop_old_frames"`
}

type StreamConfig struct {
	Protocol   string `yaml:"protocol"`
	BufferSize int    `yaml:"buffer_size"`
}

type SecurityConfig struct {
	TokenEnv     string `yaml:"token_env"`
	TLSEnabled   bool   `yaml:"tls_enabled"`
	TLSCertFile  string `yaml:"tls_cert_file"`
	TLSKeyFile   string `yaml:"tls_key_file"`
}

type LoggingConfig struct {
	Level  string `yaml:"level"`
	Format string `yaml:"format"`
}

func Load(path string, insecureAllowNoToken bool) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}
	if err := cfg.Validate(insecureAllowNoToken); err != nil {
		return nil, err
	}
	return &cfg, nil
}

// Validate checks configuration. When insecureAllowNoToken is true, an empty token is allowed (dev only).
func (c *Config) Validate(insecureAllowNoToken bool) error {
	if c.Server.Host == "" {
		return errors.New("server.host is required")
	}
	if c.Server.Port < 1 || c.Server.Port > 65535 {
		return fmt.Errorf("server.port invalid: %d", c.Server.Port)
	}
	if c.Server.MaxClients < 1 {
		return errors.New("server.max_clients must be at least 1")
	}
	if c.Server.ReadTimeoutSeconds < 1 {
		return errors.New("server.read_timeout_seconds must be at least 1")
	}
	if c.Server.WriteTimeoutSeconds < 1 {
		return errors.New("server.write_timeout_seconds must be at least 1")
	}
	if _, ok := allowedFPS[c.Capture.FPS]; !ok {
		return fmt.Errorf("capture.fps must be one of 5,10,15,24,30,60; got %d", c.Capture.FPS)
	}
	if c.Capture.Width < 1 || c.Capture.Height < 1 {
		return errors.New("capture width and height must be positive")
	}
	enc := strings.ToLower(strings.TrimSpace(c.Encoding.Type))
	if enc != "jpeg" && enc != "h264" {
		return fmt.Errorf("encoding.type must be jpeg or h264; got %q", c.Encoding.Type)
	}
	if enc == "jpeg" {
		if c.Encoding.JPEGQuality < 1 || c.Encoding.JPEGQuality > 100 {
			return fmt.Errorf("encoding.jpeg_quality must be 1-100; got %d", c.Encoding.JPEGQuality)
		}
	}
	if c.Stream.BufferSize < 1 || c.Stream.BufferSize > 32 {
		return errors.New("stream.buffer_size must be between 1 and 32")
	}
	proto := strings.ToLower(strings.TrimSpace(c.Stream.Protocol))
	if proto != "mjpeg" {
		return fmt.Errorf("stream.protocol must be mjpeg in phase 1; got %q", c.Stream.Protocol)
	}
	if c.Security.TLSEnabled {
		if c.Security.TLSCertFile == "" || c.Security.TLSKeyFile == "" {
			return errors.New("tls_enabled requires tls_cert_file and tls_key_file")
		}
	}
	token := c.ResolveToken()
	if token == "" && !insecureAllowNoToken {
		if !isLoopbackBind(c.Server.Host) {
			return errors.New("security token is required when binding to non-loopback addresses; set SCREENSHARE_TOKEN or use --insecure-allow-no-token for local dev only")
		}
	}
	return nil
}

func (c *Config) ResolveToken() string {
	envName := c.Security.TokenEnv
	if envName == "" {
		envName = "SCREENSHARE_TOKEN"
	}
	return strings.TrimSpace(os.Getenv(envName))
}

func (c *Config) ListenAddr() string {
	return fmt.Sprintf("%s:%d", c.Server.Host, c.Server.Port)
}

func (c *Config) ReadTimeout() time.Duration {
	return time.Duration(c.Server.ReadTimeoutSeconds) * time.Second
}

func (c *Config) WriteTimeout() time.Duration {
	return time.Duration(c.Server.WriteTimeoutSeconds) * time.Second
}

func isLoopbackBind(host string) bool {
	h := strings.TrimSpace(strings.ToLower(host))
	return h == "127.0.0.1" || h == "localhost" || h == "::1"
}
