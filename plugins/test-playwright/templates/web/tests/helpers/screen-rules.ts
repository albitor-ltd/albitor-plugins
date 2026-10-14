// Managed by Albitor (test-playwright). Pure rules the journey helper applies to every screen.
//
// Nothing here imports Playwright, so `npm run unit` tests it with plain Node.

export const TITLE_MAX = 60;

// titleProblem says what is wrong with a page title, or "" when nothing is.
//
// The browser tab, a bookmark and a screen reader's page-change announcement (WCAG 2.4.2) all
// read document.title, and a live app was once seen shipping a whole sentence of leaked model
// prose there. So it must be non-empty, at most 60 characters, and not read as prose.
export function titleProblem(raw: string): string {
	const title = raw.trim();
	if (title === "") {
		return "the page has no <title> - the browser tab, bookmarks and a screen reader's page-change announcement all need a real name";
	}
	if (title.length > TITLE_MAX) {
		return `the page <title> is ${title.length} characters - want a short name (at most ${TITLE_MAX}), not a sentence: "${title}"`;
	}
	// More than one sentence-ending mark followed by a space is prose, not a name:
	// "Solitaire. A classic card game. Have fun" reads as three sentences.
	const breaks = (title.match(/[.!?] /g) || []).length;
	if (breaks > 1) {
		return `the page <title> reads as prose, not a name: "${title}"`;
	}
	return "";
}

// onApp reports whether a url is an http(s) page on the origin under test. Only such pages are
// judged: a step that ends on an identity provider's hosted page is that provider's business.
// An empty origin means "not known", and then every http(s) page is judged.
export function onApp(url: string, origin: string): boolean {
	let parsed: URL;
	try {
		parsed = new URL(url);
	} catch {
		return false;
	}
	if (parsed.protocol !== "http:" && parsed.protocol !== "https:") {
		return false;
	}
	return origin === "" || parsed.origin === origin;
}

// safeOpenPath returns a path the platform's mailbox handed over only when it is a path on this
// app's own origin: never one a browser would read as another host. Anything else is undefined.
export function safeOpenPath(value: unknown): string | undefined {
	if (typeof value !== "string" || value === "") {
		return undefined;
	}
	// A control character, tab or newline is stripped by URL parsers, so "/\t/evil.example"
	// would read as "//evil.example".
	// biome-ignore lint/suspicious/noControlCharactersInRegex: that is the point of the check
	if (/[\u0000-\u001f\u007f]/.test(value)) {
		return undefined;
	}
	if (!value.startsWith("/") || value.startsWith("//") || value.startsWith("/\\")) {
		return undefined;
	}
	return value;
}
