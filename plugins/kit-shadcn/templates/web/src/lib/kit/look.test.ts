// The kit reads the platform's `--look-*` properties (look.css) and keeps its own value as the
// fallback. These tests read the shipped tokens.css as text and resolve it the way the browser
// does, once with no look and once with a look supplying properties, in both schemes.
import { existsSync, readFileSync } from 'node:fs';
import { describe, expect, it } from 'vitest';

const css = readFileSync(new URL('./tokens.css', import.meta.url), 'utf8');

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

const light = block(':root {');
const dark = { ...light, ...block(':root:not([data-theme="light"]) {') };

/** Resolve a value the way the cascade does: `look` holds the `--look-*` properties that exist. */
function resolve(
	value: string,
	scope: Record<string, string>,
	look: Record<string, string>
): string {
	const m = /^var\((--[a-z0-9-]+)\s*(?:,\s*(.+))?\)$/s.exec(value.trim());
	if (!m) return value.trim();
	const [, name, fallback] = m;
	if (look[name] !== undefined) return look[name];
	if (!name.startsWith('--look-') && scope[name] !== undefined) {
		return resolve(scope[name], scope, look);
	}
	return fallback === undefined ? '' : resolve(fallback, scope, look);
}

const token = (scope: Record<string, string>, name: string, look: Record<string, string> = {}) =>
	resolve(`var(${name})`, scope, look);

describe('with no look', () => {
	it('keeps the kit values', () => {
		expect(token(light, '--background')).toBe('#ffffff');
		expect(token(light, '--primary')).toBe('#18181b');
		expect(token(light, '--radius')).toBe('0.5rem');
		expect(token(light, '--border-width')).toBe('1px');
		expect(token(light, '--focus-width')).toBe('2px');
		expect(token(light, '--shadow')).toBe('none');
		expect(token(light, '--space')).toBe('1');
		expect(token(dark, '--background')).toBe('#09090b');
		expect(token(dark, '--primary')).toBe('#fafafa');
	});

	it('falls the second colour back to the primary and the surface to the page', () => {
		expect(token(light, '--second')).toBe(token(light, '--primary'));
		expect(token(light, '--on-second')).toBe(token(light, '--primary-foreground'));
		expect(token(light, '--surface')).toBe(token(light, '--background'));
		expect(token(dark, '--second')).toBe(token(dark, '--primary'));
	});
});

describe('with a look', () => {
	const look = {
		'--look-page': '#fafaf5',
		'--look-surface': '#ffffff',
		'--look-ink': '#1a1a1a',
		'--look-accent': '#7c3aed',
		'--look-on-accent': '#ffffff',
		'--look-second': '#0e7490',
		'--look-success': '#16a34a',
		'--look-focus': '#1e40af',
		'--look-radius': '1.5rem',
		'--look-border-width': '2px',
		'--look-focus-width': '3px',
		'--look-shadow': '0 1px 2px rgb(0 0 0 / 0.1)',
		'--look-space': '1.25',
		'--look-min-target': '48px',
		'--look-weight-bold': '800',
		'--look-tracking': '-0.02em',
		'--look-font-heading': 'Georgia, serif',
		'--look-motion-duration': '150ms'
	};

	it('maps each property onto its kit token, in both schemes', () => {
		for (const scope of [light, dark]) {
			expect(token(scope, '--background', look)).toBe('#fafaf5');
			expect(token(scope, '--surface', look)).toBe('#ffffff');
			expect(token(scope, '--foreground', look)).toBe('#1a1a1a');
			expect(token(scope, '--primary', look)).toBe('#7c3aed');
			expect(token(scope, '--primary-foreground', look)).toBe('#ffffff');
			expect(token(scope, '--second', look)).toBe('#0e7490');
			expect(token(scope, '--success', look)).toBe('#16a34a');
			expect(token(scope, '--ring', look)).toBe('#1e40af');
			expect(token(scope, '--radius', look)).toBe('1.5rem');
			expect(token(scope, '--border-width', look)).toBe('2px');
			expect(token(scope, '--focus-width', look)).toBe('3px');
			expect(token(scope, '--shadow', look)).toBe('0 1px 2px rgb(0 0 0 / 0.1)');
			expect(token(scope, '--shadow-overlay', look)).toBe(token(scope, '--shadow', look));
			expect(token(scope, '--space', look)).toBe('1.25');
			expect(token(scope, '--control-height', look)).toBe('48px');
			expect(token(scope, '--heading-weight', look)).toBe('800');
			expect(token(scope, '--heading-tracking', look)).toBe('-0.02em');
			expect(token(scope, '--heading-font', look)).toBe('Georgia, serif');
			expect(token(scope, '--motion-duration', look)).toBe('150ms');
		}
	});

	it('leaves a token on its default when its property is absent', () => {
		expect(token(light, '--destructive', look)).toBe('#b91c1c');
		expect(token(dark, '--destructive', look)).toBe('#f87171');
		expect(token(light, '--motion-ease', look)).toBe('ease');
	});

	it('uses different values in dark when the look supplies them', () => {
		const darkLook = { ...look, '--look-page': '#10101a', '--look-accent': '#c4b5fd' };
		expect(token(light, '--background', darkLook)).toBe('#10101a');
		expect(token(dark, '--primary', darkLook)).toBe('#c4b5fd');
		// What the look leaves out keeps the scheme's own default.
		expect(token(light, '--warning', darkLook)).toBe('#b45309');
		expect(token(dark, '--warning', darkLook)).toBe('#fbbf24');
	});

	it('uses the old link colour for the success toast edge until the look sets success', () => {
		expect(token(light, '--success-edge')).toBe('#1d4ed8');
		expect(token(dark, '--success-edge')).toBe('#60a5fa');
		expect(token(light, '--success-edge', look)).toBe('#16a34a');
	});

	it('takes the link from the look link, then the accent text, then the kit blue', () => {
		expect(token(light, '--link')).toBe('#1d4ed8');
		expect(token(light, '--link', { '--look-accent-text': '#5b21b6' })).toBe('#5b21b6');
		expect(
			token(light, '--link', { '--look-accent-text': '#5b21b6', '--look-link': '#0000ee' })
		).toBe('#0000ee');
	});

	it('derives the spacing steps and the row height from the density multiplier', () => {
		expect(light['--space-4']).toBe('calc(1rem * var(--space))');
		expect(light['--row-height']).toBe('calc(2.75rem * var(--look-space))');
	});
});

