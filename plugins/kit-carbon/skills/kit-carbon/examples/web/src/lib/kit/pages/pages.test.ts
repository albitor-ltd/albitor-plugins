// The page frame (__fixtures__/PagesShowcase.svelte) and the sign-in page in it, rendered on the
// server the way a build prerenders them, loaded into a DOM and audited with axe. Each page
// template is tested beside this file against its own fixture (listpage.test.ts,
// detailpage.test.ts, formpage.test.ts, dashboardpage.test.ts, settingspage.test.ts), so a template
// an app never uses goes with its fixture and its test. Colour contrast is audited from the tokens
// in ../contrast.test.ts, as jsdom has no layout.
import type { JSDOM } from 'jsdom';
import { afterEach, describe, expect, it } from 'vitest';
import { audit, serverDom } from '../__fixtures__/dom';
import SignInPageFixture from './__fixtures__/SignInPageFixture.svelte';
import { expectOneH1AndNoSkippedLevel } from './__fixtures__/helpers';

let dom: JSDOM;

function mount(props: Record<string, unknown> = {}): HTMLElement {
	dom = serverDom(SignInPageFixture, props, 'Pages');
	return dom.window.document.body;
}

afterEach(() => dom?.window.close());

const cases: Array<[name: string, props: Record<string, unknown>]> = [
	['sign-in', {}],
	['sign-in, with error', { signInError: 'Email or password is incorrect' }]
];

describe('the page frame with the sign-in page', () => {
	it.each(cases)('%s has no axe violations', async (_name, props) => {
		mount(props);
		expect(await audit(dom)).toEqual([]);
	});

	it.each(cases)('%s has exactly one h1 and no skipped heading level', (_name, props) => {
		expectOneH1AndNoSkippedLevel(mount(props));
	});
});

describe('SignInPage', () => {
	it('keeps the autocomplete hints a password manager needs and announces the error', () => {
		const root = mount({ signInError: 'Email or password is incorrect' });
		expect(root.querySelector('input[type="password"]')?.getAttribute('autocomplete')).toBe('current-password');
		expect(root.querySelector('input[type="email"]')?.getAttribute('autocomplete')).toBe('username');
		const error = root.querySelector('[role="alert"]');
		expect(error?.textContent).toContain('Error: Email or password is incorrect');
	});
});
