// Test support for the kit's tests: render a fixture on the server the way a build prerenders it,
// load it into a DOM, and audit it with axe. Fixtures live in this folder and nothing else imports
// them, so `web/knip.json` ignores the folder and the end-of-build prune never targets it.
import axe from 'axe-core';
import { JSDOM } from 'jsdom';
import type { Component } from 'svelte';
import { render } from 'svelte/server';

/** A fixture rendered on the server into a fresh DOM. Close the window after the test. */
export function serverDom(
	component: Component<any>,
	props: Record<string, unknown> = {},
	title = 'Kit'
): JSDOM {
	const { body } = render(component, { props });
	return new JSDOM(`<!doctype html><html lang="en"><head><title>${title}</title></head><body>${body}</body></html>`, {
		runScripts: 'outside-only'
	});
}

/**
 * axe over the DOM: the WCAG 2.2 AA and best-practice rules. jsdom has no layout, so colour contrast
 * is audited from the tokens in contrast.test.ts and switched off here.
 */
export async function audit(dom: JSDOM): Promise<string[]> {
	const win = dom.window as unknown as { eval(code: string): unknown; axe: typeof axe };
	win.eval(axe.source);
	const result = await win.axe.run(dom.window.document.body, {
		runOnly: { type: 'tag', values: ['wcag2a', 'wcag2aa', 'wcag21a', 'wcag21aa', 'wcag22aa', 'best-practice'] },
		rules: { 'color-contrast': { enabled: false }, region: { enabled: false } }
	});
	return result.violations.map((v) => `${v.id}: ${v.nodes.map((n) => n.html).join(' | ')}`);
}
