package security

import (
	"crypto/subtle"
	"net/http"
	"strings"
)

const headerToken = "X-Screenshare-Token"

// Authenticator validates bearer or custom header tokens.
type Authenticator struct {
	token string
}

func NewAuthenticator(token string) *Authenticator {
	return &Authenticator{token: token}
}

func (a *Authenticator) TokenConfigured() bool {
	return a.token != ""
}

func (a *Authenticator) Authorize(r *http.Request) bool {
	if a.token == "" {
		return true
	}
	got := extractToken(r)
	if got == "" {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(got), []byte(a.token)) == 1
}

func extractToken(r *http.Request) string {
	auth := strings.TrimSpace(r.Header.Get("Authorization"))
	if strings.HasPrefix(strings.ToLower(auth), "bearer ") {
		return strings.TrimSpace(auth[7:])
	}
	if h := strings.TrimSpace(r.Header.Get(headerToken)); h != "" {
		return h
	}
	return ""
}

// Middleware rejects unauthenticated requests.
func (a *Authenticator) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !a.Authorize(r) {
			w.Header().Set("WWW-Authenticate", `Bearer realm="screenshare"`)
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}
