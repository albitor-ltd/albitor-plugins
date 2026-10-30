package compose

import (
	"strings"
	"testing"

	"albitor/core"
	"albitor/core/certification"
	"albitor/core/composer"
)

// composeUnderPack composes the members with one built-in policy pack mandated, as the platform
// does for a tenant that mandates it.
func composeUnderPack(t *testing.T, plugins map[string]composer.Plugin, members []certification.Member, pack composer.Plugin) (*composer.Result, error) {
	t.Helper()
	var set []composer.Plugin
	for _, m := range members {
		p, ok := plugins[m.Plugin]
		if !ok {
			t.Fatalf("%s (%s) is not a plugin in this repository", m.Plugin, m.Socket)
		}
		set = append(set, p)
	}
	set = append(set, pack)
	return composer.Compose(composer.Input{
		Plugins: set,
		Policy:  composer.MandatePacks(pack),
		Tokens:  profileTokens(composer.DataLayerOnline),
	})
}

// TestDefaultSelectionComposesUnderEachBuiltinPolicyPack composes the default profile's selection
// with each built-in pack (policy-internal-tool, policy-gds) mandated. The default selection has
// no visual-language provider, so neither pack may refuse it. The packs' content (their checks)
// is the build's EffectivePolicy, which the composer resolves apart from the tree.
func TestDefaultSelectionComposesUnderEachBuiltinPolicyPack(t *testing.T) {
	plugins := repoSocketPlugins(t)
	packs, err := composer.BuiltinPolicyPacks()
	if err != nil {
		t.Fatal(err)
	}
	def := certification.DefaultProfile()
	for _, pack := range packs {
		name := pack.Manifest.Name
		t.Run(name, func(t *testing.T) {
			if _, err := composeUnderPack(t, plugins, def.Members, pack); err != nil {
				t.Fatalf("the default selection does not compose under %s: %v", name, composer.Problems(err))
			}
			pol, err := composer.ResolveEffectivePolicy(pack.Manifest.Policy.ID, nil)
			if err != nil {
				t.Fatalf("%s: effective policy: %v", name, err)
			}
			if len(pol.Checks) == 0 {
				t.Errorf("%s carries no checks", name)
			}
			if pack.Manifest.Policy.ID == "internal-tool" {
				var axe bool
				for _, c := range pol.Checks {
					axe = axe || c.ID == core.CheckAxeWCAGScan
				}
				if !axe {
					t.Errorf("internal tool checks lack %s: %v", core.CheckAxeWCAGScan, pol.Checks)
				}
			}
		})
	}
}

// TestVisualLanguageUnderEachBuiltinPolicyPack adds each optional visual-language provider to the
// default selection. Internal Tool states no socket constraint; GDS forbids the socket, and that
// refusal, naming policy-gds, is the only problem.
func TestVisualLanguageUnderEachBuiltinPolicyPack(t *testing.T) {
	plugins := repoSocketPlugins(t)
	packs, err := composer.BuiltinPolicyPacks()
	if err != nil {
		t.Fatal(err)
	}
	def := certification.DefaultProfile()
	for _, look := range def.Optional {
		members := append(append([]certification.Member(nil), def.Members...), look)
		for _, pack := range packs {
			name := pack.Manifest.Name
			t.Run(look.Plugin+"/"+name, func(t *testing.T) {
				_, err := composeUnderPack(t, plugins, members, pack)
				probs := composer.Problems(err)
				if name == composer.PackInternalTool {
					if err != nil {
						t.Fatalf("internal tool refuses %s: %v", look.Plugin, probs)
					}
					return
				}
				if len(probs) != 1 || probs[0].Code != composer.CodePolicy ||
					!strings.Contains(probs[0].Message, "visual-language") || probs[0].Plugins[0] != composer.PackGDS {
					t.Fatalf("gds must refuse %s with the visual-language rule and nothing else: %v (%v)", look.Plugin, probs, err)
				}
			})
		}
	}
}
