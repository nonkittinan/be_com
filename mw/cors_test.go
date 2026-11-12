package mw

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func okHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) })
}

func TestCORSAllowsListedOrigin(t *testing.T) {
	h := CORS([]string{"http://localhost:3000", " https://site.example "}, okHandler())
	for _, origin := range []string{"http://localhost:3000", "https://site.example"} {
		req := httptest.NewRequest(http.MethodGet, "/api/equipment", nil)
		req.Header.Set("Origin", origin)
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		if got := rec.Header().Get("Access-Control-Allow-Origin"); got != origin {
			t.Fatalf("origin %q: Allow-Origin = %q", origin, got)
		}
	}
}

func TestCORSIgnoresUnlistedOrigin(t *testing.T) {
	h := CORS([]string{"http://localhost:3000"}, okHandler())
	req := httptest.NewRequest(http.MethodGet, "/api/equipment", nil)
	req.Header.Set("Origin", "https://evil.example")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Fatalf("origin ที่ไม่อยู่ในรายชื่อต้องไม่ได้ header ได้ %q", got)
	}
}

func TestCORSPreflightReturns204WithoutReachingHandler(t *testing.T) {
	reached := false
	h := CORS([]string{"http://localhost:3000"}, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { reached = true }))
	req := httptest.NewRequest(http.MethodOptions, "/api/admin/equipment", nil)
	req.Header.Set("Origin", "http://localhost:3000")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusNoContent || reached {
		t.Fatalf("preflight: code=%d reached=%v", rec.Code, reached)
	}
}
