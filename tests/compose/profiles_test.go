package compose

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"albitor/core"
	"albitor/core/authtemplate"
	"albitor/core/certification"
	"albitor/core/composer"
)

// profileTokens are every platform token a composition on either lane can name, with sentinel
// values that pass their patterns (the platform's certificationTokens do the same).
func profileTokens(layer string) authtemplate.Values {
	return authtemplate.Values{
		authtemplate.TokenProjectName: "plantrain",
		authtemplate.TokenRepo:        "acme/plantrain",
		composer.TokenRegion:          "eu-west-2",
		composer.TokenDNSMode:         "none",
		composer.TokenTFStateBucket:   "albitor-tfstate-123456789012",
		composer.TokenTFStateKey:      "plantrain/dev/terraform.tfstate",
		composer.TokenDeployRoleARN:   "arn:aws:iam::123456789012:role/plantrain-dev",
		composer.TokenAccountID:       "123456789012",
		composer.TokenTenantID:        "tenant",
		composer.TokenTeamID:          "team",
		composer.TokenAppID:           "app",
		composer.TokenEnvironment:     "dev",
		composer.TokenPipelineTag:     "v0.0.0",
		composer.TokenDataLayer:       layer,
	}
}

// repoSocketPlugins loads every schema-2 socket plugin in plugins/, by name.
func repoSocketPlugins(t *testing.T) map[string]composer.Plugin {
	t.Helper()
	root := repoRoot(t)
	manifests, err := filepath.Glob(filepath.Join(root, "plugins", "*", ".claude-plugin", "plugin.json"))
	if err != nil {
		t.Fatal(err)
	}
	out := map[string]composer.Plugin{}
	for _, path := range manifests {
		raw, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		var probe struct {
			Schema int    `json:"schema"`
			Kind   string `json:"kind"`
		}
		if err := json.Unmarshal(raw, &probe); err != nil || probe.Schema != 2 || probe.Kind != "socket" {
			continue
		}
		p := loadDir(t, filepath.Dir(filepath.Dir(path)))
		out[p.Manifest.Name] = p
	}
	return out
}

func composeMembers(t *testing.T, plugins map[string]composer.Plugin, members []certification.Member, layer string) error {
	t.Helper()
	var set []composer.Plugin
	for _, m := range members {
		p, ok := plugins[m.Plugin]
		if !ok {
			t.Fatalf("profile member %s (%s) is not a plugin in this repository", m.Plugin, m.Socket)
		}
		set = append(set, p)
	}
	res, err := composer.Compose(composer.Input{Plugins: set, Tokens: profileTokens(layer)})
	if err != nil {
		return err
	}
	for path, body := range res.Files {
		if strings.Contains(string(body), "{{albitor.") {
			t.Errorf("%s is composed with an unrendered {{albitor.*}} placeholder", path)
		}
	}
	return nil
}

// TestEveryCertifiedProfileComposesFromTheRealPlugins composes each of the platform's certified
// profiles (certification.Profiles: default, byoa-aws) from this repository's own plugin trees,
// loaded as the platform loads a pinned version, with both data layers, and then each profile with
// every other plugin here that fills one of its sockets in its member's place. Every one must
// compose with no problem: a slot fragment no host template places (no_slot_template), a sidecar
// the platform never reads, a token the platform does not own or an unmet requirement fails here
// before it can fail a build's compose (as dev build 63716a5c did).
func TestEveryCertifiedProfileComposesFromTheRealPlugins(t *testing.T) {
	plugins := repoSocketPlugins(t)
	bySocket := map[core.SocketName][]string{}
	for name, p := range plugins {
		for _, r := range p.Manifest.Provides {
			bySocket[r.Socket] = append(bySocket[r.Socket], name)
		}
	}
	for _, names := range bySocket {
		sort.Strings(names)
	}
	for _, prof := range certification.Profiles() {
		for _, layer := range []string{composer.DataLayerOnline, composer.DataLayerOffline} {
			if err := composeMembers(t, plugins, prof.Members, layer); err != nil {
				t.Errorf("profile %s (%s): %v", prof.Name, layer, composer.Problems(err))
			}
		}
		for i, m := range prof.Members {
			for _, alt := range bySocket[m.Socket] {
				if alt == m.Plugin {
					continue
				}
				members := append([]certification.Member(nil), prof.Members...)
				members[i] = certification.Member{Socket: m.Socket, Plugin: alt}
				if err := composeMembers(t, plugins, members, composer.DataLayerOnline); err != nil {
					t.Errorf("profile %s with %s for %s: %v", prof.Name, alt, m.Socket, composer.Problems(err))
				}
			}
		}
	}
}
