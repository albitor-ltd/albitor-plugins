// Managed by Albitor (kit-carbon). The page templates, each in each state, rendered on the server
// the way a build prerenders them, loaded into a DOM and audited with axe, plus the behaviours a
// template promises: one <h1>, a status or alert for loading and failure, an error summary, a
// labelled form. Colour contrast is audited from the tokens in ../contrast.test.ts, as jsdom has
// no layout.
import axe from 'axe-core';
import { JSDOM } from 'jsdom';
import type { ComponentProps } from 'svelte';
import { render } from 'svelte/server';
import { afterEach, describe, expect, it } from 'vitest';
import PagesShowcase from './PagesShowcase.svelte';

let dom: JSDOM;

function mount(props: ComponentProps<typeof PagesShowcase>): HTMLElement {
	const { body } = render(PagesShowcase, { props });
	dom = new JSDOM(`<!doctype html><html lang="en"><head><title>Pages</title></head><body>${body}</body></html>`, {
		runScripts: 'outside-only'
	});
	return dom.window.document.body;
}

async function audit(): Promise<string[]> {
	const win = dom.window as unknown as { eval(code: string): unknown; axe: typeof axe };
	win.eval(axe.source);
	const result = await win.axe.run(dom.window.document.body, {
		runOnly: { type: 'tag', values: ['wcag2a', 'wcag2aa', 'wcag21a', 'wcag21aa', 'wcag22aa', 'best-practice'] },
		rules: { 'color-contrast': { enabled: false }, region: { enabled: false } }
	});
	return result.violations.map((v) => `${v.id}: ${v.nodes.map((n) => n.html).join(' | ')}`);
}

afterEach(() => dom?.window.close());

type Case = [name: string, props: ComponentProps<typeof PagesShowcase>];

const cases: Case[] = [
	['list', { page: 'list' }],
	['list, loading', { page: 'list', status: 'loading' }],
	['list, failed', { page: 'list', status: 'error' }],
	['list, empty', { page: 'list', rows: [] }],
	['detail', { page: 'detail' }],
	['detail, failed', { page: 'detail', status: 'error' }],
	['form', { page: 'form' }],
	['form, with errors', { page: 'form', errors: ['Enter a name'] }],
	['dashboard', { page: 'dashboard' }],
	['dashboard, loading', { page: 'dashboard', status: 'loading' }],
	['settings', { page: 'settings' }],
	['settings, with errors', { page: 'settings', errors: ['Enter a display name'] }],
	['sign-in', { page: 'sign-in' }],
	['sign-in, with error', { page: 'sign-in', signInError: 'Email or password is incorrect' }]
];

describe('every page template', () => {
	it.each(cases)('%s has no axe violations', async (_name, props) => {
		mount(props);
		expect(await audit()).toEqual([]);
	});

	it.each(cases)('%s has exactly one h1 and no skipped heading level', (_name, props) => {
		const root = mount(props);
		expect(root.querySelectorAll('h1')).toHaveLength(1);
		const levels = [...root.querySelectorAll('h1, h2, h3, h4')].map((h) => Number(h.tagName[1]));
		levels.forEach((level, i) => {
			if (i > 0) expect(level - levels[i - 1]).toBeLessThanOrEqual(1);
		});
	});
});

describe('states', () => {
	it('announces loading politely, where the content will appear', () => {
		const root = mount({ page: 'list', status: 'loading' });
		expect(root.querySelector('[role="status"]')?.textContent).toContain('Loading reports');
		expect(root.querySelector('table')).toBeNull();
	});

	it('shows a failure as an alert with a retry button, not only a toast', () => {
		const root = mount({ page: 'list', status: 'error' });
		const alert = root.querySelector('[role="alert"]');
		expect(alert?.textContent).toContain('Something went wrong');
		expect(alert?.querySelector('button')?.textContent?.trim()).toBe('Try again');
	});

	it('says what to do next when a list is empty', () => {
		const root = mount({ page: 'list', rows: [] });
		expect(root.querySelector('[data-kit="empty-state"]')?.textContent).toContain('Create the first report');
		expect(root.querySelector('[data-kit="empty-state"] button')?.textContent?.trim()).toBe('New report');
	});
});

describe('ListPage', () => {
	it('names the table and the filter group', () => {
		const root = mount({ page: 'list' });
		expect(root.querySelector('table caption')?.textContent).toBe('Reports');
		expect(root.querySelector('[role="group"][aria-label="Filters"] label')?.textContent).toBe('Search reports');
	});
});

describe('DetailPage', () => {
	it('pairs every term with its value and links back to the list', () => {
		const root = mount({ page: 'detail' });
		expect([...root.querySelectorAll('dt')].map((t) => t.textContent)).toEqual(['Owner', 'Status', 'Notes']);
		expect(root.querySelectorAll('dd')).toHaveLength(3);
		expect(root.querySelector('main a[href="/reports"]')?.textContent).toBe('All reports');
	});

	it('says "Not set" for an empty value rather than leaving a blank', () => {
		const root = mount({ page: 'detail' });
		expect(root.querySelectorAll('dd')[2]?.textContent).toBe('Not set');
	});
});

describe('FormPage', () => {
	it('shows no summary while the form is valid', () => {
		const root = mount({ page: 'form' });
		expect(root.querySelector('.summary')).toBeNull();
		expect(root.querySelector('form')?.hasAttribute('novalidate')).toBe(true);
	});

	it('puts the problems in a focusable alert above the form, beside the field error', () => {
		const root = mount({ page: 'form', errors: ['Enter a name'] });
		const summary = root.querySelector('.summary');
		expect(summary?.getAttribute('role')).toBe('alert');
		expect(summary?.getAttribute('tabindex')).toBe('-1');
		expect(summary?.querySelector('h2')?.textContent).toBe('There is a problem');
		expect(summary?.nextElementSibling?.tagName).toBe('FORM');
		expect(root.querySelector('input[aria-invalid="true"]')).not.toBeNull();
	});

	it('names the action and offers a way out', () => {
		const root = mount({ page: 'form' });
		expect(root.querySelector('button[type="submit"]')?.textContent?.trim()).toBe('Create report');
		expect(root.querySelector('main a[href="/reports"]')?.textContent).toBe('Cancel');
	});
});

describe('DashboardPage', () => {
	it('reads each figure with its label', () => {
		const root = mount({ page: 'dashboard' });
		const first = root.querySelector('.stat');
		expect(first?.querySelector('dt')?.textContent).toBe('Open items');
		expect(first?.querySelector('dd')?.textContent).toBe('12');
	});
});

describe('SettingsPage', () => {
	it('names each group by its heading and has one save', () => {
		const root = mount({ page: 'settings' });
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

describe('SignInPage', () => {
	it('keeps the autocomplete hints a password manager needs and announces the error', () => {
		const root = mount({ page: 'sign-in', signInError: 'Email or password is incorrect' });
		expect(root.querySelector('input[type="password"]')?.getAttribute('autocomplete')).toBe('current-password');
		expect(root.querySelector('input[type="email"]')?.getAttribute('autocomplete')).toBe('username');
		const error = root.querySelector('[role="alert"]');
		expect(error?.textContent).toContain('Error: Email or password is incorrect');
	});
});
