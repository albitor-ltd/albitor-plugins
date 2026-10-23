// Colour contrast of the GOV.UK colours the kit wears, read from the installed govuk-frontend
// stylesheet (the very file the app is built from), not from a copy. axe cannot judge contrast
// without a layout engine, so this is the check that stands in for it: every foreground and
// background pair the kit's components use must meet WCAG 2.2 AA. GOV.UK is not themeable, so the
// shipped values are the effective values.
import { readFileSync } from 'node:fs';
import { fileURLToPath } from 'node:url';
import * as sass from 'sass';
import { describe, expect, it } from 'vitest';

const css = readFileSync(
	new URL('../../../node_modules/govuk-frontend/dist/govuk/govuk-frontend.min.css', import.meta.url),
	'utf8'
);

/** The functional colour `--govuk-<name>` the stylesheet declares at its top level. */
function colour(name: string): string {
	const m = new RegExp(`--govuk-${name}:(#[0-9a-f]{3,6})[;}]`, 'i').exec(css);
	if (!m) throw new Error(`govuk-frontend.min.css declares no --govuk-${name}`);
	return full(m[1]);
}

/** The `property` of the first `selector{...}` rule that sets it, as #rrggbb. */
function ruleColour(selector: string, property: string): string {
	const escaped = selector.replace(/[.*+?^${}()|[\]\\]/g, '\\$&');
	const re = new RegExp(`${escaped}\\{(?:[^}]*;)?${property}:(#[0-9a-f]{3,6})`, 'i');
	const m = re.exec(css);
	if (!m) throw new Error(`govuk-frontend.min.css has no ${selector} ${property}`);
	return full(m[1]);
}

/** #rgb or #rrggbb to #rrggbb. */
function full(hex: string): string {
	const m = /^#([0-9a-f]{3}|[0-9a-f]{6})$/i.exec(hex);
	if (!m) throw new Error(`not a #rgb or #rrggbb colour: ${hex}`);
	const h = m[1];
	return h.length === 3 ? `#${[...h].map((c) => c + c).join('')}` : `#${h}`;
}

function channel(v: number): number {
	const s = v / 255;
	return s <= 0.03928 ? s / 12.92 : ((s + 0.055) / 1.055) ** 2.4;
}

function luminance(hex: string): number {
	const n = parseInt(full(hex).slice(1), 16);
	return (
		0.2126 * channel((n >> 16) & 255) + 0.7152 * channel((n >> 8) & 255) + 0.0722 * channel(n & 255)
	);
}

function ratio(a: string, b: string): number {
	const [hi, lo] = [luminance(a), luminance(b)].sort((x, y) => y - x);
	return (hi + 0.05) / (lo + 0.05);
}

// The kit's own stylesheet, compiled, for what the app sets there: the brand colour (the one token
// an app off GOV.UK changes) and the header text colour chosen to sit on it.
const kitCss = sass.compile(fileURLToPath(new URL('./govuk.scss', import.meta.url)), {
	loadPaths: [fileURLToPath(new URL('../../../node_modules', import.meta.url))],
	quietDeps: true,
}).css;

function kitProperty(name: string): string {
	const m = new RegExp(`--${name}:\\s*(#[0-9a-f]{3,6})[;}]`, 'i').exec(kitCss);
	if (!m) throw new Error(`govuk.scss compiles to no --${name}`);
	return full(m[1]);
}

const brand = kitProperty('govuk-brand-colour');
const headerText = kitProperty('kit-header-text');

const text = colour('text-colour');
const bg = colour('body-background-colour');
const surface = colour('surface-background-colour');

// [name, foreground, background, minimum ratio]: 4.5 for text, 3 for UI components (1.4.11).
const pairs: Array<[string, string, string, number]> = [
	['text on the page', text, bg, 4.5],
	['secondary text on the page', colour('secondary-text-colour'), bg, 4.5],
	['link on the page', colour('link-colour'), bg, 4.5],
	['link, hovered, on the page', colour('link-hover-colour'), bg, 4.5],
	['link, visited, on the page', colour('link-visited-colour'), bg, 4.5],
	['error text on the page', colour('error-colour'), bg, 4.5],
	['text on a surface (service navigation)', colour('surface-text-colour'), surface, 4.5],
	['link on a surface (service navigation)', colour('link-colour'), surface, 4.5],
	['text on the focus colour', colour('focus-text-colour'), colour('focus-colour'), 4.5],
	['input border on the page', colour('input-border-colour'), bg, 3],
	['error border on the page', colour('error-colour'), bg, 3],
	['primary button text', ruleColour('.govuk-button', 'color'), ruleColour('.govuk-button', 'background-color'), 4.5],
	[
		'warning button text',
		ruleColour('.govuk-button', 'color'),
		ruleColour('.govuk-button--warning', 'background-color'),
		4.5
	],
	['secondary button text', text, ruleColour('.govuk-button--secondary', 'background-color'), 4.5],
	// The kit's text-only header, in AppShell.svelte: the header text on the brand colour. Whatever
	// brand colour an app sets in govuk.scss, govuk.scss picks the header text to meet this.
	['header text on the brand band', headerText, brand, 4.5],
	// The footer band in Footer.svelte (#f3f3f3, GOV.UK's footer background): text and links on it.
	['text on the footer band', text, '#f3f3f3', 4.5],
	['link on the footer band', colour('link-colour'), '#f3f3f3', 4.5]
];

describe('GOV.UK colours', () => {
	it.each(pairs)('%s meets its minimum ratio', (_name, fg, back, min) => {
		const r = ratio(fg, back);
		expect(r, `${fg} on ${back} is ${r.toFixed(2)}:1`).toBeGreaterThanOrEqual(min);
	});
});

describe('the ratio helper', () => {
	it('is 21:1 for black on white and 1:1 for a colour on itself', () => {
		expect(ratio('#000000', '#ffffff')).toBeCloseTo(21, 5);
		expect(ratio('#767676', '#767676')).toBe(1);
	});
	it('rejects a colour that is not #rgb or #rrggbb', () => {
		expect(() => luminance('red')).toThrow();
	});
});
