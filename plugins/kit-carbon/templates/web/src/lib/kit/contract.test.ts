// The component-kit@2 standard shell contract: AppShell, Footer and SignInPage, exported from the
// kit's index with the same props in every kit, so an auth plugin composes against the contract and
// not against one kit. Rendered on the server the way a build prerenders, loaded into a DOM and
// audited with axe (colour contrast is audited from the tokens in contrast.test.ts, and jsdom has
// no layout, so it is off here). Managed: do not edit to make it pass.
import axe from 'axe-core';
import { JSDOM } from 'jsdom';
import { render } from 'svelte/server';
import { afterEach, describe, expect, it } from 'vitest';
import * as kit from './index';
import ContractProbe from './ContractProbe.svelte';

let dom: JSDOM;

function mount(props: Record<string, unknown> = {}): HTMLElement {
	const { body } = render(ContractProbe, { props });
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

afterEach(() => dom?.window.close());

describe('component-kit@2 exports', () => {
	it.each(['AppShell', 'Footer', 'SignInPage'])('the kit index exports %s', (name) => {
		expect(typeof (kit as Record<string, unknown>)[name]).toBe('function');
	});
});

describe('AppShell with a footer', () => {
	it('has no axe violations, with the footer and with a failed sign-in', async () => {
		mount();
		expect(await audit()).toEqual([]);
		dom.window.close();
		mount({ error: 'Email or password is incorrect' });
		expect(await audit()).toEqual([]);
	});

	it('puts the footer after main, outside it, as the one contentinfo landmark', () => {
		const root = mount();
		const footers = root.querySelectorAll('footer');
		expect(footers).toHaveLength(1);
		expect(root.querySelector('main footer')).toBeNull();
		const main = root.querySelector('main')!;
		expect(main.compareDocumentPosition(footers[0]) & dom.window.Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy();
		expect(root.querySelectorAll('main')).toHaveLength(1);
	});

	it('keeps the skip link first and pointed at main', () => {
		const root = mount();
		const first = root.querySelector('a, button') as HTMLAnchorElement;
		expect(first.textContent).toMatch(/skip to main content/i);
		const target = first.getAttribute('href')!.slice(1);
		expect(root.querySelector(`main#${target}`)).not.toBeNull();
	});

	it('renders the links in a navigation labelled Footer, and the text beside them', () => {
		const root = mount();
		const footer = root.querySelector('footer')!;
		const nav = footer.querySelector('nav')!;
		expect(nav.getAttribute('aria-label')).toBe('Footer');
		expect([...nav.querySelectorAll('a')].map((a) => [a.getAttribute('href'), a.textContent?.trim()])).toEqual([
			['/privacy', 'Privacy'],
			['/accessibility', 'Accessibility statement']
		]);
		expect(footer.textContent).toContain('Example Ltd');
	});

	it('renders a footer link label as text, never as markup', () => {
		const root = mount({ extraLink: '<img src=x onerror=alert(1)>' });
		expect(root.querySelector('footer img')).toBeNull();
		expect(root.querySelector('footer')!.textContent).toContain('<img src=x onerror=alert(1)>');
	});

	it('renders two links to the same destination, both of them', () => {
		const root = mount({ repeatLink: true });
		const labels = [...root.querySelectorAll('footer nav a')].map((a) => a.textContent?.trim());
		expect(labels).toEqual(['Privacy', 'Accessibility statement', 'Privacy again']);
	});

	it('renders no footer when the app passes none', () => {
		const root = mount({ withFooter: false });
		expect(root.querySelector('footer')).toBeNull();
	});
});

/** The alerts with something in them: a kit may keep an empty live region in the page. */
function filled(root: HTMLElement): Element[] {
	return [...root.querySelectorAll('[role=alert]')].filter((a) => a.textContent?.trim());
}

describe('SignInPage', () => {
	it('has one h1 and a labelled email and password with the right autocomplete', () => {
		const root = mount();
		expect(root.querySelectorAll('h1')).toHaveLength(1);
		const email = root.querySelector('input[type=email]') as HTMLInputElement;
		const password = root.querySelector('input[type=password]') as HTMLInputElement;
		expect(email.getAttribute('autocomplete')).toBe('username');
		expect(password.getAttribute('autocomplete')).toBe('current-password');
		for (const input of [email, password]) {
			expect(root.querySelector(`label[for="${input.id}"]`)).not.toBeNull();
		}
	});

	it('starts the email field from the email it is given, and empty without one', () => {
		const given = mount({ email: 'a@example.test' });
		expect((given.querySelector('input[type=email]') as HTMLInputElement).value).toBe('a@example.test');
		expect((mount().querySelector('input[type=email]') as HTMLInputElement).value).toBe('');
	});

	it('announces a failure once, as an alert, in the words it was given', () => {
		const root = mount({ error: 'Email or password is incorrect' });
		expect(filled(root)).toHaveLength(1);
		expect(filled(root)[0].textContent).toContain('Email or password is incorrect');
	});

	it('shows no alert before a failure', () => {
		expect(filled(mount())).toHaveLength(0);
	});

	it('offers the reset and sign-up links it is given, and disables the button while busy', () => {
		const root = mount({ busy: true });
		expect(root.querySelector('a[href="/reset"]')).not.toBeNull();
		expect(root.querySelector('a[href="/sign-up"]')).not.toBeNull();
		expect((root.querySelector('button[type=submit]') as HTMLButtonElement).disabled).toBe(true);
	});
});
