package ownedscope

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"testing"
)

// fakeT stands in for *testing.T so a test can run Run and read what it would have failed.
type fakeT struct {
	errs     []string
	cleanups []func()
}

type fatal struct{}

func (f *fakeT) Helper() {}
func (f *fakeT) Errorf(format string, args ...any) {
	f.errs = append(f.errs, fmt.Sprintf(format, args...))
}
func (f *fakeT) Fatalf(format string, args ...any) {
	f.Errorf(format, args...)
	panic(fatal{})
}
func (f *fakeT) Cleanup(fn func()) { f.cleanups = append(f.cleanups, fn) }

// run executes fn as a test body, then the cleanups in reverse, and returns the failures.
func (f *fakeT) run(fn func()) []string {
	func() {
		defer func() {
			if r := recover(); r != nil {
				if _, ok := r.(fatal); !ok {
					panic(r)
				}
			}
		}()
		fn()
	}()
	for i := len(f.cleanups) - 1; i >= 0; i-- {
		f.cleanups[i]()
	}
	return f.errs
}

const contract = `
openapi: 3.1.0
x-data-model:
  entities:
    Army:
      ownership: per-user
      reads: each user reads only the armies they made
    Rule:
      ownership: shared
      reads: everyone reads every rule
paths:
  /api/armies:
    get:
      x-entity: Army
    post:
      x-entity: Army
  /api/armies/{id}:
    get:
      x-entity: Army
    put:
      x-entity: Army
    delete:
      x-entity: Army
  /api/rules:
    get:
      x-entity: Rule
`

func writeContract(t *testing.T, doc string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "openapi.yaml")
	if err := os.WriteFile(p, []byte(doc), 0o600); err != nil {
		t.Fatal(err)
	}
	return p
}

// armies is a tiny in-memory API of armies. With perOwner it keeps one keyspace per caller; without
// it every caller shares one, which is the defect the test exists to catch.
type armies struct {
	mu       sync.Mutex
	perOwner bool
	data     map[string]map[string]json.RawMessage
}

func newArmies(perOwner bool) *armies {
	return &armies{perOwner: perOwner, data: map[string]map[string]json.RawMessage{}}
}

func (a *armies) space(r *http.Request) map[string]json.RawMessage {
	who := "shared"
	if a.perOwner {
		who = r.Header.Get("X-Who")
	}
	if a.data[who] == nil {
		a.data[who] = map[string]json.RawMessage{}
	}
	return a.data[who]
}

func (a *armies) Routes() http.Handler {
	mux := http.NewServeMux()
	write := func(w http.ResponseWriter, status int, v any) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_ = json.NewEncoder(w).Encode(v)
	}
	mux.HandleFunc("GET /api/armies", func(w http.ResponseWriter, r *http.Request) {
		a.mu.Lock()
		defer a.mu.Unlock()
		items := make([]json.RawMessage, 0)
		for _, v := range a.space(r) {
			items = append(items, v)
		}
		write(w, 200, map[string]any{"items": items})
	})
	mux.HandleFunc("POST /api/armies", func(w http.ResponseWriter, r *http.Request) {
		a.mu.Lock()
		defer a.mu.Unlock()
		var in struct {
			ID string `json:"id"`
		}
		var raw json.RawMessage
		if json.NewDecoder(r.Body).Decode(&raw) != nil || json.Unmarshal(raw, &in) != nil || in.ID == "" {
			write(w, 400, map[string]string{"error": "bad"})
			return
		}
		a.space(r)[in.ID] = raw
		write(w, 201, raw)
	})
	mux.HandleFunc("GET /api/armies/{id}", func(w http.ResponseWriter, r *http.Request) {
		a.mu.Lock()
		defer a.mu.Unlock()
		v, ok := a.space(r)[r.PathValue("id")]
		if !ok {
			write(w, 404, map[string]string{"error": "not found"})
			return
		}
		write(w, 200, v)
	})
	mux.HandleFunc("PUT /api/armies/{id}", func(w http.ResponseWriter, r *http.Request) {
		a.mu.Lock()
		defer a.mu.Unlock()
		sp := a.space(r)
		if _, ok := sp[r.PathValue("id")]; !ok {
			write(w, 404, map[string]string{"error": "not found"})
			return
		}
		var raw json.RawMessage
		_ = json.NewDecoder(r.Body).Decode(&raw)
		sp[r.PathValue("id")] = raw
		write(w, 200, raw)
	})
	mux.HandleFunc("DELETE /api/armies/{id}", func(w http.ResponseWriter, r *http.Request) {
		a.mu.Lock()
		defer a.mu.Unlock()
		sp := a.space(r)
		if _, ok := sp[r.PathValue("id")]; !ok {
			write(w, 404, map[string]string{"error": "not found"})
			return
		}
		delete(sp, r.PathValue("id"))
		w.WriteHeader(204)
	})
	return mux
}

