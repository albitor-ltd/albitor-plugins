// Package ownedscope is the two-owner contract test: it proves that one signed-in person cannot
// read, change, delete or list another person's records, against the app's real routes, in this
// process, with two injected identities. It is test support; nothing in the running API imports it.
//
// WHAT IT CHECKS. The contract (openapi.yaml) declares each entity per-user or shared
// (x-data-model) and names the entity each route reads or writes (x-entity). For every route of a
// per-user entity, the first identity makes a record, then the second identity:
//
//   - GET the record, PUT or PATCH it, DELETE it: each must be REFUSED (404 or 403);
//   - lists the entity: the answer must not contain the record (or be refused);
//   - creates a record with the same id: the first identity's record must be left intact.
//
// After each attempt the first identity's record is read back and must be unchanged.
//
// WHICH ROUTES. The set the test must cover is computed by the PLATFORM's rule, which this package
// copies (readContract; testdata holds both to the same answers): for every collection (a POST with
// no path parameter, neither public nor admin-only) that has records under it, the create, the list
// and every route under one record (the record, and actions such as POST /things/{id}/archive),
// except a collection whose create names an entity the data model declares shared. A route that
// names no entity is still required, under its collection's path as the Row's Entity.
//
// ADMIN-ONLY ROUTES. For every operation the contract declares admin-only (x-admin-only: true,
// x-required-groups / x-groups naming admin in any case, or a security requirement asking for an
// admin scope) the second identity, who must not be an admin, calls it through an entry in
// Config.AdminProbes and must be answered 403 exactly: a 404 is a route that does not exist, which
// says nothing about who may call it.
//
// WAIVERS are narrow: only a read (GET), never a create, update, delete, action or admin-only
// route; a reason of at least three words that names the entity or the route; and no more than a
// quarter (at least one) of the operations the test must cover (the platform's required set), and
// one reason on no more than two operations. The test fails on a waiver that breaks a rule.
//
// WHAT THIS DOES NOT COVER. It runs the app's router in this process with two injected identities,
// so it proves the handlers and the store scope records by the identity they are given. It does not
// prove that the deployed sign-in puts the right identity there (token checks, the gateway, the
// authoriser), that a route the contract does not list is scoped, that a store behind a different
// wiring in production scopes the same way, or that the response bodies hide nothing the tests do
// not look at.
//
// NEVER SKIPPED, ONE FILE PER PACKAGE. The report is one file for the test package: tests in the
// same package merge into it (read, merge by method and path, rewrite atomically through a
// temporary file and a rename), so none truncates another's entries. The test needs no network, no
// docker and no environment variable but the report path, and must never be skipped.
//
// The report is read by the platform, which runs `go test -run ^TestOwnedDataScope` in every package
// and reads OWNED_DATA_SCOPE_REPORT: the tests of THIS package are therefore named TestScopeHelper...
// and never write to that file (each sets the variable itself).
//
// Keep the test. Register a Factory for every per-user entity you add. Never weaken it: do not
// widen the refused statuses, drop a row, or waive a route to make it pass.
package ownedscope

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"
)

// ReportEnv names the file the report is written to when it is set.
const ReportEnv = "OWNED_DATA_SCOPE_REPORT"

// DefaultOpenAPIPath is the contract as seen from a test in internal/app.
const DefaultOpenAPIPath = "../../openapi.yaml"

// The outcomes of the report.
const (
	Refused = "refused"
	Leaked  = "leaked"
	Waived  = "waived"
)

// TB is the part of *testing.T the test uses, so the helper can be tested with a stand-in.
type TB interface {
	Helper()
	Errorf(format string, args ...any)
	Fatalf(format string, args ...any)
	Cleanup(func())
}

// Identity is one signed-in person. Apply returns the request acting as them: it puts the
// identity where the app's identity package reads it (identity.With on the request context, in
// every auth plugin).
type Identity struct {
	Name  string
	Apply func(*http.Request) *http.Request
}

// Response is what a request was answered.
type Response struct {
	Status int
	Body   []byte
}

