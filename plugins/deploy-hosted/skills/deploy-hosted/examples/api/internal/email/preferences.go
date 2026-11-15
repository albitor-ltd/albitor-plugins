package email

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"sort"

	"webapp/internal/gateway"
)

// Preference is one mail a person may choose about: an optional or consent template of the app.
// The gateway keeps the choice; the app holds none.
type Preference struct {
	TemplateID string `json:"template_id"`
	// Basis is "transactional" (optional mail the person can switch off) or "consent" (sent
	// only after they opt in).
	Basis string `json:"basis"`
	// Subscribed is the choice now: optional mail is on until the person turns it off, consent
	// mail is off until they turn it on.
	Subscribed bool `json:"subscribed"`
}

// ErrNotChoosable is a template that is not optional or consent, or is not the app's.
var ErrNotChoosable = errors.New("email: that mail is not one a person can choose about")

// Preferences lists the app's optional and consent templates with the person's choice on each.
// A transactional, non-optional template is not listed: it is not a choice.
func (s *Sender) Preferences(ctx context.Context, address string) ([]Preference, error) {
	if err := checkAddress(address); err != nil {
		return nil, err
	}
	var resp struct {
		Templates []struct {
			TemplateID  string `json:"TemplateId"`
			LawfulBasis string `json:"LawfulBasis"`
			Subscribed  bool   `json:"Subscribed"`
		} `json:"Templates"`
	}
	if err := s.gw.Call(ctx, Service, "GetEmailPreferences", map[string]string{"Address": address}, &resp); err != nil {
		return nil, err
	}
	out := make([]Preference, 0, len(resp.Templates))
	for _, t := range resp.Templates {
		out = append(out, Preference{TemplateID: t.TemplateID, Basis: t.LawfulBasis, Subscribed: t.Subscribed})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].TemplateID < out[j].TemplateID })
	return out, nil
}

// SetPreferences records choices by template id. Switching a consent template on states that
// the person did it themselves (ConsentSource user_action), so call SetPreferences only from a
// request the person made: a toggle they pressed or a sign-up box they ticked, never from a
// default, an import or a script. Turning anything off needs no assertion.
func (s *Sender) SetPreferences(ctx context.Context, address string, choices map[string]bool) error {
	if err := checkAddress(address); err != nil {
		return err
	}
	if len(choices) == 0 || len(choices) > 12 {
		return errors.New("email: choose between one and twelve templates")
	}
	for id := range choices {
		if !templatePattern.MatchString(id) {
			return fmt.Errorf("email: %q is not a template id", id)
		}
	}
	req := struct {
		Address       string          `json:"Address"`
		Templates     map[string]bool `json:"Templates"`
		ConsentSource string          `json:"ConsentSource"`
	}{address, choices, "user_action"}
	if err := s.gw.Call(ctx, Service, "SetEmailPreferences", req, nil); err != nil {
		var ge *gateway.Error
		if errors.As(err, &ge) && ge.Status == http.StatusBadRequest && (ge.Code == "not_optional" || ge.Code == "consent_required") {
			return fmt.Errorf("%w (%s)", ErrNotChoosable, ge.Code)
		}
		return err
	}
	return nil
}
