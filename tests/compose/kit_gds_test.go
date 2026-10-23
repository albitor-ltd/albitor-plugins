package compose

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"albitor/core/composer"
	"albitor/core/governance"
)

const gds = "kit-gds"

// TestKitGDSPassesComponentKitConformance runs the platform's own certification harness over the
// plugin's snapshot (manifest parse, the component-kit@1 conformance suite, a compose and the
// floor gate against web-svelte). scripts/compose-check.sh --build builds, tests and audits the
// composed web.
func TestKitGDSPassesComponentKitConformance(t *testing.T) { kitPassesConformance(t, gds) }

// TestKitGDSFailsClosed pins that the harness can fail this plugin, so the test above cannot pass
// vacuously.
func TestKitGDSFailsClosed(t *testing.T) { kitFailsClosed(t, gds) }

// TestKitGDSComposesWithWebSvelte is the golden for the composed kit files, and writes the whole
// composed tree to COMPOSE_KIT_GDS_OUT for the build script.
func TestKitGDSComposesWithWebSvelte(t *testing.T) { kitComposes(t, gds, "COMPOSE_KIT_GDS_OUT") }

func TestKitGDSShipsExactlyWhatItDeclares(t *testing.T) { kitShipsExactly(t, gds) }

// TestKitGDSIsTheGOVUKKit: the manifest says what the platform keys on. Name and keywords make
// core.IsGOVUKDesignSystem recognise it, it is not themeable (GOV.UK's look is not overridable by a
// visual language), and it carries the two GOV.UK skills on the design-system capability.
func TestKitGDSIsTheGOVUKKit(t *testing.T) {
	m := loadDir(t, kitDir(t, gds)).Manifest
	if m.Themeable {
		t.Error("kit-gds must not be themeable")
	}
	raw, err := os.ReadFile(filepath.Join(kitDir(t, gds), ".claude-plugin", "plugin.json"))
	if err != nil {
		t.Fatal(err)
	}
	var meta struct {
		Keywords []string `json:"keywords"`
	}
	if err := json.Unmarshal(raw, &meta); err != nil {
		t.Fatal(err)
	}
	kw := strings.Join(meta.Keywords, " ")
	for _, want := range []string{"design-system", "gds", "component-kit"} {
		if !strings.Contains(kw, want) {
			t.Errorf("keywords lack %q: %v", want, meta.Keywords)
		}
	}
	for _, skill := range []string{"govuk-design-system", "govuk-accessibility"} {
		b, err := os.ReadFile(filepath.Join(kitDir(t, gds), "skills", skill, "SKILL.md"))
		if err != nil {
			t.Fatal(err)
		}
		s := string(b)
		if !strings.HasPrefix(s, "---\nname: "+skill+"\ndescription: ") {
			t.Errorf("%s: SKILL.md needs front matter naming it", skill)
		}
		if !strings.Contains(s, "metadata:\n  capability: design-system\n") {
			t.Errorf("%s: SKILL.md needs metadata.capability: design-system", skill)
		}
		if strings.Contains(s, "/gds:") {
			t.Errorf("%s: SKILL.md points at the retired /gds: commands", skill)
		}
	}
}

// TestKitGDSSetsRootClasses: the kit fills web.root-classes and web-svelte's app.html carries the
// GOV.UK template classes on <html> and <body>.
func TestKitGDSSetsRootClasses(t *testing.T) {
	res := composeKit(t, gds)
	html := string(res.Files["web/src/app.html"])
	for _, want := range []string{`<html lang="en" class="govuk-template">`, `<body class="govuk-template__body" `} {
		if !strings.Contains(html, want) {
			t.Errorf("app.html lacks %q:\n%s", want, html)
		}
	}
}

func TestKitGDSFileModes(t *testing.T) {
	res := composeKit(t, gds)
	var lock composer.Lock
	if err := json.Unmarshal(res.Files[".albitor/lock.json"], &lock); err != nil {
		t.Fatal(err)
	}
	modes := map[string]string{}
	for _, f := range lock.Files {
		modes[f.Path] = f.Mode
	}
	// The accessibility gate (contrast, axe, and the Showcase they render) is platform-owned; the
	// components are the app's to edit.
	for p, want := range map[string]string{
		"web/vitest.config.ts":             "managed",
		"web/src/lib/kit/contrast.test.ts": "managed",
		"web/src/lib/kit/kit.test.ts":      "managed",
		"web/src/lib/kit/dialog.test.ts":   "managed",
		"web/src/lib/kit/Showcase.svelte":  "managed",
		"web/src/lib/kit/AppShell.svelte":  "example",
		"web/src/lib/kit/Button.svelte":    "example",
		"web/src/lib/kit/Dialog.svelte":    "example",
		"web/src/lib/kit/Field.svelte":     "example",
		"web/src/lib/kit/Table.svelte":     "example",
		"web/src/lib/kit/govuk.scss":       "example",
		"web/src/lib/kit/index.ts":         "example",
	} {
		if modes[p] != want {
			t.Errorf("%s: mode %q, want %q", p, modes[p], want)
		}
	}
}

