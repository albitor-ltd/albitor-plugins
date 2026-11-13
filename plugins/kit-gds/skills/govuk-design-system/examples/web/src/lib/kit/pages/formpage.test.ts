// FormPage in each state (__fixtures__/FormPageFixture.svelte), rendered on the server the way a
// build prerenders it, loaded into a DOM and audited with axe. Delete this file and the fixture
// with FormPage.svelte if the app has no such page.
import type { JSDOM } from 'jsdom';
import { afterEach, describe, expect, it } from 'vitest';
import { audit, serverDom } from '../__fixtures__/dom';
import FormPageFixture from './__fixtures__/FormPageFixture.svelte';
import { expectOneH1AndNoSkippedLevel, expectSummaryLinksResolve } from './__fixtures__/helpers';

let dom: JSDOM;

function mount(props: Record<string, unknown> = {}): HTMLElement {
	dom = serverDom(FormPageFixture, props, 'Pages');
	return dom.window.document.body;
}

afterEach(() => dom?.window.close());

const cases: Array<[name: string, props: Record<string, unknown>]> = [
	['form', {}],
	['form, with errors', { errors: [{ field: 'name', message: 'Enter a name' }] }],
	['form, busy', { busy: true }]
];

describe('FormPage in each state', () => {
	it.each(cases)('%s has no axe violations', async (_name, props) => {
		mount(props);
		expect(await audit(dom)).toEqual([]);
	});

	it.each(cases)('%s has exactly one h1 and no skipped heading level', (_name, props) => {
		expectOneH1AndNoSkippedLevel(mount(props));
	});
});

describe('FormPage', () => {
	it('shows no summary while the form is valid', () => {
		const root = mount();
		expect(root.querySelector('.govuk-error-summary')).toBeNull();
		expect(root.querySelector('form')?.hasAttribute('novalidate')).toBe(true);
	});

	it('puts the problems in a focusable summary above the heading, beside the field error', () => {
		const root = mount({ errors: [{ field: 'name', message: 'Enter a name' }] });
		const summary = root.querySelector('.govuk-error-summary');
		expect(summary?.getAttribute('tabindex')).toBe('-1');
		expect(summary?.querySelector('[role="alert"]')).not.toBeNull();
		expect(summary?.querySelector('h2')?.textContent).toBe('There is a problem');
		const h1 = root.querySelector('h1')!;
		expect(summary!.compareDocumentPosition(h1) & 4).toBeTruthy();
		expect(root.querySelector('input[aria-invalid="true"]')).not.toBeNull();
	});

	it('links every summary entry to the control it names', () => {
		const root = mount({ errors: [{ field: 'name', message: 'Enter a name' }] });
		expectSummaryLinksResolve(root);
	});

	it('lists a problem with no one field as plain text', () => {
		const root = mount({ errors: [{ message: 'Try again later' }] });
		expect(root.querySelector('.govuk-error-summary li')?.textContent?.trim()).toBe('Try again later');
		expect(root.querySelector('.govuk-error-summary a')).toBeNull();
	});

	it('disables the button while saving so it cannot submit twice', () => {
		const root = mount({ busy: true });
		const button = root.querySelector('button[type="submit"]');
		expect(button?.hasAttribute('disabled')).toBe(true);
		expect(button?.textContent?.trim()).toBe('Saving…');
	});

	it('names the action and offers a way out', () => {
		const root = mount();
		expect(root.querySelector('button[type="submit"]')?.textContent?.trim()).toBe('Create report');
		expect(root.querySelector('main a.govuk-link[href="/reports"]')?.textContent).toBe('Cancel');
	});
});
