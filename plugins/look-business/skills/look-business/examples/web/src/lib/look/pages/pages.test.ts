// The look's page examples, rendered on the server the way a build prerenders them, loaded into a
// DOM and audited with axe: each has exactly one h1, no skipped heading level, and no violations,
// in the states it takes. Delete a page's line with the page. Colour contrast is audited from the
// tokens in ../../kit/contrast.test.ts, as jsdom has no layout.
import type { Component } from 'svelte';
import { afterEach, describe, expect, it } from 'vitest';
import { audit, serverDom } from '../../kit/__fixtures__/dom';
import FirstRun from './FirstRun.svelte';
import Invoice from './Invoice.svelte';
import Invoices from './Invoices.svelte';
import NewInvoice from './NewInvoice.svelte';
import Overview from './Overview.svelte';
import Preferences from './Preferences.svelte';
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
	['setup, step 3', Setup, { step: 3 }]
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
