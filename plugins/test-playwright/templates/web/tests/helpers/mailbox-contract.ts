// Managed by Albitor (test-playwright). The pure half of the walk mailbox (see mailbox.ts).
import { safeOpenPath } from "./screen-rules.ts";

export type WalkVerification = {
	// The code to type into the app's confirm screen, when the email carried one.
	code?: string;
	// Whether the platform followed the email's confirmation link, when it carried a link it
	// could confirm with a request.
	followed?: boolean;
	// The confirmation link's path and query on this app, when the link opens a page: the spec
	// opens it in the browser (page.goto(open)) so the app's own script can confirm the address.
	open?: string;
};

// parseVerification reads the platform's answer to email `n`. It trusts nothing in it: the code
// is a string, and an `open` path must be a path on this app's own origin. It throws when no
// usable answer is in it, which fails the step that needed the email.
export function parseVerification(raw: string, n: number): WalkVerification {
	let got: { code?: unknown; followed?: unknown; open?: unknown };
	try {
		got = JSON.parse(raw) as typeof got;
	} catch {
		throw new Error(`email ${n}: the walk's mailbox answer is not JSON`);
	}
	if (typeof got.code === "string" && got.code !== "") {
		return { code: got.code };
	}
	if (got.followed === true) {
		return { followed: true };
	}
	const open = safeOpenPath(got.open);
	if (open) {
		return { open };
	}
	throw new Error(`email ${n}: no verification code or followable link reached the walk's mailbox`);
}
