// Colour contrast of the Carbon themes the kit wears, computed from the installed
// carbon-components-svelte stylesheet (the very file the app loads), not from a copy. axe cannot
// judge contrast without a layout engine, so this is the check that stands in for it: every
// foreground and background pair the kit's components use must meet WCAG 2.2 AA in the light theme
// (`white`) and the dark theme (`g100`). Carbon is not themeable by a visual-language plugin, so
// the shipped values are the effective values.
import { readFileSync } from 'node:fs';
import { describe, expect, it } from 'vitest';
import { pickTheme } from './theme';

const css = readFileSync(
	new URL('../../../node_modules/carbon-components-svelte/css/all.css', import.meta.url),
	'utf8'
);

/** The --cds-* custom properties of the block that opens with `selector{`. */
function theme(selector: string): Record<string, string> {
	const start = css.indexOf(`${selector}{--cds-`);
	if (start < 0) throw new Error(`all.css has no ${selector} theme block`);
	const open = css.indexOf('{', start);
	const close = css.indexOf('}', open);
	const out: Record<string, string> = {};
	for (const m of css.slice(open + 1, close).matchAll(/(--cds-[a-z0-9-]+):([^;]+)/g)) {
		out[m[1]] = m[2].trim();
	}
	return out;
}

const light = theme(':root');
const dark = theme('[theme=g100]');

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

// [foreground, background, minimum ratio]: 4.5 for text, 3 for UI components and focus (1.4.11).
const pairs: Array<[string, string, number]> = [
	['--cds-text-primary', '--cds-background', 4.5],
	['--cds-text-primary', '--cds-layer', 4.5],
	['--cds-text-primary', '--cds-layer-selected', 4.5],
	['--cds-text-secondary', '--cds-background', 4.5],
	['--cds-text-secondary', '--cds-layer', 4.5],
	// Hint text under a field, on the page and on a panel.
	['--cds-text-helper', '--cds-background', 4.5],
	['--cds-text-helper', '--cds-layer', 4.5],
	['--cds-text-error', '--cds-background', 4.5],
	['--cds-link-primary', '--cds-background', 4.5],
	['--cds-text-on-color', '--cds-button-primary', 4.5],
	['--cds-text-on-color', '--cds-button-secondary', 4.5],
	['--cds-text-on-color', '--cds-button-danger-primary', 4.5],
	// A control's edge, the focus outline and the primary fill against the page.
	['--cds-border-strong', '--cds-background', 3],
	['--cds-focus', '--cds-background', 3],
	['--cds-button-primary', '--cds-background', 3]
];

describe.each([
	['light (white)', light],
	['dark (g100)', dark]
])('%s theme', (_name, tokens) => {
	it('declares every token the pairs use as a hex colour', () => {
		for (const [fg, bg] of pairs) {
			expect(() => full(tokens[fg]), fg).not.toThrow();
			expect(() => full(tokens[bg]), bg).not.toThrow();
		}
	});

	it.each(pairs)('%s on %s meets %s:1', (fg, bg, min) => {
		const r = ratio(tokens[fg], tokens[bg]);
		expect(r, `${tokens[fg]} on ${tokens[bg]} is ${r.toFixed(2)}:1`).toBeGreaterThanOrEqual(min);
	});
});

describe('the two themes', () => {
	it('are the light and dark ends: dark text on light, light text on dark', () => {
		expect(luminance(light['--cds-background'])).toBeGreaterThan(luminance(light['--cds-text-primary']));
		expect(luminance(dark['--cds-background'])).toBeLessThan(luminance(dark['--cds-text-primary']));
	});
});

describe('pickTheme', () => {
	it('follows the visitor preference', () => {
		expect(pickTheme(undefined, false)).toBe('white');
		expect(pickTheme(undefined, true)).toBe('g100');
		expect(pickTheme(null, true)).toBe('g100');
	});
	it('lets data-theme force a scheme either way', () => {
		expect(pickTheme('light', true)).toBe('white');
		expect(pickTheme('dark', false)).toBe('g100');
	});
	it('ignores a data-theme it does not know', () => {
		expect(pickTheme('sepia', true)).toBe('g100');
		expect(pickTheme('sepia', false)).toBe('white');
	});
});

describe('the ratio helper', () => {
	it('is 21:1 for black on white and 1:1 for a colour on itself', () => {
		expect(ratio('#000000', '#ffffff')).toBeCloseTo(21, 5);
		expect(ratio('#767676', '#767676')).toBe(1);
	});
	it('reads #rgb as #rrggbb', () => {
		expect(ratio('#000', '#fff')).toBeCloseTo(21, 5);
	});
	it('rejects a colour it cannot read', () => {
		expect(() => luminance('red')).toThrow();
	});
});
