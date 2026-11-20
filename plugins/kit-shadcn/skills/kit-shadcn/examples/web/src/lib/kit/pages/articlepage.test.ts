// ArticlePage in each state (ArticlePageFixture), rendered on the server the way a build prerenders it,
// loaded into a DOM and audited with axe. Delete this file and the fixture with ArticlePage.svelte.
import type { JSDOM } from 'jsdom';
import { afterEach, describe, expect, it } from 'vitest';
import { audit, serverDom } from '../__fixtures__/dom';
import Fixture from './__fixtures__/ArticlePageFixture.svelte';
import { expectOneH1AndNoSkippedLevel } from './__fixtures__/helpers';

let dom: JSDOM;

function mount(props: Record<string, unknown> = {}): HTMLElement {
	dom = serverDom(Fixture, props, 'Pages');
	return dom.window.document.body;
}

afterEach(() => dom?.window.close());

const cases: Array<[name: string, props: Record<string, unknown>]> = [
	['article', {}],
	['article, loading', { status: 'loading' }],
	['article, failed', { status: 'error' }]
];

describe('ArticlePage in each state', () => {
	it.each(cases)('%s has no axe violations', async (_name, props) => {
		mount(props);
		expect(await audit(dom)).toEqual([]);
	});

	it.each(cases)('%s has exactly one h1 and no skipped heading level', (_name, props) => {
		expectOneH1AndNoSkippedLevel(mount(props));
	});
});

describe('ArticlePage', () => {
	it('sets the body in an article with the byline, date and reading time before it', () => {
		const root = mount();
		expect(root.querySelector('article .meta')?.textContent).toBe('Sam Rowe · 2026-11-20 · 2 min read');
		expect(root.querySelector('article h2')?.textContent).toBe('Weekends');
	});

	it('limits the line length to a reading width', () => {
		const root = mount();
		expect(root.querySelector('article')).not.toBeNull();
		// The width is in the stylesheet; the class that carries it is the contract.
		expect(root.querySelector('article .prose')).not.toBeNull();
	});

	it('names the related-reading nav and lists its links', () => {
		const root = mount();
		const nav = root.querySelector('nav.related');
		expect(nav?.getAttribute('aria-labelledby')).toBe('related-heading');
		expect(nav?.querySelector('a')?.textContent).toBe('Parking at the hall');
	});
});
