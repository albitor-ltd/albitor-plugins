package app

import (
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// TestFeaturesDoNotImportEachOther holds the layout: every package registry.go imports is a
// feature, and a feature imports no sibling feature and not this package. A feature that needs
// another's data reads it through a shared domain or store package that neither feature owns.
// Two features that import each other do not compile (an import cycle), and one that imports
// app cannot be listed in registry.go at all.
func TestFeaturesDoNotImportEachOther(t *testing.T) {
	module := moduleName(t)
	prefix := module + "/internal/"
	features := importsOf(t, "registry.go", prefix)
	if len(features) == 0 {
		return
	}
	for name := range features {
		dir := filepath.Join("..", name)
		files, err := filepath.Glob(filepath.Join(dir, "*.go"))
		if err != nil || len(files) == 0 {
			t.Errorf("registry.go imports %s, which has no Go files at %s", name, dir)
			continue
		}
		for _, f := range files {
			for imp := range importsOf(t, f, prefix) {
				if imp == "app" || (imp != name && features[imp]) {
					t.Errorf("%s imports %s%s: a feature must not import app or a sibling feature; go through a shared domain or store package", f, prefix, imp)
				}
			}
		}
	}
}

// importsOf returns the first path segment under prefix of every import in the Go file.
func importsOf(t *testing.T, file, prefix string) map[string]bool {
	t.Helper()
	parsed, err := parser.ParseFile(token.NewFileSet(), file, nil, parser.ImportsOnly)
	if err != nil {
		t.Fatalf("parse %s: %v", file, err)
	}
	out := map[string]bool{}
	for _, spec := range parsed.Imports {
		path, err := strconv.Unquote(spec.Path.Value)
		if err != nil {
			continue
		}
		if rest, ok := strings.CutPrefix(path, prefix); ok {
			out[strings.SplitN(rest, "/", 2)[0]] = true
		}
	}
	return out
}

func moduleName(t *testing.T) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("..", "..", "go.mod"))
	if err != nil {
		t.Fatalf("read go.mod: %v", err)
	}
	for _, line := range strings.Split(string(b), "\n") {
		if m, ok := strings.CutPrefix(strings.TrimSpace(line), "module "); ok {
			return strings.TrimSpace(m)
		}
	}
	t.Fatal("go.mod has no module line")
	return ""
}
