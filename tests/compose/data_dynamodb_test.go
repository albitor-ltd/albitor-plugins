package compose

import (
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"

	"albitor/core/composer"
)

// composeData composes data-dynamodb over api-go beside web-svelte and the stub sockets. api-go is
// loaded without its slot templates (the loader convention for them lands separately), which is
// enough here: neither plugin fills a slot this test reads.
func composeData(t *testing.T) *composer.Result {
	t.Helper()
	root := repoRoot(t)
	plugins := append(stubs(t, root, true),
		loadDir(t, filepath.Join(root, "plugins", "web-svelte")),
		loadDir(t, filepath.Join(root, "plugins", "api-go")),
		loadDir(t, filepath.Join(root, "plugins", "data-dynamodb")),
	)
	res, err := composer.Compose(composer.Input{Plugins: plugins, Tokens: tokens})
	if err != nil {
		t.Fatalf("data-dynamodb does not compose over api-go: %v", err)
	}
	return res
}

func TestDataDynamoDBComposesOverAPIGo(t *testing.T) {
	res := composeData(t)

	var lock composer.Lock
	if err := json.Unmarshal(res.Files[".albitor/lock.json"], &lock); err != nil {
		t.Fatal(err)
	}
	modes := map[string]string{}
	for _, f := range lock.Files {
		modes[f.Path] = f.Mode
	}
	for _, p := range []string{
		"api/internal/store/store.go", "api/internal/store/memory.go", "api/internal/store/dynamodb.go",
		"api/internal/store/sigv4.go", "api/internal/store/open.go",
		"api/internal/store/storetest/suite.go", "api/internal/store/memory_test.go",
	} {
		if _, ok := res.Files[p]; !ok {
			t.Errorf("%s is not in the composed tree", p)
		}
		if modes[p] != "example" {
			t.Errorf("%s should be an example (seeded once, app-owned), got %q", p, modes[p])
		}
	}

	// The plugin writes only inside its own directory: nothing from it lands under .github/, and
	// the conformance test that needs DynamoDB Local is not shipped to apps.
	for p := range res.Files {
		if strings.HasPrefix(p, ".github/") {
			t.Errorf("a plugin wrote %s", p)
		}
		if strings.Contains(p, "dynamodb_local") {
			t.Errorf("the DynamoDB Local test was shipped to the app: %s", p)
		}
	}
	if strings.Contains(string(res.Files["api/go.mod"]), "require") {
		t.Errorf("api/go.mod gained a dependency; the store is stdlib-only:\n%s", res.Files["api/go.mod"])
	}

	// The services.tables fragment reaches albitor.yaml, as the closed manifest vocabulary.
	yaml := string(res.Files["albitor.yaml"])
	for _, want := range []string{"tables:", `name: "data"`, "partition_key:", `name: "pk"`, "sort_key:", `name: "sk"`} {
		if !strings.Contains(yaml, want) {
			t.Errorf("albitor.yaml lacks %q:\n%s", want, yaml)
		}
	}
}

func TestDataDynamoDBNeedsAnAPIRuntime(t *testing.T) {
	root := repoRoot(t)
	plugins := append(stubs(t, root, true),
		loadDir(t, filepath.Join(root, "plugins", "web-svelte")),
		loadDir(t, filepath.Join(root, "plugins", "data-dynamodb")),
	)
	if _, err := composer.Compose(composer.Input{Plugins: plugins, Tokens: tokens}); err == nil {
		t.Fatal("data-dynamodb composed with no api-runtime; its Go would have no module to live in")
	}
}
