package compose

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"albitor/core/composer"
	"albitor/core/governance"
)

const carbon = "kit-carbon"

// TestKitCarbonPassesComponentKitConformance runs the platform's own certification harness over
// the plugin's snapshot (manifest parse, the component-kit@1 conformance suite, a compose and the
// floor gate against web-svelte). scripts/compose-check.sh --build builds, tests and audits the
// composed web.
func TestKitCarbonPassesComponentKitConformance(t *testing.T) { kitPassesConformance(t, carbon) }

// TestKitCarbonFailsClosed pins that the harness can fail this plugin, so the test above cannot
// pass vacuously.
func TestKitCarbonFailsClosed(t *testing.T) { kitFailsClosed(t, carbon) }

// TestKitCarbonComposesWithWebSvelte is the golden for the composed kit files, and writes the whole
// composed tree to COMPOSE_KIT_CARBON_OUT for the build script.
func TestKitCarbonComposesWithWebSvelte(t *testing.T) {
	kitComposes(t, carbon, "COMPOSE_KIT_CARBON_OUT")
}

// TestKitCarbonHasTheSameShellPiecesAsKitShadcn: a build swaps one kit for the other, so the pieces
// the root layout and the pages import must exist under the same names and paths.
func TestKitCarbonHasTheSameShellPiecesAsKitShadcn(t *testing.T) {
	paths := func(kit string) map[string]bool {
		out := map[string]bool{}
		for _, f := range loadDir(t, kitDir(t, kit)).Manifest.Contributes.Files {
			out[f.Path] = true
		}
		return out
	}
	shadcn, carbonPaths := paths("kit-shadcn"), paths(carbon)
	for _, p := range []string{
		"web/src/lib/kit/AppShell.svelte", "web/src/lib/kit/Button.svelte",
		"web/src/lib/kit/Dialog.svelte", "web/src/lib/kit/EmptyState.svelte",
		"web/src/lib/kit/Field.svelte", "web/src/lib/kit/Table.svelte",
		"web/src/lib/kit/Toaster.svelte", "web/src/lib/kit/toast.svelte.ts",
		"web/src/lib/kit/index.ts", "web/src/lib/kit/Showcase.svelte",
		"web/src/lib/kit/kit.test.ts", "web/src/lib/kit/dialog.test.ts",
		"web/src/lib/kit/contrast.test.ts", "web/vitest.config.ts",
	} {
		if !shadcn[p] {
			t.Fatalf("test list is stale: kit-shadcn does not ship %s", p)
		}
		if !carbonPaths[p] {
			t.Errorf("kit-carbon does not ship %s, which kit-shadcn does", p)
		}
	}
	// The barrel exports the same names.
	exports := func(kit string) string {
		b, err := os.ReadFile(filepath.Join(kitDir(t, kit), "templates", "web", "src", "lib", "kit", "index.ts"))
		if err != nil {
			t.Fatal(err)
		}
		return string(b)
	}
	for _, name := range []string{"AppShell", "Button", "Dialog", "EmptyState", "Field", "Table", "Toaster", "toast", "Column", "Toast", "ToastTone"} {
		if !strings.Contains(exports("kit-shadcn"), name) {
			t.Fatalf("test list is stale: kit-shadcn's index.ts has no %s", name)
		}
		if !strings.Contains(exports(carbon), name) {
			t.Errorf("kit-carbon's index.ts does not export %s", name)
		}
	}
}

func TestKitCarbonFileModes(t *testing.T) {
	res := composeKit(t, carbon)
	var lock composer.Lock
	if err := json.Unmarshal(res.Files[".albitor/lock.json"], &lock); err != nil {
		t.Fatal(err)
	}
	modes := map[string]string{}
	for _, f := range lock.Files {
		modes[f.Path] = f.Mode
	}
	// The accessibility gate (contrast, axe, and the Showcase they render) is platform-owned; the
	// components and the theme choice are the app's to edit.
	for p, want := range map[string]string{
		"web/vitest.config.ts":              "managed",
		"web/src/lib/kit/contrast.test.ts":  "managed",
		"web/src/lib/kit/kit.test.ts":       "managed",
		"web/src/lib/kit/dialog.test.ts":    "managed",
		"web/src/lib/kit/shell.test.ts":     "managed",
		"web/src/lib/kit/Showcase.svelte":   "managed",
		"web/src/lib/kit/AppShell.svelte":   "example",
		"web/src/lib/kit/Button.svelte":     "example",
		"web/src/lib/kit/Dialog.svelte":     "example",
		"web/src/lib/kit/Field.svelte":      "example",
		"web/src/lib/kit/Table.svelte":      "example",
		"web/src/lib/kit/Toaster.svelte":    "example",
		"web/src/lib/kit/EmptyState.svelte": "example",
		"web/src/lib/kit/toast.svelte.ts":   "example",
		"web/src/lib/kit/theme.ts":          "example",
	} {
		if modes[p] != want {
			t.Errorf("%s: mode %q, want %q", p, modes[p], want)
		}
	}
}

