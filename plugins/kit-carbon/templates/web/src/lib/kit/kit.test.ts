// The kit's shell pieces, rendered on the server the way a build prerenders them, loaded into a
// DOM and audited with axe. The test runs in Node and builds its own jsdom, because Vitest's
// jsdom environment would compile the components for the browser and this renders them for the
// server. jsdom has no layout, so colour contrast is audited from the tokens in contrast.test.ts
// and switched off here.
import axe from 'axe-core';
import { JSDOM } from 'jsdom';
import { render } from 'svelte/server';
import { afterEach, beforeEach, describe, expect, it } from 'vitest';
import Showcase from './Showcase.svelte';
import { toast } from './toast.svelte';

let dom: JSDOM;

async function mount(props: Record<string, unknown> = {}): Promise<HTMLElement> {
	const { body } = render(Showcase, { props });
	dom = new JSDOM(`<!doctype html><html lang="en"><head><title>Kit</title></head><body>${body}</body></html>`, {
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

beforeEach(() => toast.clear());
afterEach(() => dom?.window.close());

describe('the shell showcase', () => {
	it('has no axe violations', async () => {
		toast.push({ message: 'Saved', tone: 'success' });
		await mount();
		expect(await audit()).toEqual([]);
	});
});

describe('AppShell', () => {
	it('puts a skip link first and points it at the main landmark', async () => {
		const root = await mount();
		const first = root.querySelector('a, button');
		expect(first?.textContent).toMatch(/skip to main content/i);
		const target = (first as HTMLAnchorElement).getAttribute('href')!.slice(1);
		expect(root.querySelector(`main#${target}`)).not.toBeNull();
	});

	it('marks only the current page in the nav and labels the landmark', async () => {
		const root = await mount({ path: '/reports' });
		const nav = root.querySelector('nav')!;
		expect(nav.getAttribute('aria-label')).toBe('Main');
		const current = [...nav.querySelectorAll('[aria-current]')];
		expect(current).toHaveLength(1);
		expect(current[0].getAttribute('aria-current')).toBe('page');
		expect(current[0].textContent).toContain('Reports');
	});

	it('treats a nested path as inside its section, not as the page itself', async () => {
		const root = await mount({ path: '/reports/2026' });
		const current = [...root.querySelectorAll('nav [aria-current]')];
		expect(current).toHaveLength(1);
		expect(current[0].getAttribute('aria-current')).toBe('true');
	});

	it('does not treat a sibling path that shares a prefix as current', async () => {
		const root = await mount({ path: '/reports-archive' });
		expect(root.querySelectorAll('nav [aria-current]')).toHaveLength(0);
	});

	it('renders a nav label as text, never as markup', async () => {
		const root = await mount({ path: '/', extraNav: '<img src=x onerror=alert(1)>' });
		expect(root.querySelector('nav img')).toBeNull();
		expect(root.querySelector('nav')!.textContent).toContain('<img src=x onerror=alert(1)>');
	});
});

describe('main landmark', () => {
	it('can take focus when the skip link jumps to it', async () => {
		const root = await mount();
		expect(root.querySelector('main')!.getAttribute('tabindex')).toBe('-1');
	});
});

describe('Button', () => {
	it('is type=button unless told otherwise, so it never submits a form by accident', async () => {
		const root = await mount();
		const byText = (t: string) =>
			[...root.querySelectorAll('form button')].find((b) => b.textContent?.includes(t))!;
		expect(byText('Send report').getAttribute('type')).toBe('submit');
		for (const t of ['Save draft', 'Reset', 'Delete report']) {
			expect(byText(t).getAttribute('type'), t).toBe('button');
		}
	});

	it('maps each variant onto its Carbon kind', async () => {
		const root = await mount();
		const cls = (t: string) =>
			[...root.querySelectorAll('form button')].find((b) => b.textContent?.includes(t))!.className;
		expect(cls('Send report')).toContain('bx--btn--primary');
		expect(cls('Save draft')).toContain('bx--btn--secondary');
		expect(cls('Reset')).toContain('bx--btn--ghost');
		expect(cls('Delete report')).toContain('bx--btn--danger');
	});
});

describe('Field', () => {
	it('ties the label, hint and error to the control', async () => {
		const root = await mount();
		const input = root.querySelector<HTMLInputElement>('input[type="email"]')!;
		const label = root.querySelector(`label[for="${input.id}"]`);
		expect(label?.textContent).toContain('Email address');
		expect(input.getAttribute('aria-invalid')).toBe('true');
		const described = input.getAttribute('aria-describedby')!.split(' ');
		expect(described).toHaveLength(2);
		for (const id of described) expect(root.querySelector(`#${id}`)).not.toBeNull();
		expect(root.querySelector(`#${described[1]}`)!.textContent).toContain('Enter an email address');
	});

	it('keeps a caller\'s aria-describedby and adds the hint and error ids to it', async () => {
		const root = await mount();
		const notes = root.querySelector('textarea')!;
		expect(notes.getAttribute('aria-describedby')).toBe('notes-help');
		const email = root.querySelector('input[type="email"]')!;
		// The field's own wiring comes first and is never replaced by a caller's attribute.
		expect(email.getAttribute('aria-describedby')!.split(' ')).toHaveLength(2);
	});

	it('gives two fields different ids', async () => {
		const root = await mount();
		const ids = [...root.querySelectorAll('input, textarea, select')].map((e) => e.id);
		expect(ids.every(Boolean)).toBe(true);
		expect(new Set(ids).size).toBe(ids.length);
	});
});

describe('Table', () => {
	it('has a caption and scoped headers', async () => {
		const root = await mount();
		const table = root.querySelector('table')!;
		expect(table.querySelector('caption')?.textContent).toContain('Recent reports');
		for (const th of table.querySelectorAll('thead th')) {
			expect(th.getAttribute('scope')).toBe('col');
		}
	});

	it('renders cell values as text, never as markup', async () => {
		const root = await mount({ rowName: '<b id="pwned">x</b>' });
		expect(root.querySelector('#pwned')).toBeNull();
		expect(root.querySelector('table')!.textContent).toContain('<b id="pwned">x</b>');
	});
});

describe('Toaster', () => {
	it('announces a message politely and an error assertively', async () => {
		toast.push({ message: 'Saved', tone: 'success' });
		toast.push({ message: 'Could not save', tone: 'error' });
		const root = await mount();
		expect(root.querySelector('.toaster [role="status"]')?.textContent).toContain('Saved');
		expect(root.querySelector('.toaster [role="alert"]')?.textContent).toContain('Could not save');
	});

	it('puts an error only in the assertive region and a message only in the polite one', async () => {
		toast.push({ message: 'Saved', tone: 'success' });
		toast.push({ message: 'Could not save', tone: 'error' });
		const root = await mount();
		const polite = root.querySelector('.toaster [role="status"]')!.textContent;
		const urgent = root.querySelector('.toaster [role="alert"]')!.textContent;
		expect(polite).toContain('Saved');
		expect(polite).not.toContain('Could not save');
		expect(urgent).toContain('Could not save');
		expect(urgent).not.toContain('Saved');
	});

	it('keeps the live regions in the page when nothing is queued', async () => {
		const root = await mount();
		expect(root.querySelector('.toaster [role="status"]')).not.toBeNull();
		expect(root.querySelector('.toaster [role="alert"]')).not.toBeNull();
	});

	it('caps the queue so a loop cannot flood the page', () => {
		for (let i = 0; i < 100; i++) toast.push({ message: `m${i}` });
		expect(toast.items.length).toBeLessThanOrEqual(5);
		expect(toast.items.at(-1)?.message).toBe('m99');
	});

	it('dismisses by id and ignores an unknown id', () => {
		const id = toast.push({ message: 'a' });
		toast.dismiss('nope');
		expect(toast.items).toHaveLength(1);
		toast.dismiss(id);
		expect(toast.items).toHaveLength(0);
	});

	it('rejects a message that is empty or absurdly long', () => {
		expect(() => toast.push({ message: '' })).toThrow();
		expect(() => toast.push({ message: 'x'.repeat(501) })).toThrow();
	});
});

describe('EmptyState', () => {
	it('is a heading with a reason, so the page is never blank', async () => {
		const root = await mount({ empty: true });
		const h = root.querySelector('[data-kit="empty-state"] h2, [data-kit="empty-state"] h3');
		expect(h?.textContent).toContain('No reports yet');
	});
});
