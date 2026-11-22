package notifications

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"webapp/internal/email"
	"webapp/internal/gateway"
	"webapp/internal/identity"
)

type fakePrefs struct {
	list    []email.Preference
	err     error
	setFor  string
	setWhat map[string]bool
}

func (f *fakePrefs) Preferences(_ context.Context, _ string) ([]email.Preference, error) {
	return f.list, f.err
}

func (f *fakePrefs) SetPreferences(_ context.Context, address string, choices map[string]bool) error {
	f.setFor, f.setWhat = address, choices
	return f.err
}

func notificationsHandler(f *fakePrefs) http.Handler {
	n := &Notifications{Prefs: f, Labels: map[string]Label{
		"weekly-digest": {Name: "Weekly digest", Reason: "A summary of what changed this week."},
		"offers":        {Name: "Offers", Reason: "News about offers, only if you agree."},
	}}
	mux := http.NewServeMux()
	n.Routes(mux)
	return mux
}

func callAs(h http.Handler, who identity.Identity, method, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, "/api/notifications", strings.NewReader(body))
	if who.Sub != "" {
		req = req.WithContext(identity.With(req.Context(), who))
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

var ana = identity.Identity{Sub: "u-ana", Email: "ana@example.org"}

func TestNotificationsListCarriesReasonsAndDefaults(t *testing.T) {
	f := &fakePrefs{list: []email.Preference{
		{TemplateID: "offers", Basis: "consent", Subscribed: false},
		{TemplateID: "weekly-digest", Basis: "transactional", Subscribed: true},
	}}
	rec := callAs(notificationsHandler(f), ana, http.MethodGet, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("GET = %d %s", rec.Code, rec.Body)
	}
	var out struct {
		Notifications []Notification `json:"notifications"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil || len(out.Notifications) != 2 {
		t.Fatalf("body = %s (%v)", rec.Body, err)
	}
	offers, digest := out.Notifications[0], out.Notifications[1]
	if !offers.Consent || offers.Subscribed || offers.Reason == "" {
		t.Errorf("consent mail must show as off by default, with a reason: %+v", offers)
	}
	if digest.Consent || !digest.Subscribed || digest.Reason == "" {
		t.Errorf("optional mail must show as on by default, with a reason: %+v", digest)
	}
}

func TestNotificationsListNeverHidesATemplateWithoutALabel(t *testing.T) {
	f := &fakePrefs{list: []email.Preference{{TemplateID: "mystery", Basis: "transactional", Subscribed: true}}}
	rec := callAs(notificationsHandler(f), ana, http.MethodGet, "")
	var out struct {
		Notifications []Notification `json:"notifications"`
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &out)
	if len(out.Notifications) != 1 || out.Notifications[0].Reason == "" {
		t.Errorf("an unlabelled template is listed with a default reason, got %s", rec.Body)
	}
}

func TestNotificationsSaveUsesTheVerifiedAddress(t *testing.T) {
	f := &fakePrefs{}
	rec := callAs(notificationsHandler(f), ana, http.MethodPut, `{"choices":{"offers":true,"weekly-digest":false}}`)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("PUT = %d %s", rec.Code, rec.Body)
	}
	if f.setFor != "ana@example.org" || !f.setWhat["offers"] || f.setWhat["weekly-digest"] {
		t.Errorf("saved %q %v", f.setFor, f.setWhat)
	}
	// An address in the body is not a field: a person cannot name someone else's.
	rec = callAs(notificationsHandler(f), ana, http.MethodPut, `{"address":"bo@example.org","choices":{"offers":true}}`)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("a body with an address = %d, want 400", rec.Code)
	}
}

func TestNotificationsNeedASignedInPerson(t *testing.T) {
	h := notificationsHandler(&fakePrefs{})
	for _, method := range []string{http.MethodGet, http.MethodPut} {
		if rec := callAs(h, identity.Identity{}, method, `{"choices":{"offers":true}}`); rec.Code != http.StatusUnauthorized {
			t.Errorf("%s with no identity = %d, want 401", method, rec.Code)
		}
	}
	if rec := callAs(h, identity.Identity{Sub: "u-x"}, http.MethodGet, ""); rec.Code != http.StatusUnauthorized {
		t.Errorf("an identity with no email = %d, want 401", rec.Code)
	}
}

func TestNotificationsFailuresAreOpaque(t *testing.T) {
	notAChoice := &fakePrefs{err: email.ErrNotChoosable}
	if rec := callAs(notificationsHandler(notAChoice), ana, http.MethodPut, `{"choices":{"x":true}}`); rec.Code != http.StatusUnprocessableEntity {
		t.Errorf("not a choice = %d, want 422", rec.Code)
	}
	paused := &fakePrefs{err: &gateway.Error{Status: http.StatusForbidden, Code: "tenant_paused"}}
	rec := callAs(notificationsHandler(paused), ana, http.MethodGet, "")
	if rec.Code != http.StatusServiceUnavailable || strings.Contains(rec.Body.String(), "tenant_paused") {
		t.Errorf("a paused tenant = %d %s, want 503 with no code", rec.Code, rec.Body)
	}
	broken := &fakePrefs{err: errors.New("boom")}
	if rec := callAs(notificationsHandler(broken), ana, http.MethodGet, ""); rec.Code != http.StatusInternalServerError || strings.Contains(rec.Body.String(), "boom") {
		t.Errorf("a fault = %d %s, want an opaque 500", rec.Code, rec.Body)
	}
}
