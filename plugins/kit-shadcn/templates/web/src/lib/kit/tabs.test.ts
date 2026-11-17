// @vitest-environment jsdom
// Tabs, mounted in a real (jsdom) browser tree (__fixtures__/TabsFixture.svelte): Bits UI's roving
// focus and selection need the client, so the server render cannot see them. Delete this file and
// the fixture with Tabs.svelte if the app has none.
import axe from 'axe-core';
import { flushSync, mount, tick, unmount } from 'svelte';
import { afterEach, describe, expect, it } from 'vitest';
import TabsFixture from './__fixtures__/TabsFixture.svelte';

let app: ReturnType<typeof mount> | undefined;

afterEach(() => {
	if (app) unmount(app);
	app = undefined;
	document.body.innerHTML = '';
});

async function show() {
	app = mount(TabsFixture, { target: document.body });
	flushSync();
	await tick();
}

const tabs = () => [...document.querySelectorAll<HTMLElement>('[role="tab"]')];
const panel = () => document.querySelector<HTMLElement>('[role="tabpanel"]:not([hidden])');

describe('Tabs', () => {
	it('has no axe violations', async () => {
		await show();
		const result = await axe.run(document.body, {
			runOnly: {
				type: 'tag',
				values: ['wcag2a', 'wcag2aa', 'wcag21a', 'wcag21aa', 'wcag22aa', 'best-practice']
			},
			rules: { 'color-contrast': { enabled: false }, region: { enabled: false } }
		});
		expect(
			result.violations.map((v) => `${v.id}: ${v.nodes.map((n) => n.html).join(' | ')}`)
		).toEqual([]);
	});

	it('is a named tablist with the first tab selected and its panel shown', async () => {
		await show();
		expect(document.querySelector('[role="tablist"]')?.getAttribute('aria-label')).toBe(
			'Report sections'
		);
		expect(tabs().map((t) => [t.textContent?.trim(), t.getAttribute('aria-selected')])).toEqual([
			['Summary', 'true'],
			['Detail', 'false'],
			['History', 'false']
		]);
		expect(panel()?.textContent).toContain('The summary view.');
	});

	it('selects a tab on click and shows only its panel, tied to the tab', async () => {
		await show();
		tabs()[1].click();
		await tick();
		await tick();
		expect(tabs()[1].getAttribute('aria-selected')).toBe('true');
		expect(panel()?.textContent).toContain('The detail view.');
		expect(panel()?.getAttribute('aria-labelledby')).toBe(tabs()[1].id);
		expect(tabs()[1].getAttribute('aria-controls')).toBe(panel()?.id);
	});

	it('moves between tabs with the arrow keys', async () => {
		await show();
		tabs()[0].focus();
		tabs()[0].dispatchEvent(new KeyboardEvent('keydown', { key: 'ArrowRight', bubbles: true }));
		await tick();
		await tick();
		expect(document.activeElement).toBe(tabs()[1]);
	});
});
