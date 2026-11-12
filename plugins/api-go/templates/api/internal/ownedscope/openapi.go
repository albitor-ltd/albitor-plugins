package ownedscope

import (
	"fmt"
	"os"
	"regexp"
	"sort"
	"strings"
)

// Operation is one method on one path of the contract that names a per-user entity.
type Operation struct {
	Method string // upper case
	Path   string // the template as the contract writes it, "/api/armies/{id}"
	Entity string // the x-entity it names
}

// Key is how a waiver names the operation: "GET /api/armies/{id}".
func (o Operation) Key() string { return o.Method + " " + o.Path }

// shape is what an operation does to a record, read from its method and path.
type shape int

const (
	shapeOther shape = iota
	shapeCreate
	shapeList
	shapeRead
	shapeUpdate
	shapeDelete
)

var paramPattern = regexp.MustCompile(`^\{([^{}/]+)\}$`)

// shapeOf classifies an operation: a collection is a path whose last segment is a name, an item a
// path whose last segment is one parameter. Anything else (an action, a nested route) is other.
func shapeOf(o Operation) shape {
	segs := strings.Split(strings.Trim(o.Path, "/"), "/")
	last := segs[len(segs)-1]
	item := paramPattern.MatchString(last)
	switch {
	case !item && o.Method == "POST":
		return shapeCreate
	case !item && o.Method == "GET":
		return shapeList
	case item && o.Method == "GET":
		return shapeRead
	case item && (o.Method == "PUT" || o.Method == "PATCH"):
		return shapeUpdate
	case item && o.Method == "DELETE":
		return shapeDelete
	}
	return shapeOther
}

// itemParam is the name of the path's last parameter ("id" in /api/armies/{id}), or "".
func itemParam(path string) string {
	segs := strings.Split(strings.Trim(path, "/"), "/")
	if m := paramPattern.FindStringSubmatch(segs[len(segs)-1]); m != nil {
		return m[1]
	}
	return ""
}

var httpMethods = map[string]bool{"get": true, "put": true, "post": true, "delete": true, "patch": true}

var (
	keyLine      = regexp.MustCompile(`^(\s*)("[^"]*"|'[^']*'|[^\s#'"][^:]*?):(?:\s+(.*))?$`)
	flowEntity   = regexp.MustCompile(`x-entity:\s*["']?([A-Za-z0-9_.-]+)["']?`)
	flowAdmin    = regexp.MustCompile(`x-admin-only:\s*true\b`)
	adminWord    = regexp.MustCompile(`\badmin\b`)
	commentTail  = regexp.MustCompile(`\s+#.*$`)
	blockScalar  = regexp.MustCompile(`^[|>][-+0-9]*$`)
	perUserValue = "per-user"
)

// Contract is what the test reads from openapi.yaml.
type Contract struct {
	// PerUser are the operations that name a per-user entity and are not admin-only.
	PerUser []Operation
	// Admin are the operations the contract declares admin-only (`x-admin-only: true`, or
	// `x-required-groups` / `x-groups` naming admin), whatever entity they name.
	Admin []Operation
}

// ReadContract reads the contract at path. The module is stdlib only, so this is a small reader
// for the block-style YAML the contract is written in, not a general parser: it follows keys by
// indentation and understands the shapes the platform's own reader uses:
//
//	x-data-model:
//	  entities:
//	    Army:
//	      ownership: per-user
//	paths:
//	  /api/armies/{id}:
//	    get:
//	      x-entity: Army
//	  /api/admin/armies:
//	    get:
//	      x-admin-only: true
//
// An operation written in flow style on one line (`get: { x-entity: Army }`) is read too.
func ReadContract(path string) (Contract, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return Contract{}, fmt.Errorf("read the contract: %w", err)
	}
	return readContract(string(raw)), nil
}

func readContract(doc string) Contract {
	type frame struct {
		indent int
		key    string
	}
	type info struct {
		method, path, entity string
		admin                bool
	}
	var stack []frame
	ownership := map[string]string{}
	infos := map[string]*info{}
	var order []string
	get := func(path, method string) *info {
		k := strings.ToUpper(method) + " " + path
		if infos[k] == nil {
			infos[k] = &info{method: strings.ToUpper(method), path: path}
			order = append(order, k)
		}
		return infos[k]
	}
	isGroups := func(k string) bool { return k == "x-required-groups" || k == "x-groups" }
	skipBelow := -1 // inside a block scalar: ignore lines indented deeper than this

	for _, line := range strings.Split(strings.ReplaceAll(doc, "\r\n", "\n"), "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		indent := len(line) - len(strings.TrimLeft(line, " "))
		if skipBelow >= 0 {
			if indent > skipBelow {
				continue
			}
			skipBelow = -1
		}
		if strings.HasPrefix(trimmed, "- ") || trimmed == "-" {
			// A list under x-groups / x-required-groups names the groups that may call the route.
			if n := len(stack); n == 4 && stack[0].key == "paths" && httpMethods[stack[2].key] &&
				isGroups(stack[3].key) && indent >= stack[3].indent && adminWord.MatchString(trimmed) {
				get(stack[1].key, stack[2].key).admin = true
			}
			continue
		}
		m := keyLine.FindStringSubmatch(commentTail.ReplaceAllString(strings.TrimRight(line, " "), ""))
		if m == nil {
			continue
		}
		key := strings.Trim(m[2], `"'`)
		value := strings.TrimSpace(m[3])
		for len(stack) > 0 && stack[len(stack)-1].indent >= indent {
			stack = stack[:len(stack)-1]
		}
		if blockScalar.MatchString(value) {
			skipBelow = indent
			stack = append(stack, frame{indent, key})
			continue
		}
		at := func(i int) string {
			if i < len(stack) {
				return stack[i].key
			}
			return ""
		}
		switch {
		case len(stack) == 3 && at(0) == "x-data-model" && at(1) == "entities" && key == "ownership":
			ownership[at(2)] = strings.Trim(value, `"'`)
		case len(stack) == 2 && at(0) == "paths" && httpMethods[key]:
			in := get(at(1), key)
			if strings.HasPrefix(value, "{") {
				if m := flowEntity.FindStringSubmatch(value); m != nil {
					in.entity = m[1]
				}
				if flowAdmin.MatchString(value) || (strings.Contains(value, "groups") && adminWord.MatchString(value)) {
					in.admin = true
				}
			}
		case len(stack) == 3 && at(0) == "paths" && httpMethods[at(2)]:
			in := get(at(1), at(2))
			switch {
			case key == "x-entity":
				in.entity = strings.Trim(value, `"'`)
			case key == "x-admin-only" && strings.EqualFold(value, "true"):
				in.admin = true
			case isGroups(key) && adminWord.MatchString(value):
				in.admin = true
			}
		}
		stack = append(stack, frame{indent, key})
	}

	var c Contract
	for _, k := range order {
		in := infos[k]
		o := Operation{Method: in.method, Path: in.path, Entity: in.entity}
		switch {
		case in.admin:
			c.Admin = append(c.Admin, o)
		case in.entity != "" && ownership[in.entity] == perUserValue:
			c.PerUser = append(c.PerUser, o)
		}
	}
	for _, ops := range [][]Operation{c.PerUser, c.Admin} {
		ops := ops
		sort.Slice(ops, func(i, j int) bool {
			if ops[i].Path != ops[j].Path {
				return ops[i].Path < ops[j].Path
			}
			return ops[i].Method < ops[j].Method
		})
	}
	return c
}
