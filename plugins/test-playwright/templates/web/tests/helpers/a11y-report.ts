// Managed by Albitor (test-playwright). The pure half of the accessibility audit: which axe tags
// the audit runs, how a waiver is validated and how violations are reported. Nothing here imports
// Playwright or axe, so `npm run unit` tests it with plain Node.

// Every Level A and AA success criterion of WCAG 2.0, 2.1 and 2.2, and nothing else: the
// Internal Tool governance profile's bar. Best-practice rules and Level AAA are not asked for.
export const WCAG_TAGS = ["wcag2a", "wcag2aa", "wcag21a", "wcag21aa", "wcag22aa"] as const;

export type AxeNode = { target: unknown[]; html?: string };
export type AxeViolation = {
	id: string;
	impact?: string | null;
	help: string;
	helpUrl: string;
	nodes: AxeNode[];
};

// A waiver switches one rule off for one audit. It always carries a reason, which the failure
// output repeats, so a waiver is a decision someone can read and not a silent mute.
export type Waiver = { rule: string; reason: string };

const RULE_ID = /^[a-z][a-z0-9-]{0,63}$/;
const MAX_NODES_SHOWN = 10;
const PAGE_ROOTS = /^(html|body|main|:root|\*|\[role=["']?main["']?\])$/;

// checkWaivers returns what is wrong with a list of waivers, or [] when it is fit to use.
export function checkWaivers(waivers: readonly Waiver[]): string[] {
	const problems: string[] = [];
	const seen = new Set<string>();
	for (const w of waivers) {
		if (!RULE_ID.test(w.rule)) {
			problems.push(`"${w.rule}" is not an axe rule id`);
		}
		if (typeof w.reason !== "string" || w.reason.trim() === "") {
			problems.push(`the waiver for "${w.rule}" needs a reason`);
		}
		if (seen.has(w.rule)) {
			problems.push(`"${w.rule}" is waived twice`);
		}
		seen.add(w.rule);
	}
	return problems;
}

// checkSelectors returns what is wrong with an include or exclude list. A selector that is the
// whole page, or its main landmark, would switch the audit off without a waiver's reason, so it is
// refused: exclude the one third-party element instead.
export function checkSelectors(selectors: readonly string[], kind: "include" | "exclude"): string[] {
	const problems: string[] = [];
	for (const raw of selectors) {
		if (raw.trim() === "") {
			problems.push(`an empty ${kind} selector`);
			continue;
		}
		// A list is judged part by part; a part is the whole page when every compound in it is a page
		// root, however they are combined (`body > *`, `html > body`, `:root *`).
		// :not(...) only narrows a root by a hair (`body:not(.x)` is still the page), and :is() and
		// :where() wrap a list (`:is(body)` is body), so the first is dropped and the others unwrapped.
		// Position and :has() pseudos narrow a root just as little. Arguments may hold quoted parens.
		const ARGS = String.raw`\(((?:[^()'"]|'[^']*'|"[^"]*")*)\)`;
		const DROP = new RegExp(String.raw`:(?:not|has|nth-[a-z-]+)${ARGS}|:(?:first|last|only)-(?:child|of-type)`, "gi");
		const UNWRAP = new RegExp(String.raw`:(?:is|where|matches|-webkit-any|-moz-any)${ARGS}`, "gi");
		let flat = raw;
		for (let prev = ""; prev !== flat; ) {
			prev = flat;
			flat = flat.replace(DROP, "").replace(UNWRAP, "$1");
		}
		// A part that is nothing once those are gone (`:not(:not(body))`) selects anything at all.
		const wholePage = flat.split(",").some((part) => {
			const p = part.trim().toLowerCase();
			return p === "" || p.split(/[\s>+~]+/).every((tok) => PAGE_ROOTS.test(tok));
		});
		if (wholePage) {
			problems.push(`the ${kind} selector "${raw.trim()}" is the whole page; name the one element`);
		}
	}
	return problems;
}

// unwaived drops the violations of waived rules.
export function unwaived(violations: readonly AxeViolation[], waivers: readonly Waiver[]): AxeViolation[] {
	const waived = new Set(waivers.map((w) => w.rule));
	return violations.filter((v) => !waived.has(v.id));
}

// describeViolations renders violations for a failure message: the rule, its impact, where to read
// about it and the elements at fault (at most ten each, so a page of failures stays readable).
export function describeViolations(violations: readonly AxeViolation[]): string {
	return violations
		.map((v) => {
			const shown = v.nodes.slice(0, MAX_NODES_SHOWN).map((n) => `    ${n.target.join(" ")}`);
			const more = v.nodes.length - shown.length;
			if (more > 0) {
				shown.push(`    ... and ${more} more`);
			}
			return [`- ${v.id} (${v.impact ?? "unknown"}): ${v.help}`, `  ${v.helpUrl}`, ...shown].join("\n");
		})
		.join("\n");
}
