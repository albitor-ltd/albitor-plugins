package email

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"regexp"
	"strings"
	"testing"

	"webapp/internal/gateway"
)

// fake records the call and answers as set.
type fake struct {
	service, operation string
	body               map[string]any
	answer             string
	err                error
	calls              int
}

func (f *fake) Call(_ context.Context, service, operation string, req, resp any, _ ...gateway.CallOption) error {
	f.calls++
	f.service, f.operation = service, operation
	raw, _ := json.Marshal(req)
	_ = json.Unmarshal(raw, &f.body)
	if f.err != nil {
		return f.err
	}
	if resp != nil && f.answer != "" {
		return json.Unmarshal([]byte(f.answer), resp)
	}
	return nil
}

func newSender(t *testing.T, f *fake) *Sender {
	t.Helper()
	s, err := New(f, []string{"order-receipt", "weekly-digest"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func TestSendShapesTheGatewayRequest(t *testing.T) {
	f := &fake{answer: `{"MessageId":"m1","Disposition":"sent"}`}
	res, err := newSender(t, f).Send(context.Background(), "order-receipt", "person@example.org", Vars{
		"order": Number("1042"),
		"name":  Text("Ana"),
		"track": Link("https://shop.example.org", "/orders/1042"),
	}, ForEvent("order-1042-receipt"))
	if err != nil {
		t.Fatal(err)
	}
	if res.Disposition != Sent || res.MessageID != "m1" {
		t.Errorf("result = %+v", res)
	}
	if f.service != "email" || f.operation != "SendTemplatedEmail" {
		t.Errorf("called %s/%s", f.service, f.operation)
	}
	vars, _ := f.body["Variables"].(map[string]any)
	if f.body["TemplateId"] != "order-receipt" || f.body["To"] != "person@example.org" ||
		f.body["IdempotencyKey"] != "order-1042-receipt" || vars["track"] != "https://shop.example.org/orders/1042" {
		t.Errorf("body = %v", f.body)
	}
	// Nothing the gateway refuses as an unknown field: no subject, body, sender or link host.
	for k := range f.body {
		if k != "TemplateId" && k != "To" && k != "Variables" && k != "IdempotencyKey" {
			t.Errorf("unexpected field %q", k)
		}
	}
}

func TestSendWithoutAnEventUsesAFreshKeyEachTime(t *testing.T) {
	f := &fake{answer: `{"Disposition":"sent"}`}
	s := newSender(t, f)
	keys := map[string]bool{}
	for range 3 {
		if _, err := s.Send(context.Background(), "weekly-digest", "a@example.org", Vars{"week": Text("46")}); err != nil {
			t.Fatal(err)
		}
		k, _ := f.body["IdempotencyKey"].(string)
		if !regexp.MustCompile(`^[A-Za-z0-9._:-]{8,64}$`).MatchString(k) {
			t.Fatalf("key %q is not one the gateway accepts", k)
		}
		keys[k] = true
	}
	if len(keys) != 3 {
		t.Errorf("keys = %v, want a fresh one per send", keys)
	}
}

func TestSendRefusesWhatTheGatewayWouldRefuseBeforeCalling(t *testing.T) {
	long := strings.Repeat("x", MaxVarRunes+1)
	cases := map[string]struct {
		template, to string
		vars         Vars
	}{
		"template not in the manifest": {"invoice", "a@example.org", Vars{"n": Text("1")}},
		"bad template id":              {"Order Receipt", "a@example.org", nil},
		"display name in the address":  {"order-receipt", "Ana <a@example.org>", nil},
		"two addresses":                {"order-receipt", "a@example.org,b@example.org", nil},
		"text over the limit":          {"order-receipt", "a@example.org", Vars{"n": Text(long)}},
		"text with a link":             {"order-receipt", "a@example.org", Vars{"n": Text("see https://evil.test")}},
		"text over two lines":          {"order-receipt", "a@example.org", Vars{"n": Text("a\nb")}},
		"number that is not one":       {"order-receipt", "a@example.org", Vars{"n": Number("12abc")}},
		"negative number":              {"order-receipt", "a@example.org", Vars{"n": Number("-1")}},
		"link without https":           {"order-receipt", "a@example.org", Vars{"n": Link("http://shop.example.org", "/x")}},
		"link with a userinfo host":    {"order-receipt", "a@example.org", Vars{"n": Link("https://shop.example.org@evil.test", "/x")}},
		"link path that escapes":       {"order-receipt", "a@example.org", Vars{"n": Link("https://shop.example.org", "//evil.test")}},
		"unset value":                  {"order-receipt", "a@example.org", Vars{"n": {}}},
		"bad value name":               {"order-receipt", "a@example.org", Vars{"Order": Text("1")}},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			f := &fake{answer: `{"Disposition":"sent"}`}
			if _, err := newSender(t, f).Send(context.Background(), c.template, c.to, c.vars); err == nil {
				t.Error("a send the gateway would refuse was accepted")
			}
			if f.calls != 0 {
				t.Errorf("the gateway was called %d times", f.calls)
			}
		})
	}
	t.Run("too many values", func(t *testing.T) {
		vars := Vars{}
		for i := range MaxVars + 1 {
			vars["v"+string(rune('a'+i))] = Text("x")
		}
		if _, err := newSender(t, &fake{}).Send(context.Background(), "order-receipt", "a@example.org", vars); err == nil {
			t.Error("more than twelve values were accepted")
		}
	})
}

func TestSuppressedIsAnAnswerNotAFailure(t *testing.T) {
	// A recipient who opted out, a consent mail with no opt-in and an unconfirmed template all
	// answer suppressed. The caller logs and carries on: one call, no loop, no error.
	f := &fake{answer: `{"Disposition":"suppressed"}`}
	res, err := newSender(t, f).Send(context.Background(), "weekly-digest", "a@example.org", Vars{"week": Text("46")})
	if err != nil {
		t.Fatalf("suppressed is not an error: %v", err)
	}
	if res.Disposition != Suppressed || res.MessageID != "" {
		t.Errorf("result = %+v", res)
	}
	if f.calls != 1 {
		t.Errorf("calls = %d, want 1", f.calls)
	}
}

func TestClassifyHandlesTheRefusalsWithoutRetrying(t *testing.T) {
	skip := []*gateway.Error{
		{Status: http.StatusNotFound, Code: "unknown_template"},
		{Status: http.StatusForbidden, Code: "not_declared"},
		{Status: http.StatusForbidden, Code: "tenant_paused"},
		{Status: http.StatusConflict, Code: "never_email"},
		{Status: http.StatusConflict, Code: "invite_window"},
		{Status: http.StatusTooManyRequests, Code: "daily_cap"},
		gateway.ErrCreditsExhausted,
	}
	for _, e := range skip {
		if got := Classify(e); got != OutcomeSkip {
			t.Errorf("%d %s = %v, want skip", e.Status, e.Code, got)
		}
	}
	fault := []error{
		&gateway.Error{Status: http.StatusBadRequest, Code: "invalid_variable"},
		&gateway.Error{Status: http.StatusBadRequest, Code: "bad_request"},
		&gateway.Error{Status: http.StatusServiceUnavailable, Code: "unavailable"},
		errors.New("network"),
	}
	for _, e := range fault {
		if got := Classify(e); got != OutcomeFault {
			t.Errorf("%v = %v, want fault", e, got)
		}
	}
	// Whatever the refusal, Send made one call: it has no retry of its own.
	f := &fake{err: &gateway.Error{Status: http.StatusNotFound, Code: "unknown_template"}}
	if _, err := newSender(t, f).Send(context.Background(), "weekly-digest", "a@example.org", Vars{"week": Text("46")}); err == nil {
		t.Fatal("a refusal must be an error")
	}
	if f.calls != 1 {
		t.Errorf("calls = %d, want 1: never retry a refused mail", f.calls)
	}
}

func TestPreferencesListAndSet(t *testing.T) {
	f := &fake{answer: `{"Templates":[
		{"TemplateId":"weekly-digest","LawfulBasis":"transactional","Optional":true,"Subscribed":true},
		{"TemplateId":"offers","LawfulBasis":"consent","Optional":true,"Subscribed":false}]}`}
	s, err := New(f, []string{"weekly-digest", "offers"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	prefs, err := s.Preferences(context.Background(), "a@example.org")
	if err != nil {
		t.Fatal(err)
	}
	if f.operation != "GetEmailPreferences" || len(prefs) != 2 || prefs[0].TemplateID != "offers" || prefs[0].Subscribed || !prefs[1].Subscribed {
		t.Errorf("prefs = %+v (%s)", prefs, f.operation)
	}

	if err := s.SetPreferences(context.Background(), "a@example.org", map[string]bool{"offers": true}); err != nil {
		t.Fatal(err)
	}
	if f.operation != "SetEmailPreferences" || f.body["ConsentSource"] != "user_action" {
		t.Errorf("set = %s %v: an opt-in must be asserted as the person's own act", f.operation, f.body)
	}

	f.err = &gateway.Error{Status: http.StatusBadRequest, Code: "not_optional"}
	if err := s.SetPreferences(context.Background(), "a@example.org", map[string]bool{"order-receipt": true}); !errors.Is(err, ErrNotChoosable) {
		t.Errorf("not_optional = %v, want ErrNotChoosable", err)
	}
	if err := s.SetPreferences(context.Background(), "a@example.org", map[string]bool{}); err == nil {
		t.Error("an empty choice was sent")
	}
}
