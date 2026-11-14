package ownedscope

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

const nestedContract = `
openapi: 3.1.0
x-data-model:
  entities:
    List:
      ownership: per-user
      reads: each user reads only their own lists
paths:
  /api/lists:
    get:
      x-entity: List
    post:
      x-entity: List
  /api/lists/{id}:
    get:
      x-entity: List
  /api/lists/{id}/items/{itemId}:
    get:
      x-entity: List
`

// lists is a tiny API of lists with items under them. Lists are per owner. With looseItems the item
// route finds an item by its own id alone and never checks the list in the path, the defect the
// nested check looks for.
type lists struct {
	looseItems bool
	// One keyspace per caller, "who|id": the lists, and the list each item belongs to.
	listOwner map[string]bool
	itemList  map[string]string
}

func newLists(loose bool) *lists {
	return &lists{looseItems: loose, listOwner: map[string]bool{}, itemList: map[string]string{}}
}

func (l *lists) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/lists", func(w http.ResponseWriter, r *http.Request) {
		var in struct {
			ID     string `json:"id"`
			ItemID string `json:"itemId"`
		}
		_ = json.NewDecoder(r.Body).Decode(&in)
		who := r.Header.Get("X-Who")
		l.listOwner[who+"|"+in.ID] = true
		l.itemList[who+"|"+in.ItemID] = in.ID
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"id":"` + in.ID + `"}`))
	})
	mux.HandleFunc("GET /api/lists", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"items":[` + l.mine(r.Header.Get("X-Who")) + `]}`))
	})
	mux.HandleFunc("GET /api/lists/{id}", func(w http.ResponseWriter, r *http.Request) {
		if !l.listOwner[r.Header.Get("X-Who")+"|"+r.PathValue("id")] {
			http.NotFound(w, r)
			return
		}
		_, _ = w.Write([]byte(`{"id":"` + r.PathValue("id") + `"}`))
	})
	mux.HandleFunc("GET /api/lists/{id}/items/{itemId}", func(w http.ResponseWriter, r *http.Request) {
		who, item := r.Header.Get("X-Who"), r.PathValue("itemId")
		list, found := l.itemList[who+"|"+item]
		if !found || (!l.looseItems && list != r.PathValue("id")) {
			http.NotFound(w, r)
			return
		}
		_, _ = w.Write([]byte(`{"id":"` + item + `"}`))
	})
	return mux
}

func (l *lists) mine(who string) string {
	var out []string
	for key := range l.listOwner {
		if id, ok := strings.CutPrefix(key, who+"|"); ok {
			out = append(out, `{"id":"`+id+`"}`)
		}
	}
	return strings.Join(out, ",")
}

// listRow gives each identity ids of its own, as a Factory must for the swap to mean anything.
func listRow(sameIDs bool) Row {
	return Row{
		Entity: "List",
		Factory: func(t TB, as *Actor) Record {
			n := as.Identity.Name
			if sameIDs {
				n = "x"
			}
			body := map[string]any{"id": "list-" + n, "itemId": "item-" + n}
			if res := as.Do("POST", "/api/lists", body); res.Status != http.StatusCreated {
				t.Fatalf("cannot make a list: %d", res.Status)
			}
			return Record{ID: "list-" + n, Create: body, Params: map[string]string{"id": "list-" + n, "itemId": "item-" + n}}
		},
		Probes: []Probe{{Method: "GET", Path: "/api/lists/{id}/items/{itemId}"}},
	}
}

func nestedReport(t *testing.T, h http.Handler, row Row) (Report, []string) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "report.json")
	t.Setenv(ReportEnv, path)
	cfg := Config{Handler: h, First: who("alice"), Second: who("bob"), OpenAPI: writeContract(t, nestedContract), Rows: []Row{row}}
	ft := &fakeT{}
	errs := ft.run(func() { Run(ft, cfg) })
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("no report: %v", err)
	}
	var rep Report
	if err := json.Unmarshal(raw, &rep); err != nil {
		t.Fatal(err)
	}
	return rep, errs
}

func nestedOutcome(rep Report) (string, string) {
	for _, n := range rep.Nested {
		if n.Path == "/api/lists/{id}/items/{itemId}" {
			return n.Outcome, n.Reason
		}
	}
	return "", ""
}

