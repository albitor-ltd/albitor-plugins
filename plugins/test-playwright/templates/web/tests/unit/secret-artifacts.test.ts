// Managed by Albitor (test-playwright). Run with `npm run unit` from web/tests (needs Chromium:
// `npx playwright install chromium`).
//
// The demo password must not land in anything a failed sign-in leaves behind: the per-failure
// page snapshot (error-context.md), the HTML report CI uploads, a trace. This runs the real setup
// project against a sign-in page that never succeeds, with a sentinel password and CI=1 (so a retry
// happens too), then searches every output file for the sentinel.
import assert from "node:assert/strict";
import { spawn } from "node:child_process";
import { mkdtempSync, readdirSync, readFileSync, rmSync, statSync } from "node:fs";
import { createServer } from "node:http";
import type { AddressInfo } from "node:net";
import { tmpdir } from "node:os";
import { join } from "node:path";
import { describe, it } from "node:test";
import { TESTS_DIR } from "../helpers/storage-state.ts";

const SENTINEL = "SENTINEL-pw-8841";

// A sign-in screen that takes the password and never signs anyone in.
const PAGE = `<!doctype html><title>Sign in</title><main><h1>Sign in</h1>
<form data-testid="sign-in-form" onsubmit="return false">
<label>Email <input data-testid="sign-in-email" type="email"></label>
<label>Password <input data-testid="sign-in-password" type="password"></label>
<button data-testid="sign-in-button" type="button">Sign in</button></form></main>`;

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

describe("a failed sign-in", () => {
	it("leaves the password in no snapshot, report or trace", async () => {
		const server = createServer((_req, res) => {
			res.writeHead(200, { "Content-Type": "text/html" });
			res.end(PAGE);
		});
		await new Promise<void>((resolve) => server.listen(0, "127.0.0.1", () => resolve()));
		const base = `http://127.0.0.1:${(server.address() as AddressInfo).port}`;
		const out = mkdtempSync(join(tmpdir(), "albitor-secret-"));
		try {
			// Not spawnSync: that would block the event loop that serves the page.
			const child = spawn("npx", ["playwright", "test", "--project=setup"], {
				cwd: TESTS_DIR,
				env: {
					...process.env,
					CI: "1",
					ALBITOR_LOCAL_BASE_URL: base,
					ALBITOR_DEMO_ACCOUNT_PASSWORD: SENTINEL,
					ALBITOR_TESTS_OUT: out,
					ALBITOR_JOURNEY_CREDENTIALS: "",
				},
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
			assert.notEqual(code, 0, `the sign-in was meant to fail:\n${log}`);

			const produced = files(out);
			assert.ok(produced.some((f) => f.endsWith("error-context.md")), `no failure snapshot was produced:\n${produced.join("\n")}`);
			assert.ok(produced.some((f) => f.includes("playwright-report")), "no HTML report was produced");
			assert.deepEqual(produced.filter((f) => f.endsWith(".zip")), [], "a trace was recorded");
			for (const f of produced) {
				assert.ok(!readFileSync(f).includes(SENTINEL), `the password is in ${f}`);
			}
			assert.ok(!log.includes(SENTINEL), "the password is in the console output");
		} finally {
			server.close();
			rmSync(out, { recursive: true, force: true });
		}
	});
});
