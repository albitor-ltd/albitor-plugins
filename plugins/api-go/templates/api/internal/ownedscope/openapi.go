package ownedscope

import (
	"fmt"
	"net/url"
	"os"
	"regexp"
	"slices"
	"sort"
	"strings"
)

// Operation is one operation of the contract the test must cover.
type Operation struct {
	Method string // upper case
	// Path is the request path as a client sends it: the template with the first server's base
	// path in front when the template lacks it ("/api/armies/{id}").
	Path string
	// Entity names the group the operation belongs to: the x-entity of its collection's create, or
	// the collection's path when the create names none. A Row's Entity is this name.
	Entity string
	// Collection is the path of the collection's create (POST with no path parameter). Empty for
	// an admin-only operation that belongs to no collection.
	Collection string
	// Admin is true when the contract declares the operation admin-only.
	Admin bool
}

// Key is how a waiver names the operation: "GET /api/armies/{id}".
func (o Operation) Key() string { return o.Method + " " + o.Path }

// MatchKey is how the platform compares an operation with a report: the method in upper case, the
// path without a trailing slash, the "/api" base and the parameters' names left out.
func (o Operation) MatchKey() string { return matchKey(o.Method, o.Path) }

// matchKey is the platform's operationMatchKey, written the same way: the report and the contract
// are compared by it, so a path that differs only by "/api", a trailing slash or a parameter's name
// is the same operation. Keep it identical to the platform's.
func matchKey(method, path string) string {
	path = strings.TrimSpace(path)
	if i := strings.IndexAny(path, "?#"); i >= 0 {
		path = path[:i]
	}
	var b strings.Builder
	for i := 0; i < len(path); i++ {
		if path[i] != '{' {
			b.WriteByte(path[i])
			continue
		}
		j := strings.IndexByte(path[i:], '}')
		if j < 0 {
			b.WriteString(path[i:])
			break
		}
		b.WriteString("{}")
		i += j
	}
	p := strings.TrimRight(b.String(), "/")
	if p == "/api" {
		p = ""
	}
	p = strings.TrimPrefix(p, "/api")
	return strings.ToUpper(strings.TrimSpace(method)) + " " + p
}

// keyOf reads a waiver's or a probe's "METHOD /path" as the platform compares it.
func keyOf(s string) string {
	method, path, _ := strings.Cut(strings.TrimSpace(s), " ")
	return matchKey(method, path)
}

// shape is what an operation does to a record, read from its place under its collection.
type shape int

const (
	shapeOther shape = iota
	shapeCreate
	shapeList
	shapeRead
	shapeUpdate
	shapeDelete
)

var templateParam = regexp.MustCompile(`\{[^{}/]*\}`)

// pathParams lists the {name} parameters of a path template in order.
func pathParams(p string) []string {
	var out []string
	for _, m := range templateParam.FindAllString(p, -1) {
		out = append(out, strings.Trim(m, "{}"))
	}
	return out
}

// shapeOf classifies an operation by where it sits under its collection: the collection's own path
// is the create (POST) and the list (GET); "<collection>/{id}" is the record (GET, PUT or PATCH,
// DELETE); everything else under the collection is an action on a record
// (POST /things/{id}/archive) and is driven by a Probe, never mistaken for a create.
func shapeOf(o Operation) shape {
	if o.Admin || o.Collection == "" {
		return shapeOther
	}
	p, c := strings.TrimRight(o.Path, "/"), strings.TrimRight(o.Collection, "/")
	if p == c {
		switch o.Method {
		case "POST":
			return shapeCreate
		case "GET":
			return shapeList
		}
		return shapeOther
	}
	rest, ok := strings.CutPrefix(p, c+"/")
	if !ok || strings.Contains(rest, "/") || len(pathParams(rest)) != 1 || !strings.HasPrefix(rest, "{") || !strings.HasSuffix(rest, "}") {
		return shapeOther
	}
	switch o.Method {
	case "GET":
		return shapeRead
	case "PUT", "PATCH":
		return shapeUpdate
	case "DELETE":
		return shapeDelete
	}
	return shapeOther
}

// itemParam is the name of the one parameter of an operation under a record ("id" in
// /api/armies/{id} and in /api/armies/{id}/archive), or "".
func itemParam(o Operation) string {
	if names := pathParams(o.Path); len(names) == 1 && o.Collection != "" && strings.TrimRight(o.Path, "/") != strings.TrimRight(o.Collection, "/") {
		return names[0]
	}
	return ""
}

