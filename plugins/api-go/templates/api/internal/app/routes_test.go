package app

import (
	"net/http"
	"testing"

	"webapp/internal/apptest"
)

// TestRoutesMountWithoutConflict fails when two features register the same pattern: the mux
// panics on a duplicate, so mounting every registrar once is the check.
func TestRoutesMountWithoutConflict(t *testing.T) {
	if rec := apptest.Serve(t, Routes, http.MethodGet, "/api/not-a-route", ""); rec.Code != http.StatusNotFound {
		t.Fatalf("unregistered path: status %d, want 404", rec.Code)
	}
}
