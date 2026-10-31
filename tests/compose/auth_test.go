package compose

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"albitor/core"
	"albitor/core/certification"
	"albitor/core/composer"
)

// authPeers is the default profile as this repository owns it (Svelte, shadcn, online, Go,
// Playwright) and a stub for the deploy target, which lives elsewhere. The auth member is the
// candidate under test.
func authPeers(t *testing.T) catalogue {
	t.Helper()
	root := repoRoot(t)
	c := catalogue{}
	for _, n := range []string{"web-svelte", "kit-shadcn", "kit-carbon", "kit-gds", "api-go", "test-playwright"} {
		c[n] = loadDir(t, filepath.Join(root, "plugins", n))
	}
	c["deploy-stub"] = inline(t, `{"schema":2,"name":"deploy-stub","kind":"socket","provides":["deploy-target@1"]}`)
	return c
}

func authProfile(withAPI bool) certification.Profile {
	members := []certification.Member{
		{Socket: core.SocketWebFramework, Plugin: "web-svelte"},
		{Socket: core.SocketComponentKit, Plugin: "kit-shadcn"},
		{Socket: core.SocketDeployTarget, Plugin: "deploy-stub"},
		{Socket: core.SocketTestHarness, Plugin: "test-playwright"},
		{Socket: core.SocketAuth, Plugin: "auth-candidate"},
	}
	if withAPI {
		members = append(members, certification.Member{Socket: core.SocketAPIRuntime, Plugin: "api-go"})
	}
	return certification.Profile{Name: "auth-check", Members: members}
}

func authDir(t *testing.T, name string) string { return filepath.Join(repoRoot(t), "plugins", name) }

// TestAuthPluginsConform runs the platform's own certification harness over each plugin's
// snapshot: the manifest through the real parser, the auth@1 conformance suite, then a compose and
// the floor gate in the default profile. No Builder is set, so the best verdict is static_only;
// scripts/compose-check.sh --build --journeys builds the composed tree and signs in locally.
func TestAuthPluginsConform(t *testing.T) {
	for _, name := range []string{"auth-cognito", "auth-none"} {
		t.Run(name, func(t *testing.T) {
			h := &certification.Harness{
				Tokens:   tokens,
				Profiles: []certification.Profile{authProfile(true)},
				Peers:    authPeers(t),
				Suites:   certification.DefaultSuites(),
			}
			cert, err := h.Certify(context.Background(), snapshot(t, authDir(t, name)))
			if err != nil {
				t.Fatal(err)
			}
			if cert.Verdict == certification.VerdictFailed {
				t.Fatalf("certification failed: %v", cert.Failures)
			}
			if len(cert.Conformance) == 0 {
				t.Fatal("no conformance checks ran for auth@1")
			}
			for _, c := range cert.Conformance {
				if !c.Passed {
					t.Errorf("conformance %s: %s", c.ID, c.Detail)
				}
			}
			if len(cert.Profiles) != 1 || !cert.Profiles[0].Compatible || len(cert.Profiles[0].ComposeErrors) != 0 {
				t.Fatalf("profile result: %+v", cert.Profiles)
			}
		})
	}
}

// TestAuthPluginsFailClosed pins that the harness can fail these plugins: a declared file that is
// not shipped is a failed verdict, so the test above cannot pass vacuously.
func TestAuthPluginsFailClosed(t *testing.T) {
	for name, missing := range map[string]string{
		"auth-cognito": "templates/api/internal/shell/identity_verify.go",
		"auth-none":    "templates/api/internal/identity/identity.go",
	} {
		t.Run(name, func(t *testing.T) {
			files := snapshot(t, authDir(t, name))
			if _, ok := files[missing]; !ok {
				t.Fatalf("%s does not ship %s; the case is stale", name, missing)
			}
			delete(files, missing)
			h := &certification.Harness{
				Tokens:   tokens,
				Profiles: []certification.Profile{authProfile(true)},
				Peers:    authPeers(t),
				Suites:   certification.DefaultSuites(),
			}
			cert, err := h.Certify(context.Background(), files)
			if err != nil {
				t.Fatal(err)
			}
			if cert.Verdict != certification.VerdictFailed {
				t.Fatalf("a plugin missing %s must fail, got %s", missing, cert.Verdict)
			}
		})
	}
}

