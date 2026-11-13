// The kit's shell, rendered on the server the way a build prerenders it, loaded into a DOM and
// audited with axe. The test runs in Node and builds its own jsdom, because Vitest's
// jsdom environment would compile the components for the browser and this renders them for the
// server. jsdom has no layout, so colour contrast is audited from the tokens in contrast.test.ts
// and switched off here. The other pieces are tested beside this file, each against its own
// fixture in __fixtures__/ (field.test.ts, table.test.ts, emptystate.test.ts, dialog.test.ts), so
// a piece an app never uses can go with its fixture and its test.
import type { JSDOM } from 'jsdom';
import { afterEach, beforeEach, describe, expect, it } from 'vitest';
import { audit, serverDom } from './__fixtures__/dom';
import ShellFixture from './__fixtures__/ShellFixture.svelte';
import { toast } from './toast.svelte';

let dom: JSDOM;

function mount(props: Record<string, unknown> = {}): HTMLElement {
	dom = serverDom(ShellFixture, props);
	return dom.window.document.body;
}

beforeEach(() => toast.clear());
afterEach(() => dom?.window.close());

describe('the shell', () => {
	it('has no axe violations', async () => {
		toast.push({ message: 'Saved', tone: 'success' });
		mount();
		expect(await audit(dom)).toEqual([]);
	});
});

describe('AppShell', () => {
	it('puts a skip link first and points it at the main landmark', () => {
		const root = mount();
		const first = root.querySelector('a, button');
		expect(first?.textContent).toMatch(/skip to main content/i);
		const target = (first as HTMLAnchorElement).getAttribute('href')!.slice(1);
		expect(root.querySelector(`main#${target}`)).not.toBeNull();
	});

	it('marks only the current page in the nav and labels the landmark', () => {
		const root = mount({ path: '/reports' });
		const nav = root.querySelector('nav')!;
		expect(nav.getAttribute('aria-label')).toBe('Main');
		const current = [...nav.querySelectorAll('[aria-current]')];
		expect(current).toHaveLength(1);
		expect(current[0].getAttribute('aria-current')).toBe('page');
		expect(current[0].textContent).toContain('Reports');
	});

	it('treats a nested path as inside its section, not as the page itself', () => {
		const root = mount({ path: '/reports/2026' });
		const current = [...root.querySelectorAll('nav [aria-current]')];
		expect(current).toHaveLength(1);
		expect(current[0].getAttribute('aria-current')).toBe('true');
	});

	it('does not treat a sibling path that shares a prefix as current', () => {
		const root = mount({ path: '/reports-archive' });
		expect(root.querySelectorAll('nav [aria-current]')).toHaveLength(0);
	});

	it('renders a nav label as text, never as markup', () => {
		const root = mount({ path: '/', extraNav: '<img src=x onerror=alert(1)>' });
		expect(root.querySelector('nav img')).toBeNull();
		expect(root.querySelector('nav')!.textContent).toContain('<img src=x onerror=alert(1)>');
	});
});

describe('Toaster', () => {
	it('announces a message politely and an error assertively', () => {
		toast.push({ message: 'Saved', tone: 'success' });
		toast.push({ message: 'Could not save', tone: 'error' });
		const root = mount();
		expect(root.querySelector('[role="status"]')?.textContent).toContain('Saved');
		expect(root.querySelector('[role="alert"]')?.textContent).toContain('Could not save');
	});

	it('keeps the live regions in the page when nothing is queued', () => {
		const root = mount();
		expect(root.querySelector('[role="status"]')).not.toBeNull();
		expect(root.querySelector('[role="alert"]')).not.toBeNull();
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
