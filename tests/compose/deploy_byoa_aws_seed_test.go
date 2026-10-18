package compose

import (
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// fakeAWS is an `aws` that records every call and answers from files, so the scripts the deploy
// target ships can be run end to end with no account:
//
//	params/<name>  the SSM parameter's value (a missing file is ParameterNotFound)
//	denied/<name>  the parameter read fails with AccessDenied
//	users          usernames that already exist in the pool, one per line
//	status/<user>  admin-get-user's UserStatus for a user (a missing file is UserNotFoundException)
//
// admin-set-user-password's request file is copied to set.log, because that file is the only place
// the password may be.
const fakeAWS = `#!/usr/bin/env bash
echo "$*" >>"$FAKE/calls.log"
arg() { local want="$1"; shift; while [ $# -gt 0 ]; do [ "$1" = "$want" ] && { echo "$2"; return; }; shift; done; }
case "$1 $2" in
  "ssm get-parameter")
    name="$(basename "$(arg --name "$@")")"
    if [ -e "$FAKE/denied/$name" ]; then echo "An error occurred (AccessDeniedException)" >&2; exit 254; fi
    if [ ! -e "$FAKE/params/$name" ]; then echo "An error occurred (ParameterNotFound)" >&2; exit 254; fi
    cat "$FAKE/params/$name"
    ;;
  "cognito-idp admin-create-user")
    user="$(arg --username "$@")"
    if grep -qx "$user" "$FAKE/users" 2>/dev/null; then echo "An error occurred (UsernameExistsException)" >&2; exit 254; fi
    echo "$user" >>"$FAKE/users"
    ;;
  "cognito-idp admin-set-user-password")
    cat "$(arg --cli-input-json "$@" | sed 's#^file://##')" >>"$FAKE/set.log"
    echo >>"$FAKE/set.log"
    ;;
  "cognito-idp admin-get-user")
    user="$(arg --username "$@")"
    [ -e "$FAKE/status/$user" ] || { echo "An error occurred (UserNotFoundException)" >&2; exit 254; }
    cat "$FAKE/status/$user"
    ;;
  *) echo "unexpected aws call: $*" >&2; exit 99 ;;
esac
`

type scriptRun struct {
	dir    string
	out    string
	err    error
	output string
}

// runScript runs one shipped script against a fake aws. files seeds the fake's state.
func runScript(t *testing.T, script string, files map[string]string, args ...string) *scriptRun {
	t.Helper()
	dir := t.TempDir()
	bin := filepath.Join(dir, "bin")
	if err := os.MkdirAll(bin, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(bin, "aws"), []byte(fakeAWS), 0o755); err != nil {
		t.Fatal(err)
	}
	for name, body := range files {
		p := filepath.Join(dir, name)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	ghOut := filepath.Join(dir, "gh_output")
	cmd := exec.Command("bash", append([]string{filepath.Join(byoaDir(t), "templates", "scripts", script)}, args...)...)
	cmd.Env = append(os.Environ(),
		"PATH="+bin+string(os.PathListSeparator)+os.Getenv("PATH"),
		"FAKE="+dir, "GITHUB_REPOSITORY=acme/tracker", "GITHUB_OUTPUT="+ghOut)
	out, err := cmd.CombinedOutput()
	status, _ := os.ReadFile(ghOut)
	return &scriptRun{dir: dir, out: strings.TrimSpace(string(status)), err: err, output: string(out)}
}

func (r *scriptRun) read(t *testing.T, name string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(r.dir, name))
	if err != nil && !os.IsNotExist(err) {
		t.Fatal(err)
	}
	return string(b)
}

const testPool = "eu-west-2_AbCdEf123"

// testSegment is deliberately not the repository name the harness sets (acme/tracker): the path
// segment is what the platform wrote, not what GitHub calls the repository.
const testSegment = "tracker-app"

var bothPasswords = map[string]string{
	"params/ALBITOR_DEMO_ACCOUNT_PASSWORD":     "demo-Pw-0123456789",
	"params/ALBITOR_VERIFIER_ACCOUNT_PASSWORD": "verifier-Pw-0123456789",
}

// TestSeedPlatformAccountsSeedsBothAccountsWithoutPuttingAPasswordOnACommandLine proves the
// seeding ADR-0071 asks of every deploy: both accounts created without email, each with its own
// permanent password, and neither password in any argument another process could list.
func TestSeedPlatformAccountsSeedsBothAccountsWithoutPuttingAPasswordOnACommandLine(t *testing.T) {
	r := runScript(t, "seed-platform-accounts.sh", bothPasswords, testPool, testSegment)
	if r.err != nil {
		t.Fatalf("seed failed: %v\n%s", r.err, r.output)
	}
	if r.out != "seed_status=seeded" {
		t.Errorf("seed_status = %q, want seeded", r.out)
	}
	users := r.read(t, "users")
	for _, u := range []string{"demo@albitor-demo.invalid", "verifier@albitor-verify.invalid"} {
		if !strings.Contains(users, u) {
			t.Errorf("%s was not created; created: %q", u, users)
		}
	}
	calls := r.read(t, "calls.log")
	if !strings.Contains(calls, "--message-action SUPPRESS") {
		t.Errorf("an account was created without SUPPRESS, so the pool would email a reserved domain:\n%s", calls)
	}
	if !strings.Contains(calls, "/albitor/apps/tracker-app/secrets/ALBITOR_DEMO_ACCOUNT_PASSWORD") ||
		!strings.Contains(calls, "/albitor/apps/tracker-app/secrets/ALBITOR_VERIFIER_ACCOUNT_PASSWORD") {
		t.Errorf("a password was not read from the app's secret path:\n%s", calls)
	}
	for _, pw := range bothPasswords {
		if strings.Contains(calls, pw) || strings.Contains(r.output, pw) {
			t.Errorf("a password reached a command line or the log:\n%s\n%s", calls, r.output)
		}
		if !strings.Contains(r.read(t, "set.log"), pw) {
			t.Errorf("a password never reached admin-set-user-password's request file")
		}
	}
	if !strings.Contains(r.read(t, "set.log"), `"Permanent": true`) {
		t.Errorf("a password was set without Permanent, leaving the account in FORCE_CHANGE_PASSWORD")
	}
}

// TestSeedPlatformAccountsIsIdempotent: the second deploy finds both accounts and resets them.
func TestSeedPlatformAccountsIsIdempotent(t *testing.T) {
	files := map[string]string{"users": "demo@albitor-demo.invalid\nverifier@albitor-verify.invalid\n"}
	for k, v := range bothPasswords {
		files[k] = v
	}
	r := runScript(t, "seed-platform-accounts.sh", files, testPool, testSegment)
	if r.err != nil || r.out != "seed_status=seeded" {
		t.Fatalf("a second seed = %v %q\n%s", r.err, r.out, r.output)
	}
	if strings.Count(r.read(t, "set.log"), `"Permanent": true`) != 2 {
		t.Errorf("an existing account's password was not reset")
	}
}

// TestSeedPlatformAccountsSkipsLoudlyOnlyForAMissingParameter pins the fail-closed split: no
// parameter is a warning the app's owner cannot fix; every other read failure fails the deploy.
func TestSeedPlatformAccountsSkipsLoudlyOnlyForAMissingParameter(t *testing.T) {
	r := runScript(t, "seed-platform-accounts.sh", map[string]string{"params/ALBITOR_VERIFIER_ACCOUNT_PASSWORD": "v-Pw-0123456789"}, testPool, testSegment)
	if r.err != nil {
		t.Fatalf("a missing demo parameter failed the deploy: %v\n%s", r.err, r.output)
	}
	if r.out != "seed_status=skipped_unprovisioned:demo" || !strings.Contains(r.output, "::warning title=seed-skipped::") {
		t.Errorf("status %q, output %q: want a named, annotated skip", r.out, r.output)
	}
	if !strings.Contains(r.read(t, "users"), "verifier@albitor-verify.invalid") {
		t.Error("the verifier was not seeded because the demo parameter was missing")
	}

	denied := map[string]string{"denied/ALBITOR_DEMO_ACCOUNT_PASSWORD": "", "params/ALBITOR_VERIFIER_ACCOUNT_PASSWORD": "v-Pw-0123456789"}
	r = runScript(t, "seed-platform-accounts.sh", denied, testPool, testSegment)
	if r.err == nil {
		t.Fatalf("a denied read was treated as a skip:\n%s", r.output)
	}
	if !strings.Contains(r.read(t, "users"), "verifier@albitor-verify.invalid") {
		t.Error("a failed demo read stopped the verifier being seeded: both are attempted")
	}

	placeholder := map[string]string{"params/ALBITOR_DEMO_ACCOUNT_PASSWORD": "__albitor_pending__", "params/ALBITOR_VERIFIER_ACCOUNT_PASSWORD": "v-Pw-0123456789"}
	r = runScript(t, "seed-platform-accounts.sh", placeholder, testPool, testSegment)
	if r.err != nil || r.out != "seed_status=skipped_unprovisioned:demo" || strings.Contains(r.read(t, "users"), "demo@") {
		t.Errorf("the reservation placeholder was seeded as a password: %v %q", r.err, r.out)
	}
}

// TestSeedPlatformAccountsNeedsNoPoolAndRefusesAnythingElse: an app without auth seeds nothing, and
// a value that is not a pool id never reaches the cli.
func TestSeedPlatformAccountsNeedsNoPoolAndRefusesAnythingElse(t *testing.T) {
	r := runScript(t, "seed-platform-accounts.sh", bothPasswords, "", testSegment)
	if r.err != nil || r.out != "seed_status=no_pool" || r.read(t, "calls.log") != "" {
		t.Errorf("no pool = %v %q calls %q", r.err, r.out, r.read(t, "calls.log"))
	}
	r = runScript(t, "seed-platform-accounts.sh", bothPasswords, "eu-west-2_x; echo pwned", testSegment)
	if r.err == nil || r.read(t, "calls.log") != "" {
		t.Errorf("a malformed pool id reached the cli: %v", r.err)
	}
}

// TestSeedPlatformAccountsFailsLoudlyWithoutTheSecretsSegment: the segment is what the platform
// wrote into deploy.env. Falling back to the repository name would read another path, find no
// parameter and skip quietly, so its absence is a reported failure instead.
func TestSeedPlatformAccountsFailsLoudlyWithoutTheSecretsSegment(t *testing.T) {
	for _, segment := range []string{"", "../escape", "a b", "x;id"} {
		r := runScript(t, "seed-platform-accounts.sh", bothPasswords, testPool, segment)
		if r.err == nil || r.out != "seed_status=failed:no_secrets_segment" || r.read(t, "calls.log") != "" {
			t.Errorf("segment %q: err %v status %q calls %q, want a reported failure with no aws call", segment, r.err, r.out, r.read(t, "calls.log"))
		}
	}
	// No pool is still no pool: the segment only matters when there is something to seed.
	if r := runScript(t, "seed-platform-accounts.sh", bothPasswords, ""); r.err != nil || r.out != "seed_status=no_pool" {
		t.Errorf("no pool and no segment = %v %q", r.err, r.out)
	}
}

// TestInviteUserSendsOneInvitationAndNeverTouchesAnAccountThatSignedIn.
func TestInviteUserSendsOneInvitationAndNeverTouchesAnAccountThatSignedIn(t *testing.T) {
	r := runScript(t, "invite-user.sh", nil, testPool, "Ada.Lovelace@Example.com")
	if r.err != nil {
		t.Fatalf("invite failed: %v\n%s", r.err, r.output)
	}
	calls := r.read(t, "calls.log")
	if !strings.Contains(calls, "--desired-delivery-mediums EMAIL") {
		t.Errorf("the invitation does not name its delivery medium:\n%s", calls)
	}
	if !strings.Contains(calls, "admin-create-user") || !strings.Contains(calls, "--username ada.lovelace@example.com") {
		t.Errorf("the address was not lowercased into one admin-create-user:\n%s", calls)
	}
	if strings.Contains(calls, "SUPPRESS") || strings.Contains(calls, "admin-set-user-password") {
		t.Errorf("an invitation must email a temporary password and set none itself:\n%s", calls)
	}

	r = runScript(t, "invite-user.sh", map[string]string{"status/ada@example.com": "FORCE_CHANGE_PASSWORD"}, testPool, "ada@example.com")
	if r.err != nil || !strings.Contains(r.read(t, "calls.log"), "--message-action RESEND --desired-delivery-mediums EMAIL") {
		t.Errorf("a pending invitation was not resent: %v\n%s", r.err, r.read(t, "calls.log"))
	}

	r = runScript(t, "invite-user.sh", map[string]string{"status/ada@example.com": "CONFIRMED"}, testPool, "ada@example.com")
	if r.err != nil || strings.Contains(r.read(t, "calls.log"), "admin-create-user") {
		t.Errorf("an account that has signed in was touched: %v\n%s", r.err, r.read(t, "calls.log"))
	}
}

func TestInviteUserRefusesWhatIsNotAPlainAddress(t *testing.T) {
	for _, email := range []string{"", "no-at-sign", "a b@example.com", "x@y.com;rm -rf /", "demo@albitor-demo.invalid", "$(id)@example.com", "a@b"} {
		r := runScript(t, "invite-user.sh", nil, testPool, email)
		if r.err == nil || r.read(t, "calls.log") != "" {
			t.Errorf("%q was invited (err %v, calls %q)", email, r.err, r.read(t, "calls.log"))
		}
	}
	if r := runScript(t, "invite-user.sh", nil, "", "ada@example.com"); r.err == nil || r.read(t, "calls.log") != "" {
		t.Errorf("an app with no pool was invited to: %v", r.err)
	}
}

// TestDeployWorkflowSeedsOnlyAfterTheApplyAndOnlyOnDev: the seed needs the pool the apply makes,
// runs on the build target alone (Albitor's P6), and comes after the bundle ships, so a failed seed
// cannot leave the new CSP fronting the old shell. The receipt that reports it comes last.
func TestDeployWorkflowSeedsOnlyAfterTheApplyAndOnlyOnDev(t *testing.T) {
	sub := byoaSubject(t)
	s := string(sub.Tree["templates/.github/workflows/deploy.yml"])
	apply, ship, seed, receipt := strings.Index(s, "- name: terraform apply\n"), strings.Index(s, "name: ship the web bundle"), strings.Index(s, "name: seed the platform accounts"), strings.Index(s, "name: print the app url")
	if apply < 0 || ship < apply || seed < ship || receipt < seed {
		t.Fatalf("want apply, ship, seed, receipt in that order (apply %d, ship %d, seed %d, receipt %d)", apply, ship, seed, receipt)
	}
	step := s[seed:receipt]
	if !strings.Contains(step, "if: ${{ env.ENVIRONMENT == 'dev' }}") {
		t.Error("the seed is not limited to the dev environment")
	}
	if !strings.Contains(step, "scripts/seed-platform-accounts.sh") {
		t.Error("the seed step does not run the managed script")
	}
	if !strings.Contains(step, "id: seed") {
		t.Error("the seed step has no id, so its seed_status cannot reach the receipt")
	}
	if regexp.MustCompile(`(?i)password`).MatchString(step[strings.Index(step, "run:"):]) {
		t.Error("the seed step handles a password itself; only the script may")
	}
}

// TestInviteWorkflowKeepsTheEmailOutOfAnyScript: the dispatched address reaches the script through
// the environment only.
func TestInviteWorkflowKeepsTheEmailOutOfAnyScript(t *testing.T) {
	sub := byoaSubject(t)
	s := string(sub.Tree["templates/.github/workflows/invite.yml"])
	for _, run := range regexp.MustCompile(`(?s)run: \|\n(.*?)\n      - |run: \|\n(.*)$`).FindAllString(s, -1) {
		if strings.Contains(run, "inputs.") || strings.Contains(run, "github.event") {
			t.Errorf("a run script reads an input directly:\n%s", run)
		}
	}
	if !strings.Contains(s, `"$INVITE_EMAIL"`) {
		t.Error("the invite step does not pass the address through the environment")
	}
	if strings.Count(s, "\n      id-token: write\n") != 1 || strings.Contains(s, "\npermissions:\n  contents: read\n  id-token") {
		t.Error("id-token: write must be held by the one job alone")
	}
	if !strings.Contains(s, "github.event.repository.default_branch") {
		t.Error("the invite is not limited to the default branch")
	}
}

// TestDeployReceiptCarriesWhatThePlatformReads: the platform's sign-in check reads these keys from
// the one receipt line (core.DeployReceipt: url, commit_sha, auth_user_pool_id, auth_client_id,
// seed_status). A key dropped here reads as "no sign-in" and the check goes unverified, so each key
// is pinned, and each reaches the receipt through a job output.
func TestDeployReceiptCarriesWhatThePlatformReads(t *testing.T) {
	sub := byoaSubject(t)
	s := string(sub.Tree["templates/.github/workflows/deploy.yml"])
	start := strings.Index(s, "printf '::notice title=deploy-receipt::")
	if start < 0 {
		t.Fatal("no deploy receipt")
	}
	receipt := s[start : start+strings.Index(s[start:], "\n")]
	for _, key := range []string{"url", "commit_sha", "auth_user_pool_id", "auth_client_id", "seed_status"} {
		if !strings.Contains(receipt, `"`+key+`"`) {
			t.Errorf("the receipt does not carry %q:\n%s", key, receipt)
		}
	}
	for _, out := range []string{"auth_user_pool_id", "auth_client_id", "seed_status"} {
		if !strings.Contains(s, out+": ${{ steps.app_url.outputs."+out+" }}") {
			t.Errorf("the deploy job does not expose %q as an output", out)
		}
		if !strings.Contains(s, "needs.deploy.outputs."+out) {
			t.Errorf("the ship job does not read %q", out)
		}
	}
	if !strings.Contains(s, "steps.seed.outputs.seed_status") {
		t.Error("the receipt's seed_status is not the seed step's")
	}
	if !strings.Contains(s, "ALBITOR_APP_SECRETS_SEGMENT") || !strings.Contains(s, `"$SECRETS_SEGMENT"`) {
		t.Error("the seed is not given the platform-written secrets segment")
	}
}

// TestDeployWorkflowHandsTheLiveShellToTheApply: the apply changes the CSP before the bundle ships,
// so the HTML the bucket serves now (not what an earlier plan expected) is copied out before the
// apply and handed to it, with a first deploy (no bucket in the state) copying nothing.
func TestDeployWorkflowHandsTheLiveShellToTheApply(t *testing.T) {
	sub := byoaSubject(t)
	s := string(sub.Tree["templates/.github/workflows/deploy.yml"])
	read := strings.Index(s, "name: copy the live shell")
	if read < 0 {
		t.Fatal("no step copies the live shell")
	}
	apply := strings.Index(s[read:], "- name: terraform apply\n")
	if apply < 0 {
		t.Fatal("no apply follows the copy of the live shell")
	}
	apply += read
	step := s[read:apply]
	if !strings.Contains(step, `.spa_bucket.value // ""`) || !strings.Contains(step, "--include '*.html'") {
		t.Error("the copy does not take the bucket from the state, tolerating its absence, and copy its HTML")
	}
	if strings.Contains(step, "csp_script_hashes") {
		t.Error("the live hashes come from what the state planned, not what the bucket serves")
	}
	if !strings.Contains(s[apply:], "TF_VAR_previous_web_build_dir: ${{ steps.live_shell.outputs.dir }}") {
		t.Error("the apply is not handed the live shell")
	}
}
