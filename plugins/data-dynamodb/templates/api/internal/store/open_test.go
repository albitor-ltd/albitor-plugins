package store

import (
	"strings"
	"testing"
)

func env(kv ...string) func(string) string {
	m := map[string]string{}
	for i := 0; i < len(kv); i += 2 {
		m[kv[i]] = kv[i+1]
	}
	return func(k string) string { return m[k] }
}

var aws = []string{"AWS_REGION", "eu-west-2", "AWS_ACCESS_KEY_ID", "AKID", "AWS_SECRET_ACCESS_KEY", "s", "AWS_SESSION_TOKEN", "t"}

func TestTableEnv(t *testing.T) {
	if got := TableEnv("data"); got != "ALBITOR_TABLE_DATA" {
		t.Errorf("TableEnv(data) = %s", got)
	}
	if got := TableEnv("order-lines"); got != "ALBITOR_TABLE_ORDER_LINES" {
		t.Errorf("TableEnv(order-lines) = %s", got)
	}
}

func TestOpenLocalRunUsesMemory(t *testing.T) {
	s, err := Open(env("ALBITOR_LOCAL", "1"), DataTable)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := s.(*Memory); !ok {
		t.Fatalf("local run opened %T, want *Memory", s)
	}
}

func TestOpenDeployedUsesDynamoDB(t *testing.T) {
	s, err := Open(env(append(aws, "ALBITOR_TABLE_DATA", "hosted-x--data", "AWS_LAMBDA_FUNCTION_NAME", "fn")...), DataTable)
	if err != nil {
		t.Fatal(err)
	}
	d, ok := s.(*DynamoDB)
	if !ok {
		t.Fatalf("deployed run opened %T, want *DynamoDB", s)
	}
	if d.table != "hosted-x--data" || d.endpoint != "https://dynamodb.eu-west-2.amazonaws.com" {
		t.Errorf("table %q endpoint %q", d.table, d.endpoint)
	}
}

// Missing AWS configuration must never select the fake: a deployed app serving
// from memory loses its data on the next cold start and reports nothing.
func TestOpenRefusesToFallBackToMemory(t *testing.T) {
	cases := map[string][]string{
		"no table and no local flag":    aws,
		"local flag on a function":      {"ALBITOR_LOCAL", "1", "AWS_LAMBDA_FUNCTION_NAME", "fn"},
		"local flag beside a table":     {"ALBITOR_LOCAL", "1", "ALBITOR_TABLE_DATA", "t-data"},
		"local flag spelled loosely":    {"ALBITOR_LOCAL", "true"},
		"local flag with a blank table": {"ALBITOR_LOCAL", "0", "ALBITOR_TABLE_DATA", " "},
	}
	for name, kv := range cases {
		s, err := Open(env(kv...), DataTable)
		if err == nil {
			t.Errorf("%s: opened %T, want an error", name, s)
		}
		if _, isMem := s.(*Memory); isMem {
			t.Errorf("%s: fell back to memory", name)
		}
	}
	// A table but no credentials or region.
	if _, err := Open(env("ALBITOR_TABLE_DATA", "app-data"), DataTable); err == nil {
		t.Error("opened without region or credentials")
	}
}

func TestOpenEndpointOverrideIsLoopbackOnly(t *testing.T) {
	base := append(append([]string{}, aws...), "ALBITOR_TABLE_DATA", "app-data")
	if _, err := Open(env(append(base, EndpointEnv, "http://127.0.0.1:8000")...), DataTable); err != nil {
		t.Errorf("loopback override refused: %v", err)
	}
	_, err := Open(env(append(base, EndpointEnv, "https://attacker.example")...), DataTable)
	if err == nil {
		t.Fatal("a remote endpoint override was accepted")
	}
	if strings.Contains(err.Error(), "attacker") {
		t.Errorf("the error echoes the rejected endpoint: %v", err)
	}
}

func TestOpenIgnoresEndpointOverrideOnALambda(t *testing.T) {
	base := append(append([]string{}, aws...), "ALBITOR_TABLE_DATA", "app-data", "AWS_LAMBDA_FUNCTION_NAME", "fn")
	s, err := Open(env(append(base, EndpointEnv, "http://127.0.0.1:8000")...), DataTable)
	if err != nil {
		t.Fatal(err)
	}
	if d := s.(*DynamoDB); d.endpoint != "https://dynamodb.eu-west-2.amazonaws.com" {
		t.Errorf("a deployed function honoured the override: %s", d.endpoint)
	}
	// Even a hostile value is ignored there, not rejected: nothing in the environment can steer it.
	if _, err := Open(env(append(base, EndpointEnv, "https://attacker.example")...), DataTable); err != nil {
		t.Errorf("a deployed function failed on an ignored override: %v", err)
	}
}

func TestOpenRefusesABadLogicalName(t *testing.T) {
	for _, name := range []string{"", "Data", "a", "x_y", "../x", "data\n", strings.Repeat("a", 40)} {
		if _, err := Open(env("ALBITOR_LOCAL", "1"), name); err == nil {
			t.Errorf("Open(%q) accepted", name)
		}
	}
}
