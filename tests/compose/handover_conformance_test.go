package compose

import (
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"albitor/core/certification"
	"albitor/core/composer"
)

// TestEveryCataloguePluginComposesHandoverClean composes every socket plugin in this repository
// into each certified profile, on both data layers, and runs the platform's handover scan
// (certification.HandoverFindings, the code ingest runs; the runner's differential test binds it
// to the build's gates/handover_reference.sh) over the composed tree. A token that survives into
// a delivered app and is not a named contract token fails here, before a tag, whichever plugin
// carries it (albitor-ltd/albitor#4517, #4505).
//
// A plugin the profile tests never place (object-s3 fills no profile socket) is composed as an
// addition to each profile's members, or in place of the member for its socket if the profile has
// one. The test is fail-closed: every plugin must be composed at least once, so a plugin cannot
// go unscanned by being left out of every combination.
func TestEveryCataloguePluginComposesHandoverClean(t *testing.T) {
	plugins := repoSocketPlugins(t)
	if len(plugins) < 10 {
		t.Fatalf("found only %d socket plugins; the catalogue walk is broken", len(plugins))
	}
	scanned := map[string]bool{}
	for _, prof := range certification.Profiles() {
		for name, p := range plugins {
			members := append([]certification.Member(nil), prof.Members...)
			for _, r := range p.Manifest.Provides {
				replaced := false
				for i, m := range members {
					if m.Socket == r.Socket {
						members[i] = certification.Member{Socket: r.Socket, Plugin: name}
						replaced = true
					}
				}
				if !replaced {
					members = append(members, certification.Member{Socket: r.Socket, Plugin: name})
				}
			}
			for _, layer := range []string{composer.DataLayerOnline, composer.DataLayerOffline} {
				var set []composer.Plugin
				for _, m := range members {
					set = append(set, plugins[m.Plugin])
				}
				res, err := composer.Compose(composer.Input{Plugins: set, Tokens: profileTokens(layer)})
				if err != nil {
					if declaredConflict(err) {
						continue // the plugin says it cannot share a tree with a member: no such app exists
					}
					t.Errorf("profile %s with %s (%s): %v", prof.Name, name, layer, composer.Problems(err))
					continue
				}
				for _, m := range members {
					scanned[m.Plugin] = true
				}
				for _, f := range certification.HandoverFindings(res.Files) {
					t.Errorf("profile %s with %s (%s): %s", prof.Name, name, layer, f)
				}
			}
		}
	}
	var missed []string
	for name := range plugins {
		if !scanned[name] {
			missed = append(missed, name)
		}
	}
	sort.Strings(missed)
	if len(missed) > 0 {
		t.Errorf("plugins never composed, so never scanned: %v", missed)
	}
}

// declaredConflict reports whether the compose failed only because a plugin declares a conflict
// with another in the set (and the unrendered services that follow from the excluded plugin), so
// the combination is one the catalogue itself rules out. Any other problem is a real failure.
func declaredConflict(err error) bool {
	probs := composer.Problems(err)
	conflict := false
	for _, p := range probs {
		switch p.Code {
		case composer.CodePluginConflict:
			conflict = true
		case composer.CodeManifestUnrendered:
		default:
			return false
		}
	}
	return conflict
}

// TestEveryPluginExampleTreeIsHandoverClean scans the example trees the plugins' skills ship
// (plugins/*/skills/*/examples), which an agent copies into the app and the composer never places,
// so the composed-tree test above cannot see them. Each tree is scanned as the app would hold it,
// with its path relative to the examples directory. The test is fail-closed: it requires the
// walk to find the example trees it knows exist.
func TestEveryPluginExampleTreeIsHandoverClean(t *testing.T) {
	root := repoRoot(t)
	dirs, err := filepath.Glob(filepath.Join(root, "plugins", "*", "skills", "*", "examples"))
	if err != nil {
		t.Fatal(err)
	}
	if len(dirs) < 5 {
		t.Fatalf("found only %d example trees; the walk is broken", len(dirs))
	}
	sawObjects := false
	for _, dir := range dirs {
		files := map[string][]byte{}
		err := filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
			if err != nil || d.IsDir() || !d.Type().IsRegular() {
				return err
			}
			body, err := os.ReadFile(p)
			if err != nil {
				return err
			}
			rel, err := filepath.Rel(dir, p)
			if err != nil {
				return err
			}
			files[filepath.ToSlash(rel)] = body
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
		short, _ := filepath.Rel(root, dir)
		if strings.HasPrefix(short, "plugins/object-s3/") && len(files) > 0 {
			sawObjects = true
		}
		for _, f := range certification.HandoverFindings(files) {
			t.Errorf("%s: %s", short, f)
		}
	}
	if !sawObjects {
		t.Error("the object-s3 example tree was not scanned")
	}
}