// Actor sends requests as one identity.
type Actor struct {
	Identity Identity
	Handler  http.Handler
}

// Do sends method path with body (nil for none, a string or []byte as it is, anything else as
// JSON) and returns the answer.
func (a *Actor) Do(method, path string, body any) Response {
	var rd *bytes.Reader
	switch b := body.(type) {
	case nil:
		rd = bytes.NewReader(nil)
	case []byte:
		rd = bytes.NewReader(b)
	case string:
		rd = bytes.NewReader([]byte(b))
	default:
		raw, err := json.Marshal(b)
		if err != nil {
			return Response{Status: -1, Body: []byte("cannot encode the body: " + err.Error())}
		}
		rd = bytes.NewReader(raw)
	}
	req := httptest.NewRequest(method, path, rd)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if a.Identity.Apply != nil {
		req = a.Identity.Apply(req)
	}
	rec := httptest.NewRecorder()
	a.Handler.ServeHTTP(rec, req)
	return Response{Status: rec.Code, Body: rec.Body.Bytes()}
}

// Record is a record the Factory made as the first identity.
type Record struct {
	// ID is the id the record is addressed by in the item path.
	ID string
	// Create is the body that created it; the second identity sends it again to collide on the id unless Collide is set.
	Create any
	// Update is a valid body for PUT or PATCH. Empty means Create.
	Update any
	// Collide is a valid create body with the same id and different content, which the second
	// identity sends to try to take the record over. Empty means Create.
	Collide any
	// Params gives the value of every other path parameter (a parent's id), by name.
	Params map[string]string
}

// Factory makes one record of an entity as the first identity (with as.Do, which also lets it
// make a parent first) and returns it. It must create the record itself, through the app's own
// routes, and fail the test if it cannot.
type Factory func(t TB, as *Actor) Record

// Probe drives a route the standard shapes do not cover (an action on a record, say). The second
// identity sends Method Path (the template, with {id} and Params filled in as for a record) with
// Body, and must be refused.
type Probe struct {
	Method string
	Path   string
	Body   any
	// Params gives the value of each path parameter of an admin-only route (a record's id), by name.
	// A Probe in a Row also takes the record's ID and Params.
	Params map[string]string
}

// Row is one per-user entity in the table.
type Row struct {
	// Entity is the x-entity of the collection's create, or the collection's path when the create
	// names none.
	Entity  string
	Factory Factory
	// Probes cover routes of the entity that are not create, list, read, update or delete.
	Probes []Probe
	// Waive gives, by "METHOD /path/{id}", the reason a read or list operation is not tested. A
	// waiver is a statement a reviewer reads, not a way to make the test pass, and the test refuses
	// one that is not allowed: only a read (GET) can be waived, never a create, update, delete,
	// action or admin-only route; the reason is at least three
	// words and names the entity or the route; and waivers are at most a quarter of the operations
	// the contract requires the test to cover.
	Waive map[string]string
}

// Config is one run.
type Config struct {
	// Handler is the app's real routes, with no sign-in in front of them.
	Handler http.Handler
	// First owns the records; Second attacks them. Their subjects must differ.
	First, Second Identity
	// OpenAPI is the contract's path. Empty means DefaultOpenAPIPath.
	OpenAPI string
	// Rows is the table, one row per per-user entity.
	Rows []Row
	// Elsewhere lists path prefixes whose operations ANOTHER test package covers (a sign-in plugin's
	// own routes, say, tested beside the code that serves them): this test neither covers nor
	// reports them. The platform merges the reports of every package and still requires each
	// operation, so an operation nobody covers still fails the check. Never use it to drop a route of
	// your own.
	Elsewhere []string
	// Only, when not empty, limits this test to the operations whose path starts with one of these
	// prefixes: every other operation is dropped the way Elsewhere drops it (neither covered nor
	// reported), because another test package owns it. A sign-in plugin's test of its own routes
	// sets this to its prefix, so an app's per-user entities are left to the app's own test. The
	// platform still requires every operation of the merged report. Never use it to drop a route
	// of your own.
	Only []string
	// AdminProbes drive each operation the contract declares admin-only. The second identity must
	// not be an admin; it must be answered 403. An admin-only operation cannot be waived.
	AdminProbes []Probe
}

