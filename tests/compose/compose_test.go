// Package compose runs the real Albitor plugin manifest parser and composer over this repository's
// schema-2 plugins. It needs a checkout of albitor-ltd/albitor: scripts/compose-check.sh wires one
// in through a go.work file, so this module declares no dependency on it.
package compose

import (
	"encoding/json"
	"flag"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"albitor/core/authtemplate"
	"albitor/core/composer"
	"albitor/core/pluginmanifest"
)

var update = flag.Bool("update", false, "rewrite the golden file")

var tokens = authtemplate.Values{
	authtemplate.TokenProjectName: "plantrain",
	authtemplate.TokenRepo:        "acme/plantrain",
	composer.TokenDataLayer:       composer.DataLayerOnline,
	composer.TokenAppName:         "Plan Train",
}

func repoRoot(t *testing.T) string {
	t.Helper()
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	return root
}

// loadDir reads a plugin directory as the composer sees it: the validated manifest, the whole
// tree keyed by plugin-relative path (slot templates are manifest entries; a sidecar is refused).
func loadDir(t *testing.T, dir string) composer.Plugin {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(dir, ".claude-plugin", "plugin.json"))
	if err != nil {
		t.Fatal(err)
	}
	m, err := pluginmanifest.Parse(raw)
	if err != nil {
		t.Fatalf("%s: manifest rejected by the real parser: %v", dir, err)
	}
	tree := map[string][]byte{}
	err = filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		rel, _ := filepath.Rel(dir, path)
		b, err := os.ReadFile(path)
		tree[filepath.ToSlash(rel)] = b
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	// Slot templates are manifest files entries (mode slot-template, ADR-0090 amendment of
	// 2026-10-14, item 1), read by the composer from Tree exactly as the platform loads a pinned
	// version: nothing here supplies Plugin.SlotTemplates, so a sidecar the platform never reads
	// cannot make a test pass.
	for _, sidecar := range []string{"slot-templates.json", "slots/slots.json"} {
		if _, ok := tree[sidecar]; ok {
			t.Fatalf("%s: %s is a sidecar the platform never reads; declare the template in plugin.json", dir, sidecar)
		}
	}
	return composer.Plugin{Manifest: m, Tree: tree}
}

// inline builds a plugin from a manifest literal, parsed by the real parser.
func inline(t *testing.T, manifest string) composer.Plugin {
	t.Helper()
	m, err := pluginmanifest.Parse([]byte(manifest))
	if err != nil {
		t.Fatalf("stub manifest: %v", err)
	}
	return composer.Plugin{Manifest: m, Tree: map[string][]byte{}}
}

// stubs fills every exactly-one socket web-svelte does not, so a composition is complete.
func stubs(t *testing.T, root string, withAuth bool) []composer.Plugin {
	t.Helper()
	ps := []composer.Plugin{
		loadDir(t, filepath.Join(root, "tests", "compose", "stubs", "kit-stub")),
		inline(t, `{"schema":2,"name":"harness-stub","kind":"socket","provides":["test-harness@1"]}`),
		inline(t, `{"schema":2,"name":"deploy-stub","kind":"socket","provides":["deploy-target@1"]}`),
	}
	if withAuth {
		return append(ps, loadDir(t, filepath.Join(root, "tests", "compose", "stubs", "auth-stub")))
	}
	return append(ps, inline(t, `{"schema":2,"name":"auth-none-stub","kind":"socket","provides":["auth@1"]}`))
}

func compose(t *testing.T, withAuth bool) *composer.Result {
	t.Helper()
	root := repoRoot(t)
	plugins := append(stubs(t, root, withAuth), loadDir(t, filepath.Join(root, "plugins", "web-svelte")))
	res, err := composer.Compose(composer.Input{Plugins: plugins, Tokens: tokens})
	if err != nil {
		t.Fatalf("web-svelte does not compose with a stub kit: %v", err)
	}
	return res
}

// TestSchema2ManifestsParse runs the real parser over every plugin that declares schema 2. Older
// plugins have no schema key and keep ingesting as v1, which is not this test's business.
func TestSchema2ManifestsParse(t *testing.T) {
	root := repoRoot(t)
	manifests, err := filepath.Glob(filepath.Join(root, "plugins", "*", ".claude-plugin", "plugin.json"))
	if err != nil || len(manifests) == 0 {
		t.Fatalf("no plugin manifests found: %v", err)
	}
	seen := false
	for _, path := range manifests {
		raw, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		var probe map[string]json.RawMessage
		if err := json.Unmarshal(raw, &probe); err != nil {
			t.Fatalf("%s: %v", path, err)
		}
		if _, v2 := probe["schema"]; !v2 {
			continue
		}
		seen = true
		m, err := pluginmanifest.Parse(raw)
		if err != nil {
			t.Errorf("%s: %v", path, err)
			continue
		}
		if want := filepath.Base(filepath.Dir(filepath.Dir(path))); m.Name != want {
			t.Errorf("%s: name %q does not match its directory", path, m.Name)
		}
	}
	if !seen {
		t.Fatal("no schema-2 plugin found; web-svelte should be one")
	}
}

