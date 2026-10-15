package compose

import (
	"context"
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"

	"albitor/core"
	"albitor/core/certification"
	"albitor/core/composer"
)

func hostedDir(t *testing.T) string { return filepath.Join(repoRoot(t), "plugins", "deploy-hosted") }

func hostedSubject(t *testing.T) certification.Subject {
	t.Helper()
	p := loadDir(t, hostedDir(t))
	for _, s := range core.Sockets() {
		if s.Name == core.SocketDeployTarget {
			return certification.Subject{Manifest: p.Manifest, Tree: p.Tree, Socket: s}
		}
	}
	t.Fatal("the registry has no deploy-target socket")
	return certification.Subject{}
}

// TestDeployHostedConforms runs the real deploy-target@1 suite, the checks ingest runs, over the
// plugin, the workflow lint included.
func TestDeployHostedConforms(t *testing.T) {
	sub := hostedSubject(t)
	ids := map[string]bool{}
	for _, c := range certification.DefaultSuites()[core.SocketDeployTarget] {
		ids[c.ID] = true
		if err := c.Run(sub); err != nil {
			t.Errorf("conformance check %s: %v", c.ID, err)
		}
	}
	if !ids["workflows-are-safe-to-pin"] {
		t.Fatal("the deploy-target suite carries no workflow lint")
	}
}

const hostedCI = "templates/.github/workflows/ci.yml"

// TestDeployHostedCIPassesTheRealLint runs LintWorkflow with the deploy-target policy over the
// shipped workflow, so a violation names the rule.
func TestDeployHostedCIPassesTheRealLint(t *testing.T) {
	sub := hostedSubject(t)
	body, ok := sub.Tree[hostedCI]
	if !ok {
		t.Fatalf("plugin ships no %s", hostedCI)
	}
	if msgs := certification.LintWorkflow(body, certification.DefaultWorkflowPolicy(core.SocketDeployTarget)); len(msgs) != 0 {
		t.Errorf("%s fails the workflow lint:\n  %s", hostedCI, strings.Join(msgs, "\n  "))
	}
}

// TestDeployHostedCIHoldsNoCloudIdentity: the hosted CI is the gate for pull requests, so it must
// stay read-only. The lint would allow id-token on a deploy-target; this plugin must not use it,
// because the platform deploys and a pull request trigger sits beside it.
func TestDeployHostedCIHoldsNoCloudIdentity(t *testing.T) {
	body := string(hostedSubject(t).Tree[hostedCI])
	for _, banned := range []string{"id-token", "secrets.", "vars.", "aws-actions", "terraform", "contents: write"} {
		if strings.Contains(body, banned) {
			t.Errorf("ci.yml mentions %q; the hosted CI is read-only and holds no cloud identity", banned)
		}
	}
	if !strings.Contains(body, "\npermissions:\n  contents: read\n") {
		t.Error("ci.yml lacks a top-level contents: read permissions block")
	}
}

// TestDeployHostedLintIsReallyApplied proves the pass above is not vacuous: each unsafe edit of the
// shipped workflow must fail the real lint.
func TestDeployHostedLintIsReallyApplied(t *testing.T) {
	sub := hostedSubject(t)
	orig := string(sub.Tree[hostedCI])
	pol := certification.DefaultWorkflowPolicy(core.SocketDeployTarget)
	cases := map[string]func(string) string{
		"an unpinned action": func(s string) string {
			return strings.Replace(s, "actions/checkout@11bd71901bbe5b1630ceea73d27597364c9af683", "actions/checkout@v4", 1)
		},
		"a pull_request_target trigger": func(s string) string {
			return strings.Replace(s, "on:\n", "on:\n  pull_request_target:\n", 1)
		},
		"id-token at the top level": func(s string) string {
			return strings.Replace(s, "permissions:\n  contents: read\n", "permissions:\n  contents: read\n  id-token: write\n", 1)
		},
		"a secret outside the allowlist": func(s string) string {
			return strings.Replace(s, "run: go build ./...", "run: echo ${{ secrets.NPM_TOKEN }}", 1)
		},
		"a dumped environment": func(s string) string {
			return strings.Replace(s, "run: go build ./...", "run: printenv", 1)
		},
	}
	for name, edit := range cases {
		changed := edit(orig)
		if changed == orig {
			t.Fatalf("%s: the edit did not apply; the case is stale", name)
		}
		if msgs := certification.LintWorkflow([]byte(changed), pol); len(msgs) == 0 {
			t.Errorf("%s was accepted by the workflow lint", name)
		}
	}
}

// TestDeployHostedRefusesThePipelineCaller records why the platform, not this plugin, writes the
// deploy caller: a job holding id-token: write may not call a reusable workflow, which is what the
// caller of the hosted pipeline is. If the lint ever allows it, this fails and the caller can move
// into the plugin.
func TestDeployHostedRefusesThePipelineCaller(t *testing.T) {
	const caller = `name: deploy
on:
  workflow_dispatch:
    inputs:
      sha:
        required: true
        type: string
permissions: {}
jobs:
  deploy:
    permissions:
      contents: read
      id-token: write
    uses: albitor-ltd/albitor-hosted-pipeline/.github/workflows/deploy.yml@11bd71901bbe5b1630ceea73d27597364c9af683
    with:
      sha: ${{ inputs.sha }}
`
	if msgs := certification.LintWorkflow([]byte(caller), certification.DefaultWorkflowPolicy(core.SocketDeployTarget)); len(msgs) == 0 {
		t.Error("the lint now accepts a reusable-workflow deploy caller; deploy-hosted can ship it")
	}
}

