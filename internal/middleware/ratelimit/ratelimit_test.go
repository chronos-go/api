package ratelimit

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestLimiter_Returns429AndResetsWindow(t *testing.T) {
	limiter, err := New(2, time.Minute, false)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 6, 22, 12, 0, 0, 0, time.UTC)
	limiter.nowFn = func() time.Time { return now }
	handler := limiter.Middleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) }))

	for i, expected := range []int{http.StatusNoContent, http.StatusNoContent, http.StatusTooManyRequests} {
		req := httptest.NewRequest(http.MethodPost, "/login", nil)
		req.RemoteAddr = "192.0.2.1:1234"
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != expected {
			t.Fatalf("request %d: expected %d, got %d", i+1, expected, rec.Code)
		}
	}

	now = now.Add(time.Minute)
	req := httptest.NewRequest(http.MethodPost, "/login", nil)
	req.RemoteAddr = "192.0.2.1:1234"
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected reset window to allow request, got %d", rec.Code)
	}
}

func TestLimiter_SeparatesClientIPs(t *testing.T) {
	limiter, _ := New(1, time.Minute, false)
	handler := limiter.Middleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) }))
	for _, address := range []string{"192.0.2.1:1", "192.0.2.2:1"} {
		req := httptest.NewRequest(http.MethodPost, "/login", nil)
		req.RemoteAddr = address
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusNoContent {
			t.Fatalf("expected independent IP allowance, got %d", rec.Code)
		}
	}
}
