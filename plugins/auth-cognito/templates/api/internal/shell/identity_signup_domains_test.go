package shell

import (
	"strings"
	"testing"
)

func TestParseSignupDomains(t *testing.T) {
	got, err := parseSignupDomains("Example.org, *.dept.example.org\n@other.co.uk")
	if err != nil || strings.Join(got, "|") != "example.org|*.dept.example.org|other.co.uk" {
		t.Fatalf("parse = %v, %v", got, err)
	}
	if got, err := parseSignupDomains("  "); err != nil || got != nil {
		t.Fatalf("empty = %v, %v; want no rule", got, err)
	}
	for _, bad := range []string{"example", "*example.org", "a@b.org", "ex ample..org", "*.org."} {
		if _, err := parseSignupDomains(bad); err == nil {
			t.Errorf("%q should be refused so a typo closes the API", bad)
		}
	}
}

func TestSignInFromEnvRefusesAMalformedSignupDomain(t *testing.T) {
	env := map[string]string{
		"AUTH_USER_POOL_ID": "eu-west-2_abc", "AUTH_CLIENT_ID": "client", "AWS_REGION": "eu-west-2",
		"AUTH_SIGNUP_DOMAINS": "not a domain",
	}
	if s := signInFromEnv(identityTestEnv(env)); s.err == nil || s.auth != nil {
		t.Fatalf("a malformed AUTH_SIGNUP_DOMAINS = %+v, want the API closed", s)
	}
	env["AUTH_SIGNUP_DOMAINS"] = "example.org"
	if s := signInFromEnv(identityTestEnv(env)); s.err != nil || s.auth == nil || len(s.auth.signupDomains) != 1 {
		t.Fatalf("a valid AUTH_SIGNUP_DOMAINS = %+v, want it kept", s)
	}
}

func TestSignupAllowed(t *testing.T) {
	cases := []struct {
		domains []string
		email   string
		want    bool
	}{
		{nil, "anyone@anywhere.test", true},
		{[]string{"example.org"}, "a@example.org", true},
		{[]string{"example.org"}, "a@EXAMPLE.org", true},
		{[]string{"example.org"}, "a@sub.example.org", false},
		{[]string{"example.org"}, "a@notexample.org", false},
		{[]string{"*.example.org"}, "a@sub.example.org", true},
		{[]string{"*.example.org"}, "a@example.org", false},
		{[]string{"*.example.org"}, "a@evilexample.org", false},
		{[]string{"example.org", "other.org"}, "a@other.org", true},
		{[]string{"example.org"}, "no-at-sign", false},
		{[]string{"example.org"}, "@example.org", false},
	}
	for _, c := range cases {
		a := &cognitoAuth{signupDomains: c.domains}
		if got := a.signupAllowed(c.email); got != c.want {
			t.Errorf("signupAllowed(%v, %q) = %v, want %v", c.domains, c.email, got, c.want)
		}
	}
}
