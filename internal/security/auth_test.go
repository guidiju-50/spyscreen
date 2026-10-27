package security

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAuthorizeBearer(t *testing.T) {
	a := NewAuthenticator("secret")
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer secret")
	if !a.Authorize(req) {
		t.Fatal("expected authorized")
	}
	req.Header.Set("Authorization", "Bearer wrong")
	if a.Authorize(req) {
		t.Fatal("expected unauthorized")
	}
}

func TestAuthorizeCustomHeader(t *testing.T) {
	a := NewAuthenticator("secret")
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Screenshare-Token", "secret")
	if !a.Authorize(req) {
		t.Fatal("expected authorized")
	}
}

func TestMiddleware(t *testing.T) {
	a := NewAuthenticator("secret")
	h := a.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("code: %d", rec.Code)
	}
}
