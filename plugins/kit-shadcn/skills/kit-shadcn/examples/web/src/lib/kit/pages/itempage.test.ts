// ItemPage in each state (ItemPageFixture), rendered on the server the way a build prerenders it,
// loaded into a DOM and audited with axe. Delete this file and the fixture with ItemPage.svelte.
import type { JSDOM } from 'jsdom';
import { afterEach, describe, expect, it } from 'vitest';
import { audit, serverDom } from '../__fixtures__/dom';
import Fixture from './__fixtures__/ItemPageFixture.svelte';
import { expectOneH1AndNoSkippedLevel } from './__fixtures__/helpers';

let dom: JSDOM;

function mount(props: Record<string, unknown> = {}): HTMLElement {
	dom = serverDom(Fixture, props, 'Pages');
	return dom.window.document.body;
}

afterEach(() => dom?.window.close());

const cases: Array<[name: string, props: Record<string, unknown>]> = [
	['item', {}],
	['item, with errors', { errors: [{ field: 'email', message: 'Enter an email address' }] }],
	['item, sending', { busy: true }],
	['item, enquiry sent', { sent: true }],
	['item, loading', { status: 'loading' }],
	['item, failed', { status: 'error' }]
];

describe('ItemPage in each state', () => {
	it.each(cases)('%s has no axe violations', async (_name, props) => {
		mount(props);
		expect(await audit(dom)).toEqual([]);
	});

	it.each(cases)('%s has exactly one h1 and no skipped heading level', (_name, props) => {
		expectOneH1AndNoSkippedLevel(mount(props));
	});
});

describe('ItemPage', () => {
	it('lists the facts as terms and values, with a link back to the catalogue', () => {
		const root = mount();
		expect(root.querySelector('dt')?.textContent).toBe('When');
		expect(root.querySelector('dd')?.textContent).toBe('Tuesdays, 7pm');
		expect(root.querySelector('.back a')?.getAttribute('href')).toBe('/courses');
	});

	it('names the action on the submit button, not "Submit"', () => {
		expect(mount().querySelector('button[type="submit"]')?.textContent?.trim()).toBe('Book a place');
	});

	it('links each problem to its control', () => {
		const root = mount({ errors: [{ field: 'email', message: 'Enter an email address' }] });
		expect(root.querySelector('.summary a')?.getAttribute('href')).toBe('#email');
		expect(root.ownerDocument.getElementById('email')?.matches('input')).toBe(true);
		expect(root.querySelector('#email')?.getAttribute('aria-invalid')).toBe('true');
	});

	it('disables the button while sending', () => {
		expect(mount({ busy: true }).querySelector('button[type="submit"]')?.hasAttribute('disabled')).toBe(true);
	});

	it('replaces the form with a confirmation, announced politely, once sent', () => {
		const root = mount({ sent: true });
		expect(root.querySelector('form')).toBeNull();
		expect(root.querySelector('[role="status"]')?.textContent).toBe('We have emailed your booking.');
	});
});