// Entry is one operation of the report.
type Entry struct {
	Method  string `json:"method"`
	Path    string `json:"path"`
	Outcome string `json:"outcome"`
	Reason  string `json:"reason"`
}

// Report is the JSON written to OWNED_DATA_SCOPE_REPORT.
type Report struct {
	Operations []Entry `json:"operations"`
}

// Run is the whole test. Call it from TestOwnedDataScope... with the app's table.
func Run(t TB, cfg Config) []Entry {
	t.Helper()
	path := cfg.OpenAPI
	if path == "" {
		path = DefaultOpenAPIPath
	}
	c, err := ReadContract(path)
	if err != nil {
		t.Fatalf("owned data scope: %v", err)
	}
	return runOps(t, cfg, c)
}

func runOps(t TB, cfg Config, c Contract) []Entry {
	t.Helper()
	c = c.without(cfg.Elsewhere).only(cfg.Only)
	ops := c.PerUser
	var entries []Entry
	t.Cleanup(func() {
		if err := writeReport(os.Getenv(ReportEnv), entries); err != nil {
			t.Errorf("owned data scope: %v", err)
		}
	})
	if cfg.Handler == nil || cfg.First.Apply == nil || cfg.Second.Apply == nil {
		t.Fatalf("owned data scope: Config needs a Handler and an Apply for each of two identities")
	}

	rows := map[string]Row{}
	for _, r := range cfg.Rows {
		if r.Entity == "" {
			t.Fatalf("owned data scope: a table row has no Entity")
		}
		if _, dup := rows[r.Entity]; dup {
			t.Fatalf("owned data scope: entity %s has two rows", r.Entity)
		}
		for key, reason := range r.Waive {
			if strings.TrimSpace(reason) == "" {
				t.Fatalf("owned data scope: the waiver for %s (%s) has no reason", key, r.Entity)
			}
		}
		rows[r.Entity] = r
	}

	if len(cfg.Elsewhere) > 0 || len(cfg.Only) > 0 {
		// The platform counts the allowance and the reuse of a reason across the merged report of
		// every package; one package's view of it (this one) is partial, so it cannot judge a waiver.
		for _, r := range cfg.Rows {
			if len(r.Waive) > 0 {
				t.Errorf("owned data scope: %s has waivers, which are not allowed when Elsewhere or Only is set: the platform counts them across every package's report, so a package that covers only part of the contract cannot judge them", r.Entity)
			}
		}
	}

	byEntity := map[string][]Operation{}
	known := map[string]bool{}
	for _, o := range ops {
		byEntity[o.Entity] = append(byEntity[o.Entity], o)
		known[o.MatchKey()] = true
	}
	for entity := range rows {
		if len(byEntity[entity]) == 0 {
			t.Errorf("owned data scope: the table has a row for %s, but the contract has no per-user operation naming it", entity)
		}
	}
	for _, r := range rows {
		for key := range r.Waive {
			if !known[keyOf(key)] && !adminKeys0(c)[keyOf(key)] {
				t.Errorf("owned data scope: %s waives %q, which is not a per-user operation of the contract", r.Entity, key)
			}
		}
	}

	// Waiver rules. A waiver that breaks one is an error and does not count: the operation is
	// tested as if it had none.
	required := len(ops) + len(c.Admin)
	allowed := max(1, required/4)
	reasonUses := map[string]int{}
	adminKeys := map[string]bool{}
	for _, o := range c.Admin {
		adminKeys[o.MatchKey()] = true
	}
	opByKey := map[string]Operation{}
	for _, o := range ops {
		opByKey[o.MatchKey()] = o
	}
	valid := map[string]bool{} // "Entity|key" of waivers that pass the rules
	waivers := 0
	for _, r := range cfg.Rows {
		keys := make([]string, 0, len(r.Waive))
		for k := range r.Waive {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, key := range keys {
			o, per := opByKey[keyOf(key)]
			if !per {
				continue // reported as stale above, or admin-only below
			}
			if o.Method != http.MethodGet {
				t.Errorf("owned data scope: %s cannot be waived: only a read (GET) may be, never a create, update, delete or action", key)
				continue
			}
			if why := waiverReasonProblem(r.Waive[key], o); why != "" {
				t.Errorf("owned data scope: the waiver for %s %s", key, why)
				continue
			}
			norm := strings.ToLower(strings.Join(strings.Fields(r.Waive[key]), " "))
			reasonUses[norm]++
			if reasonUses[norm] > 2 {
				t.Errorf("owned data scope: the reason given for waiving %s is already used on two operations; each waiver needs a reason specific to its own operation", key)
				continue
			}
			waivers++
			if waivers > allowed {
				t.Errorf("owned data scope: too many waivers: at most a quarter (at least one) of the %d operations the contract requires to be covered (%d) may be waived; %s is over the limit", required, allowed, key)
				continue
			}
			valid[r.Entity+"|"+keyOf(key)] = true
		}
		for key := range r.Waive {
			if adminKeys[keyOf(key)] {
				t.Errorf("owned data scope: %s is admin-only and cannot be waived", key)
			}
		}
	}

	covered := map[string]bool{}
	record := func(o Operation, outcome, reason string) {
		entries = append(entries, Entry{Method: o.Method, Path: o.Path, Outcome: outcome, Reason: reason})
		covered[o.MatchKey()] = true
		if outcome == Leaked {
			t.Errorf("owned data scope: LEAKED %s: %s", o.Key(), reason)
		}
	}

	first := &Actor{Identity: cfg.First, Handler: cfg.Handler}
	second := &Actor{Identity: cfg.Second, Handler: cfg.Handler}

	entities := make([]string, 0, len(byEntity))
	for e := range byEntity {
		entities = append(entities, e)
	}
	sort.Strings(entities)
	for _, entity := range entities {
		eops := byEntity[entity]
		row, ok := rows[entity]
		if !ok {
			continue // reported below as not covered
		}
		var live []Operation
		for _, o := range eops {
			if reason, waived := waiveReason(row, o); waived && valid[row.Entity+"|"+o.MatchKey()] {
				record(o, Waived, reason)
				continue
			}
			live = append(live, o)
		}
		if len(live) == 0 {
			continue
		}
		if row.Factory == nil {
			t.Errorf("owned data scope: the row for %s has no Factory", entity)
			continue
		}
		exercise(t, row, live, first, second, record)
	}

	for _, o := range c.Admin {
		var probe *Probe
		for i := range cfg.AdminProbes {
			if keyOf(cfg.AdminProbes[i].Method+" "+cfg.AdminProbes[i].Path) == o.MatchKey() {
				probe = &cfg.AdminProbes[i]
			}
		}
		if probe == nil {
			continue // reported as missing below
		}
		p := o.Path
		ok := true
		for _, name := range pathParams(p) {
			v, has := probe.Params[name]
			if !has {
				t.Errorf("owned data scope: the admin probe for %s needs a value for {%s} in its Params", o.Key(), name)
				ok = false
				break
			}
			p = strings.ReplaceAll(p, "{"+name+"}", v)
		}
		if !ok {
			continue
		}
		res := second.Do(o.Method, p, probe.Body)
		if res.Status == http.StatusForbidden {
			record(o, Refused, "a signed-in non-admin identity called this admin-only operation and was answered 403")
		} else {
			record(o, Leaked, fmt.Sprintf("a signed-in non-admin identity called this admin-only operation and was answered %d, not a 403 (a 404 is a route that is not there, which proves nothing)", res.Status))
		}
	}

	var missing []string
	for _, o := range c.Admin {
		if !covered[o.MatchKey()] {
			missing = append(missing, o.Key()+" (admin-only)")
		}
	}
	for _, o := range ops {
		if !covered[o.MatchKey()] {
			missing = append(missing, o.Key()+" ("+o.Entity+")")
		}
	}
	if len(missing) > 0 {
		t.Errorf("owned data scope: per-user operations of the contract with no row, no probe and no waiver in the table (admin-only ones need an entry in AdminProbes): %s. "+
			"Register a Factory row for each entity (and a Probe for a route that is not create, list, read, update or delete, such as an action on a record); waive only a read (GET), with a specific written reason",
			strings.Join(missing, "; "))
	}
	return entries
}

// exercise makes a record as the first identity and attacks it as the second, for the live
// operations of one entity.
func exercise(t TB, row Row, live []Operation, first, second *Actor, record func(Operation, string, string)) {
	t.Helper()
	rec := row.Factory(t, first)
	if rec.ID == "" {
		t.Errorf("owned data scope: the Factory for %s returned no ID", row.Entity)
		return
	}
	update, collide := rec.Update, rec.Collide
	if update == nil {
		update = rec.Create
	}
	if collide == nil {
		collide = rec.Create
	}

	byShape := map[shape][]Operation{}
	for _, o := range live {
		byShape[shapeOf(o)] = append(byShape[shapeOf(o)], o)
	}
	fill := func(o Operation) (string, bool) {
		p := o.Path
		for _, name := range pathParams(p) {
			v, ok := rec.Params[name]
			if name == itemParam(o) {
				v, ok = rec.ID, true
			}
			if !ok {
				t.Errorf("owned data scope: %s needs a value for {%s}: set it in the Record.Params the %s Factory returns", o.Key(), name, row.Entity)
				return "", false
			}
			p = strings.ReplaceAll(p, "{"+name+"}", v)
		}
		return p, true
	}

	// What the first identity sees of its own record, before any attempt.
	var readPath string
	var before []byte
	if reads := byShape[shapeRead]; len(reads) > 0 {
		readPath, _ = fill(reads[0])
	} else if p, ok := itemPathOf(live, rec); ok {
		readPath = p
	}
	if readPath != "" {
		got := first.Do(http.MethodGet, readPath, nil)
		if got.Status != http.StatusOK {
			t.Errorf("owned data scope: the first identity cannot read its own %s record at GET %s (status %d); the test cannot show anything until it can", row.Entity, readPath, got.Status)
			return
		}
		before = got.Body
	}
	intact := func() (bool, string) {
		if readPath == "" {
			return true, ""
		}
		got := first.Do(http.MethodGet, readPath, nil)
		if got.Status != http.StatusOK {
			return false, fmt.Sprintf("the first identity's record is gone or unreadable afterwards (GET %s answered %d)", readPath, got.Status)
		}
		if !sameJSON(before, got.Body) {
			return false, "the first identity's record was changed"
		}
		return true, ""
	}
	refusal := func(o Operation, res Response, what string) {
		if res.Status != http.StatusNotFound && res.Status != http.StatusForbidden {
			record(o, Leaked, fmt.Sprintf("the second identity %s and was answered %d, not a 404 or 403", what, res.Status))
			return
		}
		if ok, why := intact(); !ok {
			record(o, Leaked, fmt.Sprintf("the second identity %s and was answered %d, but %s", what, res.Status, why))
			return
		}
		record(o, Refused, fmt.Sprintf("the second identity %s and was answered %d; the first identity's record is intact", what, res.Status))
	}

	for _, o := range byShape[shapeRead] {
		if p, ok := fill(o); ok {
			refusal(o, second.Do(o.Method, p, nil), "read the first identity's record")
		}
	}
	for _, o := range byShape[shapeList] {
		p, ok := fill(o)
		if !ok {
			continue
		}
		own := first.Do(http.MethodGet, p, nil)
		if !mentions(own.Body, rec.ID) {
			t.Errorf("owned data scope: the first identity's own list GET %s (status %d) does not show its %s record %q, so the list check would prove nothing; make the Factory's record appear on the first page", p, own.Status, row.Entity, rec.ID)
			continue
		}
		res := second.Do(o.Method, p, nil)
		switch {
		case res.Status == http.StatusNotFound || res.Status == http.StatusForbidden:
			record(o, Refused, fmt.Sprintf("the second identity listed and was answered %d", res.Status))
		case res.Status/100 == 2 && !mentions(res.Body, rec.ID):
			record(o, Refused, "the second identity's list does not include the first identity's record")
		case res.Status/100 == 2:
			record(o, Leaked, "the second identity's list includes the first identity's record")
		default:
			record(o, Leaked, fmt.Sprintf("the second identity listed and was answered %d, not a 2xx without the record or a 404 or 403", res.Status))
		}
	}
	for _, o := range byShape[shapeUpdate] {
		if p, ok := fill(o); ok {
			refusal(o, second.Do(o.Method, p, update), "updated the first identity's record")
		}
	}
	for _, o := range byShape[shapeOther] {
		matched := false
		for _, pr := range row.Probes {
			if keyOf(pr.Method+" "+pr.Path) == o.MatchKey() {
				matched = true
				if p, ok := fill(o); ok {
					refusal(o, second.Do(o.Method, p, pr.Body), "called "+o.Key()+" on the first identity's record")
				}
			}
		}
		if !matched {
			continue // not covered: reported as missing
		}
	}
	for _, o := range byShape[shapeDelete] {
		if p, ok := fill(o); ok {
			refusal(o, second.Do(o.Method, p, nil), "deleted the first identity's record")
		}
	}

	// The collision comes last: a record the second identity makes under the same id (in its own
	// keyspace) would otherwise answer the attempts above.
	for _, o := range byShape[shapeCreate] {
		p, ok := fill(o)
		if !ok {
			continue
		}
		res := second.Do(o.Method, p, collide)
		if ok, why := intact(); !ok {
			record(o, Leaked, fmt.Sprintf("the second identity created a record with the same id (answered %d), but %s", res.Status, why))
		} else {
			record(o, Refused, fmt.Sprintf("the second identity created a record with the same id (answered %d); the first identity's record is intact", res.Status))
		}
	}
	// Tidy up: the first identity removes its own record when the contract lets it.
	for _, o := range byShape[shapeDelete] {
		if p, ok := fill(o); ok {
			first.Do(http.MethodDelete, p, nil)
		}
	}
}

// itemPathOf finds the item path of the entity among its live operations, so a record can be read
// back for the intact check even when only the update or delete route is in the table.
func itemPathOf(live []Operation, rec Record) (string, bool) {
	for _, o := range live {
		if s := shapeOf(o); s == shapeUpdate || s == shapeDelete || s == shapeRead {
			p := o.Path
			for _, name := range pathParams(p) {
				v, ok := rec.Params[name]
				if name == itemParam(o) {
					v, ok = rec.ID, true
				}
				if !ok {
					return "", false
				}
				p = strings.ReplaceAll(p, "{"+name+"}", v)
			}
			return p, true
		}
	}
	return "", false
}

// mentions reports whether a JSON answer carries id as a string anywhere.
func mentions(body []byte, id string) bool {
	quoted, err := json.Marshal(id)
	if err != nil {
		return false
	}
	return bytes.Contains(body, quoted)
}

// sameJSON compares two JSON documents by value.
func sameJSON(a, b []byte) bool {
	var x, y any
	if json.Unmarshal(a, &x) != nil || json.Unmarshal(b, &y) != nil {
		return bytes.Equal(a, b)
	}
	xa, _ := json.Marshal(x)
	yb, _ := json.Marshal(y)
	return bytes.Equal(xa, yb)
}

// writeReport merges entries into the report at path and rewrites it atomically (a temporary file
// in the same directory, then a rename), or does nothing when path is empty. Entries already in the
// file (another test of the same package) are kept; one with the same method and path is replaced.
func writeReport(path string, entries []Entry) error {
	if path == "" {
		return nil
	}
	merged := map[string]Entry{}
	if raw, err := os.ReadFile(path); err == nil {
		var prev Report
		if json.Unmarshal(raw, &prev) == nil {
			for _, e := range prev.Operations {
				merged[e.Method+" "+e.Path] = e
			}
		}
	}
	for _, e := range entries {
		merged[e.Method+" "+e.Path] = e
	}
	out := make([]Entry, 0, len(merged))
	for _, e := range merged {
		out = append(out, e)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Path != out[j].Path {
			return out[i].Path < out[j].Path
		}
		return out[i].Method < out[j].Method
	})
	raw, err := json.Marshal(Report{Operations: out})
	if err != nil {
		return fmt.Errorf("encode the report: %w", err)
	}
	return atomicWrite(path, append(raw, '\n'))
}

