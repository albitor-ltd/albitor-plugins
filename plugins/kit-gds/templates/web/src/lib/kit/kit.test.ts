// The kit's shell, rendered on the server the way a build prerenders it, loaded into a DOM and
// audited with axe. The test runs in Node and builds its own jsdom, because Vitest's
// jsdom environment would compile the components for the browser and this renders them for the
// server. jsdom has no layout, so colour contrast is audited from the stylesheet in contrast.test.ts
// and switched off here. The other pieces are tested beside this file, each against its own
// fixture in __fixtures__/ (button.test.ts, field.test.ts, table.test.ts, errorsummary.test.ts,
// dialog.test.ts), so a piece an app never uses can go with its fixture and its test.
import { JSDOM } from 'jsdom';
import { afterEach, describe, expect, it } from 'vitest';
import { audit as auditDom, serverDom } from './__fixtures__/dom';
import { fileURLToPath } from 'node:url';
import * as sass from 'sass';
import ShellFixture from './__fixtures__/ShellFixture.svelte';

let dom: JSDOM;

async function mount(props: Record<string, unknown> = {}): Promise<HTMLElement> {
	dom = serverDom(ShellFixture, props);
	return dom.window.document.body;
}

const audit = () => auditDom(dom);

afterEach(() => dom?.window.close());

describe('the shell', () => {
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

describe('GOV.UK branding', () => {
	it('renders a text-only header: no Crown, no logotype, no image', async () => {
		const root = await mount();
		const header = root.querySelector('header')!;
		expect(header.textContent).toContain('Example app');
		expect(header.querySelector('svg, img, picture')).toBeNull();
		expect(root.innerHTML).not.toMatch(/crown|crest|logotype|GOV\.UK/i);
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
