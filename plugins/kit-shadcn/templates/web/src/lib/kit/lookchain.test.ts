// The kit's tokens and the platform's look.css resolve together. look.css is written by the
// platform, so the font properties below are a copy of what it emits (`--look-font-*` read the
// font look's `--font-body`, `--font-data` and `--font-label`, falling back to the body face). The
// resolver follows var() chains across both files the way the cascade does, and a cycle makes the
// property invalid (empty), which is the failure this test guards: the kit must not name its own
// token `--font-data`, which a font plugin binds and look.css reads.
import { readFileSync } from 'node:fs';
import { describe, expect, it } from 'vitest';

const kit = readFileSync(new URL('./tokens.css', import.meta.url), 'utf8');

const LOOK_CSS = `
:root {
  --look-font-body: var(--font-body, "Segoe UI", system-ui, sans-serif);
  --look-font-heading: var(--font-heading, "Segoe UI", system-ui, sans-serif);
  --look-font-data: var(--font-data, var(--look-font-body));
  --look-font-label: var(--font-label, var(--look-font-body));
}
`;

function declarations(css: string, marker: string): Record<string, string> {
	const start = css.indexOf(marker);
	if (start < 0) throw new Error(`no ${marker} block`);
	const open = css.indexOf('{', start);
	const close = css.indexOf('}', open);
	const out: Record<string, string> = {};
	for (const m of css.slice(open + 1, close).matchAll(/(--[a-z0-9-]+)\s*:\s*([^;]+);/g)) {
		out[m[1]] = m[2].trim();
	}
	return out;
}

/** Split `a, b` at the first top-level comma. */
function splitFallback(inner: string): [string, string | undefined] {
	let depth = 0;
	for (let i = 0; i < inner.length; i++) {
		if (inner[i] === '(') depth++;
		else if (inner[i] === ')') depth--;
		else if (inner[i] === ',' && depth === 0) return [inner.slice(0, i).trim(), inner.slice(i + 1).trim()];
	}
	return [inner.trim(), undefined];
}

/** The computed value of `name`, or null when it is invalid (a cycle, or nothing at all). */
function compute(name: string, scope: Record<string, string>, visiting: string[] = []): string | null {
	if (visiting.includes(name)) return null;
	if (scope[name] === undefined) return null;
	return substitute(scope[name], scope, [...visiting, name]);
}

function substitute(value: string, scope: Record<string, string>, visiting: string[]): string | null {
	const start = value.indexOf('var(');
	if (start < 0) return value.trim();
	let depth = 0;
	let end = -1;
	for (let i = start + 3; i < value.length; i++) {
		if (value[i] === '(') depth++;
		else if (value[i] === ')' && --depth === 0) {
			end = i;
			break;
		}
	}
	const [ref, fallback] = splitFallback(value.slice(start + 4, end));
	let inner: string | null;
	if (visiting.includes(ref)) return null; // a cycle poisons the whole value
	if (scope[ref] !== undefined) inner = compute(ref, scope, visiting);
	else inner = fallback === undefined ? null : substitute(fallback, scope, visiting);
	if (inner === null) return null;
	const rest = substitute(value.slice(end + 1), scope, visiting);
	return rest === null ? null : value.slice(0, start) + inner + rest;
}

const kitRoot = declarations(kit, ':root {');
const look = declarations(LOOK_CSS, ':root {');
const body = '"Segoe UI", system-ui, sans-serif';

describe('kit tokens with the platform look.css', () => {
	it('reads the body face for data and labels when no font plugin binds --font-data', () => {
		const scope = { ...kitRoot, ...look };
		expect(compute('--kit-font-data', scope)).toBe(body);
		expect(compute('--kit-font-label', scope)).toBe(body);
	});

	it('reads the font look face when a font plugin binds --font-data and --font-label', () => {
		const scope = { ...kitRoot, ...look, '--font-data': '"IBM Plex Mono", monospace', '--font-label': 'Barlow' };
		expect(compute('--kit-font-data', scope)).toBe('"IBM Plex Mono", monospace');
		expect(compute('--kit-font-label', scope)).toBe('Barlow');
	});

	it('falls back to the kit body face when there is no look at all', () => {
		expect(compute('--kit-font-data', kitRoot)).toBe(compute('--font-sans', kitRoot));
		expect(compute('--kit-font-data', kitRoot)).not.toBeNull();
	});

	it('does not name a kit token --font-data or --font-label, which font plugins own', () => {
		expect(kitRoot['--font-data']).toBeUndefined();
		expect(kitRoot['--font-label']).toBeUndefined();
	});

	it('the resolver sees the cycle the old names made', () => {
		const scope = {
			...look,
			'--font-data': 'var(--look-font-data, var(--font-sans))',
			'--font-sans': 'serif'
		};
		expect(compute('--font-data', scope)).toBeNull();
	});
});