func who(name string) Identity {
	return Identity{Name: name, Apply: func(r *http.Request) *http.Request {
		r.Header.Set("X-Who", name)
		return r
	}}
}

func armyRow() Row {
	return Row{Entity: "Army", Factory: func(t TB, as *Actor) Record {
		body := map[string]any{"id": "army-1", "name": "Vanguard"}
		if res := as.Do("POST", "/api/armies", body); res.Status != 201 {
			t.Fatalf("cannot make an army: %d %s", res.Status, res.Body)
		}
		return Record{ID: "army-1", Create: body, Update: map[string]any{"id": "army-1", "name": "Renamed"}}
	}}
}

func config(t *testing.T, h http.Handler, rows ...Row) Config {
	return Config{Handler: h, First: who("alice"), Second: who("bob"), OpenAPI: writeContract(t, contract), Rows: rows}
}

func TestScopeHelperAcceptsAScopedHandler(t *testing.T) {
	t.Setenv(ReportEnv, "") // the platform reads this file: a helper test must never write to it
	ft := &fakeT{}
	var entries []Entry
	errs := ft.run(func() { entries = Run(ft, config(t, newArmies(true).Routes(), armyRow())) })
	if len(errs) != 0 {
		t.Fatalf("a per-owner handler failed: %v", errs)
	}
	if len(entries) != 5 {
		t.Fatalf("want one entry for each of 5 operations, got %d: %+v", len(entries), entries)
	}
	for _, e := range entries {
		if e.Outcome != Refused || e.Reason == "" {
			t.Errorf("%s %s: outcome %q reason %q, want refused with a reason", e.Method, e.Path, e.Outcome, e.Reason)
		}
	}
}

func TestScopeHelperCatchesALeakingHandler(t *testing.T) {
	t.Setenv(ReportEnv, "") // the platform reads this file: a helper test must never write to it
	ft := &fakeT{}
	var entries []Entry
	errs := ft.run(func() { entries = Run(ft, config(t, newArmies(false).Routes(), armyRow())) })
	if len(errs) == 0 {
		t.Fatal("a handler that shares one keyspace between users passed")
	}
	leaked := map[string]bool{}
	for _, e := range entries {
		if e.Outcome == Leaked {
			leaked[e.Method+" "+e.Path] = true
		}
	}
	for _, want := range []string{"GET /api/armies", "GET /api/armies/{id}", "PUT /api/armies/{id}"} {
		if !leaked[want] {
			t.Errorf("%s was not reported leaked; leaked: %v", want, leaked)
		}
	}
	if !strings.Contains(strings.Join(errs, "\n"), "LEAKED") {
		t.Errorf("the failure does not say LEAKED: %v", errs)
	}
}

func TestScopeHelperCatchesACollisionThatOverwrites(t *testing.T) {
	t.Setenv(ReportEnv, "") // the platform reads this file: a helper test must never write to it
	ft := &fakeT{}
	row := armyRow()
	f := row.Factory
	row.Factory = func(t TB, as *Actor) Record {
		rec := f(t, as)
		rec.Collide = map[string]any{"id": "army-1", "name": "Taken over"}
		return rec
	}
	var entries []Entry
	ft.run(func() { entries = Run(ft, config(t, newArmies(false).Routes(), row)) })
	found := false
	for _, e := range entries {
		if e.Method == "POST" && e.Outcome == Leaked && strings.Contains(e.Reason, "changed") {
			found = true
		}
	}
	if !found {
		t.Errorf("a create by the second identity that overwrote the first identity's record was not reported: %+v", entries)
	}
}

func TestScopeHelperFailsOnMissingCoverage(t *testing.T) {
	t.Setenv(ReportEnv, "") // the platform reads this file: a helper test must never write to it
	ft := &fakeT{}
	errs := ft.run(func() { Run(ft, config(t, newArmies(true).Routes())) })
	joined := strings.Join(errs, "\n")
	if !strings.Contains(joined, "no row, no probe and no waiver") || !strings.Contains(joined, "GET /api/armies/{id}") {
		t.Errorf("a per-user operation with no row did not fail the coverage assertion: %v", errs)
	}
	if strings.Contains(joined, "/api/rules") {
		t.Errorf("a shared entity's route was demanded: %v", errs)
	}
}

