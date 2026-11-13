// ErrorSummary, rendered on the server inside the shell (__fixtures__/ErrorSummaryFixture.svelte)
// and audited with axe. Delete this file and the fixture with ErrorSummary.svelte if the app has no
// form errors to summarise.
import type { JSDOM } from 'jsdom';
import { afterEach, describe, expect, it } from 'vitest';
import { audit as auditDom, serverDom } from './__fixtures__/dom';
import { render } from 'svelte/server';
import ErrorSummary from './ErrorSummary.svelte';
import ErrorSummaryFixture from './__fixtures__/ErrorSummaryFixture.svelte';

let dom: JSDOM;

async function mount(props: Record<string, unknown> = {}): Promise<HTMLElement> {
	dom = serverDom(ErrorSummaryFixture, props);
	return dom.window.document.body;
}

const audit = () => auditDom(dom);

afterEach(() => dom?.window.close());

describe('ErrorSummary fixture', () => {
	it('has no axe violations', async () => {
		await mount();
		expect(await audit()).toEqual([]);
	});
});

describe('ErrorSummary', () => {
	it('links each entry to its field and is announced as an alert', async () => {
		const root = await mount();
		const summary = root.querySelector('.govuk-error-summary')!;
		expect(summary.querySelector('[role="alert"] h2')?.textContent).toBe('There is a problem');
		const link = summary.querySelector<HTMLAnchorElement>('a')!;
		expect(link.getAttribute('href')).toBe('#email');
		expect(root.querySelector('#email')).not.toBeNull();
		// Focusable by script, not by tab: it takes focus when it appears.
		expect(summary.getAttribute('tabindex')).toBe('-1');
	});

	it('renders a message as text, and an entry with no field as plain text', async () => {
		const { body } = render(ErrorSummary, {
			props: { errors: [{ message: '<img src=x onerror=alert(1)>' }] }
		});
		expect(body).not.toContain('<img');
		expect(body).not.toContain('<a');
	});
});
