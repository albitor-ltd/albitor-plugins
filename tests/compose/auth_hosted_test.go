package compose

import (
	"context"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"albitor/core"
	"albitor/core/certification"
	"albitor/core/composer"
)

// TestAuthHostedConforms runs the platform's own certification harness over auth-hosted: the
// manifest through the real parser, the auth@1 conformance suite, then a compose and the floor gate
// in the default profile.
func TestAuthHostedConforms(t *testing.T) {
	h := &certification.Harness{
		Tokens:   tokens,
		Profiles: []certification.Profile{authProfile(true)},
		Peers:    authPeers(t),
		Suites:   certification.DefaultSuites(),
	}
	cert, err := h.Certify(context.Background(), snapshot(t, authDir(t, "auth-hosted")))
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
}

// TestAuthHostedFailsClosed pins that the harness can fail the plugin: a declared file that is not
// shipped is a failed verdict, so the test above cannot pass vacuously.
func TestAuthHostedFailsClosed(t *testing.T) {
	missing := "templates/api/internal/shell/identity_hosted.go"
	files := snapshot(t, authDir(t, "auth-hosted"))
	if _, ok := files[missing]; !ok {
		t.Fatalf("auth-hosted does not ship %s; the case is stale", missing)
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
}

func TestAuthHostedFillsTheGuardAndDeclaresServicesAuth(t *testing.T) {
	res, err := composeAuth(t, "auth-hosted")
	if err != nil {
		t.Fatalf("auth-hosted does not compose in the default shell: %v", composer.Problems(err))
	}
	providers := string(res.Files["web/src/lib/shell/providers.generated.ts"])
	for _, want := range []string{"const { requireSession } = scope;", "checkGuards([requireSession])"} {
		if !strings.Contains(providers, want) {
			t.Errorf("providers.generated.ts lacks %q:\n%s", want, providers)
		}
	}
	if !strings.Contains(string(res.Files["web/src/lib/shell/scope.d/auth-hosted.ts"]), "requireSession") {
		t.Error("scope.d/auth-hosted.ts must export requireSession")
	}
	yaml := string(res.Files["albitor.yaml"])
	for _, want := range []string{"auth:", `provider: "cognito"`, `posture: "private"`, `sign_up: "open"`} {
		if !strings.Contains(yaml, want) {
			t.Errorf("albitor.yaml lacks %q:\n%s", want, yaml)
		}
	}
	for path := range res.Files {
		if strings.HasSuffix(path, ".tf") || strings.HasPrefix(path, "infrastructure/") {
			t.Errorf("auth-hosted must ship no infrastructure, found %s", path)
		}
	}

	// The router is rendered with this plugin's slots.
	router := string(res.Files["api/internal/shell/router.go"])
	for _, want := range []string{"middlewareOf(IdentityVerify)", "routesOf(IdentityRoutes)"} {
		if !strings.Contains(router, want) {
			t.Errorf("router.go lacks %q", want)
		}
	}

	modes := lockModes(t, res)
	for _, p := range []string{
		"api/internal/identity/identity.go",
		"api/internal/shell/identity_verify.go",
		"api/internal/shell/identity_token.go",
		"api/internal/shell/identity_hosted.go",
		"web/src/lib/auth/session.ts",
		"web/src/lib/auth/guard.ts",
		"web/src/lib/auth/client.ts",
		"web/src/lib/shell/scope.d/auth-hosted.ts",
	} {
		if modes[p] != "managed" {
			t.Errorf("%s should be managed, got %q", p, modes[p])
		}
	}
	for _, p := range []string{
		"web/src/routes/(auth)/sign-in/+page.svelte",
		"web/src/routes/(auth)/sign-out/+page.svelte",
		"web/src/routes/(auth)/sign-out/nav.ts",
		"web/src/routes/(auth)/not-allowed/+page.svelte",
		"web/src/routes/(auth)/account/+page.svelte",
		"web/src/routes/(auth)/account/nav.ts",
		"web/src/routes/(auth)/account/delete/+page.svelte",
		"web/tests/journeys/sign-in.spec.ts",
	} {
		if modes[p] != "example" {
			t.Errorf("%s should be an app-owned example, got %q", p, modes[p])
		}
	}
}

// TestAuthHostedScreensComposeOnEveryComponentKit2Kit: the screens use the component-kit@2 exports
// only, so every kit that provides that major gets them.
func TestAuthHostedScreensComposeOnEveryComponentKit2Kit(t *testing.T) {
	root := repoRoot(t)
	peers := authPeers(t)
	for _, kit := range []string{"kit-shadcn", "kit-carbon", "kit-gds"} {
		t.Run(kit, func(t *testing.T) {
			out, err := composer.Compose(composer.Input{Plugins: []composer.Plugin{
				loadDir(t, authDir(t, "auth-hosted")),
				loadDir(t, filepath.Join(root, "plugins", kit)),
				peers["web-svelte"], peers["api-go"], peers["test-playwright"], peers["deploy-stub"],
			}, Tokens: tokens})
			if err != nil {
				t.Fatalf("auth-hosted does not compose on %s: %v", kit, composer.Problems(err))
			}
			index := string(out.Files["web/src/lib/kit/index.ts"])
			if !strings.Contains(index, "Button") {
				t.Errorf("%s's index.ts does not export Button, which the screens import", kit)
			}
		})
	}
}

// TestAuthHostedRequiresWhatItUses: component-kit@2 (the screens import its exports), an API, and
// a deploy target (which realises the pool from services.auth); and it is not conditional on a kit.
func TestAuthHostedRequiresWhatItUses(t *testing.T) {
	p := loadDir(t, authDir(t, "auth-hosted"))
	want := map[core.SocketName]bool{core.SocketComponentKit: false, core.SocketAPIRuntime: false, core.SocketDeployTarget: false, core.SocketWebFramework: false}
	for _, r := range p.Manifest.Requires {
		if _, ok := want[r.Socket]; ok && (r.Socket != core.SocketComponentKit || r.Major == 2) {
			want[r.Socket] = true
		}
	}
	for socket, ok := range want {
		if !ok {
			t.Errorf("auth-hosted does not require %s", socket)
		}
	}
	for _, f := range p.Manifest.Contributes.Files {
		for _, sel := range f.When {
			if sel.Socket == core.SocketComponentKit {
				t.Errorf("%s is conditional on a component-kit (%s)", f.Path, sel.Plugin)
			}
		}
	}
	if _, err := composeAuth(t, "auth-hosted", "web-svelte", "kit-shadcn", "test-playwright", "deploy-stub"); err == nil {
		t.Error("auth-hosted composed with no api-runtime")
	}
	if _, err := composeAuth(t, "auth-hosted", "web-svelte", "kit-shadcn", "api-go", "test-playwright"); err == nil {
		t.Error("auth-hosted composed with no deploy target")
	}
}

// TestAuthHostedHasNoPasswordFieldAndNamesNoKit: the app never sees a password, and no screen
// names a kit.
func TestAuthHostedHasNoPasswordFieldAndNamesNoKit(t *testing.T) {
	err := filepath.WalkDir(filepath.Join(authDir(t, "auth-hosted"), "templates", "web", "src"), func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		b, rerr := os.ReadFile(path)
		if rerr != nil {
			return rerr
		}
		if strings.HasSuffix(path, ".test.ts") {
			return nil
		}
		if m := regexp.MustCompile(`(?i)type=["']password["']|autocomplete=["'](new-|current-)password`).Find(b); m != nil {
			t.Errorf("%s has a password field (%s)", path, m)
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

func TestAuthHostedStaysInItsDirectories(t *testing.T) {
	allowed := []string{
		"api/internal/identity/", "api/internal/shell/identity_", "web/src/lib/auth/",
		"web/src/lib/shell/scope.d/auth-hosted.ts", "web/src/routes/(auth)/", "web/tests/journeys/sign-in.spec.ts",
	}
	p := loadDir(t, authDir(t, "auth-hosted"))
	for _, f := range p.Manifest.Contributes.Files {
		ok := false
		for _, prefix := range allowed {
			ok = ok || strings.HasPrefix(f.Path, prefix)
		}
		if !ok {
			t.Errorf("auth-hosted contributes %q outside its own directories", f.Path)
		}
	}
	err := filepath.WalkDir(authDir(t, "auth-hosted"), func(path string, d os.DirEntry, err error) error {
		if err == nil && strings.Contains(filepath.ToSlash(path), "/.github") {
			t.Errorf("auth-hosted must not carry .github/ files: %s", path)
		}
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
}

// TestAuthHostedWritesTreeForBuild writes the composed default shell with auth-hosted, once per
// kit, for the build step: COMPOSE_OUT_AUTH_HOSTED (kit-shadcn), COMPOSE_OUT_AUTH_HOSTED_CARBON and
// COMPOSE_OUT_AUTH_HOSTED_GDS, each when set.
func TestAuthHostedWritesTreeForBuild(t *testing.T) {
	for kit, env := range map[string]string{
		"kit-shadcn": "COMPOSE_OUT_AUTH_HOSTED",
		"kit-carbon": "COMPOSE_OUT_AUTH_HOSTED_CARBON",
		"kit-gds":    "COMPOSE_OUT_AUTH_HOSTED_GDS",
	} {
		t.Run(kit, func(t *testing.T) {
			out := os.Getenv(env)
			if out == "" {
				t.Skipf("%s not set", env)
			}
			res, err := composeAuth(t, "auth-hosted", "web-svelte", kit, "api-go", "test-playwright", "deploy-stub")
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