func TestScopeHelperWaiverNeedsAReasonAndMustBeReal(t *testing.T) {
	t.Setenv(ReportEnv, "") // the platform reads this file: a helper test must never write to it
	ft := &fakeT{}
	row := armyRow()
	row.Waive = map[string]string{"DELETE /api/armies/{id}": "  "}
	errs := ft.run(func() { Run(ft, config(t, newArmies(true).Routes(), row)) })
	if len(errs) == 0 || !strings.Contains(errs[0], "has no reason") {
		t.Errorf("a waiver without a reason was accepted: %v", errs)
	}

	ft = &fakeT{}
	row = armyRow()
	row.Waive = map[string]string{"GET /api/armies": "the armies list is paged by a proxy"}
	var entries []Entry
	errs = ft.run(func() { entries = Run(ft, config(t, newArmies(true).Routes(), row)) })
	if len(errs) != 0 {
		t.Fatalf("a waived list failed: %v", errs)
	}
	var waived *Entry
	for i, e := range entries {
		if e.Method == "GET" && e.Path == "/api/armies" {
			waived = &entries[i]
		}
	}
	if waived == nil || waived.Outcome != Waived || waived.Reason != "the armies list is paged by a proxy" {
		t.Errorf("the waiver is not in the report with its reason: %+v", entries)
	}

	ft = &fakeT{}
	row = armyRow()
	row.Waive = map[string]string{"GET /api/nothing": "stale"}
	errs = ft.run(func() { Run(ft, config(t, newArmies(true).Routes(), row)) })
	if len(errs) == 0 || !strings.Contains(strings.Join(errs, "\n"), "not a per-user operation") {
		t.Errorf("a stale waiver was accepted: %v", errs)
	}
}

func TestScopeHelperRejectsAFactoryRowForNothing(t *testing.T) {
	t.Setenv(ReportEnv, "") // the platform reads this file: a helper test must never write to it
	ft := &fakeT{}
	errs := ft.run(func() {
		Run(ft, config(t, newArmies(true).Routes(), armyRow(), Row{Entity: "Rule", Factory: armyRow().Factory}))
	})
	if len(errs) == 0 || !strings.Contains(strings.Join(errs, "\n"), "row for Rule") {
		t.Errorf("a row for a shared entity was accepted: %v", errs)
	}
}

func TestScopeHelperWritesTheReport(t *testing.T) {
	report := filepath.Join(t.TempDir(), "report.json")
	t.Setenv(ReportEnv, report)
	ft := &fakeT{}
	ft.run(func() { Run(ft, config(t, newArmies(false).Routes(), armyRow())) })

	raw, err := os.ReadFile(report)
	if err != nil {
		t.Fatalf("no report written: %v", err)
	}
	var got struct {
		Operations []struct {
			Method  string `json:"method"`
			Path    string `json:"path"`
			Outcome string `json:"outcome"`
			Reason  string `json:"reason"`
		} `json:"operations"`
	}
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatalf("the report is not the documented JSON: %v\n%s", err, raw)
	}
	if len(got.Operations) != 5 {
		t.Fatalf("want 5 operations in the report, got %d: %s", len(got.Operations), raw)
	}
	outcomes := map[string]string{}
	for _, o := range got.Operations {
		if o.Outcome != Refused && o.Outcome != Leaked && o.Outcome != Waived {
			t.Errorf("%s %s: outcome %q is not one of refused, leaked, waived", o.Method, o.Path, o.Outcome)
		}
		outcomes[o.Method+" "+o.Path] = o.Outcome
	}
	if outcomes["GET /api/armies/{id}"] != Leaked {
		t.Errorf("the leaking read is not in the report as leaked: %v", outcomes)
	}
}

func TestScopeHelperReportsAnEmptyContract(t *testing.T) {
	report := filepath.Join(t.TempDir(), "report.json")
	t.Setenv(ReportEnv, report)
	ft := &fakeT{}
	cfg := config(t, newArmies(true).Routes())
	cfg.OpenAPI = writeContract(t, "openapi: 3.1.0\npaths:\n  /api/health:\n    get:\n      summary: up\n")
	if errs := ft.run(func() { Run(ft, cfg) }); len(errs) != 0 {
		t.Fatalf("a contract with no per-user entity failed: %v", errs)
	}
	raw, err := os.ReadFile(report)
	if err != nil || strings.TrimSpace(string(raw)) != `{"operations":[]}` {
		t.Errorf("want an empty operations list, got %q (%v)", raw, err)
	}
}

func TestScopeHelperReadsTheContract(t *testing.T) {
	t.Setenv(ReportEnv, "")
	doc := `
x-data-model:
  entities:
    Army:
      ownership: per-user
      reads: "mine"   # a comment
    Note:
      ownership: shared
      reads: all
paths:
  /api/armies:
    post: { x-entity: Army }
  "/api/armies/{id}":
    description: |
      get:
        x-entity: Note
    get: { summary: read, x-entity: Army }
    delete:
      summary: remove
      x-entity: "Army"
  /api/notes:
    post:
      x-entity: Note
    get:
      x-entity: Note
  /api/notes/{id}:
    get:
      x-entity: Note
  /api/armies/{id}/orders:
    post:
      x-entity: Army
`
	c, err := readContract(doc)
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	for _, o := range c.PerUser {
		got = append(got, o.Key())
	}
	sort.Strings(got)
	want := "DELETE /api/armies/{id}, GET /api/armies/{id}, POST /api/armies, POST /api/armies/{id}/orders"
	if strings.Join(got, ", ") != want {
		t.Errorf("operations = %v, want %s (the shared Note collection skipped)", got, want)
	}
}

