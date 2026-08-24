package main

import (
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestLivenessEndpoint(t *testing.T) {
	t.Parallel()

	app := &application{logger: slog.Default()}
	request := httptest.NewRequest(http.MethodGet, "/health/live", nil)
	response := httptest.NewRecorder()

	app.routes().ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, response.Code)
	}
	if contentType := response.Header().Get("Content-Type"); contentType != "application/json" {
		t.Fatalf("expected JSON response, got %q", contentType)
	}
}
