package widgets

import "net/http"

// Register mounts this feature's routes (Go 1.22 method-and-path patterns). It is the one
// symbol internal/app/registry.go uses. A feature that needs a store opens it here, once, and
// keeps it in a variable or a struct its handlers share.
func Register(r *http.ServeMux) {
	r.HandleFunc("GET /api/widgets", listWidgets)
	r.HandleFunc("POST /api/widgets", createWidget)
}