func TestScopeHelperRefusesBadWaivers(t *testing.T) {
	t.Setenv(ReportEnv, "") // the platform reads this file: a helper test must never write to it
	for name, tc := range map[string]struct {
		waive map[string]string
		want  string
	}{
		"delete":     {map[string]string{"DELETE /api/armies/{id}": "armies are never deleted here"}, "only a read (GET)"},
		"update":     {map[string]string{"PUT /api/armies/{id}": "armies are never updated here"}, "only a read (GET)"},
		"create":     {map[string]string{"POST /api/armies": "armies are made by import"}, "only a read (GET)"},
		"two words":  {map[string]string{"GET /api/armies": "armies elsewhere"}, "at least three words"},
		"not about":  {map[string]string{"GET /api/armies": "covered by another test somewhere"}, "does not name the entity"},
		"too many":   {map[string]string{"GET /api/armies": "armies list is covered", "GET /api/armies/{id}": "army read is covered"}, "too many waivers"},
		"admin-only": {map[string]string{"GET /api/admin/armies": "admin armies are checked elsewhere"}, "admin-only"},
	} {
		doc := contract
		if name == "admin-only" {
			doc = contract + "  /api/admin/armies:\n    get:\n      x-admin-only: true\n"
		}
		ft := &fakeT{}
		row := armyRow()
		row.Waive = tc.waive
		cfg := config(t, newArmies(true).Routes(), row)
		cfg.OpenAPI = writeContract(t, doc)
		var entries []Entry
		errs := strings.Join(ft.run(func() { entries = Run(ft, cfg) }), "\n")
		if !strings.Contains(errs, tc.want) {
			t.Errorf("%s: want a failure containing %q, got: %s", name, tc.want, errs)
		}
		for _, e := range entries {
			if e.Outcome == Waived && name != "too many" {
				t.Errorf("%s: a refused waiver still produced a waived entry: %+v", name, e)
			}
		}
	}
}

const adminContract = contract + `  /api/admin/armies/{id}:
    delete:
      x-admin-only: true
  /api/admin/stats:
    get: { x-required-groups: [admin] }
  /api/admin/export:
    get:
      x-groups:
        - admin
`

func adminHandler(admins map[string]bool, leakStats bool) http.Handler {
	base := newArmies(true).Routes()
	mux := http.NewServeMux()
	mux.HandleFunc("/api/admin/", func(w http.ResponseWriter, r *http.Request) {
		if !admins[r.Header.Get("X-Who")] && !(leakStats && r.URL.Path == "/api/admin/stats") {
			w.WriteHeader(http.StatusForbidden)
			return
		}
		w.WriteHeader(http.StatusOK)
	})
	mux.Handle("/", base)
	return mux
}

func adminProbes() []Probe {
	return []Probe{
		{Method: "DELETE", Path: "/api/admin/armies/{id}", Params: map[string]string{"id": "army-1"}},
		{Method: "GET", Path: "/api/admin/stats"},
		{Method: "GET", Path: "/api/admin/export"},
	}
}

func TestScopeHelperRefusesANonAdminOnAdminOperations(t *testing.T) {
	t.Setenv(ReportEnv, "") // the platform reads this file: a helper test must never write to it
	ft := &fakeT{}
	cfg := config(t, adminHandler(map[string]bool{"alice": true}, false), armyRow())
	cfg.OpenAPI = writeContract(t, adminContract)
	cfg.AdminProbes = adminProbes()
	var entries []Entry
	if errs := ft.run(func() { entries = Run(ft, cfg) }); len(errs) != 0 {
		t.Fatalf("a guarded admin API failed: %v", errs)
	}
	admin := 0
	for _, e := range entries {
		if strings.HasPrefix(e.Path, "/api/admin") {
			admin++
			if e.Outcome != Refused {
				t.Errorf("%s %s: %s", e.Method, e.Path, e.Outcome)
			}
		}
	}
	if admin != 3 {
		t.Errorf("want 3 admin entries (block, flow and list forms), got %d: %+v", admin, entries)
	}
}

func TestScopeHelperCatchesAnAdminOperationOpenToAUser(t *testing.T) {
	t.Setenv(ReportEnv, "") // the platform reads this file: a helper test must never write to it
	ft := &fakeT{}
	cfg := config(t, adminHandler(map[string]bool{"alice": true}, true), armyRow())
	cfg.OpenAPI = writeContract(t, adminContract)
	cfg.AdminProbes = adminProbes()
	errs := strings.Join(ft.run(func() { Run(ft, cfg) }), "\n")
	if !strings.Contains(errs, "LEAKED GET /api/admin/stats") {
		t.Errorf("an open admin route passed: %s", errs)
	}
}

