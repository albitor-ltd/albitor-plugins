// @vitest-environment jsdom
// The dialog, mounted in a real (jsdom) browser tree: its axe audit and behaviour. The browser owns
// the focus trap, Escape and returning focus (a native <dialog>), and jsdom implements none of
// them, so those are not asserted here; what the kit owns is asserted: the open state in and out,
// the accessible name and description, and the Close button. jsdom has no layout, so colour
// contrast is audited from the stylesheet (contrast.test.ts).
import axe from 'axe-core';
import { flushSync, mount, tick, unmount } from 'svelte';
import { afterEach, describe, expect, it } from 'vitest';
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

const dialog = () => document.querySelector<HTMLDialogElement>('dialog');
const isOpen = () => dialog()?.hasAttribute('open') ?? false;

describe('Dialog', () => {
	it('is closed until asked for', async () => {
		await show();
		expect(dialog()).not.toBeNull();
		expect(isOpen()).toBe(false);
	});

	it('is named by its title and described by its description', async () => {
		await show({ dialogOpen: true });
		const d = dialog()!;
		expect(isOpen()).toBe(true);
		expect(document.getElementById(d.getAttribute('aria-labelledby')!)?.textContent).toContain(
			'Delete report'
		);
		expect(document.getElementById(d.getAttribute('aria-describedby')!)?.textContent).toContain(
			'cannot be undone'
		);
	});

	it('opens from a button', async () => {
		await show();
		const opener = [...document.querySelectorAll('button')].find(
			(b) => !dialog()!.contains(b) && b.textContent?.includes('Delete report')
		)!;
		opener.click();
		await tick();
		await tick();
		expect(isOpen()).toBe(true);
	});

	it('closes from its Close button', async () => {
		await show({ dialogOpen: true });
		const close = [...dialog()!.querySelectorAll('button')].find(
			(b) => b.textContent?.trim() === 'Close'
		)!;
		close.click();
		await tick();
		await tick();
		expect(isOpen()).toBe(false);
	});

	it('puts the first focus on the safe action, never the destructive one', async () => {
		await show({ dialogOpen: true });
		const focused = document.activeElement as HTMLElement | null;
		expect(dialog()!.contains(focused)).toBe(true);
		expect(focused?.textContent?.trim()).toBe('Close');
		expect(focused?.className).not.toContain('govuk-button--warning');
	});

	it('has no axe violations while open', async () => {
		await show({ dialogOpen: true });
		expect(isOpen()).toBe(true);
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
