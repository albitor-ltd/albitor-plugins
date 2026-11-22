package widgets

import (
	"net/http"
	"strings"
	"testing"

	"webapp/internal/apptest"
)

func TestListWidgetsEmptyIsAnArrayNotNull(t *testing.T) {
	rec := apptest.Serve(t, Register, http.MethodGet, "/api/widgets", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), `"items":[]`) {
		t.Fatalf("body %s: items must be [] and never null", rec.Body.String())
	}
}

func TestCreateWidgetNeedsAName(t *testing.T) {
	if rec := apptest.Serve(t, Register, http.MethodPost, "/api/widgets", `{}`); rec.Code != http.StatusBadRequest {
		t.Fatalf("status %d, want 400", rec.Code)
	}
	rec := apptest.Serve(t, Register, http.MethodPost, "/api/widgets", `{"name":"a"}`)
	if rec.Code != http.StatusCreated {
		t.Fatalf("status %d, want 201", rec.Code)
	}
}
