// Managed by Albitor (test-playwright). Run with `npm run unit` from web/tests (needs Chromium:
// `npx playwright install chromium`).
//
// The demo password must not land in anything a failed sign-in leaves behind: the per-failure
// page snapshot (error-context.md), the HTML report CI uploads, a trace. This runs the real setup
// project against sign-in pages that fail in different ways, with a sentinel password and CI=1
// (so a retry happens too), then searches every output file for the sentinel. The HTML report
// embeds its data as a base64 zip, so the scan decodes it; a scan of the raw bytes would pass
// whatever the setup did.
import assert from "node:assert/strict";
import { spawn } from "node:child_process";
import { mkdirSync, mkdtempSync, readdirSync, readFileSync, rmSync, statSync, writeFileSync } from "node:fs";
import { createServer } from "node:http";
import type { AddressInfo } from "node:net";
import { tmpdir } from "node:os";
import { join } from "node:path";
import { describe, it } from "node:test";
import { inflateRawSync } from "node:zlib";
import { DEMO_STORAGE_STATE, hasDemoSession, TESTS_DIR } from "../helpers/storage-state.ts";

const SENTINEL = "SENTINEL-pw-8841";

// The sign-in screens a sign-in is tried against, served by path. Each is one way it can go, and
// each would leave the password somewhere different if the setup were careless:
//   /never     takes the password and signs no one in
//   /disabled  the sign-in button is disabled, so click() waits for it until its timeout
//   /overlay   something covers the button, so click() waits for it until its timeout
//   /bound     a controlled input in the style of a UI framework (see below) that DOES sign in
const FIELDS = (extra: string) => `<!doctype html><title>Sign in</title><main><h1>Sign in</h1>
<form id="sign-in-form" onsubmit="return false">
<label>Email address <input id="sign-in-email" type="email"></label>
<label>Password <input id="sign-in-password" type="password"></label>
${extra}</form></main>`;

const BUTTON = '<button id="sign-in-button" type="button">Sign in</button>';

const PAGES: Record<string, string> = {
	"/never": FIELDS(BUTTON),
	"/disabled": FIELDS(BUTTON.replace("<button ", "<button disabled ")),
	// A cold edge: the page reaches `load` only after ~27s (a script that is slow to arrive), and the
	// button is disabled. The test timeout then fires late in the sign-in, not in a step of its own.
	"/slow": `${FIELDS(BUTTON.replace("<button ", "<button disabled "))}<script src="/slow.js"></script>`,
	"/overlay": `${FIELDS(BUTTON)}<div id="veil" style="position:fixed;inset:0;background:#fff8"></div>`,
	// A framework-bound input. Like React, it replaces the value property on the INSTANCE to track
	// what it last saw, and only believes a change when an input event arrives with a value that
	// differs from the tracked one. Setting the value through the prototype's native setter and then
	// dispatching "input" is what satisfies it. The button signs in only with the right password.
	"/bound": `${FIELDS(BUTTON)}
<script>
const input = document.querySelector('#sign-in-password');
const proto = Object.getOwnPropertyDescriptor(HTMLInputElement.prototype, 'value');
let tracked = '';
Object.defineProperty(input, 'value', {
	configurable: true,
	get() { return proto.get.call(this); },
	set(v) { tracked = String(v); proto.set.call(this, v); },
});
let state = '';
input.addEventListener('input', () => {
	if (proto.get.call(input) !== tracked) { state = proto.get.call(input); tracked = state; }
});
document.querySelector('#sign-in-button').addEventListener('click', () => {
	if (state === ${JSON.stringify(SENTINEL)}) {
		document.cookie = 'session=ok; path=/';
		history.pushState({}, '', '/done');
		document.querySelector('#sign-in-form').remove();
	}
});
</script>`,
};

function files(dir: string): string[] {
	const out: string[] = [];
	for (const name of readdirSync(dir)) {
		const p = join(dir, name);
		if (statSync(p).isDirectory()) {
			out.push(...files(p));
		} else {
			out.push(p);
		}
	}
	return out;
}

