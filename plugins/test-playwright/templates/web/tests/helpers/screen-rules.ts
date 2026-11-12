// Starting point (test-playwright). Pure rules the journey helper applies to every screen.
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

// CODE_CONTEXT_SELECTOR matches the elements whose text a person is meant to read as code or to
// type into, or that never render: text inside any of them is not "source text shown by mistake".
export const CODE_CONTEXT_SELECTOR =
	"pre, code, textarea, input, select, script, style, noscript, template, [contenteditable]:not([contenteditable=false])";

// sourceTextProblem says what is wrong when the screen's visible text contains developer source
// text, or "" when nothing is.
//
// `chunks` is the text of each visible text node, already stripped of everything a person is meant
// to read as code: <pre>, <code>, <textarea>, inputs, contenteditable regions, scripts and styles
// (see visibleTextChunks in journey.ts). A live app once rendered 18 lines of `//` comments above
// its header on every signed-in screen, because two components had comment lines above <script>:
// the markup compiler treats them as text. A person sees that; no other check noticed.
//
// A line is source text when, with URLs removed, it starts with `//` or `/*`, or contains `<!--`.
// A `//` in the middle of a sentence ("a // b") is fine.
export function sourceTextProblem(chunks: string[]): string {
	for (const chunk of chunks) {
		for (const raw of chunk.split(/\r?\n/)) {
			const line = withoutUrls(raw).trim();
			if (line.startsWith("//") || line.startsWith("/*") || line.includes("<!--")) {
				const shown = raw.trim().slice(0, 80);
				return `the screen shows source text to the person: "${shown}" - a comment or markup left where the page renders it (a component's file-level note must go in an HTML comment or inside <script>)`;
			}
		}
	}
	return "";
}

// withoutUrls drops absolute and protocol-relative URLs, which legitimately contain `//`.
function withoutUrls(line: string): string {
	return line
		.replace(/[a-z][a-z0-9+.-]*:\/\/\S*/gi, "")
		.replace(/^(\s*)\/\/[^\s/]+\.[^\s]*/, "$1");
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

// authFor returns the headers to send a request with the journey identity's bearer added, or
// undefined when the request must go as it is: it is not to the app's origin (a CDN, an identity
// provider, an analytics host must never see the token), the origin is unknown, or the request
// already carries its own Authorization.
export function authFor(
	url: string,
	origin: string,
	headers: Record<string, string>,
	auth: Record<string, string>,
): Record<string, string> | undefined {
	if (origin === "" || !onApp(url, origin)) {
		return undefined;
	}
	if (Object.keys(headers).some((k) => k.toLowerCase() === "authorization")) {
		return undefined;
	}
	return { ...headers, ...auth };
}
