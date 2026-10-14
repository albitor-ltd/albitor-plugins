package compose

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"albitor/core/composer"
)

// composeOffline composes web-svelte with the real client-offline in place of the client-data stub.
func composeOffline(t *testing.T) *composer.Result {
	t.Helper()
	root := repoRoot(t)
	plugins := []composer.Plugin{
		loadDir(t, filepath.Join(root, "tests", "compose", "stubs", "kit-stub")),
		inline(t, `{"schema":2,"name":"harness-stub","kind":"socket","provides":["test-harness@1"]}`),
		inline(t, `{"schema":2,"name":"deploy-stub","kind":"socket","provides":["deploy-target@1"]}`),
		inline(t, `{"schema":2,"name":"auth-none-stub","kind":"socket","provides":["auth@1"]}`),
		loadDir(t, filepath.Join(root, "plugins", "web-svelte")),
		loadDir(t, filepath.Join(root, "plugins", "client-offline")),
	}
	res, err := composer.Compose(composer.Input{Plugins: plugins, Tokens: tokens})
	if err != nil {
		t.Fatalf("client-offline does not compose with web-svelte: %v", err)
	}
	return res
}

func TestClientOfflineFillsClientDataAndTheProvidersSlot(t *testing.T) {
	res := composeOffline(t)

	providers := string(res.Files["web/src/lib/shell/providers.generated.ts"])
	for _, want := range []string{"const { SyncProvider } = scope;", "checkProviders([SyncProvider])"} {
		if !strings.Contains(providers, want) {
			t.Errorf("providers.generated.ts lacks %q:\n%s", want, providers)
		}
	}
	if !strings.Contains(string(res.Files["web/src/lib/shell/scope.d/client-offline.ts"]), "SyncProvider") {
		t.Error("scope.d/client-offline.ts must export SyncProvider")
	}

	var lock composer.Lock
	if err := json.Unmarshal(res.Files[".albitor/lock.json"], &lock); err != nil {
		t.Fatal(err)
	}
	filled := false
	for _, s := range lock.Sockets {
		if s.Socket == "client-data" && len(s.Plugins) == 1 && s.Plugins[0] == "client-offline" {
			filled = true
		}
	}
	if !filled {
		t.Errorf("client-data is not filled by client-offline: %+v", lock.Sockets)
	}
	for _, f := range lock.Files {
		if !strings.HasPrefix(f.Path, "web/src/lib/sync/") {
			continue
		}
		want := "managed"
		if f.Path == "web/src/lib/sync/collections.ts" {
			want = "example" // the app lists its own collections here
		}
		if f.Mode != want {
			t.Errorf("%s should be %s, got %q", f.Path, want, f.Mode)
		}
	}
}

// TestClientOfflineShipsTheServiceWorker pins the offline promise at composition: the composed tree
// carries the worker that precaches the shell, so the build emits it into the deploy output.
func TestClientOfflineShipsTheServiceWorker(t *testing.T) {
	res := composeOffline(t)
	sw := string(res.Files["web/src/service-worker/index.ts"])
	if sw == "" {
		t.Fatal("the composed tree has no src/service-worker/index.ts, so no worker reaches the deploy output")
	}
	for _, want := range []string{"cache.addAll", "/index.html", "/api/"} {
		if !strings.Contains(sw, want) {
			t.Errorf("service worker lacks %q", want)
		}
	}
	if _, ok := res.Files["web/src/lib/sync/shell.test.ts"]; !ok {
		t.Error("shell.test.ts, which guards the worker in the app's own tests, is not composed")
	}
}

// TestClientOfflineStaysInItsDirectories: it writes only under web/src/lib/sync and its own
// scope.d module, never .github/, never the shell's service worker or config.
func TestClientOfflineStaysInItsDirectories(t *testing.T) {
	root := repoRoot(t)
	p := loadDir(t, filepath.Join(root, "plugins", "client-offline"))
	for _, f := range p.Manifest.Contributes.Files {
		ok := strings.HasPrefix(f.Path, "web/src/lib/sync/") || f.Path == "web/src/lib/shell/scope.d/client-offline.ts"
		if !ok {
			t.Errorf("client-offline contributes %q outside its own directories", f.Path)
		}
	}
	err := filepath.WalkDir(filepath.Join(root, "plugins", "client-offline"), func(path string, d os.DirEntry, err error) error {
		if err == nil && strings.Contains(filepath.ToSlash(path), "/.github") {
			t.Errorf("client-offline must not carry .github/ files: %s", path)
		}
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
}

// TestClientOfflineAndOnlineCannotBothFillClientData: the socket has exactly one provider.
func TestClientOfflineAndOnlineCannotBothFillClientData(t *testing.T) {
	root := repoRoot(t)
	plugins := []composer.Plugin{
		loadDir(t, filepath.Join(root, "tests", "compose", "stubs", "kit-stub")),
		inline(t, `{"schema":2,"name":"harness-stub","kind":"socket","provides":["test-harness@1"]}`),
		inline(t, `{"schema":2,"name":"deploy-stub","kind":"socket","provides":["deploy-target@1"]}`),
		inline(t, `{"schema":2,"name":"auth-none-stub","kind":"socket","provides":["auth@1"]}`),
		loadDir(t, filepath.Join(root, "plugins", "web-svelte")),
		loadDir(t, filepath.Join(root, "plugins", "client-online")),
		loadDir(t, filepath.Join(root, "plugins", "client-offline")),
	}
	if _, err := composer.Compose(composer.Input{Plugins: plugins, Tokens: tokens}); err == nil {
		t.Fatal("client-online and client-offline composed together")
	}
}

func TestClientOfflineWithoutAWebFrameworkIsRefused(t *testing.T) {
	root := repoRoot(t)
	plugins := []composer.Plugin{
		loadDir(t, filepath.Join(root, "tests", "compose", "stubs", "kit-stub")),
		inline(t, `{"schema":2,"name":"harness-stub","kind":"socket","provides":["test-harness@1"]}`),
		inline(t, `{"schema":2,"name":"deploy-stub","kind":"socket","provides":["deploy-target@1"]}`),
		inline(t, `{"schema":2,"name":"auth-none-stub","kind":"socket","provides":["auth@1"]}`),
		loadDir(t, filepath.Join(root, "plugins", "client-offline")),
	}
	if _, err := composer.Compose(composer.Input{Plugins: plugins, Tokens: tokens}); err == nil {
		t.Fatal("client-offline composed with no web-framework")
	}
}

// TestClientOfflineWritesTreeForBuild writes the composed tree for the build step, when
// COMPOSE_OUT_OFFLINE is set.
func TestClientOfflineWritesTreeForBuild(t *testing.T) {
	out := os.Getenv("COMPOSE_OUT_OFFLINE")
	if out == "" {
		t.Skip("COMPOSE_OUT_OFFLINE not set")
	}
	for path, content := range composeOffline(t).Files {
		dst := filepath.Join(out, filepath.FromSlash(path))
		if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(dst, content, 0o644); err != nil {
			t.Fatal(err)
		}
	}
}
