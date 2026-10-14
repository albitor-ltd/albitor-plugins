// @vitest-environment jsdom
// The dialog, mounted in a real (jsdom) browser tree. Bits UI portals the content in the client,
// so the server render in kit.test.ts cannot see it; this file is the dialog's axe and behaviour
// audit. jsdom has no layout, so colour contrast is audited from the tokens (contrast.test.ts).
import axe from 'axe-core';
import { flushSync, mount, tick, unmount } from 'svelte';
import { afterEach, describe, expect, it, vi } from 'vitest';
import Showcase from './Showcase.svelte';

let app: ReturnType<typeof mount> | undefined;

afterEach(() => {
	if (app) unmount(app);
	app = undefined;
	document.body.innerHTML = '';
});

async function show(props: Record<string, unknown> = {}) {
	app = mount(Showcase, { target: document.body, props });
	flushSync();
	await tick();
}

const dialog = () => document.querySelector<HTMLElement>('[role="dialog"]');

describe('Dialog', () => {
	it('is closed until asked for', async () => {
		await show();
		expect(dialog()).toBeNull();
	});

	it('is named by its title and described by its description', async () => {
		await show({ dialogOpen: true });
		const d = dialog()!;
		expect(d).not.toBeNull();
		expect(document.getElementById(d.getAttribute('aria-labelledby')!)?.textContent).toContain(
			'Delete report'
		);
		expect(document.getElementById(d.getAttribute('aria-describedby')!)?.textContent).toContain(
			'cannot be undone'
		);
	});

	it('opens from a button, moves focus inside, and closes on Escape', async () => {
		await show();
		const opener = [...document.querySelectorAll('button')].find((b) =>
			b.textContent?.includes('Delete report')
		)!;
		opener.focus();
		opener.click();
		await tick();
		await tick();
		const d = dialog()!;
		expect(d).not.toBeNull();
		// Bits UI moves focus in on a later frame.
		await vi.waitFor(() => expect(d.contains(document.activeElement)).toBe(true));

		document.activeElement!.dispatchEvent(
			new KeyboardEvent('keydown', { key: 'Escape', bubbles: true })
		);
		await tick();
		await tick();
		expect(dialog()).toBeNull();
	});

	it('closes from its Close button', async () => {
		await show({ dialogOpen: true });
		const close = [...document.querySelectorAll<HTMLButtonElement>('[role="dialog"] button')].find(
			(b) => b.textContent?.trim() === 'Close'
		)!;
		close.click();
		await tick();
		await tick();
		expect(dialog()).toBeNull();
	});

	it('has no axe violations while open', async () => {
		await show({ dialogOpen: true });
		expect(dialog()).not.toBeNull();
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
});