func TestScopeHelperFailsOnAnAdminOperationWithNoProbe(t *testing.T) {
	t.Setenv(ReportEnv, "") // the platform reads this file: a helper test must never write to it
	ft := &fakeT{}
	cfg := config(t, adminHandler(nil, false), armyRow())
	cfg.OpenAPI = writeContract(t, adminContract)
	cfg.AdminProbes = adminProbes()[:1]
	errs := strings.Join(ft.run(func() { Run(ft, cfg) }), "\n")
	if !strings.Contains(errs, "GET /api/admin/stats (admin-only)") {
		t.Errorf("an admin-only operation with no probe was accepted: %s", errs)
	}
}

func TestScopeHelperReportIsAtomicAndMergesTests(t *testing.T) {
	dir := t.TempDir()
	report := filepath.Join(dir, "report.json")
	t.Setenv(ReportEnv, report)
	// An earlier test of the same package left one entry; a second test must not truncate it.
	prev := `{"operations":[{"method":"GET","path":"/api/other","outcome":"refused","reason":""}]}`
	if err := os.WriteFile(report, []byte(prev), 0o600); err != nil {
		t.Fatal(err)
	}
	ft := &fakeT{}
	ft.run(func() { Run(ft, config(t, newArmies(true).Routes(), armyRow())) })
	entries, _ := os.ReadDir(dir)
	for _, e := range entries {
		if strings.Contains(e.Name(), ".tmp-") {
			t.Errorf("a temporary file was left behind: %s", e.Name())
		}
	}
	raw, err := os.ReadFile(report)
	if err != nil {
		t.Fatal(err)
	}
	var got Report
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatal(err)
	}
	if len(got.Operations) != 6 {
		t.Errorf("want the earlier entry and the 5 new ones, got %d: %s", len(got.Operations), raw)
	}
	// Running again replaces, not duplicates.
	ft = &fakeT{}
	ft.run(func() { Run(ft, config(t, newArmies(true).Routes(), armyRow())) })
	raw, _ = os.ReadFile(report)
	if err := json.Unmarshal(raw, &got); err != nil || len(got.Operations) != 6 {
		t.Errorf("a second run duplicated entries: %s", raw)
	}
}

func TestScopeHelperWaiverReasonsAreNotReused(t *testing.T) {
	t.Setenv(ReportEnv, "") // the platform reads this file: a helper test must never write to it
	doc := contract + "  /api/armies/{id}/units:\n    get:\n      x-entity: Army\n"
	ft := &fakeT{}
	row := armyRow()
	same := "armies are read through the roster service"
	row.Waive = map[string]string{"GET /api/armies": same, "GET /api/armies/{id}": same, "GET /api/armies/{id}/units": same}
	cfg := config(t, newArmies(true).Routes(), row)
	cfg.OpenAPI = writeContract(t, doc)
	errs := strings.Join(ft.run(func() { Run(ft, cfg) }), "\n")
	if !strings.Contains(errs, "already used on two operations") {
		t.Errorf("one reason on three waivers was accepted: %s", errs)
	}
}

func TestScopeHelperAdminOnlyWithNoPerUserEntities(t *testing.T) {
	doc := `openapi: 3.1.0
x-data-model:
  entities:
    Rule:
      ownership: shared
      reads: everyone
paths:
  /api/rules:
    get:
      x-entity: Rule
  /api/admin/purge:
    post:
      x-admin-only: true
`
	report := filepath.Join(t.TempDir(), "report.json")
	t.Setenv(ReportEnv, report)
	cfg := config(t, adminHandler(map[string]bool{"alice": true}, false))
	cfg.OpenAPI = writeContract(t, doc)
	cfg.AdminProbes = []Probe{{Method: "POST", Path: "/api/admin/purge"}}
	ft := &fakeT{}
	var entries []Entry
	if errs := ft.run(func() { entries = Run(ft, cfg) }); len(errs) != 0 {
		t.Fatalf("admin-only alone failed: %v", errs)
	}
	if len(entries) != 1 || entries[0].Outcome != Refused || entries[0].Path != "/api/admin/purge" {
		t.Errorf("want the one admin-only entry, refused: %+v", entries)
	}
	// Without the probe it is owed, and cannot be waived.
	cfg.AdminProbes = nil
	ft = &fakeT{}
	errs := strings.Join(ft.run(func() { Run(ft, cfg) }), "\n")
	if !strings.Contains(errs, "POST /api/admin/purge (admin-only)") {
		t.Errorf("an admin-only operation was not required when the data is shared: %s", errs)
	}
}

// TestMain keeps the whole package off the platform's report file, whatever a test forgets.
func TestMain(m *testing.M) {
	_ = os.Unsetenv(ReportEnv)
	os.Exit(m.Run())
}