func TestScopeHelperNestedRouteRefusedWhenScoped(t *testing.T) {
	rep, errs := nestedReport(t, newLists(false).Routes(), listRow(false))
	if len(errs) != 0 {
		t.Fatalf("a scoped handler must pass: %v", errs)
	}
	if got, why := nestedOutcome(rep); got != Refused {
		t.Fatalf("nested outcome = %q (%s), want refused", got, why)
	}
}

func TestScopeHelperNestedLeakIsAdvisoryOnly(t *testing.T) {
	rep, errs := nestedReport(t, newLists(true).Routes(), listRow(false))
	if len(errs) != 0 {
		t.Fatalf("a nested leak must not fail the test (advisory): %v", errs)
	}
	if got, _ := nestedOutcome(rep); got != Leaked {
		t.Fatalf("nested outcome = %q, want leaked", got)
	}
}

func TestScopeHelperNestedNotDrivenWithoutDistinctIDs(t *testing.T) {
	rep, errs := nestedReport(t, newLists(true).Routes(), listRow(true))
	if len(errs) != 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}
	if got, why := nestedOutcome(rep); got != NotDriven || !strings.Contains(why, "same") {
		t.Fatalf("nested outcome = %q (%s), want not-driven because the ids are the same", got, why)
	}
}

func TestScopeHelperNestedFactoryFailureAsSecondIsNotDriven(t *testing.T) {
	row := listRow(false)
	inner := row.Factory
	row.Factory = func(t TB, as *Actor) Record {
		if as.Identity.Name == "bob" {
			t.Fatalf("bob cannot make a list")
		}
		return inner(t, as)
	}
	rep, errs := nestedReport(t, newLists(false).Routes(), row)
	if len(errs) != 0 {
		t.Fatalf("a Factory failing for the second identity must not fail the test: %v", errs)
	}
	if got, why := nestedOutcome(rep); got != NotDriven || !strings.Contains(why, "bob cannot make a list") {
		t.Fatalf("nested outcome = %q (%s), want not-driven", got, why)
	}
}

func TestScopeHelperStampsEveryRow(t *testing.T) {
	rep, _ := nestedReport(t, newLists(false).Routes(), listRow(false))
	if len(rep.Operations) == 0 || len(rep.Nested) == 0 {
		t.Fatalf("empty report: %+v", rep)
	}
	src := SourceDigest()
	if src == "" || len(src) != 64 {
		t.Fatalf("source digest = %q", src)
	}
	check := func(method, path, outcome, reason string, p *Provenance) {
		t.Helper()
		if p == nil || p.Helper != HelperName || p.Kit != KitVersion || p.Source != src {
			t.Fatalf("%s %s: bad stamp %+v", method, path, p)
		}
		if p.Digest != rowDigest(KitVersion, src, method, path, outcome, reason) {
			t.Fatalf("%s %s: the row digest does not match its row", method, path)
		}
		if p.Digest == rowDigest(KitVersion, src, method, path, "refused", reason+" ") {
			t.Fatalf("the digest ignores the reason")
		}
	}
	for _, e := range rep.Operations {
		check(e.Method, e.Path, e.Outcome, e.Reason, e.Provenance)
	}
	for _, e := range rep.Nested {
		check(e.Method, e.Path, e.Outcome, e.Reason, e.Provenance)
	}
}

// The stamp's source digest is what the platform recomputes from the files on disk.
func TestScopeHelperSourceDigestMatchesTheFiles(t *testing.T) {
	var all []byte
	for _, name := range sourceFiles {
		raw, err := os.ReadFile(name)
		if err != nil {
			t.Fatal(err)
		}
		all = append(all, []byte(name+"\n"+strconv.Itoa(len(raw))+"\n")...)
		all = append(all, raw...)
		all = append(all, '\n')
	}
	sum := sha256.Sum256(all)
	if got := hex.EncodeToString(sum[:]); got != SourceDigest() {
		t.Fatalf("SourceDigest does not match the files on disk: %s vs %s", got, SourceDigest())
	}
	t.Logf("api-go %s source digest: %s", KitVersion, SourceDigest())
}
