// Package widgets is an example feature: one Go package per feature, under internal/<feature>.
// Copy it to api/internal/widgets and rename "widgets" to the app's own noun. It owns its
// handlers, its routes and its tests, and imports only shared packages (apikit, identity, a
// store, outbound), never another feature and never app.
package widgets

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"webapp/internal/apikit"
)

// Widget is the record this feature serves.
type Widget struct {
	Name string `json:"name"`
}

func listWidgets(w http.ResponseWriter, r *http.Request) {
	apikit.ServeList(w, r, slog.Default(), func(q apikit.PageQuery) (apikit.Page[Widget], error) {
		// Read one page using q.Cursor and q.Limit; mint next_cursor with apikit.EncodeCursor.
		return apikit.Page[Widget]{Items: make([]Widget, 0)}, nil
	})
}

func createWidget(w http.ResponseWriter, r *http.Request) {
	var in Widget
	body := http.MaxBytesReader(w, r.Body, 1<<20)
	if err := json.NewDecoder(body).Decode(&in); err != nil || in.Name == "" {
		apikit.WriteError(w, http.StatusBadRequest, "name is required")
		return
	}
	// Take the owner from the verified identity, never the body; write through the store
	// (with data-dynamodb, store.Owned(s, r) scopes every call to that owner).
	apikit.WriteJSON(w, http.StatusCreated, in)
}
