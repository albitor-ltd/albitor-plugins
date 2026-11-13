// SettingsPage in each state (__fixtures__/SettingsPageFixture.svelte), rendered on the server the
// way a build prerenders it, loaded into a DOM and audited with axe. Delete this file and the
// fixture with SettingsPage.svelte and SettingsSection.svelte if the app has no settings page.
import type { JSDOM } from 'jsdom';
import { afterEach, describe, expect, it } from 'vitest';
import { audit, serverDom } from '../__fixtures__/dom';
import { toast } from '../toast.svelte';
import SettingsPageFixture from './__fixtures__/SettingsPageFixture.svelte';
import { expectOneH1AndNoSkippedLevel, expectSummaryLinksResolve } from './__fixtures__/helpers';

let dom: JSDOM;

function mount(props: Record<string, unknown> = {}): HTMLElement {
	dom = serverDom(SettingsPageFixture, props, 'Pages');
	return dom.window.document.body;
}

afterEach(() => {
	dom?.window.close();
	toast.clear();
});

const cases: Array<[name: string, props: Record<string, unknown>, saved?: boolean]> = [
	['settings', {}],
	['settings, with errors', { errors: [{ field: 'display-name', message: 'Enter a display name' }] }],
	['settings, busy', { busy: true }],
	['settings, saved', {}, true]
];

describe('SettingsPage in each state', () => {
	it.each(cases)('%s has no axe violations', async (_name, props, saved) => {
		if (saved) toast.push({ message: 'Saved', tone: 'success' });
		mount(props);
		expect(await audit(dom)).toEqual([]);
	});

	it.each(cases)('%s has exactly one h1 and no skipped heading level', (_name, props, saved) => {
		if (saved) toast.push({ message: 'Saved', tone: 'success' });
		expectOneH1AndNoSkippedLevel(mount(props));
	});
});

describe('SettingsPage', () => {
	it('links every summary entry to the control it names', () => {
		const root = mount({ errors: [{ field: 'display-name', message: 'Enter a display name' }] });
		expectSummaryLinksResolve(root);
	});

	it('names each group by its heading and has one save', () => {
		const root = mount();
		const sections = [...root.querySelectorAll('section[aria-labelledby]')];
		expect(sections).toHaveLength(2);
		for (const s of sections) {
			const heading = s.querySelector(`#${s.getAttribute('aria-labelledby')}`);
			expect(heading?.tagName).toBe('H2');
		}
		expect(root.querySelectorAll('button[type="submit"]')).toHaveLength(1);
		expect(root.querySelector('fieldset legend')?.textContent).toBe('Email me about');
	});
});
