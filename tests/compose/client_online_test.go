package compose

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"albitor/core/composer"
)

// composeOnline composes web-svelte with the real client-online in place of the client-data stub.
func composeOnline(t *testing.T) *composer.Result {
	t.Helper()
	root := repoRoot(t)
	plugins := []composer.Plugin{
		loadDir(t, filepath.Join(root, "tests", "compose", "stubs", "kit-stub")),
		inline(t, `{"schema":2,"name":"harness-stub","kind":"socket","provides":["test-harness@1"]}`),
		inline(t, `{"schema":2,"name":"deploy-stub","kind":"socket","provides":["deploy-target@1"]}`),
		inline(t, `{"schema":2,"name":"auth-none-stub","kind":"socket","provides":["auth@1"]}`),
		loadDir(t, filepath.Join(root, "plugins", "web-svelte")),
		loadDir(t, filepath.Join(root, "plugins", "client-online")),
	}
	res, err := composer.Compose(composer.Input{Plugins: plugins, Tokens: tokens})
	if err != nil {
		t.Fatalf("client-online does not compose with web-svelte: %v", err)
	}
	return res
}

func TestClientOnlyFillsClientDataAndTheProvidersSlot(t *testing.T) {
	res := composeOnline(t)

	providers := string(res.Files["web/src/lib/shell/providers.generated.ts"])
	for _, want := range []string{"const { QueryProvider } = scope;", "checkProviders([QueryProvider])"} {
		if !strings.Contains(providers, want) {
			t.Errorf("providers.generated.ts lacks %q:\n%s", want, providers)
		}
	}
	if !strings.Contains(string(res.Files["web/src/lib/shell/scope.d/client-online.ts"]), "QueryProvider") {
		t.Error("scope.d/client-online.ts must export QueryProvider")
	}

	var lock composer.Lock
	if err := json.Unmarshal(res.Files[".albitor/lock.json"], &lock); err != nil {
		t.Fatal(err)
	}
	filled := false
	for _, s := range lock.Sockets {
		if s.Socket == "client-data" && len(s.Plugins) == 1 && s.Plugins[0] == "client-online" {
			filled = true
		}
	}
	if !filled {
		t.Errorf("client-data is not filled by client-online: %+v", lock.Sockets)
	}
	for _, f := range lock.Files {
		if strings.HasPrefix(f.Path, "web/src/lib/data/") && f.Mode != "managed" {
			t.Errorf("%s should be managed, got %q", f.Path, f.Mode)
		}
	}
}

// TestClientOnlyShipsNoWorkflowAndStaysInItsDirectories pins the path discipline: the plugin only
// writes under web/src/lib/data and web/src/lib/shell/scope.d, never .github/ or a shared file.
func TestClientOnlyShipsNoWorkflowAndStaysInItsDirectories(t *testing.T) {
	root := repoRoot(t)
	p := loadDir(t, filepath.Join(root, "plugins", "client-online"))
	for _, f := range p.Manifest.Contributes.Files {
		ok := strings.HasPrefix(f.Path, "web/src/lib/data/") || strings.HasPrefix(f.Path, "web/src/lib/shell/scope.d/client-online")
		if !ok {
			t.Errorf("client-online contributes %q outside its own directories", f.Path)
		}
		if f.Mode != "managed" {
			t.Errorf("%s: platform plumbing must be managed, got %q", f.Path, f.Mode)
		}
	}
	err := filepath.WalkDir(filepath.Join(root, "plugins", "client-online"), func(path string, d os.DirEntry, err error) error {
		if err == nil && strings.Contains(filepath.ToSlash(path), "/.github") {
			t.Errorf("client-online must not carry .github/ files: %s", path)
		}
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
}

// TestClientOnlyWithoutAWebFrameworkIsRefused: the provider needs the slot host and API client.
func TestClientOnlyWithoutAWebFrameworkIsRefused(t *testing.T) {
	root := repoRoot(t)
	plugins := []composer.Plugin{
		loadDir(t, filepath.Join(root, "tests", "compose", "stubs", "kit-stub")),
		inline(t, `{"schema":2,"name":"harness-stub","kind":"socket","provides":["test-harness@1"]}`),
		inline(t, `{"schema":2,"name":"deploy-stub","kind":"socket","provides":["deploy-target@1"]}`),
		inline(t, `{"schema":2,"name":"auth-none-stub","kind":"socket","provides":["auth@1"]}`),
		loadDir(t, filepath.Join(root, "plugins", "client-online")),
	}
	if _, err := composer.Compose(composer.Input{Plugins: plugins, Tokens: tokens}); err == nil {
		t.Fatal("client-online composed with no web-framework")
	}
}

// writeOnline writes the composed tree for the build step, when COMPOSE_OUT_ONLINE is set.
func TestClientOnlyWritesTreeForBuild(t *testing.T) {
	out := os.Getenv("COMPOSE_OUT_ONLINE")
	if out == "" {
		t.Skip("COMPOSE_OUT_ONLINE not set")
	}
	for path, content := range composeOnline(t).Files {
		dst := filepath.Join(out, filepath.FromSlash(path))
		if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(dst, content, 0o644); err != nil {
			t.Fatal(err)
		}
	}
}
