package server

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/guidiju-50/spyscreen/internal/capture"
	"github.com/guidiju-50/spyscreen/internal/config"
	"github.com/guidiju-50/spyscreen/internal/encoder"
	"github.com/guidiju-50/spyscreen/internal/metrics"
	"github.com/guidiju-50/spyscreen/internal/security"
	"github.com/guidiju-50/spyscreen/internal/streaming"
	"github.com/guidiju-50/spyscreen/web"
)

type Server struct {
	cfg     *config.Config
	auth    *security.Authenticator
	log     *slog.Logger
	metrics *metrics.Collector
	bc      *streaming.Broadcaster
	httpSrv *http.Server

	capturer capture.ScreenCapturer
	capOpts  capture.Options
	encCfg   encoder.Config
	lifetime context.Context

	mu            sync.Mutex
	streamClients int
}

type Options struct {
	Config   *config.Config
	Logger   *slog.Logger
	Capturer capture.ScreenCapturer
	CapOpts  capture.Options
	Lifetime context.Context
	AuthToken string // optional override (tests)
}

func New(opts Options) (*Server, error) {
	if opts.Config == nil {
		return nil, errors.New("config required")
	}
	log := opts.Logger
	if log == nil {
		log = slog.Default()
	}
	m := metrics.New()
	bc := streaming.NewBroadcaster(m, opts.Config.Encoding.DropOldFrames)

	lifetime := opts.Lifetime
	if lifetime == nil {
		lifetime = context.Background()
	}

	token := opts.AuthToken
	if token == "" {
		token = opts.Config.ResolveToken()
	}

	s := &Server{
		cfg:      opts.Config,
		auth:     security.NewAuthenticator(token),
		log:      log,
		metrics:  m,
		bc:       bc,
		capturer: opts.Capturer,
		capOpts:  opts.CapOpts,
		lifetime: lifetime,
		encCfg: encoder.Config{
			Type:        strings.ToLower(opts.Config.Encoding.Type),
			JPEGQuality: opts.Config.Encoding.JPEGQuality,
		},
	}
	return s, nil
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/status", s.withAuth(s.handleStatus))
	mux.HandleFunc("/api/displays", s.withAuth(s.handleDisplays))
	mux.HandleFunc("/api/start", s.withAuth(s.handleStart))
	mux.HandleFunc("/api/stop", s.withAuth(s.handleStop))
	mux.HandleFunc("/api/metrics", s.withAuth(s.handleNotImplemented))
	mux.HandleFunc("/ws", s.withAuth(s.handleNotImplemented))
	mux.Handle("/stream", s.withAuthHandler(streaming.MJPEGHandler(s.bc, s.metrics)))

	mux.Handle("/", http.FileServer(http.FS(web.Assets)))

	return s.loggingMiddleware(s.clientLimitMiddleware(mux))
}

func (s *Server) ListenAndServe() error {
	s.httpSrv = &http.Server{
		Addr:         s.cfg.ListenAddr(),
		Handler:      s.Handler(),
		ReadTimeout:  s.cfg.ReadTimeout(),
		WriteTimeout: 0, // streaming endpoints manage their own deadlines
	}
	if s.cfg.Security.TLSEnabled {
		return s.httpSrv.ListenAndServeTLS(s.cfg.Security.TLSCertFile, s.cfg.Security.TLSKeyFile)
	}
	return s.httpSrv.ListenAndServe()
}

func (s *Server) Shutdown(ctx context.Context) error {
	_ = s.bc.Stop()
	if s.httpSrv != nil {
		return s.httpSrv.Shutdown(ctx)
	}
	return nil
}

func (s *Server) handleNotImplemented(w http.ResponseWriter, _ *http.Request) {
	http.Error(w, "not implemented", http.StatusNotImplemented)
}

func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	wid, ht := s.bc.Dimensions()
	if wid == 0 {
		wid = s.capOpts.Width
		ht = s.capOpts.Height
	}
	snap := s.metrics.Snapshot(s.bc.Running(), wid, ht, s.bc.LastError())
	_ = json.NewEncoder(w).Encode(snap)
}

func (s *Server) handleDisplays(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if s.capturer == nil {
		http.Error(w, "capturer unavailable", http.StatusServiceUnavailable)
		return
	}
	list, err := s.capturer.ListDisplays(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(list)
}

func (s *Server) handleStart(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if s.capturer == nil {
		http.Error(w, "capturer unavailable", http.StatusServiceUnavailable)
		return
	}
	if s.bc.Running() {
		w.WriteHeader(http.StatusOK)
		return
	}
	enc, err := encoder.New(s.encCfg)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if err := s.bc.Start(s.lifetime, s.capturer, enc); err != nil {
		if errors.Is(err, streaming.ErrAlreadyRunning) {
			w.WriteHeader(http.StatusOK)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	s.log.Info("capture started")
	w.WriteHeader(http.StatusOK)
}

func (s *Server) handleStop(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if err := s.bc.Stop(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	s.log.Info("capture stopped")
	w.WriteHeader(http.StatusOK)
}

func (s *Server) withAuth(h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !s.auth.Authorize(r) {
			w.Header().Set("WWW-Authenticate", `Bearer realm="screenshare"`)
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		h(w, r)
	}
}

func (s *Server) withAuthHandler(h http.Handler) http.Handler {
	return s.auth.Middleware(h)
}

func (s *Server) loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		s.log.Info("request", "method", r.Method, "path", r.URL.Path, "remote", r.RemoteAddr)
		next.ServeHTTP(w, r)
		s.log.Info("completed", "path", r.URL.Path, "duration_ms", time.Since(start).Milliseconds())
	})
}

func (s *Server) clientLimitMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/stream" {
			next.ServeHTTP(w, r)
			return
		}
		if !s.auth.Authorize(r) {
			w.Header().Set("WWW-Authenticate", `Bearer realm="screenshare"`)
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		s.mu.Lock()
		if s.streamClients >= s.cfg.Server.MaxClients {
			s.mu.Unlock()
			http.Error(w, "too many clients", http.StatusTooManyRequests)
			return
		}
		s.streamClients++
		s.metrics.SetClients(int64(s.streamClients))
		s.mu.Unlock()

		defer func() {
			s.mu.Lock()
			s.streamClients--
			s.metrics.SetClients(int64(s.streamClients))
			s.mu.Unlock()
			s.log.Info("stream client disconnected", "remote", r.RemoteAddr)
		}()
		s.log.Info("stream client connected", "remote", r.RemoteAddr)
		next.ServeHTTP(w, r)
	})
}