describe('type scale', () => {
	const steps = ['micro', 'small', 'body', 'lead', 'heading', 'title'];

	it('has six steps, each with a line-height, in the kit and in the look', () => {
		for (const step of steps) {
			expect(light[`--text-${step}`]).toMatch(new RegExp(`^var\\(--look-text-${step},`));
			expect(light[`--leading-${step}`]).toMatch(new RegExp(`^var\\(--look-leading-${step},`));
		}
	});

	it('falls back to a rising scale with no look', () => {
		const sizes = steps.map((s) => parseFloat(token(light, `--text-${s}`)));
		expect(sizes).toEqual([...sizes].sort((a, b) => a - b));
		expect(new Set(sizes).size).toBe(6);
	});

	it('maps the heading levels onto the steps and takes the look values when set', () => {
		expect(token(light, '--h1-size')).toBe(token(light, '--text-title'));
		expect(token(light, '--h2-size')).toBe(token(light, '--text-heading'));
		expect(token(light, '--h3-size')).toBe(token(light, '--text-lead'));
		expect(token(light, '--h4-size')).toBe(token(light, '--text-body'));
		expect(token(light, '--h6-size')).toBe(token(light, '--text-small'));
		expect(token(light, '--h1-leading')).toBe(token(light, '--leading-title'));
		expect(token(light, '--h1-size', { '--look-h1-size': '1.5rem' })).toBe('1.5rem');
		expect(token(light, '--text-lead', { '--look-text-lead': '1rem' })).toBe('1rem');
	});

	it('reads the look data and label faces, falling back to the body face', () => {
		expect(token(light, '--kit-font-data')).toBe(token(light, '--font-sans'));
		expect(token(light, '--kit-font-label')).toBe(token(light, '--font-sans'));
		expect(token(light, '--kit-font-data', { '--look-font-data': 'Mono' })).toBe('Mono');
		expect(token(light, '--kit-font-label', { '--look-font-label': 'Caps' })).toBe('Caps');
	});

	it('has no shadow on the dark page unless a look sets one (the platform writes none)', () => {
		expect(token(dark, '--shadow')).toBe('none');
		expect(token(dark, '--shadow', { '--look-shadow': 'none' })).toBe('none');
	});

	it('styles every heading level from its token', () => {
		for (let n = 1; n <= 6; n++) {
			expect(css).toMatch(new RegExp(`h${n} \\{\\s*font-size: var\\(--h${n}-size\\);\\s*line-height: var\\(--h${n}-leading\\);`));
		}
	});

	it('keeps a 44px hit area under a smaller target', () => {
		expect(light['--hit-inset']).toBe('min(0px, calc((var(--control-height) - 2.75rem) / 2))');
	});
});

it('imports the look file the platform writes, and the file exists so the build resolves it', () => {
	expect(css).toMatch(/@import '\.\.\/look\/look\.css';/);
	expect(existsSync(new URL('../look/look.css', import.meta.url))).toBe(true);
});
