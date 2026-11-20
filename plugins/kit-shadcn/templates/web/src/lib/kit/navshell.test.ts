// @vitest-environment jsdom
// NavShell through AppShell (__fixtures__/NavShellFixture.svelte), mounted in a jsdom tree: how many
// destinations make a tab bar, 4 tabs and More, or a drawer, and how More behaves. jsdom has no
// layout, so which of these the phone shows is checked from the markup and its state; colour
// contrast is in contrast.test.ts. NavShell is part of AppShell, so this test and its fixture stay unless the app
// replaces the shell wholesale.
import axe from 'axe-core';
import { flushSync, mount, tick, unmount } from 'svelte';
import { afterEach, describe, expect, it } from 'vitest';
import NavShellFixture from './__fixtures__/NavShellFixture.svelte';
import { phoneMode, splitTabs } from './NavShell.svelte';

let app: ReturnType<typeof mount> | undefined;

afterEach(() => {
	if (app) unmount(app);
	app = undefined;
	document.body.innerHTML = '';
});

async function show(props: Record<string, unknown> = {}) {
	app = mount(NavShellFixture, { target: document.body, props });
	flushSync();
	await tick();
}

const nav = () => document.querySelector<HTMLElement>('nav[aria-label="Main"]')!;
const links = () => [...nav().querySelectorAll('a')].map((a) => a.textContent?.trim());
const menu = () =>
	[...document.querySelectorAll('button')].find((b) => b.textContent?.includes('Menu'));
const more = () => nav().querySelector<HTMLButtonElement>('.more-button');

async function audit() {
	const result = await axe.run(document.body, {
		runOnly: {
			type: 'tag',
			values: ['wcag2a', 'wcag2aa', 'wcag21a', 'wcag21aa', 'wcag22aa', 'best-practice']
		},
		rules: { 'color-contrast': { enabled: false }, region: { enabled: false } }
	});
	return result.violations.map((v) => `${v.id}: ${v.nodes.map((n) => n.html).join(' | ')}`);
}

describe('the phone stance', () => {
	it('is the drawer by default: a Menu button controlling the nav', async () => {
		await show({ count: 3 });
		expect(nav().dataset.phone).toBe('drawer');
		expect(menu()?.getAttribute('aria-controls')).toBe(nav().id);
		expect(menu()?.getAttribute('aria-expanded')).toBe('false');
		menu()!.click();
		await tick();
		expect(menu()?.getAttribute('aria-expanded')).toBe('true');
		expect(nav().classList.contains('open')).toBe(true);
	});

	it.each([2, 3, 4, 5])('makes a tab bar of %i destinations, with no Menu button and no More', async (count) => {
		await show({ count, phoneNav: 'tabs' });
		expect(nav().dataset.phone).toBe('tabs');
		expect(links()).toHaveLength(count);
		expect(menu()).toBeUndefined();
		expect(more()).toBeNull();
	});

	it.each([6, 9])('makes 4 tabs and More of %i destinations, every one still reachable', async (count) => {
		await show({ count, phoneNav: 'tabs' });
		expect(nav().querySelectorAll('.items > li > a')).toHaveLength(4);
		expect(more()).not.toBeNull();
		expect(links()).toHaveLength(count);
		expect(nav().querySelectorAll('#more-list a')).toHaveLength(count - 4);
	});

	it('falls back to the drawer for a single destination, since a bar of one is not navigation', async () => {
		await show({ count: 1, phoneNav: 'tabs' });
		expect(nav().dataset.phone).toBe('drawer');
	});

	it('never puts the same destinations in a tab bar and a menu', async () => {
		await show({ count: 7, phoneNav: 'tabs' });
		expect(document.querySelectorAll('nav')).toHaveLength(1);
		expect(menu()).toBeUndefined();
	});

	it('is decided by phoneMode and splitTabs alone', () => {
		const one = [{ id: null, items: [{ label: 'A', order: 0, href: '/' }] }];
		expect(phoneMode(one, 'tabs')).toBe('drawer');
		expect(phoneMode([], 'tabs')).toBe('drawer');
		const items = Array.from({ length: 6 }, (_, i) => ({ label: `P${i}`, order: i, href: `/${i}` }));
		expect(splitTabs(items.slice(0, 5))).toEqual({ tabs: items.slice(0, 5), more: [] });
		expect(splitTabs(items)).toEqual({ tabs: items.slice(0, 4), more: items.slice(4) });
	});
});

