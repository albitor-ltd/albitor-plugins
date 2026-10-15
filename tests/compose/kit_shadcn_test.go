package compose

import (
	"context"
	"encoding/json"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"

	"albitor/core"
	"albitor/core/certification"
	"albitor/core/composer"
	"albitor/core/governance"
)

// kitProfile is the smallest legal composition around a candidate component-kit: the real
// web-svelte plus stubs for every other exactly-one socket.
func kitProfile(kit string) certification.Profile {
	return certification.Profile{Name: "kit-check", Members: []certification.Member{
		{Socket: core.SocketWebFramework, Plugin: "web-svelte"},
		{Socket: core.SocketComponentKit, Plugin: kit},
		{Socket: core.SocketAuth, Plugin: "auth-none-stub"},
		{Socket: core.SocketDeployTarget, Plugin: "deploy-stub"},
		{Socket: core.SocketTestHarness, Plugin: "harness-stub"},
	}}
}

type catalogue map[string]composer.Plugin

func (c catalogue) Plugin(n string) (composer.Plugin, bool) { p, ok := c[n]; return p, ok }

func kitPeers(t *testing.T) catalogue {
	t.Helper()
	root := repoRoot(t)
	c := catalogue{"web-svelte": loadDir(t, filepath.Join(root, "plugins", "web-svelte"))}
	for _, m := range []string{
		`{"schema":2,"name":"auth-none-stub","kind":"socket","provides":["auth@1"]}`,
		`{"schema":2,"name":"harness-stub","kind":"socket","provides":["test-harness@1"]}`,
		`{"schema":2,"name":"deploy-stub","kind":"socket","provides":["deploy-target@1"]}`,
	} {
		p := inline(t, m)
		c[p.Manifest.Name] = p
	}
	return c
}

// snapshot is the plugin directory as the platform ingests it: every file, keyed by
// plugin-relative path.
func snapshot(t *testing.T, dir string) map[string][]byte {
	t.Helper()
	files := map[string][]byte{}
	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		rel, _ := filepath.Rel(dir, path)
		b, err := os.ReadFile(path)
		files[filepath.ToSlash(rel)] = b
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	return files
}

func kitDir(t *testing.T, kit string) string { return filepath.Join(repoRoot(t), "plugins", kit) }

func composeKit(t *testing.T, kit string) *composer.Result {
	t.Helper()
	peers := kitPeers(t)
	plugins := []composer.Plugin{loadDir(t, kitDir(t, kit))}
	for _, n := range []string{"web-svelte", "auth-none-stub", "harness-stub", "deploy-stub"} {
		plugins = append(plugins, peers[n])
	}
	res, err := composer.Compose(composer.Input{Plugins: plugins, Tokens: tokens})
	if err != nil {
		t.Fatalf("%s does not compose with web-svelte: %v", kit, composer.Problems(err))
	}
	return res
}

// TestKitShadcnPassesComponentKitConformance runs the platform's own certification harness over
// the plugin's snapshot: the manifest through the real parser, the component-kit@1 conformance
// suite, then a compose and the floor gate against web-svelte. No Builder is set, so the best
// verdict is static_only; scripts/compose-check.sh --build builds, tests and audits the composed web.
func TestKitShadcnPassesComponentKitConformance(t *testing.T) { kitPassesConformance(t, "kit-shadcn") }

func kitPassesConformance(t *testing.T, kit string) {
	t.Helper()
	h := &certification.Harness{
		Tokens:   tokens,
		Profiles: []certification.Profile{kitProfile(kit)},
		Peers:    kitPeers(t),
		Suites:   certification.DefaultSuites(),
	}
	cert, err := h.Certify(context.Background(), snapshot(t, kitDir(t, kit)))
	if err != nil {
		t.Fatal(err)
	}
	if cert.Verdict == certification.VerdictFailed {
		t.Fatalf("certification failed: %v", cert.Failures)
	}
	if len(cert.Conformance) == 0 {
		t.Fatal("no conformance checks ran for component-kit@1")
	}
	for _, c := range cert.Conformance {
		if !c.Passed {
			t.Errorf("conformance %s: %s", c.ID, c.Detail)
		}
	}
	if len(cert.Profiles) != 1 || !cert.Profiles[0].Compatible || len(cert.Profiles[0].ComposeErrors) != 0 {
		t.Fatalf("profile result: %+v", cert.Profiles)
	}
}

// TestKitShadcnFailsClosed pins that the harness can fail this plugin: a declared file that is
// not shipped must be a failed verdict, so the test above cannot pass vacuously.
func TestKitShadcnFailsClosed(t *testing.T) { kitFailsClosed(t, "kit-shadcn") }