// zipEntries reads a zip held in memory through its central directory and returns each entry's
// bytes. Only stored and deflated entries, which is all Playwright's report writes.
function zipEntries(zip: Buffer): Buffer[] {
	const eocd = zip.lastIndexOf(Buffer.from([0x50, 0x4b, 0x05, 0x06]));
	assert.ok(eocd >= 0, "the embedded report is not a zip");
	const count = zip.readUInt16LE(eocd + 10);
	let p = zip.readUInt32LE(eocd + 16);
	const out: Buffer[] = [];
	for (let i = 0; i < count; i++) {
		assert.equal(zip.readUInt32LE(p), 0x02014b50, "bad central directory entry");
		const method = zip.readUInt16LE(p + 10);
		const size = zip.readUInt32LE(p + 20);
		const nameLen = zip.readUInt16LE(p + 28);
		const extraLen = zip.readUInt16LE(p + 30);
		const commentLen = zip.readUInt16LE(p + 32);
		const local = zip.readUInt32LE(p + 42);
		const dataAt = local + 30 + zip.readUInt16LE(local + 26) + zip.readUInt16LE(local + 28);
		const raw = zip.subarray(dataAt, dataAt + size);
		assert.ok(method === 0 || method === 8, `zip method ${method} is not handled`);
		out.push(method === 0 ? raw : inflateRawSync(raw));
		p += 46 + nameLen + extraLen + commentLen;
	}
	return out;
}

// contentsOf is what a file really holds: its bytes and, for the HTML report, the zip it embeds
// as base64 (report.json, with every step title and error in it), decoded.
function contentsOf(file: string): Buffer[] {
	const bytes = readFileSync(file);
	const found = [bytes];
	if (file.endsWith(".html")) {
		const m = /data:application\/zip;base64,([A-Za-z0-9+/=]+)/.exec(bytes.toString("utf8"));
		if (m) {
			found.push(...zipEntries(Buffer.from(m[1], "base64")));
		}
	}
	return found;
}

function leaks(produced: string[], secret: string): string[] {
	return produced.filter((f) => contentsOf(f).some((b) => b.includes(secret)));
}

// run starts playwright and waits for it. Not spawnSync: that would block the event loop that
// serves the page.
async function run(cwd: string, args: string[], env: Record<string, string>): Promise<{ code: number | null; log: string }> {
	const child = spawn("npx", ["playwright", "test", ...args], {
		cwd,
		env: { ...process.env, CI: "1", ALBITOR_JOURNEY_CREDENTIALS: "", ...env },
		stdio: ["ignore", "pipe", "pipe"],
	});
	let log = "";
	child.stdout.on("data", (d) => {
		log += d;
	});
	child.stderr.on("data", (d) => {
		log += d;
	});
	const code = await new Promise<number | null>((resolve) => child.on("close", resolve));
	return { code, log };
}

async function serve(): Promise<{ base: string; close: () => void }> {
	const server = createServer((req, res) => {
		if (req.url === "/slow.js") {
			const timer = setTimeout(() => {
				res.writeHead(200, { "Content-Type": "text/javascript" });
				res.end("");
			}, 27_000);
			res.on("close", () => clearTimeout(timer));
			return;
		}
		res.writeHead(200, { "Content-Type": "text/html" });
		res.end(PAGES[req.url ?? ""] ?? PAGES["/never"]);
	});
	await new Promise<void>((resolve) => server.listen(0, "127.0.0.1", () => resolve()));
	return {
		base: `http://127.0.0.1:${(server.address() as AddressInfo).port}`,
		close: () => {
			server.closeAllConnections();
			server.close();
		},
	};
}

