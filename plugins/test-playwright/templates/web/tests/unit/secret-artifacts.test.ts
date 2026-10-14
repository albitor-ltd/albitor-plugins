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
import { DEMO_STORAGE_STATE, TESTS_DIR } from "../helpers/storage-state.ts";

const SENTINEL = "SENTINEL-pw-8841";

// The sign-in screens a sign-in is tried against, served by path. Each is one way it can go, and
// each would leave the password somewhere different if the setup were careless:
//   /never     takes the password and signs no one in
//   /disabled  the sign-in button is disabled, so click() waits for it until its timeout
//   /overlay   something covers the button, so click() waits for it until its timeout
//   /bound     a controlled input in the style of a UI framework (see below) that DOES sign in
const FIELDS = (extra: string) => `<!doctype html><title>Sign in</title><main><h1>Sign in</h1>
<form data-testid="sign-in-form" onsubmit="return false">
<label>Email <input data-testid="sign-in-email" type="email"></label>
<label>Password <input data-testid="sign-in-password" type="password"></label>
${extra}</form></main>`;

const BUTTON = '<button data-testid="sign-in-button" type="button">Sign in</button>';

const PAGES: Record<string, string> = {
	"/never": FIELDS(BUTTON),
	"/disabled": FIELDS(BUTTON.replace("<button ", "<button disabled ")),
	"/overlay": `${FIELDS(BUTTON)}<div id="veil" style="position:fixed;inset:0;background:#fff8"></div>`,
	// A framework-bound input. Like React, it replaces the value property on the INSTANCE to track
	// what it last saw, and only believes a change when an input event arrives with a value that
	// differs from the tracked one. Setting the value through the prototype's native setter and then
	// dispatching "input" is what satisfies it. The button signs in only with the right password.
	"/bound": `${FIELDS(BUTTON)}
<script>
const input = document.querySelector('[data-testid=sign-in-password]');
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
document.querySelector('[data-testid=sign-in-button]').addEventListener('click', () => {
	if (state === ${JSON.stringify(SENTINEL)}) {
		document.cookie = 'session=ok; path=/';
		document.querySelector('[data-testid=sign-in-form]').remove();
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
		res.writeHead(200, { "Content-Type": "text/html" });
		res.end(PAGES[req.url ?? ""] ?? PAGES["/never"]);
	});
	await new Promise<void>((resolve) => server.listen(0, "127.0.0.1", () => resolve()));
	return { base: `http://127.0.0.1:${(server.address() as AddressInfo).port}`, close: () => server.close() };
}

describe("the demo password", () => {
	for (const route of ["/never", "/disabled", "/overlay"]) {
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
  await page.getByTestId("sign-in-password").fill(${JSON.stringify(SENTINEL)});
  await page.getByTestId("sign-in-form").waitFor({ state: "hidden", timeout: 1000 });
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