// Contract is what the test reads from openapi.yaml: the operations it must cover, by the
// platform's rule.
type Contract struct {
	// PerUser are the operations of every collection (a POST with no path parameter that is
	// neither public nor admin-only, with records under it) whose create does not name an entity
	// the data model declares shared: the create, the list and the routes under one record.
	PerUser []Operation
	// Admin are the other operations the contract declares admin-only (`x-admin-only: true`,
	// `x-required-groups` / `x-groups` naming admin in any case, or a security requirement whose
	// every alternative asks for an admin scope), whatever entity they name.
	Admin []Operation
}

// Required are the match keys of every operation the test must cover, sorted.
func (c Contract) Required() []string {
	var out []string
	for _, o := range c.PerUser {
		out = append(out, o.MatchKey())
	}
	for _, o := range c.Admin {
		out = append(out, o.MatchKey())
	}
	slices.Sort(out)
	return out
}

// ReadContract reads the contract at path. It fails, so the test fails, on a document it cannot
// read, rather than reading less of it: see parseYAML for what it reads.
func ReadContract(path string) (Contract, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return Contract{}, fmt.Errorf("read the contract: %w", err)
	}
	c, err := readContract(string(raw))
	if err != nil {
		return Contract{}, fmt.Errorf("read the contract %s: %w", path, err)
	}
	return c, nil
}

// rawOp is an operation before the collections are found.
type rawOp struct {
	Operation
	params    []string
	public    bool
	ownership string // the data model's ownership of the entity it names, "" when none or unknown
	entity    string
}

var contractMethods = map[string]bool{"get": true, "put": true, "post": true, "delete": true, "options": true, "head": true, "patch": true, "trace": true}

var (
	alwaysPublicPaths    = map[string]bool{"/": true, "/login": true, "/signin": true, "/signup": true, "/register": true, "/logout": true, "/health": true, "/healthz": true, "/favicon.ico": true, "/robots.txt": true, "/sitemap.xml": true, "/index.html": true}
	alwaysPublicPrefixes = []string{"/static/", "/assets/", "/public/", "/_app/", "/.well-known/", "/css/", "/js/", "/img/", "/images/", "/fonts/"}
)

// readContract computes the required operations the way the platform does (its probeOperations,
// perUserOperations and requiredOperations): keep it in step. The fixtures in testdata hold both
// sides to the same answers.
func readContract(doc string) (Contract, error) {
	root, err := parseYAML(doc)
	if err != nil {
		return Contract{}, err
	}
	top, _ := root.(map[string]any)
	paths, _ := top["paths"].(map[string]any)
	if len(paths) == 0 {
		return Contract{}, fmt.Errorf("the contract defines no paths")
	}
	base := ""
	servers, _ := top["servers"].([]any)
	for _, s := range servers {
		sm, _ := s.(map[string]any)
		raw, _ := sm["url"].(string)
		u, err := url.Parse(strings.TrimSpace(raw))
		if err != nil {
			continue
		}
		if b := strings.TrimRight(u.Path, "/"); strings.HasPrefix(b, "/") {
			base = b // the first base path the platform reads: its servers[0]
			break
		}
	}
	model := ownershipOf(top)
	docSecurity, docHasSecurity := top["security"].([]any)

	var all []rawOp
	for template, rawItem := range paths {
		if _, ok := rawItem.(map[string]any); !ok && rawItem != nil {
			return Contract{}, fmt.Errorf("the path item %q is not a mapping", template)
		}
		item, _ := rawItem.(map[string]any)
		full := template
		if base != "" && full != base && !strings.HasPrefix(full, base+"/") {
			full = base + "/" + strings.TrimLeft(full, "/")
		}
		for key, rawOperation := range item {
			method := strings.ToLower(key)
			if !contractMethods[method] {
				continue
			}
			op, _ := rawOperation.(map[string]any)
			all = append(all, readOp(strings.ToUpper(method), template, full, op, item, docSecurity, docHasSecurity, model))
		}
	}
	sort.Slice(all, func(i, j int) bool {
		if all[i].Path != all[j].Path {
			return all[i].Path < all[j].Path
		}
		return all[i].Method < all[j].Method
	})

	// probeOperations: not the platform's own routes, not an always-public path, no HEAD, OPTIONS or TRACE.
	var ops []rawOp
	for _, o := range all {
		if platformRoute(o.Path) || alwaysPublic(o.Path) || o.Method == "HEAD" || o.Method == "OPTIONS" || o.Method == "TRACE" {
			continue
		}
		ops = append(ops, o)
	}

	var c Contract
	seen := map[string]bool{}
	add := func(o rawOp, group, collection string) {
		o.Entity, o.Collection = group, collection
		if k := o.MatchKey(); !seen[k] {
			seen[k] = true
			c.PerUser = append(c.PerUser, o.Operation)
		}
	}
	for _, create := range ops {
		if create.Method != "POST" || len(create.params) != 0 || create.public || create.Admin {
			continue
		}
		if create.ownership == "shared" {
			continue // a collection of shared records has no owner to scope to
		}
		var items []rawOp
		var list *rawOp
		for i, o := range ops {
			switch {
			case o.Method == "GET" && o.Path == create.Path && len(o.params) == 0:
				list = &ops[i]
			case len(o.params) == 1 && strings.HasPrefix(o.Path, create.Path+"/{"+o.params[0]+"}") && !o.Admin:
				items = append(items, o)
			}
		}
		if len(items) == 0 && list == nil {
			continue
		}
		group := create.entity
		if group == "" {
			group = create.Path
		}
		add(create, group, create.Path)
		if list != nil {
			add(*list, group, create.Path)
		}
		for _, o := range items {
			add(o, group, create.Path)
		}
	}
	for _, o := range ops {
		if k := o.MatchKey(); o.Admin && !seen[k] {
			seen[k] = true
			o.Operation.Collection, o.Operation.Entity = "", o.entity
			c.Admin = append(c.Admin, o.Operation)
		}
	}
	return c, nil
}