describe("the demo password", () => {
	for (const route of ["/never", "/disabled", "/overlay", "/slow"]) {
		it(`is in no snapshot, report or trace when sign-in fails (${route})`, async () => {
			const app = await serve();
			const out = mkdtempSync(join(tmpdir(), "albitor-secret-"));
			try {
				const { code, log } = await run(TESTS_DIR, ["--project=setup"], {
					ALBITOR_LOCAL_BASE_URL: app.base,
					ALBITOR_DEMO_ACCOUNT_PASSWORD: SENTINEL,
					DEMO_ACCOUNT_SIGN_IN_ROUTE: route,
					ALBITOR_TESTS_OUT: out,
				});
				assert.notEqual(code, 0, `the sign-in was meant to fail:\n${log}`);

				const produced = files(out);
				assert.ok(produced.some((f) => f.endsWith("error-context.md")), `no failure snapshot was produced:\n${produced.join("\n")}`);
				assert.ok(produced.some((f) => f.endsWith(join("playwright-report", "index.html"))), "no HTML report was produced");
				assert.deepEqual(produced.filter((f) => f.endsWith(".zip")), [], "a trace was recorded");
				assert.deepEqual(leaks(produced, SENTINEL), [], "the password is in these files");
				assert.ok(!log.includes(SENTINEL), "the password is in the console output");
			} finally {
				app.close();
				rmSync(out, { recursive: true, force: true });
			}
		});
	}

	it("still signs in against an input a UI framework controls", async () => {
		const app = await serve();
		const out = mkdtempSync(join(tmpdir(), "albitor-secret-"));
		let hadSession = true;
		try {
			statSync(DEMO_STORAGE_STATE);
		} catch {
			hadSession = false;
		}
		try {
			const { code, log } = await run(TESTS_DIR, ["--project=setup"], {
				ALBITOR_LOCAL_BASE_URL: app.base,
				ALBITOR_DEMO_ACCOUNT_PASSWORD: SENTINEL,
				DEMO_ACCOUNT_SIGN_IN_ROUTE: "/bound",
				ALBITOR_TESTS_OUT: out,
			});
			assert.equal(code, 0, `the sign-in was meant to succeed:\n${log}`);
		} finally {
			app.close();
			rmSync(out, { recursive: true, force: true });
			if (!hadSession) {
				rmSync(DEMO_STORAGE_STATE, { force: true });
			}
		}
	});

	// The scan above is only worth anything if it can fail. This runs the same page through a spec
	// that types the password with Playwright's own fill(), which prints it in the step title, and
	// requires the scan to find it in the HTML report even though the raw bytes of the file do not
	// hold it.
	it("is found by the scan when fill() is used instead of fillSecret", async () => {
		const app = await serve();
		const out = mkdtempSync(join(tmpdir(), "albitor-secret-"));
		const control = mkdtempSync(join(TESTS_DIR, ".control-"));
		try {
			writeFileSync(
				join(control, "playwright.config.ts"),
				`import { defineConfig } from "@playwright/test";
export default defineConfig({ testDir: ".", testMatch: /control\\.spec\\.ts/, retries: 1,
  reporter: [["html", { open: "never", outputFolder: ${JSON.stringify(join(out, "playwright-report"))} }]],
  outputDir: ${JSON.stringify(join(out, ".results"))}, use: { baseURL: ${JSON.stringify(app.base)}, trace: "off" } });
`,
			);
			writeFileSync(
				join(control, "control.spec.ts"),
				`import { test } from "@playwright/test";
test("fill() prints the password", async ({ page }) => {
  await page.goto("/never");
  await page.getByLabel(/^password/i).fill(${JSON.stringify(SENTINEL)});
  await page.getByLabel(/^password/i).waitFor({ state: "hidden", timeout: 1000 });
});
`,
			);
			mkdirSync(join(out, "playwright-report"), { recursive: true });
			const { code, log } = await run(control, ["-c", "."], {});
			assert.notEqual(code, 0, `the control was meant to fail:\n${log}`);
			const produced = files(out);
			const raw = produced.filter((f) => f.endsWith(".html") && readFileSync(f).includes(SENTINEL));
			assert.deepEqual(raw, [], "the raw bytes of the report already hold the password, so this control proves nothing about the decoding");
			assert.ok(
				leaks(produced, SENTINEL).some((f) => f.endsWith("index.html")),
				"the scan did not find a password that fill() put in the report",
			);
		} finally {
			app.close();
			rmSync(out, { recursive: true, force: true });
			rmSync(control, { recursive: true, force: true });
		}
	});
});

