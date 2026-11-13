// Checks the page tests share. Test support, in the folder `web/knip.json` ignores.
import { expect } from 'vitest';

/** Every summary link is an in-page anchor to a form control that exists. */
export function expectSummaryLinksResolve(root: HTMLElement): void {
	const links = [...root.querySelectorAll('.summary a')];
	expect(links.length).toBeGreaterThan(0);
	for (const link of links) {
		const href = link.getAttribute('href') ?? '';
		expect(href).toMatch(/^#.+/);
		const target = root.ownerDocument.getElementById(href.slice(1));
		expect(target?.matches('input, textarea, select')).toBe(true);
	}
}

/** Exactly one h1, and no heading level skipped. */
export function expectOneH1AndNoSkippedLevel(root: HTMLElement): void {
	expect(root.querySelectorAll('h1')).toHaveLength(1);
	const levels = [...root.querySelectorAll('h1, h2, h3, h4')].map((h) => Number(h.tagName[1]));
	levels.forEach((level, i) => {
		if (i > 0) expect(level - levels[i - 1]).toBeLessThanOrEqual(1);
	});
}
