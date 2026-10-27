# spyscreen — ScreenShare LAN (Go)

Real-time macOS screen capture over the local network, viewed in a browser on Linux (or any device on the LAN). Capture uses **ScreenCaptureKit** in a native Objective-C layer bridged to Go via cgo — not browser `getDisplayMedia()` on the Mac.

## Requirements

- **macOS 12.3+** (ScreenCaptureKit); Apple Silicon or Intel
- **Go 1.22+** and **Xcode Command Line Tools** (`xcode-select --install`)
- **Screen Recording** permission for the process doing capture (Terminal, IDE, or `bin/screenshare`)
- Linux (or other) machine on the same LAN with Chrome or Firefox

## Quick start (macOS host)

```bash
export SCREENSHARE_TOKEN="$(openssl rand -hex 24)"
cp configs/config.example.yaml configs/config.yaml   # if needed
make build-macos
./bin/screenshare -config configs/config.yaml
```

Grant **System Settings → Privacy & Security → Screen Recording** for Terminal (or your IDE) when prompted, then restart the app if macOS requires it.

## View on Linux

1. Open `http://<MAC_LAN_IP>:8080` (or `https://` if TLS is enabled).
2. Enter the same token (stored in sessionStorage, not the URL).
3. Click **Connect** (calls `POST /api/start` and opens authenticated MJPEG `GET /stream`).

Without TLS, use only on a trusted LAN. For regular use, enable `security.tls_enabled` and provide cert/key paths.

## Architecture (Phase 1)

| Component | Role |
|-----------|------|
| `internal/capture/darwin` | SCStream capture via ScreenCaptureKit |
| `internal/encoder` | JPEG (H.264 stub for Phase 4) |
| `internal/streaming` | Shared broadcaster + MJPEG multipart |
| `internal/server` | HTTP API, auth, static web UI |
| `web/` | Viewer + diagnostics placeholder (Phase 3) |

Endpoints: `/`, `/stream`, `/api/status`, `/api/displays`, `/api/start`, `/api/stop`. `/ws` and `/api/metrics` return 501 until Phase 2.

## Security

- Bearer token (`Authorization: Bearer …`) or `X-Screenshare-Token`
- Token from `SCREENSHARE_TOKEN` (or `security.token_env` in config)
- Startup fails on non-loopback bind without a token unless `--insecure-allow-no-token` (dev only)
- No token in query strings; no wide-open CORS

## Build & test

```bash
make build-macos
go test ./...
go test -race ./...

# macOS integration (Screen Recording permission required):
go test -tags=integration ./tests/integration/...
```

## Limitations

- Protected content and apps that block capture may appear blank or excluded (by design).
- macOS may show screen-recording indicators; this app does not disable them.
- Browser diagnostics **cannot** detect all capture methods (Phase 3 clarifies scenarios A/B/C).
- Latency shown in the viewer is a local estimate, not a synchronized end-to-end measurement.

## Roadmap

- **Phase 2:** WebSocket stream, `/api/metrics`, richer reconnect
- **Phase 3:** Chrome/Firefox diagnostics JSON export
- **Phase 4:** VideoToolbox H.264, optional WebRTC (Pion)