// composeAuth composes the default shell around one auth plugin: the real web-svelte, kit-shadcn,
// api-go and test-playwright, and a stub deploy target.
func composeAuth(t *testing.T, auth string, extra ...string) (*composer.Result, error) {
	t.Helper()
	peers := authPeers(t)
	plugins := []composer.Plugin{loadDir(t, authDir(t, auth))}
	names := extra
	if len(names) == 0 {
		names = []string{"web-svelte", "kit-shadcn", "api-go", "test-playwright", "deploy-stub"}
	}
	for _, n := range names {
		plugins = append(plugins, peers[n])
	}
	return composer.Compose(composer.Input{Plugins: plugins, Tokens: tokens})
}

// requireOneContractWithMe asserts the composed tree has exactly one api/openapi.yaml and that it
// defines GET /api/me, the entry the sign-in probe depends on.
func requireOneContractWithMe(t *testing.T, res *composer.Result) {
	t.Helper()
	n := 0
	for path, body := range res.Files {
		if path == "api/openapi.yaml" {
			n++
			if !strings.Contains(string(body), "\n  /api/me:\n") {
				t.Errorf("api/openapi.yaml does not define /api/me")
			}
		}
	}
	if n != 1 {
		t.Errorf("want exactly one api/openapi.yaml in the composed tree, got %d", n)
	}
}

func lockModes(t *testing.T, res *composer.Result) map[string]string {
	t.Helper()
	var lock composer.Lock
	if err := json.Unmarshal(res.Files[".albitor/lock.json"], &lock); err != nil {
		t.Fatal(err)
	}
	modes := map[string]string{}
	for _, f := range lock.Files {
		modes[f.Path] = f.Mode
	}
	return modes
}

func TestAuthCognitoFillsTheGuardAndDeclaresServicesAuth(t *testing.T) {
	res, err := composeAuth(t, "auth-cognito")
	if err != nil {
		t.Fatalf("auth-cognito does not compose in the default shell: %v", composer.Problems(err))
	}

	requireOneContractWithMe(t, res)
	providers := string(res.Files["web/src/lib/shell/providers.generated.ts"])
	for _, want := range []string{"const { requireSession } = scope;", "checkGuards([requireSession])"} {
		if !strings.Contains(providers, want) {
			t.Errorf("providers.generated.ts lacks %q:\n%s", want, providers)
		}
	}
	if !strings.Contains(string(res.Files["web/src/lib/shell/scope.d/auth-cognito.ts"]), "requireSession") {
		t.Error("scope.d/auth-cognito.ts must export requireSession")
	}

	yaml := string(res.Files["albitor.yaml"])
	for _, want := range []string{"auth:", `provider: "cognito"`, `posture: "private"`, `sign_up: "open"`} {
		if !strings.Contains(yaml, want) {
			t.Errorf("albitor.yaml lacks %q:\n%s", want, yaml)
		}
	}
	// No pool ids, ARNs or Terraform: the deploy target realises the pool from services.auth.
	for path := range res.Files {
		if strings.HasSuffix(path, ".tf") || strings.HasPrefix(path, "infrastructure/") {
			t.Errorf("auth-cognito must ship no infrastructure, found %s", path)
		}
	}

	modes := lockModes(t, res)
	managed := []string{
		"api/internal/identity/identity.go",
		"api/internal/shell/identity_verify.go",
		"api/internal/shell/identity_token.go",
		"api/internal/shell/identity_cognito.go",
		"web/src/lib/auth/session.ts",
		"web/src/lib/auth/guard.ts",
		"web/src/lib/auth/client.ts",
		"web/src/lib/shell/scope.d/auth-cognito.ts",
	}
	for _, p := range managed {
		if modes[p] != "example" {
			t.Errorf("%s should be an example (seeded once, app-owned), got %q", p, modes[p])
		}
	}
	screens := []string{
		"web/src/routes/(auth)/sign-in/+page.svelte",
		"web/src/routes/(auth)/sign-up/+page.svelte",
		"web/src/routes/(auth)/confirm/+page.svelte",
		"web/src/routes/(auth)/reset/+page.svelte",
		"web/src/routes/(auth)/sign-out/+page.svelte",
		"web/src/routes/(auth)/sign-out/nav.ts",
		"web/tests/journeys/sign-in.spec.ts", // a spec author's path: seeded, never pinned
	}
	for _, p := range screens {
		if modes[p] != "example" {
			t.Errorf("%s should be an app-owned example, got %q", p, modes[p])
		}
	}
}