// TestWebSvelteComposesWithStubKit is the golden: the generated slot files, the manifest and the
// path list of the composed tree. Run with -update after an intended change.
func TestWebSvelteComposesWithStubKit(t *testing.T) {
	res := compose(t, true)

	var b strings.Builder
	for _, p := range res.Paths() {
		if p == "web/package-lock.json" || p == ".albitor/lock.json" {
			b.WriteString("=== " + p + " (not shown) ===\n")
			continue
		}
		if strings.HasSuffix(p, ".generated.ts") || p == "web/src/app.html" || p == "albitor.yaml" || p == ".albitor/shell.json" {
			b.WriteString("=== " + p + " ===\n" + string(res.Files[p]) + "\n")
			continue
		}
		b.WriteString("=== " + p + " ===\n")
	}
	got := b.String()

	golden := filepath.Join("testdata", "web-svelte.golden.txt")
	if *update {
		if err := os.WriteFile(golden, []byte(got), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	want, err := os.ReadFile(golden)
	if err != nil {
		t.Fatalf("%v (run with -update)", err)
	}
	if string(want) != got {
		t.Errorf("composed tree differs from %s (run with -update to accept)\n--- got ---\n%s", golden, got)
	}

	if out := os.Getenv("COMPOSE_OUT"); out != "" {
		for path, content := range res.Files {
			dst := filepath.Join(out, filepath.FromSlash(path))
			if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(dst, content, 0o644); err != nil {
				t.Fatal(err)
			}
		}
	}
}

// TestSlotFilesAreOpaqueLiterals pins the behaviour the shell depends on rather than its bytes.
func TestSlotFilesAreOpaqueLiterals(t *testing.T) {
	full := compose(t, true)
	providers := string(full.Files["web/src/lib/shell/providers.generated.ts"])
	for _, want := range []string{
		"const { DataProvider, StubProvider } = scope;",
		"const { stubGuard } = scope;",
		"checkProviders([DataProvider, StubProvider])",
		"checkGuards([stubGuard])",
	} {
		if !strings.Contains(providers, want) {
			t.Errorf("providers.generated.ts lacks %q:\n%s", want, providers)
		}
	}
	nav := string(full.Files["web/src/lib/shell/nav.generated.ts"])
	if !strings.Contains(nav, `[{"label":"Account","order":900,"group":"user"}]`) {
		t.Errorf("nav.generated.ts lacks the plugin nav entry:\n%s", nav)
	}
	if !strings.Contains(nav, "import.meta.glob('/src/routes/**/nav.ts', { eager: true })") {
		t.Errorf("nav.generated.ts must collect route nav.ts modules:\n%s", nav)
	}

	// With no other plugin filling the web slots, providers hold only web-svelte's own DataProvider,
	// every other placeholder renders empty, and the generated
	// code must still be well formed.
	bare := compose(t, false)
	providers = string(bare.Files["web/src/lib/shell/providers.generated.ts"])
	for _, want := range []string{"const { DataProvider } = scope;", "const {  } = scope;", "checkProviders([DataProvider])", "checkGuards([])"} {
		if !strings.Contains(providers, want) {
			t.Errorf("empty providers.generated.ts lacks %q:\n%s", want, providers)
		}
	}
	// The composer renders an unfilled web.nav as nothing, not as [], so the template must not
	// depend on the placeholder being an expression.
	if !strings.Contains(string(bare.Files["web/src/lib/shell/nav.generated.ts"]), "([] as NavMeta[]).concat()") {
		t.Errorf("empty nav.generated.ts should concat nothing:\n%s", bare.Files["web/src/lib/shell/nav.generated.ts"])
	}
}

func TestManagedAndExampleModes(t *testing.T) {
	res := compose(t, true)
	var lock composer.Lock
	if err := json.Unmarshal(res.Files[".albitor/lock.json"], &lock); err != nil {
		t.Fatal(err)
	}
	modes := map[string]string{}
	for _, f := range lock.Files {
		modes[f.Path] = f.Mode
	}
	managed := []string{"web/src/routes/+layout.svelte", "web/src/lib/shell/nav.generated.ts", "web/vite.config.ts"}
	example := []string{"web/src/routes/+error.svelte", "web/src/routes/+page.svelte", "web/package.json", "web/src/routes/nav.ts"}
	for _, p := range managed {
		if modes[p] != "managed" {
			t.Errorf("%s should be managed, got %q", p, modes[p])
		}
	}
	for _, p := range example {
		if modes[p] != "example" {
			t.Errorf("%s should be example, got %q", p, modes[p])
		}
	}
	// A route nav.ts and a +page.ts are separate files: the shell never asks +page.ts for nav.
	var pageTS []string
	for p := range res.Files {
		if strings.HasSuffix(p, "+page.ts") {
			pageTS = append(pageTS, p)
		}
	}
	sort.Strings(pageTS)
	if len(pageTS) != 0 {
		t.Errorf("the shell seeds no +page.ts: %v", pageTS)
	}
}

// The root layout hands the platform's app_name to the kit's AppShell as its title, so the header
// names the app (albitor-ltd/albitor#3426); without the token the composition is refused.
func TestRootLayoutTitlesTheShellWithTheAppName(t *testing.T) {
	layout := string(compose(t, false).Files["web/src/routes/+layout.svelte"])
	if !strings.Contains(layout, `title="Plan Train"`) || strings.Contains(layout, "{{") {
		t.Errorf("+layout.svelte does not pass app_name to AppShell as its title:\n%s", layout)
	}

	root := repoRoot(t)
	toks := authtemplate.Values{}
	for k, v := range tokens {
		if k != composer.TokenAppName {
			toks[k] = v
		}
	}
	plugins := append(stubs(t, root, false), loadDir(t, filepath.Join(root, "plugins", "web-svelte")))
	if _, err := composer.Compose(composer.Input{Plugins: plugins, Tokens: toks}); err == nil {
		t.Fatal("web-svelte composed with no app_name token")
	}
}
