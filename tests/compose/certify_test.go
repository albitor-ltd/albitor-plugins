package compose

import (
	"context"
	"os/exec"
	"regexp"
	"sort"
	"testing"

	"albitor/core/certification"
	"albitor/core/certification/terraform"
	"albitor/core/composer"
)

type verdictCache map[string]certification.ProfileResult

func (c verdictCache) Get(k string) (certification.ProfileResult, bool) { r, ok := c[k]; return r, ok }
func (c verdictCache) Put(k string, r certification.ProfileResult)      { c[k] = r }

// moduleSource matches a module a plugin pins to one of its own release tags, which does not
// exist until the plugin is tagged (albitor-ltd/albitor#3333).
var moduleSource = regexp.MustCompile(`git::https://github\.com/albitor-ltd/albitor-plugins\.git//plugins/([a-z0-9-]+)/(modules/[a-z0-9_/-]+)\?ref=[^"]+`)

// TestEveryPluginCertifiesIntoEveryProfile runs the platform's certification harness over each
// socket plugin in this repository, as the platform runs it at ingest, with the two checks the
// platform leaves off (albitor-ltd/albitor#3333): the deploy target must declare what it renders
// (RequireRenders) and the composed albitor.yaml must be one it renders, and every composed
// Terraform root must pass `terraform init -backend=false && terraform validate` with no
// credentials. It covers every certified profile (default, byoa-aws), whichever plugin is the
// candidate, so a manifest the composer writes and a target refuses (deploy-byoa-aws v0.1.0 and
// services.auth) fails here, before a release, not at an app's first deploy.
func TestEveryPluginCertifiesIntoEveryProfile(t *testing.T) {
	if _, err := exec.LookPath("terraform"); err != nil {
		t.Skip("terraform is not installed (CI installs it)")
	}
	root := repoRoot(t)
	plugins := repoSocketPlugins(t)
	h := &certification.Harness{
		Profiles:       certification.Profiles(),
		Peers:          catalogue(plugins),
		Tokens:         profileTokens(composer.DataLayerOnline),
		RequireRenders: true,
		Cache:          verdictCache{},
		Validator: &terraform.Validator{Rewrite: func(_ string, body []byte) []byte {
			return moduleSource.ReplaceAll(body, []byte(root+"/plugins/$1/$2"))
		}},
	}
	names := make([]string, 0, len(plugins))
	for n := range plugins {
		names = append(names, n)
	}
	sort.Strings(names)
	for _, name := range names {
		t.Run(name, func(t *testing.T) {
			cert, err := h.Certify(context.Background(), plugins[name].Tree)
			if err != nil {
				t.Fatal(err)
			}
			if cert.Verdict == certification.VerdictFailed {
				t.Errorf("%s does not certify: %q", name, cert.Failures)
			}
		})
	}
}