// THE HOSTED SIGN-IN, end to end against two fake hosts. The app (127.0.0.1) sends /sign-in/<mode>
// to a "hosted UI" on localhost, a different host, whose page carries a hidden duplicate of the
// form first (as the Cognito classic UI does). A good sign-in goes back to the app's callback,
// which answers 200 with a meta refresh, as a callback that sets cookies and then redirects
// client-side may.
//   good      the callback sets the app's session cookie
//   nocookie  the callback sets nothing (the hosted host's own cookie must not count)
//   refused   the hosted page answers with its error message and stays put
//   late      the callback answers 200 with no cookie; the page its meta refresh lands on sets it
//   locale    the callback sets nothing; the page its meta refresh lands on sets only locale=en
//   inapp     not hosted at all: the app's own form, served on a sibling host (localhost) of the app
// The app's /api/auth/login sets a transient state cookie in every mode, as auth-hosted does, so
// "the app has some cookie" is true even when the callback sets none.
const HOSTED_FORM = (mode: string, error = "") => `<!doctype html><title>Sign in</title>
<form style="display:none" method="post" action="/login?mode=${mode}"><input name="username"><input name="password" type="password"><input name="signInSubmitButton" type="submit"></form>
${error}<form method="post" action="/login?mode=${mode}"><input name="username"><input name="password" type="password"><input name="signInSubmitButton" type="submit" value="Sign in"></form>`;

// A request-derived mode is only ever one of these, never echoed as received.
const MODES = ["good", "nocookie", "refused", "late", "locale", "inapp"] as const;
const modeOf = (raw: string | null): (typeof MODES)[number] => MODES.find((m) => m === raw) ?? "good";

async function serveHosted(): Promise<{ base: string; close: () => void }> {
	const hosted = createServer((req, res) => {
		const url = new URL(req.url ?? "/", "http://x");
		const mode = modeOf(url.searchParams.get("mode"));
		if (req.method === "POST") {
			req.resume();
			if (mode === "refused") {
				res.writeHead(200, { "Content-Type": "text/html" });
				res.end(HOSTED_FORM(mode, '<p id="loginErrorMessage">Incorrect username or password.</p>'));
				return;
			}
			res.writeHead(302, { Location: `${appBase}/api/auth/callback?mode=${mode}&code=abc`, "Set-Cookie": "XSRF-TOKEN=1; Path=/" });
			res.end();
			return;
		}
		res.writeHead(200, { "Content-Type": "text/html" });
		res.end(HOSTED_FORM(mode));
	});
	await new Promise<void>((resolve) => hosted.listen(0, () => resolve()));
	const hostedBase = `http://localhost:${(hosted.address() as AddressInfo).port}`;
	let appBase = "";
	const app = createServer((req, res) => {
		const url = new URL(req.url ?? "/", "http://x");
		const sign = /^\/sign-in\/(\w+)$/.exec(url.pathname);
		const signMode = sign ? modeOf(sign[1]) : "good";
		if (sign && signMode === "inapp") {
			res.writeHead(302, { Location: `http://localhost:${new URL(appBase).port}/in-app-form` });
		} else if (sign) {
			res.writeHead(302, { Location: `/api/auth/login?mode=${signMode}` });
		} else if (url.pathname === "/api/auth/login") {
			res.writeHead(302, { "Set-Cookie": "auth_state=abc; Path=/", Location: `${hostedBase}/login?mode=${modeOf(url.searchParams.get("mode"))}` });
		} else if (url.pathname === "/api/auth/callback") {
			const callbackMode = modeOf(url.searchParams.get("mode"));
			const headers: Record<string, string> = { "Content-Type": "text/html" };
			if (callbackMode === "good") {
				headers["Set-Cookie"] = "session=ok; Path=/";
			}
			res.writeHead(200, headers);
			res.end(
				`<!doctype html><meta http-equiv="refresh" content="1;url=${callbackMode === "late" ? "/landing-late" : callbackMode === "locale" ? "/landing-locale" : "/"}"><p>Signing you in</p>`,
			);
			return;
		} else if (url.pathname === "/landing-late") {
			res.writeHead(200, { "Content-Type": "text/html", "Set-Cookie": "session=ok; Path=/" });
			res.end("<!doctype html><h1>app</h1>");
			return;
		} else if (url.pathname === "/landing-locale") {
			res.writeHead(200, { "Content-Type": "text/html", "Set-Cookie": "locale=en; Path=/" });
			res.end("<!doctype html><h1>app</h1>");
			return;
		} else if (url.pathname === "/in-app-form") {
			res.writeHead(200, { "Content-Type": "text/html" });
			res.end(`<!doctype html><title>Sign in</title><form onsubmit="return false">
<label>Email address <input type="email"></label><label>Password <input type="password"></label>
<button type="button" onclick="document.cookie='session=ok; path=/'; history.pushState({}, '', '/done'); document.forms[0].remove()">Sign in</button></form>`);
			return;
		} else {
			res.writeHead(200, { "Content-Type": "text/html" });
			res.end("<!doctype html><h1>app</h1>");
			return;
		}
		res.end();
	});
	await new Promise<void>((resolve) => app.listen(0, "127.0.0.1", () => resolve()));
	appBase = `http://127.0.0.1:${(app.address() as AddressInfo).port}`;
	return {
		base: appBase,
		close: () => {
			for (const s of [app, hosted]) {
				s.closeAllConnections();
				s.close();
			}
		},
	};
}

