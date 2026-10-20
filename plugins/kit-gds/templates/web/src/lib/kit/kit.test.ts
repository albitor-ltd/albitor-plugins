// The kit's shell pieces, rendered on the server the way a build prerenders them, loaded into a
// DOM and audited with axe. The test runs in Node and builds its own jsdom, because Vitest's
// jsdom environment would compile the components for the browser and this renders them for the
// server. jsdom has no layout, so colour contrast is audited from the tokens in contrast.test.ts
// and switched off here.
import axe from 'axe-core';
import { fileURLToPath } from 'node:url';
import * as sass from 'sass';
import { JSDOM } from 'jsdom';
import { createRawSnippet } from 'svelte';
import { render } from 'svelte/server';
import { afterEach, describe, expect, it } from 'vitest';
import Button from './Button.svelte';
import ErrorSummary from './ErrorSummary.svelte';
import Showcase from './Showcase.svelte';

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

afterEach(() => dom?.window.close());

describe('the shell showcase', () => {
	it('has no axe violations', async () => {
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

describe('Table empty state', () => {
	it('says why there is nothing, with a heading, so the page is never blank', async () => {
		const root = await mount({ empty: true });
		expect(root.querySelector('table')).toBeNull();
		expect(root.querySelector('main')?.textContent).toContain('No reports yet');
		expect(
			[...root.querySelectorAll('h2')].some((h) => h.textContent?.includes('No reports yet'))
		).toBe(true);
		expect(await audit()).toEqual([]);
	});
});

describe('GOV.UK branding', () => {
	it('renders a text-only header: no Crown, no logotype, no image', async () => {
		const root = await mount();
		const header = root.querySelector('header')!;
		expect(header.textContent).toContain('Example app');
		expect(header.querySelector('svg, img, picture')).toBeNull();
		expect(root.innerHTML).not.toMatch(/crown|crest|logotype|GOV\.UK/i);
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

describe('Button', () => {
	const children = createRawSnippet(() => ({ render: () => '<span>Go</span>' }));

	it('merges a caller\'s class with govuk-button instead of replacing it', () => {
		const { body } = render(Button, { props: { class: 'my-extra', variant: 'destructive', children } });
		expect(body).toMatch(/class="govuk-button govuk-button--warning my-extra"/);
	});

	it('keeps govuk-button with no caller class', () => {
		const { body } = render(Button, { props: { children } });
		expect(body).toMatch(/class="govuk-button"/);
	});
});

describe('Table accessible name', () => {
	it('is named once, by its caption, not by a second region label', async () => {
		const root = await mount();
		expect(root.querySelector('[role="region"]')).toBeNull();
		expect(root.querySelector('.kit-scroll')?.hasAttribute('aria-label')).toBe(false);
	});
});

// The base layer in govuk.scss: bare elements, with no govuk-* class, read as GOV.UK. This is what
// the auth-cognito sign-in pages are. The stylesheet is compiled from the kit's own file and put in
// a page of bare elements; jsdom cascades the type and spacing. It does not resolve var() or match
// :link, so the link and focus rules are read from the compiled CSS.
describe('base typography', () => {
	const css = sass.compile(fileURLToPath(new URL('./govuk.scss', import.meta.url)), {
		loadPaths: [fileURLToPath(new URL('../../../node_modules', import.meta.url))],
		quietDeps: true,
		
	}).css;

	function bare(html: string): Document {
		const d = new JSDOM(
			`<!doctype html><html lang="en"><head><style>${css}</style></head><body>${html}</body></html>`
		);
		dom = d;
		return d.window.document;
	}

	it('styles a bare heading, paragraph and link as GOV.UK', () => {
		const doc = bare('<h1>Sign in</h1><p>Enter your details.</p><a href="/x">Forgotten?</a>');
		const style = (sel: string) => dom.window.getComputedStyle(doc.querySelector(sel)!);
		expect(style('h1').fontWeight).toBe('700');
		expect(style('h1').fontFamily).toMatch(/arial/i);
		expect(style('h1').marginBottom).toBe('20px');
		expect(style('p').marginBottom).toBe('15px');
		expect(style('body').fontFamily).toMatch(/arial/i);
		// jsdom does not resolve var(), so the colour is read from the rule itself.
		expect(css).toMatch(/:where\(a:link, a:visited\)\s*\{[^}]*color:\s*var\(--govuk-link-colour, #1a65a6\)/);
		expect(css).toMatch(/:where\(a:link, a:visited\)\s*\{[^}]*text-decoration:\s*underline/);
		expect(css).toMatch(/:where\(a:focus\)\s*\{[^}]*background-color:\s*var\(--govuk-focus-colour, #fd0\)/);
	});

	it('keeps the base layer below every class, so components are not overridden', () => {
		// :where() gives the layer no specificity; a .govuk-* class always wins.
		const bareRules = css.match(/:where\([^)]*\)\s*\{/g) ?? [];
		expect(bareRules.length).toBeGreaterThanOrEqual(6);
		expect(css).not.toMatch(/(^|\})\s*(a|p|h1|h2|h3|body)(:[a-z]+)?\s*[,{]/);
	});
});
