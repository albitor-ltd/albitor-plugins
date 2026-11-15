package ownedscope

import (
	"fmt"
	"net/http"
	"strings"
)

// NestedEntry is one nested-route check of the report. The nested checks are ADVISORY: they never
// fail the test, and the platform records them as evidence only.
//
// A nested route is one under a record that takes more than one path parameter
// (/api/lists/{listId}/items/{itemId}). The second identity makes a record of its own (the same
// Factory, run as them), then calls the route with the FIRST identity's parent id and its own child
// id, and must be refused (404 or 403). The first identity's record must be unchanged afterwards.
// Outcomes: "refused", "leaked", or "not-driven" (with the reason: no Probe, no Params for the
// second identity's record, the Factory failed as the second identity). The bodies come from the
// row's Probe and the records from the app's own Factory: the helper generates neither.
type NestedEntry struct {
	Method     string      `json:"method"`
	Path       string      `json:"path"`
	Outcome    string      `json:"outcome"`
	Reason     string      `json:"reason"`
	Provenance *Provenance `json:"provenance,omitempty"`
}

// NotDriven is the nested outcome of a route the helper could not drive.
const NotDriven = "not-driven"

// shadowT runs a Factory as the second identity without letting it fail the test: the nested checks
// are advisory, so a Factory that cannot make a second record is a "not-driven" row, not an error.
type shadowT struct {
	TB
	failed string
}

type shadowAbort struct{}

func (s *shadowT) Errorf(format string, args ...any) {
	if s.failed == "" {
		s.failed = fmt.Sprintf(format, args...)
	}
}

func (s *shadowT) Fatalf(format string, args ...any) {
	s.Errorf(format, args...)
	panic(shadowAbort{})
}

// shadowRecord runs the row's Factory as the second identity. ok is false, with why, when it failed.
func shadowRecord(t TB, row Row, second *Actor) (rec Record, ok bool, why string) {
	st := &shadowT{TB: t}
	defer func() {
		if r := recover(); r != nil {
			if _, isAbort := r.(shadowAbort); !isAbort {
				panic(r)
			}
			ok, why = false, st.failed
		}
	}()
	rec = row.Factory(st, second)
	if st.failed != "" {
		return rec, false, st.failed
	}
	if rec.ID == "" {
		return rec, false, "the Factory returned no ID"
	}
	return rec, true, ""
}

// restParams are the path parameters after the collection's path: the first is the parent record's.
func restParams(o Operation) []string {
	rest, ok := strings.CutPrefix(strings.TrimRight(o.Path, "/"), strings.TrimRight(o.Collection, "/"))
	if !ok {
		return nil
	}
	return pathParams(rest)
}

// fillWith fills the path's parameters from params; the parent's falls back to id.
func fillWith(path string, params map[string]string, id, parent string) string {
	for _, name := range pathParams(path) {
		v, ok := params[name]
		if !ok && name == parent {
			v = id
		}
		path = strings.ReplaceAll(path, "{"+name+"}", v)
	}
	return path
}

// driveNested runs the nested checks of one entity over its nested operations (Contract.Nested) and hands each result
// to emit. A nested route sits under a record and takes the parent's id and at least one more.
// A GET or DELETE needs no Probe; a route with a body needs one.
// It never fails the test. intact reads the first identity's record back.
func driveNested(t TB, row Row, nested []Operation, rec Record, intact func() (bool, string), second *Actor, emit func(NestedEntry)) {
	ops := nested
	if len(ops) == 0 {
		return
	}
	mine, made, why := shadowRecord(t, row, second)
	for _, o := range ops {
		out := func(outcome, reason string) {
			emit(NestedEntry{Method: o.Method, Path: o.Path, Outcome: outcome, Reason: reason})
		}
		var probe *Probe
		for i := range row.Probes {
			if keyOf(row.Probes[i].Method+" "+row.Probes[i].Path) == o.MatchKey() {
				probe = &row.Probes[i]
			}
		}
		bodyless := o.Method == http.MethodGet || o.Method == http.MethodDelete
		if probe == nil && !bodyless {
			out(NotDriven, "the table has no Probe, and so no body, for this route")
			continue
		}
		if !made {
			out(NotDriven, "the Factory could not make a record as the second identity: "+why)
			continue
		}
		parent := restParams(o)[0]
		p, missing := o.Path, ""
		for _, name := range pathParams(o.Path) {
			v, has := mine.Params[name]
			if name == parent {
				// The swap: the first identity's parent id.
				if v, has = rec.Params[name]; !has {
					v, has = rec.ID, true
				}
			}
			if !has {
				missing = name
				break
			}
			p = strings.ReplaceAll(p, "{"+name+"}", v)
		}
		if missing != "" {
			out(NotDriven, "the second identity's record has no Params value for {"+missing+"}")
			continue
		}
		if own := fillWith(o.Path, mine.Params, mine.ID, parent); own == p {
			out(NotDriven, "the swap changes nothing: the second identity's record has the same ids as the first identity's, so give each identity ids of its own in the Factory")
			continue
		}
		var body any
		if probe != nil {
			body = probe.Body
		}
		res := second.Do(o.Method, p, body)
		if res.Status != http.StatusNotFound && res.Status != http.StatusForbidden {
			out(Leaked, fmt.Sprintf("the second identity called it with the first identity's parent id and its own child id and was answered %d, not a 404 or 403", res.Status))
			continue
		}
		if ok, why := intact(); !ok {
			out(Leaked, fmt.Sprintf("answered %d, but %s", res.Status, why))
			continue
		}
		out(Refused, fmt.Sprintf("the second identity called it with the first identity's parent id and its own child id and was answered %d; the first identity's record is intact", res.Status))
	}
}