func atomicWrite(path string, data []byte) error {
	f, err := os.CreateTemp(filepath.Dir(path), filepath.Base(path)+".tmp-*")
	if err != nil {
		return fmt.Errorf("write the report to %s: %w", path, err)
	}
	tmp := f.Name()
	_, werr := f.Write(data)
	cerr := f.Close()
	if werr != nil || cerr != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("write the report to %s: %w", path, errors.Join(werr, cerr))
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("write the report to %s: %w", path, err)
	}
	return nil
}

// AsContext is a convenience for an Apply that carries the identity on the request context: it
// returns r with ctx(r.Context()) as its context.
func AsContext(ctx func(context.Context) context.Context) func(*http.Request) *http.Request {
	return func(r *http.Request) *http.Request { return r.WithContext(ctx(r.Context())) }
}

// adminKeys0 is the set of "METHOD /path" keys of the admin-only operations.
func adminKeys0(c Contract) map[string]bool {
	m := map[string]bool{}
	for _, o := range c.Admin {
		m[o.MatchKey()] = true
	}
	return m
}

// waiverReasonProblem says what is wrong with a waiver's reason, or "" when it is acceptable: at
// least three words, naming the entity or a literal segment of the route's path.
func waiverReasonProblem(reason string, o Operation) string {
	if len(strings.Fields(reason)) < 3 {
		return "needs a reason of at least three words that is specific to the operation"
	}
	low := strings.ToLower(reason)
	if strings.Contains(low, strings.ToLower(o.Entity)) {
		return ""
	}
	for _, seg := range strings.Split(o.Path, "/") {
		seg = strings.ToLower(seg)
		if seg != "" && seg != "api" && !templateParam.MatchString(seg) && strings.Contains(low, seg) {
			return ""
		}
	}
	return "has a reason that does not name the entity or the route it is for (" + o.Entity + ", " + o.Path + ")"
}

