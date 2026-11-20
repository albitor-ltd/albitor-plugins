// Component styles the server render does not carry, read from the source: each Button variant
// owns its elevation (none on a ghost), figures and tables read the data and label faces, a
// stat value sits below the h1 size, the controls keep a 44px hit area, and an empty state has no
// dashed box. Delete this file with the components it covers.
import { readFileSync } from 'node:fs';
import { describe, expect, it } from 'vitest';

const src = (name: string) => readFileSync(new URL(`./${name}`, import.meta.url), 'utf8');

/** The declarations of the first rule whose selector is exactly `selector`. */
function rule(file: string, selector: string): string {
	const text = src(file);
	const at = text.search(new RegExp(`(^|\\n)\\s*${selector.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')} \\{`));
	if (at < 0) throw new Error(`${file} has no ${selector} rule`);
	const open = text.indexOf('{', at);
	return text.slice(open + 1, text.indexOf('}', open));
}

describe('Button elevation', () => {
	it.each(['.primary', '.secondary', '.destructive'])('%s takes the look shadow', (variant) => {
		expect(rule('Button.svelte', variant)).toContain('box-shadow: var(--shadow)');
	});

	it('gives a ghost none, and the base rule carries no shadow of its own', () => {
		expect(rule('Button.svelte', '.ghost')).toContain('box-shadow: none');
		expect(rule('Button.svelte', 'button')).not.toContain('box-shadow');
	});
});

describe('data and label faces', () => {
	it('puts the table in the data face and its header and stacked labels in the label face', () => {
		expect(rule('Table.svelte', 'table')).toContain('font-family: var(--font-data)');
		expect(rule('Table.svelte', 'thead th')).toContain('font-family: var(--font-label)');
	});

	it('sets a stat value in the data face at the heading step, never the title', () => {
		const value = rule('StatTile.svelte', '.value');
		expect(value).toContain('font-family: var(--font-data)');
		expect(value).toContain('font-size: var(--h2-size)');
		expect(value).not.toContain('--text-title');
		expect(rule('StatTile.svelte', 'dt')).toContain('font-family: var(--font-label)');
	});
});

describe('hit area', () => {
	it.each(['Button.svelte', 'FilterChip.svelte'])('%s extends to 44px by --hit-inset', (file) => {
		const text = src(file);
		expect(text).toMatch(/::after \{[^}]*inset-block: var\(--hit-inset\)/);
		expect(text).toContain('position: relative');
	});

	it('Tabs extends the trigger the same way', () => {
		expect(src('Tabs.svelte')).toMatch(/kit-tabs-trigger::after\) \{[^}]*inset-block: var\(--hit-inset\)/);
	});
});

describe('EmptyState', () => {
	it('has no dashed box and takes the look art as a snippet', () => {
		const text = src('EmptyState.svelte');
		expect(text).not.toContain('dashed');
		expect(text).toContain('art?: Snippet');
	});
});