func platformRoute(path string) bool {
	p := "/" + strings.Join(strings.FieldsFunc(path, func(r rune) bool { return r == '/' }), "/")
	return p == "/api/health" || strings.HasPrefix(p+"/", "/api/_local/")
}

func alwaysPublic(path string) bool {
	p := strings.TrimSpace(path)
	if i := strings.IndexAny(p, "?#"); i >= 0 {
		p = p[:i]
	}
	if p == "" {
		return true
	}
	lower := strings.ToLower(p)
	if alwaysPublicPaths[lower] {
		return true
	}
	for _, prefix := range alwaysPublicPrefixes {
		if strings.HasPrefix(lower, prefix) {
			return true
		}
	}
	return false
}

// ownershipOf reads the data model's ownership of each entity: "per-user" or "shared", and nothing
// for an entity whose ownership is neither.
func ownershipOf(top map[string]any) map[string]string {
	out := map[string]string{}
	model, _ := top["x-data-model"].(map[string]any)
	entities, _ := model["entities"].(map[string]any)
	for name, raw := range entities {
		e, _ := raw.(map[string]any)
		own, _ := e["ownership"].(string)
		if own = strings.TrimSpace(own); own == "per-user" || own == "shared" {
			out[name] = own
		}
	}
	return out
}

func readOp(method, template, full string, op, item map[string]any, docSecurity []any, docHasSecurity bool, model map[string]string) rawOp {
	r := rawOp{Operation: Operation{Method: method, Path: full}, params: pathParams(template)}
	security, hasSecurity := op["security"].([]any)
	if !hasSecurity {
		security, hasSecurity = docSecurity, docHasSecurity
	}
	truthy := func(key string) bool {
		a, _ := op[key].(bool)
		b, _ := item[key].(bool)
		return a || b
	}
	r.public = truthy("x-public") || (hasSecurity && len(security) == 0)
	r.Admin = !r.public && (truthy("x-admin-only") ||
		namesAdmin(op["x-required-groups"]) || namesAdmin(op["x-groups"]) ||
		namesAdmin(item["x-required-groups"]) || namesAdmin(item["x-groups"]) ||
		securityNamesAdmin(security))
	name, _ := op["x-entity"].(string)
	if name == "" {
		name, _ = item["x-entity"].(string)
	}
	r.entity = strings.TrimSpace(name)
	r.ownership = model[r.entity]
	return r
}

// namesAdmin reports whether a list of groups holds "admin" (any case).
func namesAdmin(v any) bool {
	list, _ := v.([]any)
	return slices.ContainsFunc(list, func(e any) bool { s, _ := e.(string); return strings.EqualFold(s, "admin") })
}

// securityNamesAdmin reports whether a security requirement list asks for an admin scope: a scope
// that is "admin" or ends in ":admin" or "/admin". Every alternative must, since any one of them
// lets a normal user in.
func securityNamesAdmin(security []any) bool {
	if len(security) == 0 {
		return false
	}
	for _, alt := range security {
		req, _ := alt.(map[string]any)
		found := false
		for _, scopes := range req {
			list, _ := scopes.([]any)
			for _, s := range list {
				name, _ := s.(string)
				name = strings.ToLower(name)
				found = found || name == "admin" || strings.HasSuffix(name, ":admin") || strings.HasSuffix(name, "/admin")
			}
		}
		if !found {
			return false
		}
	}
	return true
}