func TestKitGDSDependenciesAreExact(t *testing.T) {
	names := kitDependenciesAreExact(t, gds)
	for _, want := range []string{"govuk-frontend", "sass", "axe-core"} {
		if !names[want] {
			t.Errorf("the kit depends on %s", want)
		}
	}
	for _, d := range loadDir(t, kitDir(t, gds)).Manifest.Contributes.Dependencies.NPM {
		if strings.HasPrefix(d, "govuk-frontend@") && d != "govuk-frontend@6.4.0" {
			t.Errorf("the markup follows GOV.UK Frontend 6.4.0, the dependency is %s", d)
		}
	}
}

// commentRe matches the comments in the kit's Svelte, TypeScript and SCSS source, which say what
// the kit does not use and so name the banned words.
var commentRe = regexp.MustCompile(`(?s)<!--.*?-->|/\*.*?\*/|(?m)^\s*//.*$`)

// TestKitGDSHasNoCrownOrLogotype: the kit is for apps that are not on GOV.UK, so its code ships no
// Crown, crest, GOV.UK logotype or image, none of GOV.UK Frontend's branded header or footer, and
// not the GDS Transport font. Comments that say so are not uses.
func TestKitGDSHasNoCrownOrLogotype(t *testing.T) {
	for p, b := range snapshot(t, kitDir(t, gds)) {
		if !strings.HasPrefix(p, "templates/") || strings.HasSuffix(p, ".test.ts") {
			continue
		}
		code := strings.ToLower(commentRe.ReplaceAllString(string(b), ""))
		for _, banned := range []string{"crest", "crown", "logotype", "govuk-header", "govuk-footer", "<svg", "<img", "gds transport", "gov.uk"} {
			if strings.Contains(code, banned) {
				t.Errorf("%s uses %q outside a comment", p, banned)
			}
		}
	}
	if matches, _ := filepath.Glob(filepath.Join(kitDir(t, gds), "preview", "*")); len(matches) > 0 {
		t.Errorf("kit-gds ships no preview page, found %v", matches)
	}
}

// TestKitGDSPreviewTokens: the preview tokens pass the platform's own validator and carry GOV.UK
// Frontend 6.4.0's palette (text, secondary text, surface, brand blue), so the preview and the app
// cannot drift; the real colours are read from the shipped stylesheet in contrast.test.ts.
func TestKitGDSPreviewTokens(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join(kitDir(t, gds), governance.PreviewTokensPath))
	if err != nil {
		t.Fatal(err)
	}
	tok, err := governance.ParsePreviewTokens(raw)
	if err != nil {
		t.Fatalf("preview tokens rejected by the platform validator: %v", err)
	}
	for name, got := range map[string]string{
		"text":           tok.Colours.Text,
		"text_secondary": tok.Colours.TextSecondary,
		"surface":        tok.Colours.Surface,
		"background":     tok.Colours.Background,
		"accent":         tok.Colours.Accent,
		"on_accent":      tok.Colours.OnAccent,
	} {
		want := map[string]string{
			"text": "#0b0c0c", "text_secondary": "#484949", "surface": "#f4f8fb",
			"background": "#ffffff", "accent": "#1d70b8", "on_accent": "#ffffff",
		}[name]
		if strings.ToLower(got) != want {
			t.Errorf("preview colour %s is %s, GOV.UK Frontend 6.4.0 says %s", name, got, want)
		}
	}
	// roles (#3710): off GOV.UK the header is never the GOV.UK blue; that blue is the band only
	// where the brand colour is withdrawn, and the focus is the yellow fill with black text.
	r := tok.Roles
	if r == nil || r.Header == nil || r.HeaderWithdrawn == nil {
		t.Fatalf("roles lack a header and a withdrawn header: %+v", r)
	}
	if strings.EqualFold(r.Header.Background, "#1d70b8") {
		t.Errorf("roles.header is the GOV.UK blue %s; off GOV.UK the band must be neutral", r.Header.Background)
	}
	if !strings.EqualFold(r.HeaderWithdrawn.Background, "#1d70b8") {
		t.Errorf("roles.header_withdrawn is %s, want #1d70b8", r.HeaderWithdrawn.Background)
	}
	if !strings.EqualFold(r.Focus, "#ffdd00") || !strings.EqualFold(r.FocusInk, "#0b0c0c") {
		t.Errorf("roles focus = %s on %s, want #ffdd00 on #0b0c0c", r.Focus, r.FocusInk)
	}
}