func kitFailsClosed(t *testing.T, kit string) {
	t.Helper()
	files := snapshot(t, kitDir(t, kit))
	delete(files, "templates/web/src/lib/kit/AppShell.svelte")
	h := &certification.Harness{
		Tokens:   tokens,
		Profiles: []certification.Profile{kitProfile(kit)},
		Peers:    kitPeers(t),
		Suites:   certification.DefaultSuites(),
	}
	cert, err := h.Certify(context.Background(), files)
	if err != nil {
		t.Fatal(err)
	}
	if cert.Verdict != certification.VerdictFailed {
		t.Fatalf("a kit missing its AppShell must fail, got %s", cert.Verdict)
	}
}

// TestKitShadcnComposesWithWebSvelte is the golden for the composed kit files, and writes the
// whole composed tree to COMPOSE_KIT_OUT for the build script.
func TestKitShadcnComposesWithWebSvelte(t *testing.T) {
	kitComposes(t, "kit-shadcn", "COMPOSE_KIT_OUT")
}

func kitComposes(t *testing.T, kit, outEnv string) {
	t.Helper()
	res := composeKit(t, kit)
	var b strings.Builder
	for _, p := range res.Paths() {
		b.WriteString("=== " + p + " ===\n")
	}
	golden := filepath.Join("testdata", kit+".golden.txt")
	if *update {
		if err := os.WriteFile(golden, []byte(b.String()), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	want, err := os.ReadFile(golden)
	if err != nil {
		t.Fatalf("%v (run with -update)", err)
	}
	if string(want) != b.String() {
		t.Errorf("composed tree differs from %s (run with -update)\n--- got ---\n%s", golden, b.String())
	}

	// The root layout web-svelte renders imports the kit at exactly this path.
	layout := string(res.Files["web/src/routes/+layout.svelte"])
	m := regexp.MustCompile(`import AppShell from '\.\./lib/kit/AppShell\.svelte'`).FindString(layout)
	if m == "" || res.Files["web/src/lib/kit/AppShell.svelte"] == nil {
		t.Errorf("web-svelte's layout needs web/src/lib/kit/AppShell.svelte")
	}

	if out := os.Getenv(outEnv); out != "" {
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

func TestKitShadcnFileModes(t *testing.T) {
	res := composeKit(t, "kit-shadcn")
	var lock composer.Lock
	if err := json.Unmarshal(res.Files[".albitor/lock.json"], &lock); err != nil {
		t.Fatal(err)
	}
	modes := map[string]string{}
	for _, f := range lock.Files {
		modes[f.Path] = f.Mode
	}
	// Tokens and the accessibility gate (contrast, axe, and the Showcase they render) are platform-owned; the components are the app's to edit.
	for p, want := range map[string]string{
		"web/src/lib/kit/tokens.css":        "managed",
		"web/src/lib/kit/contrast.test.ts":  "managed",
		"web/src/lib/kit/kit.test.ts":       "managed",
		"web/src/lib/kit/dialog.test.ts":    "managed",
		"web/src/lib/kit/Showcase.svelte":   "managed",
		"web/src/lib/kit/AppShell.svelte":   "example",
		"web/src/lib/kit/Dialog.svelte":     "example",
		"web/src/lib/kit/Field.svelte":      "example",
		"web/src/lib/kit/Table.svelte":      "example",
		"web/src/lib/kit/Toaster.svelte":    "example",
		"web/src/lib/kit/EmptyState.svelte": "example",
		"web/src/lib/kit/toast.svelte.ts":   "example",
	} {
		if modes[p] != want {
			t.Errorf("%s: mode %q, want %q", p, modes[p], want)
		}
	}
}

// TestKitShadcnShipsExactlyWhatItDeclares: no stray template, no declared file missing, every path
// clean and under web/src/lib/kit/ (the kit owns nothing else of the app), and nothing under
// .github/.
func TestKitShadcnShipsExactlyWhatItDeclares(t *testing.T) { kitShipsExactly(t, "kit-shadcn") }

func kitShipsExactly(t *testing.T, kit string) {
	t.Helper()
	m := loadDir(t, kitDir(t, kit)).Manifest
	declared := map[string]bool{}
	for _, f := range m.Contributes.Files {
		if f.Path != filepath.ToSlash(filepath.Clean(f.Path)) || strings.HasPrefix(f.Path, "/") ||
			strings.Contains(f.Path, "..") || strings.Contains(f.Path, "*") {
			t.Errorf("unclean or glob path %q", f.Path)
		}
		// The one file outside the kit directory is the Vitest config its tests need.
		if !strings.HasPrefix(f.Path, "web/src/lib/kit/") && f.Path != "web/vitest.config.ts" {
			t.Errorf("%s: the kit writes only under web/src/lib/kit/ (and web/vitest.config.ts)", f.Path)
		}
		declared["templates/"+f.Path] = true
	}
	var shipped []string
	for p := range snapshot(t, kitDir(t, kit)) {
		if strings.HasPrefix(p, "templates/") {
			shipped = append(shipped, p)
		}
		if strings.HasPrefix(p, ".github/") || strings.Contains(p, "/.github/") {
			t.Errorf("a plugin ships no .github files: %s", p)
		}
	}
	sort.Strings(shipped)
	for _, p := range shipped {
		if !declared[p] {
			t.Errorf("%s ships but the manifest does not declare it", p)
		}
		delete(declared, p)
	}
	for p := range declared {
		t.Errorf("%s declared but not shipped", p)
	}
}

var exact = regexp.MustCompile(`^(@?[a-z0-9][a-z0-9._/-]*)@\d+\.\d+\.\d+$`)

func TestKitShadcnDependenciesAreExact(t *testing.T) {
	names := kitDependenciesAreExact(t, "kit-shadcn")
	if !names["bits-ui"] {
		t.Error("the kit depends on bits-ui")
	}
}

func kitDependenciesAreExact(t *testing.T, kit string) map[string]bool {
	t.Helper()
	m := loadDir(t, kitDir(t, kit)).Manifest
	names := map[string]bool{}
	for _, d := range m.Contributes.Dependencies.NPM {
		if !exact.MatchString(d) {
			t.Errorf("dependency %q is not an exact name@x.y.z (no range, tag or URL)", d)
		}
		names[d[:strings.LastIndex(d, "@")]] = true
	}
	return names
}

// TestKitShadcnPreviewTokens: the preview tokens pass the platform's own validator and describe
// the same palette tokens.css ships, so the preview and the app cannot drift.
func TestKitShadcnPreviewTokens(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join(kitDir(t, "kit-shadcn"), governance.PreviewTokensPath))
	if err != nil {
		t.Fatal(err)
	}
	tok, err := governance.ParsePreviewTokens(raw)
	if err != nil {
		t.Fatalf("preview tokens rejected by the platform validator: %v", err)
	}
	css, err := os.ReadFile(filepath.Join(kitDir(t, "kit-shadcn"), "templates", "web", "src", "lib", "kit", "tokens.css"))
	if err != nil {
		t.Fatal(err)
	}
	root := string(css)[strings.Index(string(css), ":root {"):]
	root = root[:strings.Index(root, "}")]
	get := func(name string) string {
		m := regexp.MustCompile(`--` + name + `:\s*(#[0-9a-fA-F]{6});`).FindStringSubmatch(root)
		if m == nil {
			t.Fatalf("tokens.css lacks --%s", name)
		}
		return strings.ToLower(m[1])
	}
	for name, got := range map[string]string{
		"foreground":         tok.Colours.Text,
		"muted-foreground":   tok.Colours.TextSecondary,
		"border":             tok.Colours.Border,
		"muted":              tok.Colours.Surface,
		"background":         tok.Colours.Background,
		"primary":            tok.Colours.Accent,
		"primary-foreground": tok.Colours.OnAccent,
	} {
		if strings.ToLower(got) != get(name) {
			t.Errorf("preview colour for --%s is %s, tokens.css says %s", name, got, get(name))
		}
	}
}

func TestKitShadcnSkill(t *testing.T) {
	skill := filepath.Join(kitDir(t, "kit-shadcn"), "skills", "kit-shadcn", "SKILL.md")
	b, err := os.ReadFile(skill)
	if err != nil {
		t.Fatal(err)
	}
	s := string(b)
	if !strings.HasPrefix(s, "---\nname: kit-shadcn\ndescription: ") {
		t.Error("SKILL.md needs front matter naming kit-shadcn")
	}
	// Svelte-first: the React-only advice of the shadcn plugin must not carry over.
	for _, banned := range []string{"use client", "Next.js", "npx shadcn@latest", "React Hook Form"} {
		if strings.Contains(s, banned) {
			t.Errorf("SKILL.md carries React-era advice %q", banned)
		}
	}
	// The shared WCAG content lives in test-playwright, not here.
	if _, err := os.Stat(filepath.Join(kitDir(t, "kit-shadcn"), "skills", "kit-shadcn", "references", "wcag-2.2.md")); err == nil {
		t.Error("the WCAG reference moves to test-playwright; do not copy it here")
	}
}