describe("the hosted sign-in", () => {
	async function attempt(mode: string) {
		const servers = await serveHosted();
		const out = mkdtempSync(join(tmpdir(), "albitor-hosted-"));
		rmSync(DEMO_STORAGE_STATE, { force: true });
		try {
			const result = await run(TESTS_DIR, ["--project=setup"], {
				ALBITOR_LOCAL_BASE_URL: servers.base,
				ALBITOR_DEMO_ACCOUNT_PASSWORD: SENTINEL,
				DEMO_ACCOUNT_SIGN_IN_ROUTE: `/sign-in/${mode}`,
				// The fixtures run over plain http, where a __Host- cookie is never stored, so the session is
				// named plainly. auth_state is deliberately accepted here: only the before-submit snapshot
				// keeps the login route's state cookie from passing for a session.
				ALBITOR_SESSION_COOKIE_PATTERN: "^(session|auth_state)$",
				ALBITOR_TESTS_OUT: out,
			});
			const produced = files(out);
			return { ...result, produced, leaked: leaks(produced, SENTINEL), state: hasDemoSession() };
		} finally {
			servers.close();
			rmSync(out, { recursive: true, force: true });
			rmSync(DEMO_STORAGE_STATE, { force: true });
		}
	}

	it("signs in across hosts and saves the app's session once the callback has finished", async () => {
		const { code, log, state } = await attempt("good");
		assert.equal(code, 0, `the hosted sign-in was meant to succeed:\n${log}`);
		assert.equal(state, true, "no session was saved");
	});

	it("fails, saving no session, when the callback sets no cookie on the app's host", async () => {
		const { code, log, state } = await attempt("nocookie");
		assert.notEqual(code, 0, `a callback with no session was meant to fail:\n${log}`);
		assert.match(log, /callback set no session/);
		assert.equal(state, false, "a signed-out run saved something that looks like a session");
	});

	it("fails when only the transient state cookie the login route set is on the app", async () => {
		// The same run as "nocookie" above, named for what it pins: the app host DOES hold a cookie
		// (the state cookie), and the meta refresh lands on "/", yet no session was set.
		const { code, log, state } = await attempt("nocookie");
		assert.notEqual(code, 0, log);
		assert.match(log, /no new cookie for the app's host/);
		assert.equal(state, false);
	});

	it("fails when the landing page sets only an ordinary cookie (locale), not a session", async () => {
		const { code, log, state } = await attempt("locale");
		assert.notEqual(code, 0, log);
		assert.match(log, /callback set no session/);
		assert.equal(state, false);
	});

	it("waits out the callback's meta refresh when the session cookie arrives on the page after it", async () => {
		const { code, log, state } = await attempt("late");
		assert.equal(code, 0, `saving at the callback would have missed the cookie:\n${log}`);
		assert.equal(state, true);
	});

	it("takes the in-app path for the app's own form served on a sibling host", async () => {
		const { code, log, state } = await attempt("inapp");
		assert.equal(code, 0, `an in-app form on another host of the app was meant to sign in:\n${log}`);
		assert.equal(state, true);
	});

	it("fails fast when Cognito refuses the credentials, and the password is in nothing left behind", async () => {
		const started = Date.now();
		const { code, log, produced, leaked } = await attempt("refused");
		assert.notEqual(code, 0, `a refused sign-in was meant to fail:\n${log}`);
		assert.match(log, /Cognito refused the credentials/);
		assert.ok(Date.now() - started < 25_000, "the refusal was not noticed before the return wait ran out");
		assert.ok(produced.some((f) => f.endsWith("error-context.md")), "no failure snapshot was produced, so the scan proves nothing");
		assert.deepEqual(leaked, [], "the password is in these files");
		assert.ok(!log.includes(SENTINEL), "the password is in the console output");
	});
});