// waiveReason is the reason row waives o for, matched as the platform matches operations.
func waiveReason(row Row, o Operation) (string, bool) {
	for key, reason := range row.Waive {
		if keyOf(key) == o.MatchKey() {
			return reason, true
		}
	}
	return "", false
}

// without drops the operations whose path starts with one of the prefixes.
func (c Contract) without(prefixes []string) Contract {
	if len(prefixes) == 0 {
		return c
	}
	keep := func(ops []Operation) []Operation {
		var out []Operation
		for _, o := range ops {
			if !slices.ContainsFunc(prefixes, func(p string) bool { return strings.HasPrefix(o.Path, p) }) {
				out = append(out, o)
			}
		}
		return out
	}
	return Contract{PerUser: keep(c.PerUser), Admin: keep(c.Admin)}
}

// only keeps the operations whose path starts with one of the prefixes; no prefix keeps all.
func (c Contract) only(prefixes []string) Contract {
	if len(prefixes) == 0 {
		return c
	}
	keep := func(ops []Operation) []Operation {
		var out []Operation
		for _, o := range ops {
			if slices.ContainsFunc(prefixes, func(p string) bool { return strings.HasPrefix(o.Path, p) }) {
				out = append(out, o)
			}
		}
		return out
	}
	return Contract{PerUser: keep(c.PerUser), Admin: keep(c.Admin)}
}
