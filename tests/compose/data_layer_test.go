package compose

import (
	"encoding/json"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"albitor/core/authtemplate"
	"albitor/core/composer"
)

// web-svelte seeds the online data layer only. The offline-first layer is a worked example in the
// data-offline skill (albitor-ltd/albitor#4300): an app copies it in and points
// data-layer/index.ts at it. These tests pin both halves.

const offlineExample = "plugins/web-svelte/skills/data-offline/examples"

// composeOnline composes web-svelte with stubs for the other exactly-one sockets.
func composeOnline(t *testing.T) *composer.Result {
	t.Helper()
	root := repoRoot(t)
	toks := authtemplate.Values{}
	for k, v := range tokens {
		toks[k] = v
	}
	plugins := append(stubs(t, root, false), loadDir(t, filepath.Join(root, "plugins", "web-svelte")))
	res, err := composer.Compose(composer.Input{Plugins: plugins, Tokens: toks})
	if err != nil {
		t.Fatalf("web-svelte does not compose: %v", err)
	}
	return res
}

// composeOffline is what an app has after following the data-offline skill: the online composition
// with the example's files copied to the same paths and index.ts pointing at ./offline.
func composeOffline(t *testing.T) map[string][]byte {
	t.Helper()
	res := composeOnline(t)
	files := map[string][]byte{}
	for p, c := range res.Files {
		files[p] = c
	}
	base := filepath.Join(repoRoot(t), filepath.FromSlash(offlineExample))
	err := filepath.WalkDir(base, func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		rel, err := filepath.Rel(base, p)
		if err != nil {
			return err
		}
		c, err := os.ReadFile(p)
		files[filepath.ToSlash(rel)] = c
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	idx := "web/src/lib/data-layer/index.ts"
	files[idx] = []byte(strings.ReplaceAll(string(files[idx]), "'./online'", "'./offline'"))
	return files
}

func TestSeededEntryPointIsOnline(t *testing.T) {
	res := composeOnline(t)
	index := string(res.Files["web/src/lib/data-layer/index.ts"])
	if !strings.Contains(index, "from './online';") || strings.Contains(index, "{{") {
		t.Errorf("data-layer/index.ts does not export ./online:\n%s", index)
	}
	if entry := string(res.Files["web/src/lib/data-layer/online.ts"]); !strings.Contains(entry, "QueryProvider") {
		t.Error("data-layer/online.ts does not export QueryProvider as the DataProvider")
	}
	providers := string(res.Files["web/src/lib/shell/providers.generated.ts"])
	for _, want := range []string{"const { DataProvider } = scope;", "checkProviders([DataProvider])"} {
		if !strings.Contains(providers, want) {
			t.Errorf("providers.generated.ts lacks %q:\n%s", want, providers)
		}
	}
	if !strings.Contains(string(res.Files["web/src/lib/shell/scope.d/data-layer.ts"]), "DataProvider") {
		t.Error("scope.d/data-layer.ts must export DataProvider")
	}
}

// The seeded tree is closed: no offline file is composed, nothing seeded imports the sync layer,
// and the retired client-data socket is not in the lock.
func TestSeededTreeCarriesNoOfflineLayer(t *testing.T) {
	res := composeOnline(t)
	var lock composer.Lock
	if err := json.Unmarshal(res.Files[".albitor/lock.json"], &lock); err != nil {
		t.Fatal(err)
	}
	for _, s := range lock.Sockets {
		if s.Socket == "client-data" {
			t.Errorf("the lock still fills client-data: %+v", s)
		}
	}
	data := 0
	for _, f := range lock.Files {
		if strings.HasPrefix(f.Path, "web/src/lib/sync/") || f.Path == "web/src/lib/data-layer/offline.ts" {
			t.Errorf("%s is seeded; the offline layer is a skill example", f.Path)
		}
		if strings.HasPrefix(f.Path, "web/src/lib/data/") {
			data++
		}
	}
	if data == 0 {
		t.Error("the online data layer is not composed")
	}
	for p, c := range res.Files {
		if strings.HasSuffix(p, ".ts") || strings.HasSuffix(p, ".svelte") {
			if strings.Contains(string(c), "/sync/") || strings.Contains(string(c), "./offline") {
				t.Errorf("seeded %s imports the offline example", p)
			}
		}
	}
	pkg := string(res.Files["web/package.json"])
	for _, dep := range []string{`"@tanstack/svelte-query": "6.3.2"`, `"fake-indexeddb": "6.2.5"`} {
		if !strings.Contains(pkg, dep) {
			t.Errorf("web/package.json lacks %s", dep)
		}
	}
}

// TestOfflineExampleShipsTheServiceWorker pins the offline promise: the seeded tree carries the
// worker that precaches the shell, and the example's shell.test.ts guards it.
func TestOfflineExampleShipsTheServiceWorker(t *testing.T) {
	files := composeOffline(t)
	sw := string(files["web/src/service-worker/index.ts"])
	if sw == "" {
		t.Fatal("the composed tree has no src/service-worker/index.ts, so no worker reaches the deploy output")
	}
	for _, want := range []string{"cache.addAll", "/index.html", "/api/"} {
		if !strings.Contains(sw, want) {
			t.Errorf("service worker lacks %q", want)
		}
	}
	if _, ok := files["web/src/lib/sync/shell.test.ts"]; !ok {
		t.Error("shell.test.ts, which guards the worker in the app's own tests, is not in the example")
	}
	if !strings.Contains(string(files["web/src/lib/data-layer/index.ts"]), "from './offline';") {
		t.Error("the swapped index.ts does not export ./offline")
	}
	if !strings.Contains(string(files["web/src/lib/data-layer/offline.ts"]), "SyncProvider") {
		t.Error("data-layer/offline.ts does not export SyncProvider as the DataProvider")
	}
}

// TestDataLayersWriteTreesForBuild writes each tree for the build step, when COMPOSE_OUT_ONLINE or
// COMPOSE_OUT_OFFLINE is set. The offline tree is the seeded one with the example copied in.
func TestDataLayersWriteTreesForBuild(t *testing.T) {
	wrote := false
	trees := map[string]func() map[string][]byte{
		"COMPOSE_OUT_ONLINE":  func() map[string][]byte { return composeOnline(t).Files },
		"COMPOSE_OUT_OFFLINE": func() map[string][]byte { return composeOffline(t) },
	}
	for env, tree := range trees {
		out := os.Getenv(env)
		if out == "" {
			continue
		}
		wrote = true
		for path, content := range tree() {
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
