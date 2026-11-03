// Starting point (test-playwright). Run with `npm run unit` from web/tests.
import assert from "node:assert/strict";
import { mkdtempSync } from "node:fs";
import { tmpdir } from "node:os";
import { join } from "node:path";
import { afterEach, beforeEach, describe, it } from "node:test";
import { previewSeedPath, readPreviewSeed, seedOncePerPreview } from "../helpers/preview-seed.ts";

describe("seedOncePerPreview", () => {
	beforeEach(() => {
		process.env.TESTS_SEED_DIR = mkdtempSync(join(tmpdir(), "seed-test-"));
		delete process.env.ALBITOR_SEED_REFRESH;
	});
	afterEach(() => {
		delete process.env.TESTS_SEED_DIR;
		delete process.env.ALBITOR_SEED_REFRESH;
	});

	it("seeds once and reuses the result across walks", async () => {
		let calls = 0;
		const seed = async () => ({ id: `rec${++calls}` });
		const a = await seedOncePerPreview("https://p1.example", "gist", seed);
		const b = await seedOncePerPreview("https://p1.example", "gist", seed);
		assert.deepEqual(a, b);
		assert.equal(calls, 1);
		assert.deepEqual(readPreviewSeed("https://p1.example", "gist"), { id: "rec1" });
	});

	it("keys the cache by the preview URL and version, even with an override directory", () => {
		assert.notEqual(previewSeedPath("https://p1.example", "x"), previewSeedPath("https://p2.example", "x"));
		assert.equal(previewSeedPath("https://p1.example/", "x"), previewSeedPath("https://p1.example", "x"));
		const before = previewSeedPath("https://p1.example", "x");
		process.env.ALBITOR_PREVIEW_VERSION = "v2";
		try {
			assert.notEqual(previewSeedPath("https://p1.example", "x"), before);
		} finally {
			delete process.env.ALBITOR_PREVIEW_VERSION;
		}
	});

	it("reseeds when the saved result is no longer valid, or on refresh", async () => {
		let calls = 0;
		const seed = async () => ({ id: `rec${++calls}` });
		await seedOncePerPreview("https://p.example", "n", seed);
		const again = await seedOncePerPreview("https://p.example", "n", seed, async () => false);
		assert.equal(again.id, "rec2");
		process.env.ALBITOR_SEED_REFRESH = "1";
		assert.equal((await seedOncePerPreview("https://p.example", "n", seed)).id, "rec3");
	});

	it("fails the read when the setup has not run, and rejects a path-like name", () => {
		assert.throws(() => readPreviewSeed("https://none.example", "gist"), /has not run/);
		assert.throws(() => previewSeedPath("https://p.example", "../escape"), /Seed name/);
	});
});