func TestKitCarbonShipsExactlyWhatItDeclares(t *testing.T) { kitShipsExactly(t, carbon) }

func TestKitCarbonDependenciesAreExact(t *testing.T) {
	names := kitDependenciesAreExact(t, carbon)
	// @ibm/telemetry-js is carbon-components-svelte's postinstall; it is pinned here so code that
	// runs at install does not float to whatever is newest.
	for _, want := range []string{"carbon-components-svelte", "carbon-icons-svelte", "@ibm/telemetry-js"} {
		if !names[want] {
			t.Errorf("the kit depends on %s", want)
		}
	}
	if names["bits-ui"] {
		t.Error("kit-carbon builds on Carbon's own components, not Bits UI")
	}
}

// TestKitCarbonPreviewTokens: the preview tokens pass the platform's own validator and describe
// Carbon's white theme (the light theme the kit ships), so the preview and the app cannot drift.
func TestKitCarbonPreviewTokens(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join(kitDir(t, carbon), governance.PreviewTokensPath))
	if err != nil {
		t.Fatal(err)
	}
	tok, err := governance.ParsePreviewTokens(raw)
	if err != nil {
		t.Fatalf("preview tokens rejected by the platform validator: %v", err)
	}
	// Carbon white: text-primary, text-secondary, border-subtle, layer, background, button-primary.
	for name, pair := range map[string][2]string{
		"text":           {tok.Colours.Text, "#161616"},
		"text_secondary": {tok.Colours.TextSecondary, "#525252"},
		"border":         {tok.Colours.Border, "#e0e0e0"},
		"surface":        {tok.Colours.Surface, "#f4f4f4"},
		"background":     {tok.Colours.Background, "#ffffff"},
		"accent":         {tok.Colours.Accent, "#0f62fe"},
		"on_accent":      {tok.Colours.OnAccent, "#ffffff"},
	} {
		if strings.ToLower(pair[0]) != pair[1] {
			t.Errorf("preview colour %s is %s, Carbon white says %s", name, pair[0], pair[1])
		}
	}
}

func TestKitCarbonSkill(t *testing.T) {
	skill := filepath.Join(kitDir(t, carbon), "skills", "kit-carbon", "SKILL.md")
	b, err := os.ReadFile(skill)
	if err != nil {
		t.Fatal(err)
	}
	s := string(b)
	if !strings.HasPrefix(s, "---\nname: kit-carbon\ndescription: ") {
		t.Error("SKILL.md needs front matter naming kit-carbon")
	}
	// Svelte-first: the React and web-component advice of the legacy carbon plugin must not carry over.
	for _, banned := range []string{"@carbon/react", "@carbon/web-components", "use client", "Next.js"} {
		if strings.Contains(s, banned) {
			t.Errorf("SKILL.md carries advice for another stack: %q", banned)
		}
	}
	if _, err := os.Stat(filepath.Join(kitDir(t, carbon), "skills", "kit-carbon", "references", "wcag-2.2.md")); err == nil {
		t.Error("the WCAG reference lives in test-playwright; do not copy it here")
	}
}

// TestKitCarbonIsNotThemeable pins what "Carbon is not themeable by a visual-language pack" means
// today. component-kit@1 has no manifest marker for it (albitor-ltd/albitor#3236), so the kit says
// it three ways: it ships no `albitor-kit` cascade layer for an unlayered override to win against
// (kit-shadcn's convention), it styles from Carbon's own --cds-* properties, and the README and
// skill both say so, so a build agent and a reader are told not to pair it with a visual language.
func TestKitCarbonIsNotThemeable(t *testing.T) {
	for p, b := range snapshot(t, kitDir(t, carbon)) {
		if strings.Contains(string(b), "@layer albitor-kit") {
			t.Errorf("%s declares the albitor-kit layer, which is how a themeable kit says so", p)
		}
		if strings.HasSuffix(p, "tokens.css") {
			t.Errorf("%s: Carbon ships its own themes; the kit adds no token file", p)
		}
	}
	for _, p := range []string{"README.md", "skills/kit-carbon/SKILL.md"} {
		b, err := os.ReadFile(filepath.Join(kitDir(t, carbon), filepath.FromSlash(p)))
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(strings.ToLower(string(b)), "not themeable") {
			t.Errorf("%s must say Carbon is not themeable by a visual-language plugin", p)
		}
	}
}
