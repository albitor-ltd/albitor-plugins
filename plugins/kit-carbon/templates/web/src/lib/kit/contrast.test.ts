// Colour contrast of the Carbon themes the kit wears, computed from the installed
// carbon-components-svelte stylesheet (the very file the app loads), not from a copy. axe cannot
// judge contrast without a layout engine, so this is the check that stands in for it: every
// foreground and background pair the kit's components use must meet WCAG 2.2 AA in the light theme
// (`white`) and the dark theme (`g100`). Carbon is not themeable by a visual-language plugin, so
// the shipped values are the effective values. Which theme the page wears is tested through the
// app shell (shell.test.ts), not by importing the app-owned theme.ts.
import { readFileSync } from 'node:fs';
import { describe, expect, it } from 'vitest';

// Colour literals the ratio helper is exercised with.
const BLACK = '#000000';
const WHITE = '#ffffff';
const MID_GREY = '#767676';
const SHORT_BLACK = '#000';
const SHORT_WHITE = '#fff';

const css = readFileSync(
	new URL('../../../node_modules/carbon-components-svelte/css/all.css', import.meta.url),
	'utf8'
);

/**
 * The --cds-* custom properties of the first block that opens with `selector{` and sets a theme
 * (carbon-components-svelte 0.113 writes the dark themes as `:root[theme=g100]{...color-scheme:dark;
 * --cds-...}`, so the block need not start with a --cds- property).
 */
function theme(selector: string): Record<string, string> {
	let open = -1;
	let close = -1;
	for (let at = css.indexOf(`${selector}{`); at >= 0; at = css.indexOf(`${selector}{`, at + 1)) {
		const o = css.indexOf('{', at);
		const c = css.indexOf('}', o);
		if (css.slice(o, c).includes('--cds-layer:')) {
			open = o;
			close = c;
			break;
		}
	}
	if (open < 0) throw new Error(`all.css has no ${selector} theme block`);
	const out: Record<string, string> = {};
	for (const m of css.slice(open + 1, close).matchAll(/(--cds-[a-z0-9-]+):([^;]+)/g)) {
		out[m[1]] = m[2].trim();
	}
	return out;
}

const light = theme(':root');
// A token the g100 block leaves out is inherited from `:root` (the button and on-colour tokens are
// the same in every theme), so the dark set is the light one overlaid with the g100 block.
const dark = { ...light, ...theme(':root[theme=g100]') };

// Carbon's Gray 100 theme paints a low-contrast toast with the layer colour, not a notification
// token (only the light theme defines those). Read the rule from the g100 stylesheet and check it
// says that, then audit the toast text against the layer.
const g100 = readFileSync(
	new URL('../../../node_modules/carbon-components-svelte/css/g100.css', import.meta.url),
	'utf8'
);
for (const tone of ['info', 'success', 'error']) {
	const m = new RegExp(
		`\\.bx--toast-notification--low-contrast\\.bx--toast-notification--${tone}(?:,[^{]*)?\\{background:(#[0-9a-f]{3,6});`
	).exec(g100);
	if (!m) throw new Error(`g100.css has no low-contrast ${tone} toast rule`);
	if (full(m[1]) !== full(dark['--cds-layer'])) {
		throw new Error(`g100 ${tone} toast is ${m[1]}, no longer the layer colour`);
	}
	dark[`--cds-notification-background-${tone}`] = dark['--cds-layer'];
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
	['--cds-link-primary', '--cds-layer', 4.5],
	['--cds-text-error', '--cds-layer', 4.5],
	// The toast text on each tone's background (the Toaster uses Carbon's low-contrast toasts).
	['--cds-text-primary', '--cds-notification-background-info', 4.5],
	['--cds-text-primary', '--cds-notification-background-success', 4.5],
	['--cds-text-primary', '--cds-notification-background-error', 4.5],
	['--cds-text-on-color', '--cds-button-primary', 4.5],
	['--cds-text-on-color', '--cds-button-secondary', 4.5],
	['--cds-text-on-color', '--cds-button-danger-primary', 4.5],
	// A control's edge, the focus outline and the primary fill against the page.
	['--cds-border-strong', '--cds-background', 3],
	['--cds-focus', '--cds-background', 3],
	['--cds-button-primary', '--cds-background', 3],
	['--cds-button-danger-primary', '--cds-background', 3]
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

describe('the ratio helper', () => {
	it('is 21:1 for black on white and 1:1 for a colour on itself', () => {
		expect(ratio(BLACK, WHITE)).toBeCloseTo(21, 5);
		expect(ratio(MID_GREY, MID_GREY)).toBe(1);
	});
	it('reads #rgb as #rrggbb', () => {
		expect(ratio(SHORT_BLACK, SHORT_WHITE)).toBeCloseTo(21, 5);
	});
	it('rejects a colour it cannot read', () => {
		expect(() => luminance('red')).toThrow();
	});
});
