package app

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"webapp/internal/apikit"
	"webapp/internal/email"
	"webapp/internal/identity"
)

// Notifications is the Notifications page's API: what mail the signed-in person may choose
// about, and their choices. The gateway keeps the choices; this keeps nothing.
//
//	GET /api/notifications   the optional and consent mail, each with its reason and the choice now
//	PUT /api/notifications   {"choices": {"<template id>": true|false}}; 204
//
// The address is the verified token's email, never the request's: a person can read and change
// only their own. A PUT is a person's action, so a consent opt-in sent here is recorded as one
// (email.SetPreferences); never call it from anything the person did not do.
type Notifications struct {
	Prefs PreferenceStore
	// Labels name each optional or consent template in the person's words and give the reason
	// they are sent. The build writes them from the Emails step's answers (the touchpoint's name
	// and reason): the gateway does not return them. A template with none is listed under its
	// id with a plain default, and logged, so a missing label is visible, never a hidden toggle.
	Labels map[string]Label
	Log    *slog.Logger
}

// Label is how the page describes one template.
type Label struct {
	Name   string `json:"name"`
	Reason string `json:"reason"`
}

// PreferenceStore is the part of *email.Sender the handlers use, so tests fake it.
type PreferenceStore interface {
	Preferences(ctx context.Context, address string) ([]email.Preference, error)
	SetPreferences(ctx context.Context, address string, choices map[string]bool) error
}

// Notification is one row of the page.
type Notification struct {
	TemplateID string `json:"template_id"`
	Name       string `json:"name"`
	Reason     string `json:"reason"`
	// Consent is true for mail that is sent only if the person agrees; the page says so, and the
	// sign-up form offers the same choice.
	Consent    bool `json:"consent"`
	Subscribed bool `json:"subscribed"`
}

// Routes registers the routes. Mount it behind the app's sign-in.
func (n *Notifications) Routes(r *http.ServeMux) {
	r.HandleFunc("GET /api/notifications", n.list)
	r.HandleFunc("PUT /api/notifications", n.save)
}

func (n *Notifications) log() *slog.Logger {
	if n.Log != nil {
		return n.Log
	}
	return slog.Default()
}

func (n *Notifications) address(w http.ResponseWriter, r *http.Request) (string, bool) {
	id, ok := identity.FromContext(r.Context())
	if !ok || strings.TrimSpace(id.Email) == "" {
		apikit.WriteError(w, http.StatusUnauthorized, "sign-in required")
		return "", false
	}
	return id.Email, true
}

func (n *Notifications) list(w http.ResponseWriter, r *http.Request) {
	addr, ok := n.address(w, r)
	if !ok {
		return
	}
	prefs, err := n.Prefs.Preferences(r.Context(), addr)
	if err != nil {
		n.fail(w, r, "list", err)
		return
	}
	out := make([]Notification, 0, len(prefs))
	for _, p := range prefs {
		l, known := n.Labels[p.TemplateID]
		if !known {
			n.log().WarnContext(r.Context(), "notifications: no label for template", "template", p.TemplateID,
				"correlation_id", apikit.CorrelationIDFromContext(r.Context()))
			l = Label{Name: p.TemplateID, Reason: "We send this to keep you informed."}
		}
		out = append(out, Notification{TemplateID: p.TemplateID, Name: l.Name, Reason: l.Reason,
			Consent: p.Basis == "consent", Subscribed: p.Subscribed})
	}
	apikit.WriteJSON(w, http.StatusOK, map[string]any{"notifications": out})
}

func (n *Notifications) save(w http.ResponseWriter, r *http.Request) {
	addr, ok := n.address(w, r)
	if !ok {
		return
	}
	var in struct {
		Choices map[string]bool `json:"choices"`
	}
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<14))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&in); err != nil || len(in.Choices) == 0 {
		apikit.WriteError(w, http.StatusBadRequest, `send {"choices": {"<template id>": true|false}}`)
		return
	}
	if err := n.Prefs.SetPreferences(r.Context(), addr, in.Choices); err != nil {
		if errors.Is(err, email.ErrNotChoosable) {
			apikit.WriteError(w, http.StatusUnprocessableEntity, "one of those emails is not a choice")
			return
		}
		n.fail(w, r, "save", err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (n *Notifications) fail(w http.ResponseWriter, r *http.Request, what string, err error) {
	n.log().ErrorContext(r.Context(), "notifications: "+what, "correlation_id", apikit.CorrelationIDFromContext(r.Context()), "err", err)
	if email.Classify(err) == email.OutcomeSkip {
		apikit.WriteError(w, http.StatusServiceUnavailable, "email settings are unavailable right now")
		return
	}
	apikit.WriteError(w, http.StatusInternalServerError, "something went wrong")
}