// TestAuthCognitoScreensComposeOnEveryComponentKit2Kit: the screens are written against the
// component-kit@2 standard exports, not a named kit, so every kit that provides that major gets
// the same screens, and a kit that provides only component-kit@1 is refused (no implicit
// compatibility). kit-carbon is the non-GDS proof beside kit-shadcn and kit-gds.
func TestAuthCognitoScreensComposeOnEveryComponentKit2Kit(t *testing.T) {
	root := repoRoot(t)
	peers := authPeers(t)
	for _, kit := range []string{"kit-shadcn", "kit-carbon", "kit-gds"} {
		t.Run(kit, func(t *testing.T) {
			plugins := []composer.Plugin{
				loadDir(t, authDir(t, "auth-cognito")),
				loadDir(t, filepath.Join(root, "plugins", kit)),
				peers["web-svelte"], peers["api-go"], peers["test-playwright"], peers["deploy-stub"],
			}
			out, err := composer.Compose(composer.Input{Plugins: plugins, Tokens: tokens})
			if err != nil {
				t.Fatalf("auth-cognito does not compose on %s: %v", kit, composer.Problems(err))
			}
			for _, p := range []string{
				"web/src/routes/(auth)/sign-in/+page.svelte",
				"web/src/routes/(auth)/sign-up/+page.svelte",
				"web/src/routes/(auth)/confirm/+page.svelte",
				"web/src/routes/(auth)/reset/+page.svelte",
				"web/src/routes/(auth)/sign-out/+page.svelte",
				"web/src/lib/kit/index.ts",
			} {
				if _, ok := out.Files[p]; !ok {
					t.Errorf("%s: %s was not written", kit, p)
				}
			}
			// The kit's own barrel exports what the sign-in screen imports.
			index := string(out.Files["web/src/lib/kit/index.ts"])
			signIn := string(out.Files["web/src/routes/(auth)/sign-in/+page.svelte"])
			if !strings.Contains(signIn, "SignInPage") {
				t.Error("the sign-in screen must render the kit's SignInPage")
			}
			for _, name := range []string{"SignInPage", "Field", "Button"} {
				if !strings.Contains(index, name) {
					t.Errorf("%s's index.ts does not export %s, which the screens import", kit, name)
				}
			}
			if modes := lockModes(t, out); modes["web/src/routes/(auth)/sign-in/+page.svelte"] != "example" {
				t.Errorf("the sign-in screen should be an app-owned example, got %q", modes["web/src/routes/(auth)/sign-in/+page.svelte"])
			}
		})
	}
}

