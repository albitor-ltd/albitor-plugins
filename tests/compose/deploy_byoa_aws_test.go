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

func byoaDir(t *testing.T) string { return filepath.Join(repoRoot(t), "plugins", "deploy-byoa-aws") }

func byoaSubject(t *testing.T) certification.Subject {
	t.Helper()
	p := loadDir(t, byoaDir(t))
	for _, s := range core.Sockets() {
		if s.Name == core.SocketDeployTarget {
			return certification.Subject{Manifest: p.Manifest, Tree: p.Tree, Socket: s}
		}
	}
	t.Fatal("the registry has no deploy-target socket")
	return certification.Subject{}
}

// TestDeployBYOAConforms runs the real deploy-target@1 suite, the checks ingest runs, over the
// plugin, the workflow lint included.
func TestDeployBYOAConforms(t *testing.T) {
	sub := byoaSubject(t)
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

var workflowPaths = []string{
	"templates/.github/workflows/ci.yml",
	"templates/.github/workflows/deploy.yml",
	"templates/.github/workflows/destroy.yml",
	"templates/.github/workflows/invite.yml",
}

// TestDeployBYOAWorkflowsPassTheRealLint runs LintWorkflow with the deploy-target policy over each
// shipped workflow, so a violation names the workflow and the rule.
func TestDeployBYOAWorkflowsPassTheRealLint(t *testing.T) {
	sub := byoaSubject(t)
	pol := certification.DefaultWorkflowPolicy(core.SocketDeployTarget)
	for _, p := range workflowPaths {
		body, ok := sub.Tree[p]
		if !ok {
			t.Fatalf("plugin ships no %s", p)
		}
		if msgs := certification.LintWorkflow(body, pol); len(msgs) != 0 {
			t.Errorf("%s fails the workflow lint:\n  %s", p, strings.Join(msgs, "\n  "))
		}
	}
}

// TestDeployBYOALintIsReallyApplied proves the passes above are not vacuous: each unsafe edit of
// the shipped deploy workflow must fail the real lint.
func TestDeployBYOALintIsReallyApplied(t *testing.T) {
	const path = "templates/.github/workflows/deploy.yml"
	sub := byoaSubject(t)
	orig, ok := sub.Tree[path]
	if !ok {
		t.Fatalf("plugin ships no %s", path)
	}
	pol := certification.DefaultWorkflowPolicy(core.SocketDeployTarget)
	cases := map[string]func(string) string{
		"an unpinned action": func(s string) string {
			return strings.Replace(s, "actions/checkout@3d3c42e5aac5ba805825da76410c181273ba90b1", "actions/checkout@v4", 1)
		},
		"a pull request trigger beside id-token": func(s string) string {
			return strings.Replace(s, "on:\n", "on:\n  pull_request:\n", 1)
		},
		"a workflow_run trigger": func(s string) string {
			return strings.Replace(s, "on:\n", "on:\n  workflow_run:\n    workflows: [ci]\n", 1)
		},
		"id-token at the top level": func(s string) string {
			return strings.Replace(s, "permissions:\n  contents: read\n", "permissions:\n  contents: read\n  id-token: write\n", 1)
		},
		"a secret outside the allowlist": func(s string) string {
			return strings.Replace(s, "run: terraform fmt", "run: echo ${{ secrets.NPM_TOKEN }}", 1)
		},
		"a ref spliced into a script": func(s string) string {
			return strings.Replace(s, "run: terraform fmt", "run: echo ${{ github.ref_name }}", 1)
		},
		"a dumped environment": func(s string) string {
			return strings.Replace(s, "run: terraform fmt", "run: printenv", 1)
		},
	}
	for name, edit := range cases {
		changed := edit(string(orig))
		if changed == string(orig) {
			t.Fatalf("%s: the edit did not apply; the case is stale", name)
		}
		if msgs := certification.LintWorkflow([]byte(changed), pol); len(msgs) == 0 {
			t.Errorf("%s was accepted by the workflow lint", name)
		}
	}
}

func byoaPeers(t *testing.T) catalogue {
	t.Helper()
	root := repoRoot(t)
	c := catalogue{}
	for _, n := range []string{"web-svelte", "kit-shadcn", "api-go", "data-dynamodb", "auth-none", "test-playwright"} {
		c[n] = loadDir(t, filepath.Join(root, "plugins", n))
	}
	return c
}

func byoaProfile() certification.Profile {
	return certification.Profile{Name: "byoa-check", Members: []certification.Member{
		{Socket: core.SocketWebFramework, Plugin: "web-svelte"},
		{Socket: core.SocketComponentKit, Plugin: "kit-shadcn"},
		{Socket: core.SocketAuth, Plugin: "auth-none"},
		{Socket: core.SocketAPIRuntime, Plugin: "api-go"},
		{Socket: core.SocketDataStore, Plugin: "data-dynamodb"},
		{Socket: core.SocketDeployTarget, Plugin: "deploy-byoa-aws"},
		{Socket: core.SocketTestHarness, Plugin: "test-playwright"},
	}}
}

// TestDeployBYOACertifies runs the platform's harness over the plugin's snapshot in a BYOA
// profile: manifest, deploy-target@1 conformance, compose and the floor gate.
func TestDeployBYOACertifies(t *testing.T) {
	h := &certification.Harness{
		Tokens:   tokens,
		Profiles: []certification.Profile{byoaProfile()},
		Peers:    byoaPeers(t),
		Suites:   certification.DefaultSuites(),
	}
	cert, err := h.Certify(context.Background(), snapshot(t, byoaDir(t)))
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

func composeBYOA(t *testing.T) *composer.Result {
	t.Helper()
	peers := byoaPeers(t)
	plugins := []composer.Plugin{loadDir(t, byoaDir(t))}
	for _, n := range []string{"web-svelte", "kit-shadcn", "api-go", "data-dynamodb", "auth-none", "test-playwright"} {
		plugins = append(plugins, peers[n])
	}
	res, err := composer.Compose(composer.Input{Plugins: plugins, Tokens: tokens})
	if err != nil {
		t.Fatalf("deploy-byoa-aws does not compose in a BYOA profile: %v", err)
	}
	return res
}

// TestDeployBYOAComposes pins what the plugin writes, in which mode, and what it must not write.
func TestDeployBYOAComposes(t *testing.T) {
	res := composeBYOA(t)

	var lock composer.Lock
	if err := json.Unmarshal(res.Files[".albitor/lock.json"], &lock); err != nil {
		t.Fatal(err)
	}
	modes := map[string]string{}
	for _, f := range lock.Files {
		modes[f.Path] = f.Mode
	}
	// Deploy plumbing is a starting point: the workflows (still managed, the parser admits no other
	// mode) and the rendered stack the app owns from the first compose.
	for _, p := range []string{
		".github/workflows/ci.yml", ".github/workflows/deploy.yml", ".github/workflows/destroy.yml", ".github/workflows/invite.yml",
		"infrastructure/aws/main.tf", "infrastructure/aws/variables.tf", "infrastructure/aws/versions.tf",
		"infrastructure/aws/backend.tf", "infrastructure/aws/outputs.tf", "infrastructure/aws/.terraform.lock.hcl",
		"scripts/tf-unlock-stale.sh", "scripts/seed-platform-accounts.sh", "scripts/invite-user.sh",
	} {
		if _, ok := res.Files[p]; !ok {
			t.Errorf("%s is not in the composed tree", p)
		}
		// The platform's manifest parser still admits a workflow in managed mode only; every other
		// file here is seeded once and app-owned.
		want := "example"
		if strings.HasPrefix(p, ".github/workflows/") {
			want = "managed"
		}
		if modes[p] != want {
			t.Errorf("%s should be %s, got %q", p, want, modes[p])
		}
	}
	// The one app-owned file: per-environment choices (DNS, alarm address) the person makes.
	if modes["infrastructure/aws/terraform.tfvars"] != "example" {
		t.Errorf("terraform.tfvars should be an example the app owns, got %q", modes["infrastructure/aws/terraform.tfvars"])
	}
	for p := range res.Files {
		if strings.HasPrefix(p, "infrastructure/aws/env/") {
			t.Errorf("a plugin wrote %s, which the platform renders from the bound account", p)
		}
		if strings.HasPrefix(p, ".github/") && !strings.HasPrefix(p, ".github/workflows/") {
			t.Errorf("a plugin wrote %s", p)
		}
	}

	// The Dependabot entries: the workflow pins and the module and provider pins.
	dependabot := string(res.Held[".github/dependabot.yml"])
	for _, want := range []string{"github-actions", "terraform", "/infrastructure/aws"} {
		if !strings.Contains(dependabot, want) {
			t.Errorf("dependabot.yml lacks %q:\n%s", want, dependabot)
		}
	}
	if _, ok := res.Files[".github/dependabot.yml"]; ok {
		t.Error("dependabot.yml is in the Start tree; it is held until handover (ADR-0041, #1187)")
	}

	// The manifest the stack renders from is the composed one: the table api-go's store needs.
	yaml := string(res.Files["albitor.yaml"])
	for _, want := range []string{"tables:", `name: "data"`, "http:", "handler:"} {
		if !strings.Contains(yaml, want) {
			t.Errorf("albitor.yaml lacks %q:\n%s", want, yaml)
		}
	}

	// COMPOSE_OUT_BYOA writes the composed tree for scripts/compose-check.sh to plan with Terraform.
	if out := os.Getenv("COMPOSE_OUT_BYOA"); out != "" {
		for path, content := range res.Files {
			dst := filepath.Join(out, filepath.FromSlash(path))
			if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(dst, content, 0o644); err != nil {
				t.Fatal(err)
			}
		}
	}
}

var moduleSourceRE = regexp.MustCompile(`source\s*=\s*"([^"]+)"`)

// TestDeployBYOAModulesAreVersioned: the managed root consumes the plugin's modules by a git
// source whose ref is this plugin's release tag, so a root and its modules always match. The ref
// is a literal (Terraform takes no variable in a source) and moves with plugin.json's version.
func TestDeployBYOAModulesAreVersioned(t *testing.T) {
	sub := byoaSubject(t)
	version := sub.Manifest.Version
	if version == "" {
		t.Fatal("plugin.json has no version")
	}
	main := string(sub.Tree["templates/infrastructure/aws/main.tf"])
	sources := moduleSourceRE.FindAllStringSubmatch(main, -1)
	if len(sources) == 0 {
		t.Fatal("main.tf calls no module")
	}
	want := "git::https://github.com/albitor-ltd/albitor-plugins.git//plugins/deploy-byoa-aws/modules/"
	for _, m := range sources {
		src := m[1]
		if !strings.HasPrefix(src, want) || !strings.HasSuffix(src, "?ref=deploy-byoa-aws/v"+version) {
			t.Errorf("module source %q is not %s<name>?ref=deploy-byoa-aws/v%s", src, want, version)
		}
		dir := strings.TrimSuffix(strings.TrimPrefix(src, want), "?ref=deploy-byoa-aws/v"+version)
		if _, err := os.Stat(filepath.Join(byoaDir(t), "modules", dir)); err != nil {
			t.Errorf("module source %q names no module in the plugin: %v", src, err)
		}
	}
}

// TestDeployBYOAHonoursThePlatformDispatchContract pins what the platform relies on when it
// dispatches the deploy workflow to promote a commit and reads the result back
// (albitor-ltd/albitor#3178): the `environment` and `ref` inputs are declared (GitHub answers 422
// to an undeclared one), and a job named exactly core.DeployReceiptJobName ends with the
// deploy-receipt notice, after the apply.
func TestDeployBYOAHonoursThePlatformDispatchContract(t *testing.T) {
	sub := byoaSubject(t)
	body, ok := sub.Tree["templates/.github/workflows/deploy.yml"]
	if !ok {
		t.Fatal("plugin ships no deploy.yml")
	}
	s := string(body)
	for _, input := range []string{"environment", "ref"} {
		re := regexp.MustCompile(`(?m)^  workflow_dispatch:\n    inputs:\n(?:      .*\n|        .*\n)*?      ` + input + `:\n`)
		if !re.MatchString(s) {
			t.Errorf("workflow_dispatch does not declare the %q input", input)
		}
	}
	// A promotion pins ref to an older commit that predates env/<environment>.*, so the deploy job
	// must take those files from the default branch head and overlay them.
	for _, want := range []string{
		"ref: ${{ github.sha }}",
		"sparse-checkout: infrastructure/aws/env",
		"name: overlay deploy inputs",
	} {
		if !strings.Contains(s, want) {
			t.Errorf("deploy.yml does not overlay the deploy inputs from the default branch: missing %q", want)
		}
	}
	check, identity := strings.Index(s, "name: check the environment name"), strings.Index(s, "name: resolve the deploy identity")
	if check < 0 || check > identity {
		t.Error("the environment name must be validated before the deploy identity is resolved")
	}
	job := regexp.MustCompile(`(?m)^  [a-z_-]+:\n    name: ` + regexp.QuoteMeta(core.DeployReceiptJobName) + `\n    needs: \[deploy\]\n`)
	idx := job.FindStringIndex(s)
	if idx == nil {
		t.Fatalf("no job named exactly %q that needs the deploy job", core.DeployReceiptJobName)
	}
	tail := s[idx[0]:]
	notice := `::notice title=` + core.DeployReceiptTitle + `::{"url":"%s","commit_sha":"%s","auth_user_pool_id":"%s","auth_client_id":"%s","seed_status":"%s","seeded_roles":"%s"}`
	if !strings.Contains(tail, notice) {
		t.Errorf("the receipt job does not print %s", notice)
	}
	if strings.Contains(tail, "id-token") {
		t.Error("the receipt job must hold no cloud identity")
	}
}

// TestDeployBYOADestroyHonoursThePlatformContract pins what the platform relies on when it
// dispatches destroy.yml for "Delete app" (albitor-ltd/albitor#4040): the `environment` and `ref`
// inputs are declared, nothing but a manual dispatch triggers it, the cloud identity sits on the
// destroy job alone, and a job named exactly `destroy receipt` ends with the destroy-receipt notice.
func TestDeployBYOADestroyHonoursThePlatformContract(t *testing.T) {
	sub := byoaSubject(t)
	body, ok := sub.Tree["templates/.github/workflows/destroy.yml"]
	if !ok {
		t.Fatal("plugin ships no destroy.yml")
	}
	s := string(body)

	for _, input := range []string{"environment", "ref"} {
		re := regexp.MustCompile(`(?m)^  workflow_dispatch:\n    inputs:\n(?:      .*\n|        .*\n)*?      ` + input + `:\n`)
		if !re.MatchString(s) {
			t.Errorf("workflow_dispatch does not declare the %q input", input)
		}
	}

	// The trigger block is the lines between `on:` and `permissions:`: dispatch and nothing else.
	onBlock := regexp.MustCompile(`(?ms)^on:\n(.*?)^permissions:`).FindStringSubmatch(s)
	if onBlock == nil {
		t.Fatal("destroy.yml has no on: block")
	}
	events := regexp.MustCompile(`(?m)^  ([a-z_]+):`).FindAllStringSubmatch(onBlock[1], -1)
	if len(events) != 1 || events[0][1] != "workflow_dispatch" {
		t.Errorf("destroy.yml must trigger on workflow_dispatch alone, got %v", events)
	}
	for _, bad := range []string{"push:", "pull_request", "schedule:", "workflow_run", "workflow_call"} {
		if strings.Contains(onBlock[1], bad) {
			t.Errorf("destroy.yml must not trigger on %s", bad)
		}
	}

	if !strings.Contains(s, "group: deploy-${{ inputs.environment }}") || !strings.Contains(s, "cancel-in-progress: false") {
		t.Error("destroy.yml must queue in deploy.yml's per-environment concurrency group and never cancel")
	}
	if n := len(regexp.MustCompile(`(?m)^      id-token: write$`).FindAllString(s, -1)); n != 1 {
		t.Errorf("id-token: write must be held by one job, found %d", n)
	}
	check, identity := strings.Index(s, "name: check the environment name"), strings.Index(s, "name: resolve the deploy identity")
	if check < 0 || check > identity {
		t.Error("the environment name must be validated before the deploy identity is resolved")
	}
	init, unblock, destroy := strings.Index(s, "name: terraform init"), strings.Index(s, "name: unblock the deletes"), strings.Index(s, "- name: terraform destroy")
	if init < 0 || unblock < init || destroy < unblock {
		t.Error("destroy.yml must init, then unblock the deletes, then terraform destroy")
	}
	if !strings.Contains(s, `-backend-config="env/${ENVIRONMENT}.s3.tfbackend"`) {
		t.Error("destroy.yml must init from env/<environment>.s3.tfbackend")
	}
	for _, call := range []string{"delete-objects", "--no-deletion-protection-enabled", "--deletion-protection INACTIVE"} {
		if !strings.Contains(s, call) {
			t.Errorf("the unblock step lacks %q", call)
		}
	}

	job := regexp.MustCompile(`(?m)^  [a-z_-]+:\n    name: destroy receipt\n    needs: \[destroy\]\n`)
	idx := job.FindStringIndex(s)
	if idx == nil {
		t.Fatal(`no job named exactly "destroy receipt" that needs the destroy job`)
	}
	if got := strings.Count(s, "name: destroy receipt\n"); got != 1 {
		t.Errorf("exactly one job may be named destroy receipt, found %d", got)
	}

	// Fail closed: the destroy job runs on the default branch only, and the receipt job has no
	// condition of its own that could let it run after a skipped destroy.
	if !strings.Contains(s, "    if: ${{ github.ref == format('refs/heads/{0}', github.event.repository.default_branch) }}\n    runs-on: ubuntu-24.04\n    timeout-minutes: 60") {
		t.Error("the destroy job must run only from the default branch")
	}
	receiptJob := s[idx[0]:]
	if regexp.MustCompile(`(?m)^    if:`).MatchString(receiptJob) {
		t.Error("the receipt job must carry no if: that could run it after a skipped destroy")
	}
	// Dispatch inputs never reach a script by interpolation.
	for _, m := range regexp.MustCompile(`(?ms)^\s+run: \|\n((?:\s{12,}.*\n|\n)*)`).FindAllStringSubmatch(s, -1) {
		if strings.Contains(m[1], "${{ inputs.") {
			t.Errorf("a run: block interpolates a dispatch input:\n%s", m[1])
		}
	}
	for _, want := range []string{
		`select(.mode? == "managed" and .type? == "aws_dynamodb_table")`,
		`(.Errors // [])[0]`,
		`--delete "file://$batch"`,
		`terraform state list`,
		`[[ "$DESTROY_REF" =~ ^[0-9a-f]{40}$ ]]`,
		`git merge-base --is-ancestor "$DESTROY_REF" "$GITHUB_SHA"`,
		"fetch-depth: 0",
		`[ -n "$listing" ] || listing='{}'`,
		`[ "$(jq '(.Objects // []) | length' "$batch")" = 0 ]`,
		`.[0:1000]`,
		`exists "s3://${bucket}" '404|Not Found|NoSuchBucket' aws s3api head-bucket`,
		`exists "table $table" 'ResourceNotFoundException' aws dynamodb describe-table`,
		`exists "user pool $pool" 'ResourceNotFoundException' aws cognito-idp describe-user-pool`,
		`could not check whether $what exists`,
	} {
		if !strings.Contains(s, want) {
			t.Errorf("destroy.yml lacks %q", want)
		}
	}
	for _, bad := range []string{"head-bucket --bucket \"$bucket\" 2>/dev/null", "describe-table --table-name \"$table\" >/dev/null 2>&1", "describe-user-pool --user-pool-id \"$pool\" >/dev/null 2>&1"} {
		if strings.Contains(s, bad) {
			t.Errorf("destroy.yml treats every error as not found: %q", bad)
		}
	}
	refCheck, standIn := strings.Index(s, "- name: check the ref"), strings.Index(s, "- name: stand in for the build outputs")
	if refCheck < 0 || refCheck > identity || standIn < 0 || standIn > destroy {
		t.Error("the ref must be checked before the identity is resolved, and the stand-ins made before terraform destroy")
	}
	if stateCheck := strings.Index(s, "- name: check the state is empty"); stateCheck < destroy || stateCheck > idx[0] {
		t.Error("the state must be checked empty after terraform destroy and before the receipt")
	}
	tail := s[idx[0]:]
	notice := `::notice title=destroy-receipt::{"environment":"%s","commit":"%s","state":"destroyed"}`
	if !strings.Contains(tail, notice) {
		t.Errorf("the receipt job does not print %s", notice)
	}
	if strings.Contains(tail, "id-token") {
		t.Error("the receipt job must hold no cloud identity")
	}
}
