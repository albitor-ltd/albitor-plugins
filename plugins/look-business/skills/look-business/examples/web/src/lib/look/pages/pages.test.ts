// The look's page examples, rendered on the server the way a build prerenders them, loaded into a
// DOM and audited with axe: each has exactly one h1, no skipped heading level, and no violations,
// in the states it takes. Delete a page's line with the page. Colour contrast is audited from the
// tokens in ../../kit/contrast.test.ts, as jsdom has no layout.
import type { Component } from 'svelte';
import { afterEach, describe, expect, it } from 'vitest';
import { audit, serverDom } from '../../kit/__fixtures__/dom';
import FirstRun from './FirstRun.svelte';
import Hub from './Hub.svelte';
import Invoice from './Invoice.svelte';
import Invoices from './Invoices.svelte';
import Letters from './Letters.svelte';
import NewInvoice from './NewInvoice.svelte';
import Overview from './Overview.svelte';
import Preferences from './Preferences.svelte';
import Renew from './Renew.svelte';
import Setup from './Setup.svelte';

let dom: ReturnType<typeof serverDom>;
afterEach(() => dom?.window.close());

const cases: Array<[name: string, page: Component<any>, props: Record<string, unknown>]> = [
	['overview', Overview, {}],
	['overview, loading', Overview, { status: 'loading' }],
	['overview, failed', Overview, { status: 'error' }],
	['invoices', Invoices, {}],
	['invoices, none yet', Invoices, { rows: [] }],
	['invoices, loading', Invoices, { status: 'loading' }],
	['invoice', Invoice, {}],
	['new invoice', NewInvoice, {}],
	['new invoice, with errors', NewInvoice, { errors: [{ field: 'supplier', message: 'Enter a supplier' }] }],
	['settings', Preferences, {}],
	['first run', FirstRun, {}],
	['setup, step 1', Setup, { step: 1 }],
	['setup, step 3', Setup, { step: 3 }],
	['hub (portal)', Hub, {}],
	['hub, loading', Hub, { status: 'loading' }],
	['letters', Letters, {}],
	['letters, none yet', Letters, { rows: [] }],
	['renew, question', Renew, {}],
	['renew, with errors', Renew, { errors: [{ field: 'registration', message: 'Enter the registration number' }] }],
	['renew, check your answers', Renew, { phase: 'check' }],
	['renew, confirmation', Renew, { phase: 'done' }]
];

describe.each(cases)('%s', (_name, page, props) => {
	it('has no axe violations', async () => {
		dom = serverDom(page, props, 'Example');
		expect(await audit(dom)).toEqual([]);
	});

	it('has exactly one h1 and no skipped heading level', () => {
		dom = serverDom(page, props, 'Example');
		const doc = dom.window.document;
		expect(doc.querySelectorAll('h1')).toHaveLength(1);
		const levels = [...doc.querySelectorAll('h1, h2, h3, h4')].map((h) => Number(h.tagName[1]));
		levels.forEach((level, i) => {
			if (i > 0) expect(level - levels[i - 1]).toBeLessThanOrEqual(1);
		});
	});
});

describe('the portal example app', () => {
	it('keeps a link back to the hub at the top of every inner page', () => {
		for (const [page, props] of [[Letters, {}], [Renew, {}], [Renew, { phase: 'check' }], [Renew, { phase: 'done' }]] as const) {
			dom = serverDom(page, props, 'Example');
			const link = dom.window.document.querySelector('p.back a');
			expect(link?.getAttribute('href')).toBe('/');
			expect(link?.textContent?.trim()).toBe('Back to your permits');
			dom.window.close();
		}
	});

	it('links every destination from a tile on the hub', () => {
		dom = serverDom(Hub, {}, 'Example');
		const hrefs = [...dom.window.document.querySelectorAll('.tiles h3 a')].map((a) => a.getAttribute('href'));
		expect(hrefs).toEqual(['/renew', '/payments', '/letters']);
	});
});