// The fixtures in testdata/contracts are read by the platform's own reader as well: each .required
// file is the platform's answer (the required operations by its perUserOperations and
// requiredOperations, as method and path with the /api base and parameter names left out), and this
// package must give the same. A fixture changed here must be re-checked against the platform's
// own reader before its .required file is rewritten.
func TestScopeHelperRequiredSetMatchesThePlatform(t *testing.T) {
	t.Setenv(ReportEnv, "")
	files, err := filepath.Glob("testdata/contracts/*.yaml")
	if err != nil || len(files) < 5 {
		t.Fatalf("fixtures missing: %v %v", files, err)
	}
	for _, f := range files {
		raw, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		want, err := os.ReadFile(strings.TrimSuffix(f, ".yaml") + ".required")
		if err != nil {
			t.Fatalf("%s has no .required file: %v", f, err)
		}
		c, err := readContract(string(raw))
		if err != nil {
			t.Fatalf("%s: %v", f, err)
		}
		if got := strings.Join(c.Required(), "\n") + "\n"; got != string(want) {
			t.Errorf("%s: required operations differ from the platform's\n got:\n%s want:\n%s", f, got, want)
		}
	}
}

func TestScopeHelperFailsClosedOnYAMLItCannotRead(t *testing.T) {
	t.Setenv(ReportEnv, "")
	for name, doc := range map[string]string{
		"tab":              "paths:\n\t/a:\n\t\tpost: {}\n",
		"anchor":           "paths:\n  /a: &a\n    post: {}\n  /b: *a\n",
		"alias":            "paths:\n  /a:\n    post: *x\n",
		"merge":            "paths:\n  /a:\n    <<: {post: {}}\n",
		"duplicate":        "paths:\n  /a:\n    post: {}\n  /a:\n    get: {}\n",
		"two docs":         "paths:\n  /a:\n    post: {}\n---\npaths: {}\n",
		"stray":            "paths:\n  /a:\n    post: {}\n   garbage\n",
		"unclosed":         "paths: {\n  /a: {post: {}}\n",
		"no paths":         "openapi: 3.1.0\n",
		"not yaml":         "just some words",
		"item list":        "paths:\n  /a:\n    - post\n",
		"key anchor":       "paths:\n  &a /a:\n    post: {}\n",
		"key alias":        "paths:\n  /a:\n    post: {}\n  *a : {}\n",
		"key tag":          "paths:\n  !!str /a:\n    post: {}\n",
		"flow key":         "paths:\n  /a: {&k post: {}}\n",
		"item key":         "paths:\n  /a:\n    - &k post: {}\n",
		"yaml-only escape": "paths:\n  \"\\/items\":\n    post: {}\n",
		"escape value":     "paths:\n  /a:\n    post: {x-entity: \"\\/items\"}\n",
	} {
		if _, err := readContract(doc); err == nil {
			t.Errorf("%s: a contract that cannot be read was accepted", name)
		}
	}
	// And Run fails the test rather than checking nothing.
	cfg := config(t, newArmies(true).Routes(), armyRow())
	cfg.OpenAPI = writeContract(t, "paths:\n\t/a: {}\n")
	ft := &fakeT{}
	if errs := ft.run(func() { Run(ft, cfg) }); len(errs) == 0 || !strings.Contains(errs[0], "tab") {
		t.Errorf("Run accepted a contract it cannot read: %v", errs)
	}
	cfg.OpenAPI = filepath.Join(t.TempDir(), "missing.yaml")
	ft = &fakeT{}
	if errs := ft.run(func() { Run(ft, cfg) }); len(errs) == 0 {
		t.Errorf("Run accepted a missing contract")
	}
}

func TestScopeHelperAppliesTheServersBasePath(t *testing.T) {
	t.Setenv(ReportEnv, "")
	c, err := readContract("servers:\n  - url: https://example.test/v2/\npaths:\n  /boxes:\n    post: {}\n  /boxes/{id}:\n    get: {}\n")
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	for _, o := range c.PerUser {
		got = append(got, o.Key())
	}
	sort.Strings(got)
	if strings.Join(got, ",") != "GET /v2/boxes/{id},POST /v2/boxes" {
		t.Errorf("paths = %v, want the /v2 base in front", got)
	}
}

func TestScopeHelperAdminDetectionMatchesThePlatform(t *testing.T) {
	t.Setenv(ReportEnv, "")
	c, err := readContract(`paths:
  /a: {get: {x-required-groups: [Admin]}}
  /b: {get: {x-groups: [ops, ADMIN]}}
  /c: {get: {x-groups: [administrators]}}
  /d:
    x-admin-only: true
    get: {}
  /e:
    get:
      security: [{oauth: ["reports:admin"]}]
  /f:
    get:
      security: [{oauth: ["reports:admin"]}, {oauth: [read]}]
  /g: {get: {x-admin-only: "true"}}
`)
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	for _, o := range c.Admin {
		got = append(got, o.Path)
	}
	sort.Strings(got)
	if strings.Join(got, ",") != "/a,/b,/d,/e" {
		t.Errorf("admin-only = %v, want /a,/b,/d,/e (any case, a path-item key, a scope; not a substring, a mixed alternative or a string)", got)
	}
}

