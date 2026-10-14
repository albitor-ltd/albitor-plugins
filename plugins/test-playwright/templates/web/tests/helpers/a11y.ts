// Managed by Albitor (test-playwright). The accessibility audit.
//
// WHAT IT IS: axe-core, run in the page, against WCAG 2.2 Level A and AA - the bar the Internal
// Tool governance profile sets. It fails on a violation and says which rule, where to read about
// it and which elements.
//
// WHAT IT IS NOT: a conformance claim. Automated tools find roughly a third of WCAG problems.
// Reading order, meaningful text alternatives, understandable errors and keyboard operation of
// custom widgets still need a person (see the wcag-22-aa skill). A green audit means "no known
// automatable failure", never "accessible".
import AxeBuilder from "@axe-core/playwright";
import type { Page } from "@playwright/test";
import { WCAG_TAGS, type Waiver, checkSelectors, checkWaivers, describeViolations, unwaived } from "./a11y-report";

export type A11yOptions = {
	// CSS selectors the audit looks inside. Default: the whole page.
	include?: string[];
	// CSS selectors the audit skips, for markup the app does not own (a third-party embed). Never
	// the whole page or main: that switches the audit off.
	exclude?: string[];
	// Rules switched off, each with the reason. A waiver with no reason fails the audit.
	waive?: Waiver[];
};

// expectNoA11yViolations audits the page as it is now and throws if axe finds a violation of a
// WCAG 2.2 A or AA rule. Call it after the screen has settled (a journeyStep body, or a spec's
// own assertion that the screen is ready), because it audits only what is rendered.
export async function expectNoA11yViolations(page: Page, options: A11yOptions = {}): Promise<void> {
	const waivers = options.waive ?? [];
	const bad = [...checkWaivers(waivers), ...checkSelectors(options.include ?? [], "include"), ...checkSelectors(options.exclude ?? [], "exclude")];
	if (bad.length > 0) {
		throw new Error(`the accessibility audit was given an invalid option: ${bad.join("; ")}`);
	}

	let builder = new AxeBuilder({ page }).withTags([...WCAG_TAGS]);
	for (const selector of options.include ?? []) {
		builder = builder.include(selector);
	}
	for (const selector of options.exclude ?? []) {
		builder = builder.exclude(selector);
	}
	const results = await builder.analyze();

	const violations = unwaived(results.violations, waivers);
	if (violations.length > 0) {
		const waived = waivers.length > 0 ? `\nwaived: ${waivers.map((w) => `${w.rule} (${w.reason})`).join("; ")}` : "";
		throw new Error(
			`${violations.length} WCAG 2.2 A/AA accessibility violation(s) on ${new URL(page.url()).pathname}:\n${describeViolations(violations)}${waived}`,
		);
	}
}
