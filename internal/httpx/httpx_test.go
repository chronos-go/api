package httpx

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestDecodeJSON_RejectsUnknownAndMultipleValues(t *testing.T) {
	type payload struct {
		Name string `json:"name"`
	}
	for _, body := range []string{`{"name":"ok","unknown":true}`, `{"name":"one"}{"name":"two"}`} {
		req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
		var dst payload
		if err := DecodeJSON(req, &dst); err == nil {
			t.Fatalf("expected invalid body %q to fail", body)
		}
	}
}

func TestLimitBodyAndSecurityHeaders(t *testing.T) {
	handler := SecurityHeaders(LimitBody(4)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var dst map[string]any
		if err := DecodeJSON(r, &dst); err != nil {
			http.Error(w, "invalid", http.StatusBadRequest)
			return
		}
	})))
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"long":true}`))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected oversized body rejection, got %d", rec.Code)
	}
	if rec.Header().Get("X-Content-Type-Options") != "nosniff" {
		t.Fatal("missing nosniff header")
	}
}
