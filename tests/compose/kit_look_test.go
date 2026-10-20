package compose

import (
	"sort"
	"strings"
	"testing"

	"albitor/core"
	"albitor/core/certification"
	"albitor/core/composer"
)

// TestEveryKitComposesWithEveryLook is the kit x look matrix (albitor#3568). The platform places
// an app's chosen look on the visual-language socket (the look is re-theming the kit's tokens), so
// every component-kit plugin here is composed with every visual-language plugin here, on each
// certified profile, over the profile's other members:
//
//   - a kit that declares "themeable": true must compose with the look, with no problem and no
//     unrendered {{albitor.*}} placeholder, and the look's token file must land in the tree;
//   - a kit that does not declare it must be refused by the composer's kit_not_themeable check,
//     the same answer the platform gives before it ever places the look (lookMayApply), so a
//     look is never composed over a kit that cannot take it.
//
// A new kit or a new look joins the matrix by existing in plugins/; nothing here names one.
func TestEveryKitComposesWithEveryLook(t *testing.T) {
	plugins := repoSocketPlugins(t)
	providers := func(socket core.SocketName) []string {
		var names []string
		for name, p := range plugins {
			for _, r := range p.Manifest.Provides {
				if r.Socket == socket {
					names = append(names, name)
				}
			}
		}
		sort.Strings(names)
		return names
	}
	kits, looks := providers(core.SocketComponentKit), providers(core.SocketVisualLang)
	if len(kits) == 0 || len(looks) == 0 {
		t.Fatalf("matrix is empty: kits %v, looks %v", kits, looks)
	}
	for _, prof := range certification.Profiles() {
		for _, kit := range kits {
			for _, look := range looks {
				members := make([]certification.Member, 0, len(prof.Members)+1)
				for _, m := range prof.Members {
					if m.Socket == core.SocketComponentKit {
						m.Plugin = kit
					}
					members = append(members, m)
				}
				members = append(members, certification.Member{Socket: core.SocketVisualLang, Plugin: look})
				var set []composer.Plugin
				for _, m := range members {
					set = append(set, plugins[m.Plugin])
				}
				name := prof.Name + "/" + kit + "+" + look
				res, err := composer.Compose(composer.Input{Plugins: set, Tokens: profileTokens(composer.DataLayerOnline)})
				if plugins[kit].Manifest.Themeable {
					if err != nil {
						t.Errorf("%s: %v", name, composer.Problems(err))
						continue
					}
					for path, body := range res.Files {
						if strings.Contains(string(body), "{{albitor.") {
							t.Errorf("%s: %s has an unrendered placeholder", name, path)
						}
					}
					if _, ok := res.Files["web/src/lib/look/look.json"]; !ok {
						t.Errorf("%s: the look's token file is not in the composed tree", name)
					}
					continue
				}
				var refused bool
				for _, p := range composer.Problems(err) {
					refused = refused || p.Code == composer.CodeKitNotThemeable
				}
				if !refused {
					t.Errorf("%s: a kit that is not themeable composed with a look (err %v)", name, err)
				}
			}
		}
	}
}