// TestAuthCognitoNamesNoKit: no condition in the manifest and no screen mentions a kit by name
// (the plan's genericity check), and the plugin requires the exact major.
func TestAuthCognitoNamesNoKit(t *testing.T) {
	p := loadDir(t, authDir(t, "auth-cognito"))
	for _, f := range p.Manifest.Contributes.Files {
		for _, sel := range f.When {
			if sel.Socket == core.SocketComponentKit {
				t.Errorf("%s is conditional on a component-kit (%s)", f.Path, sel.Plugin)
			}
		}
	}
	found := false
	for _, r := range p.Manifest.Requires {
		if r.Socket == core.SocketComponentKit && r.Major == 2 {
			found = true
		}
	}
	if !found {
		t.Errorf("auth-cognito must require component-kit@2, requires %v", p.Manifest.Requires)
	}
	err := filepath.WalkDir(filepath.Join(authDir(t, "auth-cognito"), "templates", "web", "src", "routes"), func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		b, rerr := os.ReadFile(path)
		if rerr != nil {
			return rerr
		}
		if m := regexp.MustCompile(`(?i)kit-(shadcn|carbon|gds)`).Find(b); m != nil {
			t.Errorf("%s names a kit (%s)", path, m)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

// TestAuthCognitoRefusesAComponentKit1OnlyKit: a kit that provides only major 1 does not satisfy
// auth-cognito's requires; there is no implicit compatibility.
func TestAuthCognitoRefusesAComponentKit1OnlyKit(t *testing.T) {
	root := repoRoot(t)
	peers := authPeers(t)
	plugins := []composer.Plugin{
		loadDir(t, authDir(t, "auth-cognito")),
		loadDir(t, filepath.Join(root, "tests", "compose", "stubs", "kit-stub")),
		peers["web-svelte"], peers["api-go"], peers["test-playwright"], peers["deploy-stub"],
	}
	_, err := composer.Compose(composer.Input{Plugins: plugins, Tokens: tokens})
	if err == nil {
		t.Fatal("auth-cognito composed with a kit that provides only component-kit@1")
	}
	found := false
	for _, p := range composer.Problems(err) {
		if p.Code == composer.CodeUnmetRequires && strings.Contains(p.Message, "auth-cognito requires component-kit@2") {
			found = true
		}
	}
	if !found {
		t.Fatalf("want an unmet_requires problem for auth-cognito's component-kit@2, got: %v", composer.Problems(err))
	}
}

// TestAuthCognitoNeedsAnAPI: a web-only profile (no api-runtime) uses auth-none.
func TestAuthCognitoNeedsAnAPI(t *testing.T) {
	webOnly := []string{"web-svelte", "kit-shadcn", "test-playwright", "deploy-stub"}
	if _, err := composeAuth(t, "auth-cognito", webOnly...); err == nil {
		t.Fatal("auth-cognito composed with no api-runtime")
	}
	res, err := composeAuth(t, "auth-none", webOnly...)
	if err != nil {
		t.Fatalf("auth-none does not compose in a web-only profile: %v", composer.Problems(err))
	}
	for path := range res.Files {
		if strings.HasPrefix(path, "api/") {
			t.Errorf("a web-only composition wrote %s", path)
		}
	}
}

func TestAuthNoneShipsOnlyTheFixedOwnerSeam(t *testing.T) {
	res, err := composeAuth(t, "auth-none")
	if err != nil {
		t.Fatalf("auth-none does not compose in the default shell: %v", composer.Problems(err))
	}
	modes := lockModes(t, res)
	for _, p := range []string{"api/internal/identity/identity.go", "api/internal/identity/identity_test.go"} {
		if modes[p] != "example" {
			t.Errorf("%s should be an example (seeded once, app-owned), got %q", p, modes[p])
		}
	}
	// No screens, no guard, no middleware, no routes, no services.auth.
	providers := string(res.Files["web/src/lib/shell/providers.generated.ts"])
	if !strings.Contains(providers, "checkGuards([])") {
		t.Errorf("auth-none must not fill the route guard:\n%s", providers)
	}
	if strings.Contains(string(res.Files["albitor.yaml"]), "auth:") {
		t.Errorf("auth-none must not declare services.auth:\n%s", res.Files["albitor.yaml"])
	}
	for path := range res.Files {
		if strings.HasPrefix(path, "web/src/routes/(auth)") || strings.HasPrefix(path, "web/src/lib/auth") ||
			strings.HasPrefix(path, "api/internal/shell/identity") {
			t.Errorf("auth-none wrote %s", path)
		}
	}
}

// TestAuthPluginsStayInTheirDirectories pins the path discipline: neither writes under .github/,
// and neither contributes a file outside identity, auth and the sign-in screens.
func TestAuthPluginsStayInTheirDirectories(t *testing.T) {
	allowed := []string{
		"api/internal/identity/", "api/internal/shell/identity_", "web/src/lib/auth/",
		"web/src/lib/shell/scope.d/auth-cognito.ts", "web/src/routes/(auth)/", "web/tests/journeys/sign-in.spec.ts",
		"web/tests/auth/provider.ts", "api/openapi.yaml",
	}
	for _, name := range []string{"auth-cognito", "auth-none"} {
		p := loadDir(t, authDir(t, name))
		for _, f := range p.Manifest.Contributes.Files {
			ok := false
			for _, prefix := range allowed {
				ok = ok || strings.HasPrefix(f.Path, prefix)
			}
			if !ok {
				t.Errorf("%s contributes %q outside its own directories", name, f.Path)
			}
		}
		err := filepath.WalkDir(authDir(t, name), func(path string, d os.DirEntry, err error) error {
			if err == nil && strings.Contains(filepath.ToSlash(path), "/.github") {
				t.Errorf("%s must not carry .github/ files: %s", name, path)
			}
			return err
		})
		if err != nil {
			t.Fatal(err)
		}
	}
}

// TestAuthCognitoWritesTreeForBuild writes the composed default shell with auth-cognito, once per
// kit, for the build step: COMPOSE_OUT_AUTH (kit-shadcn), COMPOSE_OUT_AUTH_CARBON and
// COMPOSE_OUT_AUTH_GDS, each when set.
func TestAuthCognitoWritesTreeForBuild(t *testing.T) {
	for kit, env := range map[string]string{
		"kit-shadcn": "COMPOSE_OUT_AUTH",
		"kit-carbon": "COMPOSE_OUT_AUTH_CARBON",
		"kit-gds":    "COMPOSE_OUT_AUTH_GDS",
	} {
		t.Run(kit, func(t *testing.T) {
			out := os.Getenv(env)
			if out == "" {
				t.Skipf("%s not set", env)
			}
			res, err := composeAuth(t, "auth-cognito", "web-svelte", kit, "api-go", "test-playwright", "deploy-stub")
			if err != nil {
				t.Fatal(composer.Problems(err))
			}
			for path, content := range res.Files {
				dst := filepath.Join(out, filepath.FromSlash(path))
				if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(dst, content, 0o644); err != nil {
					t.Fatal(err)
				}
			}
		})
	}
}

// TestAuthPluginsShipTheAuthV2Provider pins auth@2: every auth plugin ships the seeded
// web/tests/auth/provider.ts that test-playwright's specs and setup call, naming the sign-in it
// ships; auth-none's is the one with no roles.
func TestAuthPluginsShipTheAuthV2Provider(t *testing.T) {
	for name, shape := range map[string]string{"auth-cognito": "in-app", "auth-hosted": "hosted", "auth-none": "none"} {
		p := loadDir(t, authDir(t, name))
		found := false
		for _, f := range p.Manifest.Contributes.Files {
			if f.Path == "web/tests/auth/provider.ts" {
				found = true
				if f.Mode != "example" || len(f.When) != 1 || string(f.When[0].Socket) != "test-harness" || f.When[0].Plugin != "test-playwright" {
					t.Errorf("%s: provider.ts must be an example and only with test-playwright, got %+v", name, f)
				}
			}
		}
		if !found {
			t.Errorf("%s does not contribute web/tests/auth/provider.ts", name)
		}
		src, err := os.ReadFile(filepath.Join(authDir(t, name), "templates", "web", "tests", "auth", "provider.ts"))
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		want := `createAuthV2({ shape: "` + shape + `" })`
		if !strings.Contains(string(src), want) || !strings.Contains(string(src), "export const auth") {
			t.Errorf("%s: provider.ts must export auth = %s", name, want)
		}
	}
}