// A collection whose records have an action route, and one that names no entity: both are required,
// and the action is a probe, not a create.
const actionContract = `servers: [{url: /api}]
x-data-model:
  entities:
    Army: {ownership: per-user, reads: mine}
paths:
  /armies:
    get: {x-entity: Army}
    post: {x-entity: Army}
  /armies/{id}:
    get: {x-entity: Army}
  /armies/{id}/archive:
    post: {x-entity: Army}
  /armies/{id}/export:
    get: {x-entity: Army}
`

func TestScopeHelperClassifiesActionRoutes(t *testing.T) {
	t.Setenv(ReportEnv, "")
	c, err := readContract(actionContract)
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]shape{
		"POST /api/armies":              shapeCreate,
		"GET /api/armies":               shapeList,
		"GET /api/armies/{id}":          shapeRead,
		"POST /api/armies/{id}/archive": shapeOther,
		"GET /api/armies/{id}/export":   shapeOther,
	}
	for _, o := range c.PerUser {
		if s, ok := want[o.Key()]; !ok || shapeOf(o) != s {
			t.Errorf("%s: shape %v, want %v", o.Key(), shapeOf(o), s)
		}
		delete(want, o.Key())
	}
	if len(want) != 0 {
		t.Errorf("operations not read: %v", want)
	}
}

func TestScopeHelperActionRouteNeedsAProbeAndIsRefusedWhenScoped(t *testing.T) {
	t.Setenv(ReportEnv, "")
	h := func(perOwner bool) http.Handler {
		base := newArmies(perOwner)
		mux := base.Routes().(*http.ServeMux)
		mux.HandleFunc("POST /api/armies/{id}/archive", func(w http.ResponseWriter, r *http.Request) {
			base.mu.Lock()
			defer base.mu.Unlock()
			if _, ok := base.space(r)[r.PathValue("id")]; !ok {
				w.WriteHeader(http.StatusNotFound)
				return
			}
			w.WriteHeader(http.StatusOK)
		})
		mux.HandleFunc("GET /api/armies/{id}/export", func(w http.ResponseWriter, r *http.Request) {
			base.mu.Lock()
			defer base.mu.Unlock()
			if _, ok := base.space(r)[r.PathValue("id")]; !ok {
				w.WriteHeader(http.StatusNotFound)
				return
			}
			w.WriteHeader(http.StatusOK)
		})
		return mux
	}
	cfgOf := func(handler http.Handler, row Row) Config {
		cfg := config(t, handler, row)
		cfg.OpenAPI = writeContract(t, actionContract)
		return cfg
	}
	// No probe for the action and no waiver for the export: both are missing.
	ft := &fakeT{}
	errs := strings.Join(ft.run(func() { Run(ft, cfgOf(h(true), armyRow())) }), "\n")
	if !strings.Contains(errs, "POST /api/armies/{id}/archive (Army)") || !strings.Contains(errs, "GET /api/armies/{id}/export (Army)") {
		t.Errorf("action routes were not required: %s", errs)
	}
	// With a probe for the action and the export waived (a GET may be), a scoped app passes.
	row := armyRow()
	row.Probes = []Probe{{Method: "POST", Path: "/api/armies/{id}/archive"}}
	row.Waive = map[string]string{"GET /api/armies/{id}/export": "the export is built by the army reporting job"}
	ft = &fakeT{}
	if errs := ft.run(func() { Run(ft, cfgOf(h(true), row)) }); len(errs) != 0 {
		t.Fatalf("a scoped app with the action probed failed: %v", errs)
	}
	// An action that acts on another's record is a leak.
	ft = &fakeT{}
	leaky := newArmies(false)
	mux := leaky.Routes().(*http.ServeMux)
	mux.HandleFunc("POST /api/armies/{id}/archive", func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) })
	errs = strings.Join(ft.run(func() { Run(ft, cfgOf(mux, row)) }), "\n")
	if !strings.Contains(errs, "LEAKED POST /api/armies/{id}/archive") {
		t.Errorf("an action that answers another person's record passed: %s", errs)
	}
}

func TestScopeHelperAnAdminProbeMustSeeA403(t *testing.T) {
	t.Setenv(ReportEnv, "")
	for status, leaked := range map[int]bool{http.StatusForbidden: false, http.StatusNotFound: true, http.StatusUnauthorized: true, http.StatusOK: true} {
		h := http.NewServeMux()
		h.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(status) })
		cfg := config(t, h)
		cfg.OpenAPI = writeContract(t, "paths:\n  /api/admin/purge:\n    post:\n      x-admin-only: true\n")
		cfg.AdminProbes = []Probe{{Method: "POST", Path: "/api/admin/purge"}}
		ft := &fakeT{}
		var entries []Entry
		ft.run(func() { entries = Run(ft, cfg) })
		if len(entries) != 1 || (entries[0].Outcome == Leaked) != leaked {
			t.Errorf("status %d: entries %+v, leaked want %v", status, entries, leaked)
		}
	}
}

