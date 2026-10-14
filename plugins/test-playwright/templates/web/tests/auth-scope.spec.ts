// Managed by Albitor (test-playwright). The journey identity's bearer reaches the app's own origin
// and nowhere else - not another host, and not another host a redirect from the app points at.
import { createServer, type IncomingMessage, type Server } from "node:http";
import type { AddressInfo } from "node:net";
import { expect, test } from "@playwright/test";
import { attachAuth } from "./helpers/local-auth";
import { EMPTY_STORAGE_STATE } from "./helpers/storage-state";

type Seen = { path: string; authorization: string | undefined };

function listen(handler: (req: IncomingMessage, seen: Seen[]) => { status: number; headers?: Record<string, string>; body?: string }) {
	const seen: Seen[] = [];
	const server: Server = createServer((req, res) => {
		seen.push({ path: req.url ?? "", authorization: req.headers.authorization });
		const out = handler(req, seen);
		res.writeHead(out.status, { "Access-Control-Allow-Origin": "*", "Content-Type": "text/html", ...out.headers });
		res.end(out.body ?? "<!doctype html><title>x</title><main>ok</main>");
	});
	return new Promise<{ origin: string; seen: Seen[]; close: () => Promise<void> }>((resolve) => {
		server.listen(0, "127.0.0.1", () => {
			const { port } = server.address() as AddressInfo;
			resolve({
				origin: `http://127.0.0.1:${port}`,
				seen,
				close: () => new Promise((done) => server.close(() => done())),
			});
		});
	});
}

const AUTH = { Authorization: "Bearer TOK" };

test.describe("attachAuth", () => {
	test("sends the bearer to the app origin and to no other host, redirects included", async ({ browser }) => {
		const other = await listen(() => ({ status: 200, body: "other" }));
		const app = await listen((req) => {
			if (req.url === "/redir") {
				return { status: 302, headers: { Location: `${other.origin}/landed` } };
			}
			return { status: 200 };
		});
		const context = await browser.newContext({ storageState: EMPTY_STORAGE_STATE });
		try {
			await attachAuth(context, AUTH, app.origin);
			const page = await context.newPage();
			await page.goto(`${app.origin}/`);
			// A same-origin call carries the identity.
			await page.evaluate(() => fetch("/api/ping").then((r) => r.text()));
			expect(app.seen.find((s) => s.path === "/api/ping")?.authorization).toBe("Bearer TOK");

			// A call to another host does not.
			await page.evaluate((url) => fetch(url).then((r) => r.text()), `${other.origin}/direct`);
			// A same-origin call that the app redirects to another host must not carry it there.
			// (It may fail closed rather than complete: what matters is that the bearer does not arrive.)
			await page.evaluate(() => fetch("/redir").then((r) => r.text()).catch(() => "failed closed"));
			// Nor must a navigation that is redirected.
			await page.goto(`${app.origin}/redir`);

			const landed = other.seen.filter((s) => s.path === "/direct" || s.path === "/landed");
			expect(landed.length).toBeGreaterThanOrEqual(3);
			for (const s of landed) {
				expect(s.authorization, `${s.path} received the bearer`).toBeUndefined();
			}
		} finally {
			await context.close();
			await app.close();
			await other.close();
		}
	});

	test("a same-origin redirect is followed by the browser without the bearer", async ({ browser }) => {
		const app = await listen((req) => {
			if (req.url === "/api/old") {
				return { status: 302, headers: { Location: "/api/new" } };
			}
			return { status: 200 };
		});
		const context = await browser.newContext({ storageState: EMPTY_STORAGE_STATE });
		try {
			await attachAuth(context, AUTH, app.origin);
			const page = await context.newPage();
			await page.goto(`${app.origin}/`);
			await page.evaluate(() => fetch("/api/old").then((r) => r.text()));
			expect(app.seen.find((s) => s.path === "/api/new")).toBeDefined();
			// Documented, not wanted: the hop after a redirect does not come back through the route, so an
			// API that redirects to another API path sees an anonymous request and fails closed.
			expect(app.seen.find((s) => s.path === "/api/new")?.authorization).toBeUndefined();
		} finally {
			await context.close();
			await app.close();
		}
	});

	test("script and module-preload fetches are not round-tripped", async ({ browser }) => {
		const app = await listen((req) => {
			if (req.url === "/m.js") {
				return { status: 200, headers: { "Content-Type": "text/javascript" }, body: "export const x = 1;" };
			}
			return {
				status: 200,
				body: '<!doctype html><title>x</title><link rel="modulepreload" href="/m.js"><main>ok</main>',
			};
		});
		const context = await browser.newContext({ storageState: EMPTY_STORAGE_STATE });
		try {
			await attachAuth(context, AUTH, app.origin);
			const page = await context.newPage();
			await page.goto(`${app.origin}/`);
			await expect.poll(() => app.seen.some((s) => s.path === "/m.js")).toBe(true);
			expect(app.seen.find((s) => s.path === "/m.js")?.authorization).toBeUndefined();
		} finally {
			await context.close();
			await app.close();
		}
	});

	test("a streaming call is left alone rather than buffered, and goes without the bearer", async ({ browser }) => {
		const app = await listen(() => ({ status: 200, headers: { "Content-Type": "text/event-stream" }, body: "data: 1\n\n" }));
		const context = await browser.newContext({ storageState: EMPTY_STORAGE_STATE });
		try {
			await attachAuth(context, AUTH, app.origin);
			const page = await context.newPage();
			await page.goto(`${app.origin}/`);
			await page.evaluate(() => fetch("/api/stream", { headers: { Accept: "text/event-stream" } }).then((r) => r.text()));
			expect(app.seen.find((s) => s.path === "/api/stream")?.authorization).toBeUndefined();
		} finally {
			await context.close();
			await app.close();
		}
	});

	test("a request still in flight when the context closes does not raise", async ({ browser }) => {
		const app = await listen((req) => ({ status: 200, body: req.url ?? "" }));
		const raised: string[] = [];
		const onError = (err: unknown) => raised.push(String(err));
		process.on("unhandledRejection", onError);
		process.on("uncaughtException", onError);
		try {
			for (let round = 0; round < 8; round++) {
				const context = await browser.newContext({ storageState: EMPTY_STORAGE_STATE });
				await attachAuth(context, AUTH, app.origin);
				const page = await context.newPage();
				await page.goto(`${app.origin}/`);
				await page.evaluate(() => {
					for (let i = 0; i < 40; i++) {
						fetch(`/api/slow-${i}`).catch(() => {});
					}
				});
				await context.close();
			}
			await new Promise((r) => setTimeout(r, 500));
			expect(raised).toEqual([]);
		} finally {
			process.off("unhandledRejection", onError);
			process.off("uncaughtException", onError);
			await app.close();
		}
	});

	test("leaves a request that carries its own Authorization alone", async ({ browser }) => {
		const app = await listen(() => ({ status: 200 }));
		const context = await browser.newContext({ storageState: EMPTY_STORAGE_STATE });
		try {
			await attachAuth(context, AUTH, app.origin);
			const page = await context.newPage();
			await page.goto(`${app.origin}/`);
			await page.evaluate(() => fetch("/own", { headers: { Authorization: "Bearer OWN" } }).then((r) => r.text()));
			expect(app.seen.find((s) => s.path === "/own")?.authorization).toBe("Bearer OWN");
		} finally {
			await context.close();
			await app.close();
		}
	});
});
