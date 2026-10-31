package compose

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"albitor/core/authtemplate"
	"albitor/core/composer"
)

// web-svelte ships both client data layers and the platform's data_layer token selects one
// (albitor-ltd/albitor ADR-0090, amendment of 2026-10-15). These tests replace the old
// client-online and client-offline plugin tests: the same files, now composed from web-svelte.

// composeLayer composes web-svelte with stubs for the other exactly-one sockets and the given
// data layer.
func composeLayer(t *testing.T, layer string) *composer.Result {
	t.Helper()
	root := repoRoot(t)
	toks := authtemplate.Values{}
	for k, v := range tokens {
		toks[k] = v
	}
	toks[composer.TokenDataLayer] = layer
	plugins := append(stubs(t, root, false), loadDir(t, filepath.Join(root, "plugins", "web-svelte")))
	res, err := composer.Compose(composer.Input{Plugins: plugins, Tokens: toks})
	if err != nil {
		t.Fatalf("web-svelte does not compose with the %s data layer: %v", layer, err)
	}
	return res
}

func TestDataLayerTokenSelectsTheEntryPoint(t *testing.T) {
	for layer, provider := range map[string]string{composer.DataLayerOnline: "QueryProvider", composer.DataLayerOffline: "SyncProvider"} {
		res := composeLayer(t, layer)
		index := string(res.Files["web/src/lib/data-layer/index.ts"])
		if !strings.Contains(index, "from './"+layer+"';") || strings.Contains(index, "{{") {
			t.Errorf("%s: data-layer/index.ts does not select ./%s:\n%s", layer, layer, index)
		}
		if entry := string(res.Files["web/src/lib/data-layer/"+layer+".ts"]); !strings.Contains(entry, provider) {
			t.Errorf("%s: data-layer/%s.ts does not export %s as the DataProvider", layer, layer, provider)
		}
		providers := string(res.Files["web/src/lib/shell/providers.generated.ts"])
		for _, want := range []string{"const { DataProvider } = scope;", "checkProviders([DataProvider])"} {
			if !strings.Contains(providers, want) {
				t.Errorf("%s: providers.generated.ts lacks %q:\n%s", layer, want, providers)
			}
		}
		if !strings.Contains(string(res.Files["web/src/lib/shell/scope.d/data-layer.ts"]), "DataProvider") {
			t.Errorf("%s: scope.d/data-layer.ts must export DataProvider", layer)
		}
	}
}

// Without a data_layer value the composition is refused, never rendered with an empty import.
func TestDataLayerTokenIsRequired(t *testing.T) {
	root := repoRoot(t)
	toks := authtemplate.Values{}
	for k, v := range tokens {
		if k != composer.TokenDataLayer {
			toks[k] = v
		}
	}
	plugins := append(stubs(t, root, false), loadDir(t, filepath.Join(root, "plugins", "web-svelte")))
	if _, err := composer.Compose(composer.Input{Plugins: plugins, Tokens: toks}); err == nil {
		t.Fatal("web-svelte composed with no data_layer token")
	}
}

// Both layers ship whichever is selected, managed except the app's collections list, and the
// retired client-data socket is not in the lock.
func TestBothDataLayersShipManaged(t *testing.T) {
	res := composeLayer(t, composer.DataLayerOnline)
	var lock composer.Lock
	if err := json.Unmarshal(res.Files[".albitor/lock.json"], &lock); err != nil {
		t.Fatal(err)
	}
	for _, s := range lock.Sockets {
		if s.Socket == "client-data" {
			t.Errorf("the lock still fills client-data: %+v", s)
		}
	}
	data, sync := 0, 0
	for _, f := range lock.Files {
		switch {
		case strings.HasPrefix(f.Path, "web/src/lib/data/"):
			data++
		case strings.HasPrefix(f.Path, "web/src/lib/sync/"):
			sync++
		default:
			continue
		}
		want := "example"
		if f.Mode != want {
			t.Errorf("%s should be %s, got %q", f.Path, want, f.Mode)
		}
	}
	if data == 0 || sync == 0 {
		t.Errorf("an online composition ships %d online and %d offline layer files; want both", data, sync)
	}
	pkg := string(res.Files["web/package.json"])
	for _, dep := range []string{`"@tanstack/svelte-query": "6.3.2"`, `"fake-indexeddb": "6.2.5"`} {
		if !strings.Contains(pkg, dep) {
			t.Errorf("web/package.json lacks %s", dep)
		}
	}
}

// TestOfflineLayerShipsTheServiceWorker pins the offline promise at composition: the composed
// tree carries the worker that precaches the shell, so the build emits it into the deploy output.
func TestOfflineLayerShipsTheServiceWorker(t *testing.T) {
	res := composeLayer(t, composer.DataLayerOffline)
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

// TestDataLayersWriteTreesForBuild writes each composed tree for the build step, when
// COMPOSE_OUT_ONLINE or COMPOSE_OUT_OFFLINE is set.
func TestDataLayersWriteTreesForBuild(t *testing.T) {
	wrote := false
	for env, layer := range map[string]string{"COMPOSE_OUT_ONLINE": composer.DataLayerOnline, "COMPOSE_OUT_OFFLINE": composer.DataLayerOffline} {
		out := os.Getenv(env)
		if out == "" {
			continue
		}
		wrote = true
		for path, content := range composeLayer(t, layer).Files {
			dst := filepath.Join(out, filepath.FromSlash(path))
			if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(dst, content, 0o644); err != nil {
				t.Fatal(err)
			}
		}
	}
	if !wrote {
		t.Skip("COMPOSE_OUT_ONLINE and COMPOSE_OUT_OFFLINE not set")
	}
}
