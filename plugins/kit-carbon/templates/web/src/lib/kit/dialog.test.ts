// @vitest-environment jsdom
// The dialog, mounted in a real (jsdom) browser tree. Carbon's Modal is client behaviour (focus
// trap, Escape, returning focus), so the server render in kit.test.ts cannot see it; this file is
// the dialog's axe and behaviour audit. Carbon keeps a closed Modal in the page, inert, so "open"
// means the `is-visible` class on its wrapper. jsdom has no layout, so colour contrast is audited from the tokens (contrast.test.ts).
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

/** Carbon restores focus on `transitionend` of the modal's transform. */
function endTransition() {
	const e = new Event('transitionend', { bubbles: true });
	Object.defineProperty(e, 'propertyName', { value: 'transform' });
	document.querySelector('.bx--modal')!.dispatchEvent(e);
}

const dialog = () => document.querySelector<HTMLElement>('.bx--modal.is-visible [role="dialog"]');

describe('Dialog', () => {
	it('is closed until asked for', async () => {
		await show();
		expect(dialog()).toBeNull();
	});

	it('is named by its title and described by its description', async () => {
		await show({ dialogOpen: true });
		const d = dialog()!;
		expect(d).not.toBeNull();
		expect(d.getAttribute('aria-label')).toBe('Delete report');
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
		// Carbon moves focus in on a later frame.
		await vi.waitFor(() => expect(d.contains(document.activeElement)).toBe(true));

		document.activeElement!.dispatchEvent(
			new KeyboardEvent('keydown', { key: 'Escape', bubbles: true })
		);
		await tick();
		await tick();
		expect(dialog()).toBeNull();
		// Carbon returns focus when the closing transition ends, and jsdom runs no transitions.
		endTransition();
		// Focus goes back to the control that opened it.
		await vi.waitFor(() => expect(document.activeElement).toBe(opener));
	});

	it('traps focus: Tab from the last control wraps to the first, Shift+Tab the other way', async () => {
		await show();
		const opener = [...document.querySelectorAll('button')].find((b) =>
			b.textContent?.includes('Delete report')
		)!;
		opener.focus();
		opener.click();
		await tick();
		await tick();
		const d = dialog()!;
		await vi.waitFor(() => expect(d.contains(document.activeElement)).toBe(true));

		const controls = [...d.querySelectorAll<HTMLElement>('button, [href], input, [tabindex="0"]')];
		expect(controls.length).toBeGreaterThan(1);
		const [first, last] = [controls[0], controls[controls.length - 1]];
		last.focus();
		last.dispatchEvent(new KeyboardEvent('keydown', { key: 'Tab', bubbles: true, cancelable: true }));
		expect(document.activeElement).toBe(first);
		first.dispatchEvent(
			new KeyboardEvent('keydown', { key: 'Tab', shiftKey: true, bubbles: true, cancelable: true })
		);
		expect(document.activeElement).toBe(last);
	});

	it('closes from its Close button', async () => {
		await show({ dialogOpen: true });
		const close = document.querySelector<HTMLButtonElement>(
			'.bx--modal.is-visible [role="dialog"] button[aria-label="Close"]'
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
			// heading-order: Carbon's modal title is an h3 whatever the page's outline above it.
			rules: {
				'color-contrast': { enabled: false },
				region: { enabled: false },
				'heading-order': { enabled: false }
			}
		});
		expect(
			result.violations.map((v) => `${v.id}: ${v.nodes.map((n) => n.html).join(' | ')}`)
		).toEqual([]);
	});
});
