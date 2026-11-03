// Starting point (test-playwright).
import { createHash } from "node:crypto";
import { mkdirSync, readFileSync, renameSync, writeFileSync } from "node:fs";
import { tmpdir, userInfo } from "node:os";
import { join } from "node:path";

// Seed a deployed preview ONCE, through the app's real API, and reuse the result across every
// walk of that preview.
//
// Some records are slow or costly to make for real: a media upload that goes through the app's
// pipeline and a supplier, a document that must be processed before a screen has anything to show.
// A walk that makes its own on every run pays that each time and fails on the first slow step. So
// a setup step (a `seed.setup.ts` project that `chromium` depends on, beside auth.setup.ts) calls
// seedOncePerPreview with a function that drives the app's own API as the demo identity, and
// the specs read the saved ids.
//
// NEVER ADD A TEST-ONLY ROUTE TO THE APP (a /test/seed or /api/test-fixtures endpoint). It ships
// to the deployed app, where anyone holding the demo account can write records that skip the
// real pipeline. The seed goes through the routes a person uses, or it does not exist.
//
// The cache is keyed by the preview's URL, so a new preview seeds afresh and a stale one is never
// reused. PREVIEW_SEED_REFRESH=1 forces a reseed of the same URL. Pass `stillValid` to check a
// saved result against the live app (for example, GET the record) and reseed when it is gone.
// Seeded records are not removed by fixtureScope: the preview is thrown away, not cleaned.
// Two setup runs started at the same time on one preview both find no saved seed and both seed
// (the last write wins); run one walk per preview at a time, or have `seed` tolerate a duplicate.

// The key is the preview URL plus, when the walk sets one, PREVIEW_SEED_VERSION (a deploy id or
// commit): a redeploy of the same URL with a new version seeds afresh. TESTS_SEED_DIR only moves the
// parent directory; the per-preview key is always kept, so two previews never share a seed.
function seedDir(baseURL: string): string {
	const id = `${baseURL.replace(/\/+$/, "")}|${process.env.PREVIEW_SEED_VERSION || ""}`;
	const key = createHash("sha256").update(id).digest("hex").slice(0, 16);
	const parent = process.env.TESTS_SEED_DIR || tmpdir();
	return join(parent, `tests-seed-${userInfo().uid}-${key}`);
}

export function previewSeedPath(baseURL: string, name: string): string {
	if (!/^[a-z0-9][a-z0-9-]{0,63}$/.test(name)) {
		throw new Error(`Seed name must be lowercase letters, digits and hyphens: ${name}`);
	}
	return join(seedDir(baseURL), `${name}.json`);
}

export async function seedOncePerPreview<T>(
	baseURL: string,
	name: string,
	seed: () => Promise<T>,
	stillValid?: (saved: T) => Promise<boolean>,
): Promise<T> {
	if (!baseURL) throw new Error("seedOncePerPreview needs the preview's base URL");
	const path = previewSeedPath(baseURL, name);
	if (process.env.PREVIEW_SEED_REFRESH !== "1") {
		let saved: T | undefined;
		try {
			saved = JSON.parse(readFileSync(path, "utf8")) as T;
		} catch {
			saved = undefined; // absent or unreadable: seed
		}
		if (saved !== undefined && (!stillValid || (await stillValid(saved)))) return saved;
	}
	const fresh = await seed();
	mkdirSync(seedDir(baseURL), { recursive: true, mode: 0o700 });
	const tmp = `${path}.${process.pid}.tmp`;
	writeFileSync(tmp, JSON.stringify(fresh), { mode: 0o600 });
	renameSync(tmp, path);
	return fresh;
}

// Read what the setup step saved. Throws when the setup did not run, so a spec fails for the
// right reason instead of browsing an empty app.
export function readPreviewSeed<T>(baseURL: string, name: string): T {
	try {
		return JSON.parse(readFileSync(previewSeedPath(baseURL, name), "utf8")) as T;
	} catch {
		throw new Error(`No saved seed "${name}" for ${baseURL}: the seed setup project has not run`);
	}
}