describe('the tab bar highlight', () => {
	it('marks a tab current for a nested page too, so the bar highlights it', async () => {
		await show({ count: 4, phoneNav: 'tabs', path: '/p2/deep' });
		const current = [...nav().querySelectorAll('a[aria-current]')];
		expect(current).toHaveLength(1);
		expect(current[0].getAttribute('aria-current')).toBe('true');
		expect(current[0].textContent?.trim()).toBe('Page 2');
	});

	it('says the current page is in More for a nested page under a More destination', async () => {
		await show({ count: 7, phoneNav: 'tabs', path: '/p6/deep' });
		expect(more()!.textContent).toContain('(current page is in More)');
	});
});

describe('More', () => {
	it('is closed until pressed, then opens the rest as a list, and closes on Escape with focus back', async () => {
		await show({ count: 7, phoneNav: 'tabs' });
		const button = more()!;
		expect(button.getAttribute('aria-expanded')).toBe('false');
		expect(nav().querySelector('#more-list')?.classList.contains('open')).toBe(false);
		button.click();
		await tick();
		expect(button.getAttribute('aria-expanded')).toBe('true');
		expect(nav().querySelector('#more-list')?.classList.contains('open')).toBe(true);
		const inside = nav().querySelector<HTMLElement>('#more-list a')!;
		inside.focus();
		inside.dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape', bubbles: true }));
		await tick();
		expect(button.getAttribute('aria-expanded')).toBe('false');
		expect(document.activeElement).toBe(button);
	});

	it('never carries aria-current itself; the page in More is current there, and More says so', async () => {
		await show({ count: 7, phoneNav: 'tabs', path: '/p6' });
		const button = more()!;
		expect(button.hasAttribute('aria-current')).toBe(false);
		expect(button.textContent).toContain('(current page is in More)');
		const current = [...nav().querySelectorAll('[aria-current]')];
		expect(current).toHaveLength(1);
		expect(current[0].textContent?.trim()).toBe('Page 6');
		expect(current[0].closest('#more-list')).not.toBeNull();
	});

	it('says nothing about the current page when it is a tab', async () => {
		await show({ count: 7, phoneNav: 'tabs', path: '/p2' });
		expect(more()!.textContent).not.toContain('current page');
	});
});

describe('layout and icons', () => {
	it('puts the nav in a sidebar for the side layout, still one Main navigation', async () => {
		await show({ count: 3, layout: 'side' });
		expect(document.querySelectorAll('nav[aria-label="Main"]')).toHaveLength(1);
		expect(nav().closest('aside')).not.toBeNull();
		expect(nav().classList.contains('side')).toBe(true);
		expect(nav().closest('header')).toBeNull();
	});

	it('keeps the nav in the header for the top layout', async () => {
		await show({ count: 3 });
		expect(nav().closest('header')).not.toBeNull();
	});

	it('draws an icon beside a label, hidden from assistive technology', async () => {
		await show({ count: 3, withIcons: true });
		const first = nav().querySelector('a')!;
		expect(first.querySelector('svg')?.getAttribute('aria-hidden')).toBe('true');
		expect(first.textContent?.trim()).toBe('Page 1');
		expect(nav().querySelectorAll('svg')).toHaveLength(1);
	});
});

describe('with no destinations', () => {
	it.each([
		['top', { count: 0, layout: 'top', phoneNav: 'tabs' }],
		['side', { count: 0, layout: 'side', phoneNav: 'tabs' }],
		['drawer', { count: 0 }]
	])('renders no nav landmark, no Menu button and no sidebar (%s)', async (_name, props) => {
		await show(props);
		expect(document.querySelector('nav')).toBeNull();
		expect(document.querySelector('aside')).toBeNull();
		expect(menu()).toBeUndefined();
		expect(document.querySelector('main#main')).not.toBeNull();
		expect(document.querySelector('header')).not.toBeNull();
		expect(await audit()).toEqual([]);
	});
});

describe('accessibility', () => {
	it.each([
		['drawer', { count: 3 }],
		['tab bar', { count: 4, phoneNav: 'tabs' }],
		['tabs and More', { count: 7, phoneNav: 'tabs', path: '/p6' }],
		['sidebar', { count: 3, layout: 'side', withIcons: true }]
	])('%s has no axe violations', async (_name, props) => {
		await show(props);
		expect(await audit()).toEqual([]);
	});
});
