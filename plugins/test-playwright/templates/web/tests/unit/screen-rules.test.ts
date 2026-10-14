// Managed by Albitor (test-playwright). Run with `npm run unit` from web/tests.
import assert from "node:assert/strict";
import { describe, it } from "node:test";
import { authFor, onApp, safeOpenPath, titleProblem } from "../helpers/screen-rules.ts";

describe("titleProblem", () => {
	it("accepts a short name", () => {
		assert.equal(titleProblem("Claims queue"), "");
		assert.equal(titleProblem("  Home  "), "");
	});
	it("refuses an empty title", () => {
		assert.match(titleProblem(""), /no <title>/);
		assert.match(titleProblem("   "), /no <title>/);
	});
	it("refuses a title over 60 characters", () => {
		assert.match(titleProblem("x".repeat(61)), /61 characters/);
		assert.equal(titleProblem("x".repeat(60)), "");
	});
	it("refuses prose of more than one sentence", () => {
		assert.match(titleProblem("Solitaire. A classic card game. Have fun"), /prose/);
		assert.equal(titleProblem("Solitaire. A card game"), "");
	});
});

describe("onApp", () => {
	it("judges only http(s) pages on the app origin", () => {
		assert.equal(onApp("http://localhost:3000/a", "http://localhost:3000"), true);
		assert.equal(onApp("http://localhost:3001/a", "http://localhost:3000"), false);
		assert.equal(onApp("https://idp.example/login", "http://localhost:3000"), false);
	});
	it("refuses non-http schemes and unparseable urls", () => {
		assert.equal(onApp("about:blank", ""), false);
		assert.equal(onApp("javascript:alert(1)", ""), false);
		assert.equal(onApp("not a url", ""), false);
	});
	it("judges every http(s) page when the origin is unknown", () => {
		assert.equal(onApp("https://anywhere.example/", ""), true);
	});
});

describe("safeOpenPath", () => {
	it("accepts a path on the app's own origin", () => {
		assert.equal(safeOpenPath("/confirm?code=123"), "/confirm?code=123");
	});
	it("refuses anything a browser would read as another host", () => {
		for (const bad of ["//evil.example/x", "/\\evil.example", "https://evil.example/", "confirm", "", undefined, 7]) {
			assert.equal(safeOpenPath(bad), undefined, String(bad));
		}
	});
	it("refuses control characters and tab or newline tricks", () => {
		assert.equal(safeOpenPath("/a\n//evil.example"), undefined);
		assert.equal(safeOpenPath("/\t/evil.example"), undefined);
	});
});

describe("authFor", () => {
	const auth = { Authorization: "Bearer t" };
	it("adds the identity to a request to the app's origin only", () => {
		assert.deepEqual(authFor("http://localhost:3000/api/x", "http://localhost:3000", { a: "1" }, auth), {
			a: "1",
			Authorization: "Bearer t",
		});
		assert.equal(authFor("https://cdn.example/x.js", "http://localhost:3000", {}, auth), undefined);
		assert.equal(authFor("http://localhost:3001/", "http://localhost:3000", {}, auth), undefined);
	});
	it("never sends a bearer when the origin is unknown", () => {
		assert.equal(authFor("http://localhost:3000/", "", {}, auth), undefined);
	});
	it("does not overwrite an Authorization header the request already carries", () => {
		const got = authFor("http://localhost:3000/", "http://localhost:3000", { authorization: "Bearer own" }, auth);
		assert.equal(got, undefined);
	});
});
