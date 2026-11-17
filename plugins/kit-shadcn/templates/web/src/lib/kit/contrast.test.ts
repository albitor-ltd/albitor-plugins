// Colour contrast of the kit's tokens, computed from the shipped tokens.css. axe cannot judge
// contrast without a layout engine, so this is the check that stands in for it: every foreground
// and background pair the components use must meet WCAG 2.2 AA in the light and the dark scheme.
import { readFileSync } from 'node:fs';
import { describe, expect, it } from 'vitest';

// Colour literals the ratio helper is exercised with.
const BLACK = '#000000';
const WHITE = '#ffffff';
const MID_GREY = '#767676';

const css = readFileSync(new URL('./tokens.css', import.meta.url), 'utf8');

/** The custom properties declared by the first block that follows `marker`. */
function block(marker: string): Record<string, string> {
	const start = css.indexOf(marker);
	if (start < 0) throw new Error(`tokens.css has no ${marker} block`);
	const open = css.indexOf('{', start);
	const close = css.indexOf('}', open);
	const out: Record<string, string> = {};
	for (const m of css.slice(open + 1, close).matchAll(/(--[a-z0-9-]+)\s*:\s*([^;]+);/g)) {
		out[m[1]] = m[2].trim();
	}
	return out;
}

/** Resolve `var(--name, fallback)` against `scope` (the kit's own tokens) with no look present, so
 * the audit covers the kit's defaults: a `--look-*` property that is absent takes its fallback. */
function resolve(value: string, scope: Record<string, string>): string {
	const m = /^var\((--[a-z0-9-]+)\s*(?:,\s*(.+))?\)$/s.exec(value.trim());
	if (!m) return value.trim();
	const [, name, fallback] = m;
	if (name.startsWith('--look-')) return fallback === undefined ? '' : resolve(fallback, scope);
	if (scope[name] !== undefined) return resolve(scope[name], scope);
	return fallback === undefined ? '' : resolve(fallback, scope);
}

const lightRaw = block(':root {');
const darkRaw = block(':root:not([data-theme="light"]) {');
const darkForcedRaw = block(':root[data-theme="dark"] {');

/** A scheme's tokens: the light block, with the dark block laid over it, every var() resolved. */
function scheme(...layers: Array<Record<string, string>>): Record<string, string> {
	const raw = Object.assign({}, ...layers) as Record<string, string>;
	return Object.fromEntries(Object.keys(raw).map((k) => [k, resolve(raw[k], raw)]));
}

const light = scheme(lightRaw);
const dark = scheme(lightRaw, darkRaw);
const darkForced = scheme(lightRaw, darkForcedRaw);

function channel(v: number): number {
	const s = v / 255;
	return s <= 0.03928 ? s / 12.92 : ((s + 0.055) / 1.055) ** 2.4;
}

function luminance(hex: string): number {
	const m = /^#([0-9a-f]{6})$/i.exec(hex);
	if (!m) throw new Error(`not a #rrggbb colour: ${hex}`);
	const n = parseInt(m[1], 16);
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
	['--foreground', '--background', 4.5],
	['--foreground', '--muted', 4.5],
	['--muted-foreground', '--background', 4.5],
	['--muted-foreground', '--muted', 4.5],
	['--primary-foreground', '--primary', 4.5],
	['--destructive-foreground', '--destructive', 4.5],
	['--destructive', '--background', 4.5],
	['--link', '--background', 4.5],
	['--input', '--background', 3],
	// The focus outline is drawn 2px off the control (outline-offset), so it sits on the page
	// background, not on the control's own fill.
	['--ring', '--background', 3],
	['--primary', '--background', 3],
	// Cards, the dialog, toasts and field backgrounds sit on the surface.
	['--foreground', '--surface', 4.5],
	['--muted-foreground', '--surface', 4.5],
	['--input', '--surface', 3],
	// The header band and its nav hover.
	['--header-ink', '--header', 4.5],
	['--header-ink', '--muted', 4.5],
	// The second colour: fill with its ink, the chip with its text, and the text on the page.
	['--on-second', '--second', 4.5],
	['--second-text', '--second-subtle', 4.5],
	['--second-text', '--background', 4.5],
	['--second', '--background', 3],
	// Success and warning: ink on the fill, and the fill against the page.
	['--on-success', '--success', 4.5],
	['--on-warning', '--warning', 4.5],
	['--success', '--background', 3],
	['--warning', '--background', 3],
	// Progress and charts: the fill is a graphic, so 3:1 against its track and its surface; the
	// sparkline's end dot is the second colour on the surface.
	['--primary', '--muted', 3],
	['--primary', '--surface', 3],
	['--second', '--surface', 3],
	// Stat tile: the delta arrows on the surface.
	['--success', '--surface', 3],
	['--destructive', '--surface', 4.5]
];

describe.each([
	['light', light],
	['dark by preference', dark],
	['dark by attribute', darkForced]
])('%s scheme', (_name, tokens) => {
	it('declares every token the pairs use as #rrggbb', () => {
		for (const [fg, bg] of pairs) {
			expect(tokens[fg], fg).toMatch(/^#[0-9a-f]{6}$/i);
			expect(tokens[bg], bg).toMatch(/^#[0-9a-f]{6}$/i);
		}
	});

	it.each(pairs)('%s on %s meets %s:1', (fg, bg, min) => {
		const r = ratio(tokens[fg], tokens[bg]);
		expect(r, `${tokens[fg]} on ${tokens[bg]} is ${r.toFixed(2)}:1`).toBeGreaterThanOrEqual(min);
	});
});

it('the two dark blocks agree', () => {
	expect(darkForcedRaw).toEqual(darkRaw);
	expect(darkForced).toEqual(dark);
});

describe('the ratio helper', () => {
	it('is 21:1 for black on white and 1:1 for a colour on itself', () => {
		expect(ratio(BLACK, WHITE)).toBeCloseTo(21, 5);
		expect(ratio(MID_GREY, MID_GREY)).toBe(1);
	});
	it('rejects a colour that is not #rrggbb', () => {
		expect(() => luminance('red')).toThrow();
	});
});
