package compose

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"albitor/core"
	"albitor/core/certification"
	"albitor/core/composer"
)

// harnessSubject is the plugin as the certification harness sees it: the validated manifest and the
// whole tree.
func harnessSubject(t *testing.T) certification.Subject {
	t.Helper()
	p := loadDir(t, filepath.Join(repoRoot(t), "plugins", "test-playwright"))
	for _, s := range core.Sockets() {
		if s.Name == core.SocketTestHarness {
			return certification.Subject{Manifest: p.Manifest, Tree: p.Tree, Socket: s}
		}
	}
	t.Fatal("the registry has no test-harness socket")
	return certification.Subject{}
}

// TestTestPlaywrightConforms runs the real conformance suite of the test-harness socket, the same
// checks ingest runs, over the plugin.
func TestTestPlaywrightConforms(t *testing.T) {
	sub := harnessSubject(t)
	checks := certification.DefaultSuites()[core.SocketTestHarness]
	ids := map[string]bool{}
	for _, c := range checks {
		ids[c.ID] = true
		if err := c.Run(sub); err != nil {
			t.Errorf("conformance check %s: %v", c.ID, err)
		}
	}
	if !ids["workflows-are-safe-to-pin"] {
		t.Fatal("the test-harness suite carries no workflow lint")
	}
}

// TestWorkflowLintIsReallyAppliedToTheWorkflow proves the conformance pass above is not vacuous:
// each unsafe edit of the shipped workflow must fail the real lint.
func TestWorkflowLintIsReallyAppliedToTheWorkflow(t *testing.T) {
	const path = "templates/.github/workflows/journeys.yml"
	sub := harnessSubject(t)
	orig, ok := sub.Tree[path]
	if !ok {
		t.Fatalf("plugin ships no %s", path)
	}
	var lint certification.Check
	for _, c := range certification.DefaultSuites()[core.SocketTestHarness] {
		if c.ID == "workflows-are-safe-to-pin" {
			lint = c
		}
	}

	cases := map[string]func(string) string{
		"an unpinned action": func(s string) string {
			return strings.Replace(s, "actions/checkout@11bd71901bbe5b1630ceea73d27597364c9af683", "actions/checkout@v4", 1)
		},
		"a secret outside the allowlist": func(s string) string {
			return strings.Replace(s, `CI: "true"`, "CI: ${{ secrets.NPM_TOKEN }}", 1)
		},
		"a write permission": func(s string) string {
			return strings.Replace(s, "contents: read", "contents: write", 1)
		},
		"pull_request_target": func(s string) string {
			return strings.Replace(s, "  pull_request:\n", "  pull_request_target:\n", 1)
		},
		"a dumped environment": func(s string) string {
			return strings.Replace(s, "run: npm run unit", "run: printenv", 1)
		},
	}
	for name, edit := range cases {
		changed := edit(string(orig))
		if changed == string(orig) {
			t.Fatalf("%s: the edit did not apply; the case is stale", name)
		}
		tree := map[string][]byte{}
		for k, v := range sub.Tree {
			tree[k] = v
		}
		tree[path] = []byte(changed)
		bad := sub
		bad.Tree = tree
		if err := lint.Run(bad); err == nil {
			t.Errorf("%s was accepted by the workflow lint", name)
		}
	}
}

// composeDefault composes the default-profile shell around the harness: web-svelte, the real
// test-playwright, and stubs for the sockets this repository does not own yet.
func composeDefault(t *testing.T) *composer.Result {
	t.Helper()
	root := repoRoot(t)
	plugins := []composer.Plugin{
		loadDir(t, filepath.Join(root, "tests", "compose", "stubs", "kit-stub")),
		inline(t, `{"schema":2,"name":"deploy-stub","kind":"socket","provides":["deploy-target@1"]}`),
		inline(t, `{"schema":2,"name":"auth-none-stub","kind":"socket","provides":["auth@1"]}`),
		loadDir(t, filepath.Join(root, "plugins", "web-svelte")),
		loadDir(t, filepath.Join(root, "plugins", "test-playwright")),
	}
	res, err := composer.Compose(composer.Input{Plugins: plugins, Tokens: tokens})
	if err != nil {
		t.Fatalf("test-playwright does not compose with web-svelte: %v", err)
	}
	return res
}