func hostedPeerNames() []string {
	return []string{"web-svelte", "kit-shadcn", "api-go", "data-dynamodb", "auth-none", "test-playwright"}
}

func hostedPeers(t *testing.T) catalogue {
	t.Helper()
	root := repoRoot(t)
	c := catalogue{}
	for _, n := range hostedPeerNames() {
		c[n] = loadDir(t, filepath.Join(root, "plugins", n))
	}
	return c
}

func hostedProfile() certification.Profile {
	return certification.Profile{Name: "hosted-check", Members: []certification.Member{
		{Socket: core.SocketWebFramework, Plugin: "web-svelte"},
		{Socket: core.SocketComponentKit, Plugin: "kit-shadcn"},
		{Socket: core.SocketAuth, Plugin: "auth-none"},
		{Socket: core.SocketAPIRuntime, Plugin: "api-go"},
		{Socket: core.SocketDataStore, Plugin: "data-dynamodb"},
		{Socket: core.SocketDeployTarget, Plugin: "deploy-hosted"},
		{Socket: core.SocketTestHarness, Plugin: "test-playwright"},
	}}
}

// TestDeployHostedCertifies runs the platform's harness over the plugin's snapshot in a hosted
// profile: manifest, deploy-target@1 conformance, compose and the floor gate.
func TestDeployHostedCertifies(t *testing.T) {
	h := &certification.Harness{
		Tokens:   tokens,
		Profiles: []certification.Profile{hostedProfile()},
		Peers:    hostedPeers(t),
		Suites:   certification.DefaultSuites(),
	}
	cert, err := h.Certify(context.Background(), snapshot(t, hostedDir(t)))
	if err != nil {
		t.Fatal(err)
	}
	if cert.Verdict == certification.VerdictFailed {
		t.Fatalf("certification failed: %v", cert.Failures)
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

func composeHosted(t *testing.T, deploy ...string) (*composer.Result, error) {
	t.Helper()
	root := repoRoot(t)
	plugins := []composer.Plugin{}
	for _, n := range append(deploy, hostedPeerNames()...) {
		plugins = append(plugins, loadDir(t, filepath.Join(root, "plugins", n)))
	}
	return composer.Compose(composer.Input{Plugins: plugins, Tokens: tokens})
}

// TestDeployHostedComposes pins what the plugin writes, in which mode, and what it must not write.
func TestDeployHostedComposes(t *testing.T) {
	res, err := composeHosted(t, "deploy-hosted")
	if err != nil {
		t.Fatalf("deploy-hosted does not compose in a hosted profile: %v", err)
	}
	var lock composer.Lock
	if err := json.Unmarshal(res.Files[".albitor/lock.json"], &lock); err != nil {
		t.Fatal(err)
	}
	modes := map[string]string{}
	for _, f := range lock.Files {
		modes[f.Path] = f.Mode
	}
	// The gate is pinned: a build cannot weaken it.
	if _, ok := res.Files[".github/workflows/ci.yml"]; !ok {
		t.Error(".github/workflows/ci.yml is not in the composed tree")
	}
	if modes[".github/workflows/ci.yml"] != "managed" {
		t.Errorf("ci.yml should be managed, got %q", modes[".github/workflows/ci.yml"])
	}
	// No infrastructure and no deploy path: the platform compiles albitor.yaml and deploys.
	for p := range res.Files {
		if strings.HasPrefix(p, "infrastructure/") || p == ".github/workflows/deploy.yml" || p == "scripts/tf-unlock-stale.sh" {
			t.Errorf("deploy-hosted composed %s; a hosted repository holds no infrastructure or deploy workflow", p)
		}
		if strings.HasPrefix(p, ".github/") && !strings.HasPrefix(p, ".github/workflows/") {
			t.Errorf("a plugin wrote %s", p)
		}
	}
	// The Dependabot entry is the workflow pins only: there is no Terraform to pin.
	dependabot := string(res.Held[".github/dependabot.yml"])
	if !strings.Contains(dependabot, "github-actions") {
		t.Errorf("dependabot.yml lacks github-actions:\n%s", dependabot)
	}
	if strings.Contains(dependabot, "terraform") {
		t.Errorf("dependabot.yml names terraform:\n%s", dependabot)
	}
	// The manifest the platform compiles is the composed one.
	yaml := string(res.Files["albitor.yaml"])
	for _, want := range []string{"tables:", "http:", "handler:"} {
		if !strings.Contains(yaml, want) {
			t.Errorf("albitor.yaml lacks %q:\n%s", want, yaml)
		}
	}
}

// TestDeployHostedIsExactlyOne: deploy-target admits one plugin, so the two roster targets never
// compose together, and a profile needs one.
func TestDeployHostedIsExactlyOne(t *testing.T) {
	if _, err := composeHosted(t, "deploy-hosted", "deploy-byoa-aws"); err == nil {
		t.Fatal("deploy-hosted and deploy-byoa-aws composed together; deploy-target is exactly-one")
	}
	if _, err := composeHosted(t); err == nil {
		t.Fatal("a profile with no deploy target composed")
	}
}
