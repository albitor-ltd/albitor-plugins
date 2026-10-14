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