// TestTestPlaywrightComposes is the golden for the harness: the paths it writes, and the modes the
// platform pins them under.
func TestTestPlaywrightComposes(t *testing.T) {
	res := composeDefault(t)

	var paths []string
	for _, p := range res.Paths() {
		if strings.HasPrefix(p, "web/tests/") || strings.HasPrefix(p, ".github/workflows/") {
			paths = append(paths, p)
		}
	}
	got := strings.Join(paths, "\n") + "\n"
	golden := filepath.Join("testdata", "test-playwright.golden.txt")
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
		t.Errorf("composed harness paths differ from %s (run with -update to accept)\n--- got ---\n%s", golden, got)
	}

	// COMPOSE_OUT writes the composed tree for a local run of the journeys (see the README).
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

	var lock composer.Lock
	if err := json.Unmarshal(res.Files[".albitor/lock.json"], &lock); err != nil {
		t.Fatal(err)
	}
	modes := map[string]string{}
	for _, f := range lock.Files {
		modes[f.Path] = f.Mode
	}
	// The harness and the CI that runs it are pinned: a builder cannot weaken either.
	for _, p := range []string{
		".github/workflows/journeys.yml",
		"web/tests/playwright.config.ts",
		"web/tests/auth.setup.ts",
		"web/tests/helpers/journey.ts",
		"web/tests/helpers/seed.ts",
		"web/tests/helpers/a11y.ts",
		"web/tests/package-lock.json",
	} {
		if modes[p] != "managed" {
			t.Errorf("%s should be managed, got %q", p, modes[p])
		}
	}
	// A journey spec is the spec author's, never the harness's.
	for p := range modes {
		if strings.HasPrefix(p, "web/tests/journeys/") && modes[p] != "example" {
			t.Errorf("%s should be an example, got %q", p, modes[p])
		}
		if strings.HasPrefix(p, "web/tests/fixtures/") {
			t.Errorf("the harness must not write into the spec author's fixtures dir: %s", p)
		}
	}

	// The spec-author paths and helper names the planner prompt hard-codes must all exist.
	for _, p := range []string{
		"web/tests/auth.setup.ts",
		"web/tests/helpers/journey.ts",
		"web/tests/helpers/seed.ts",
		"web/tests/helpers/mailbox.ts",
		"web/tests/helpers/fixture-scope.ts",
	} {
		if _, ok := res.Files[p]; !ok {
			t.Errorf("composed tree lacks %s", p)
		}
	}
	for _, name := range []string{"journeyStep", "appAuthHeaders", "fixtureScope", "walkMailboxAvailable", "awaitVerification", "walkAccount", "EMPTY_STORAGE_STATE", "hasDemoSession"} {
		found := false
		for p, b := range res.Files {
			if strings.HasPrefix(p, "web/tests/helpers/") && strings.Contains(string(b), name) {
				found = true
			}
		}
		if !found {
			t.Errorf("no helper defines or exports %s, which the spec-author prompt names", name)
		}
	}

	// Dependabot follows the harness's own package and its workflow pins.
	dependabot := string(res.Held[".github/dependabot.yml"])
	for _, want := range []string{"/web/tests", "github-actions"} {
		if !strings.Contains(dependabot, want) {
			t.Errorf("dependabot.yml lacks %q:\n%s", want, dependabot)
		}
	}
}

// TestOnlyAHarnessMayShipAWorkflow keeps the sanctioned path narrow: the same workflow from a
// plugin that fills another socket is refused by the composer.
func TestOnlyAHarnessMayShipAWorkflow(t *testing.T) {
	root := repoRoot(t)
	harness := loadDir(t, filepath.Join(root, "plugins", "test-playwright"))
	other := loadDir(t, filepath.Join(root, "tests", "compose", "stubs", "kit-stub"))
	other.Tree["templates/.github/workflows/journeys.yml"] = harness.Tree["templates/.github/workflows/journeys.yml"]
	raw := `{"schema":2,"name":"kit-stub","kind":"socket","provides":["component-kit@1"],"requires":["web-framework@1"],` +
		`"contributes":{"files":[{"path":".github/workflows/journeys.yml","mode":"managed"}]}}`
	other.Manifest = inline(t, raw).Manifest
	plugins := []composer.Plugin{
		other,
		inline(t, `{"schema":2,"name":"deploy-stub","kind":"socket","provides":["deploy-target@1"]}`),
		inline(t, `{"schema":2,"name":"auth-none-stub","kind":"socket","provides":["auth@1"]}`),
		inline(t, `{"schema":2,"name":"harness-stub","kind":"socket","provides":["test-harness@1"]}`),
		loadDir(t, filepath.Join(root, "plugins", "web-svelte")),
	}
	_, err := composer.Compose(composer.Input{Plugins: plugins, Tokens: tokens})
	if err == nil {
		t.Fatal("a component-kit plugin wrote .github/workflows/journeys.yml")
	}
	if !strings.Contains(err.Error(), "workflows") {
		t.Errorf("refused, but not for the workflow path: %v", err)
	}
}