func TestScopeHelperWaiverAllowanceCountsThePlatformsRequiredSet(t *testing.T) {
	t.Setenv(ReportEnv, "")
	// Two collections, one naming no entity: 4 + 2 = 6 required operations, so one waiver is allowed
	// (6/4 = 1) and a second is not. The shared Rule collection adds none.
	doc := `servers: [{url: /api}]
x-data-model:
  entities:
    Army: {ownership: per-user, reads: mine}
    Rule: {ownership: shared, reads: all}
paths:
  /armies: {post: {x-entity: Army}, get: {x-entity: Army}}
  /armies/{id}: {get: {x-entity: Army}, delete: {x-entity: Army}}
  /rules: {post: {x-entity: Rule}, get: {x-entity: Rule}}
  /rules/{id}: {get: {x-entity: Rule}, delete: {x-entity: Rule}}
  /notes: {post: {}}
  /notes/{id}: {get: {}}
`
	c, err := readContract(doc)
	if err != nil {
		t.Fatal(err)
	}
	if n := len(c.PerUser) + len(c.Admin); n != 6 {
		t.Fatalf("required = %d (%v), want 6", n, c.Required())
	}
	cfg := config(t, newArmies(true).Routes(), armyRow())
	cfg.OpenAPI = writeContract(t, doc)
	row := armyRow()
	row.Waive = map[string]string{"GET /api/armies": "the armies list is paged elsewhere", "GET /api/armies/{id}": "reading an army goes through the roster"}
	cfg.Rows = []Row{row, {Entity: "/api/notes", Factory: armyRow().Factory, Waive: map[string]string{"GET /api/notes/{id}": "notes are read by the notes service"}}}
	ft := &fakeT{}
	errs := strings.Join(ft.run(func() { Run(ft, cfg) }), "\n")
	if !strings.Contains(errs, "too many waivers") {
		t.Errorf("waivers over a quarter of the platform's required set were accepted: %s", errs)
	}
}

// A package that covers part of the contract (the sign-in plugin's own routes) sets Only, so the
// app's per-user entities, which it does not own, owe it nothing; its admin-only routes are probed.
func TestScopeHelperOnlyLeavesOtherPackagesOperationsAlone(t *testing.T) {
	doc := `servers: [{url: /api}]
x-data-model:
  entities:
    Note: {ownership: per-user, reads: mine}
paths:
  /notes: {post: {x-entity: Note}, get: {x-entity: Note}}
  /notes/{id}: {get: {x-entity: Note}, delete: {x-entity: Note}}
  /auth/users: {get: {x-admin-only: true}, post: {x-admin-only: true}}
`
	t.Setenv(ReportEnv, filepath.Join(t.TempDir(), "report.json"))
	// The sign-in routes refuse everyone but alice; the app's notes routes do not exist here at all.
	mux := http.NewServeMux()
	mux.HandleFunc("/api/auth/", func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Who") != "alice" {
			w.WriteHeader(http.StatusForbidden)
		}
	})
	cfg := config(t, mux)
	cfg.OpenAPI = writeContract(t, doc)
	cfg.AdminProbes = []Probe{{Method: "GET", Path: "/api/auth/users"}, {Method: "POST", Path: "/api/auth/users"}}
	// Without Only the Note operations are owed a row.
	ft := &fakeT{}
	if errs := strings.Join(ft.run(func() { Run(ft, cfg) }), "\n"); !strings.Contains(errs, "/api/notes") {
		t.Errorf("the per-user entity was not required without Only: %s", errs)
	}
	cfg.Only = []string{"/api/auth/"}
	ft = &fakeT{}
	var entries []Entry
	if errs := ft.run(func() { entries = Run(ft, cfg) }); len(errs) != 0 {
		t.Fatalf("Only did not drop the operations of the app's own entity: %v", errs)
	}
	if len(entries) != 2 {
		t.Errorf("want the two admin-only entries, got %+v", entries)
	}
}

func TestScopeHelperNoWaiversWhenOnlyOrElsewhereIsSet(t *testing.T) {
	t.Setenv(ReportEnv, "")
	for name, set := range map[string]func(*Config){
		"only":      func(c *Config) { c.Only = []string{"/api/"} },
		"elsewhere": func(c *Config) { c.Elsewhere = []string{"/api/rules"} },
	} {
		cfg := config(t, newArmies(true).Routes(), armyRow())
		row := armyRow()
		row.Waive = map[string]string{"GET /api/armies": "the armies list is paged elsewhere"}
		cfg.Rows = []Row{row}
		set(&cfg)
		ft := &fakeT{}
		if errs := strings.Join(ft.run(func() { Run(ft, cfg) }), "\n"); !strings.Contains(errs, "not allowed when Elsewhere or Only is set") {
			t.Errorf("%s: a waiver was accepted: %s", name, errs)
		}
	}
}
