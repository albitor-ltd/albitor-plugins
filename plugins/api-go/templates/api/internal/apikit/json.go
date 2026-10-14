// Package apikit is what an app's handlers import: the JSON response writers,
// the paging envelope and its cursors, and the correlation id on the request
// context. It imports nothing from the app, so app code and the shell can both
// depend on it.
package apikit

import (
	"encoding/json"
	"net/http"
)

// WriteJSON answers status with v encoded as JSON.
func WriteJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// WriteError answers status with {"error": msg}. Every error body has this
// shape, so a client has one place to read the message.
func WriteError(w http.ResponseWriter, status int, msg string) {
	WriteJSON(w, status, map[string]string{"error": msg})
}
