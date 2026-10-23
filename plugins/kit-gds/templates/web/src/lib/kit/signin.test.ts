// @vitest-environment jsdom
// Managed by Albitor (kit-gds). SignInPage's error behaviour, the same in every kit (component-kit@2):
// the failure stays in the page until the next attempt's result replaces it, and takes focus on
// every failed attempt, a repeated sentence included. Mounted in a jsdom tree because focus is a
// client behaviour.
import { flushSync, mount, tick, unmount } from 'svelte';
import { afterEach, describe, expect, it } from 'vitest';
import SignInHarness from './SignInHarness.svelte';

let app: (ReturnType<typeof mount> & { set(next: { error?: string; email?: string; busy?: boolean }): void }) | undefined;

afterEach(() => {
	if (app) unmount(app);
	app = undefined;
	document.body.innerHTML = '';
});

async function show() {
	app = mount(SignInHarness, { target: document.body }) as typeof app;
	flushSync();
	await tick();
}

async function set(next: { error?: string; email?: string; busy?: boolean }) {
	app!.set(next);
	flushSync();
	await tick();
}

const alert = () => document.querySelector<HTMLElement>('[role="alert"]');
const email = () => document.querySelector<HTMLInputElement>('input[type="email"]')!;
/** Focus is on the alert, or on the summary that holds it. */
const alertHasFocus = () => {
	const a = document.activeElement;
	return !!a && (a.matches('[role="alert"]') || !!a.querySelector('[role="alert"]'));
};
const SAME = 'Email or password is incorrect';

describe('SignInPage errors', () => {
	it('shows no alert and takes no focus before a failure', async () => {
		await show();
		expect(alert()).toBeNull();
		expect(alertHasFocus()).toBe(false);
	});

	it('focuses the error when an attempt fails', async () => {
		await show();
		await set({ busy: true });
		await set({ busy: false, error: SAME });
		expect(alert()?.textContent).toContain(SAME);
		expect(alertHasFocus()).toBe(true);
	});

	it('mounts and focuses the first error once', async () => {
		await show();
		let focused = 0;
		document.addEventListener('focusin', () => (focused += 1));
		await set({ error: SAME });
		expect(focused).toBe(1);
		expect(document.querySelectorAll('[role="alert"]')).toHaveLength(1);
	});

	it('keeps the error in the page while the next attempt is busy', async () => {
		await show();
		await set({ error: SAME });
		email().focus();
		await set({ busy: true });
		expect(alert()?.textContent).toContain(SAME);
		expect(alertHasFocus()).toBe(false);
		expect((document.querySelector('button[type="submit"]') as HTMLButtonElement).disabled).toBe(true);
	});

	it('focuses the error again when the same sentence fails again', async () => {
		await show();
		await set({ error: SAME });
		expect(alertHasFocus()).toBe(true);
		email().focus();
		await set({ busy: true });
		await set({ busy: false });
		expect(alert()?.textContent).toContain(SAME);
		expect(alertHasFocus()).toBe(true);
	});

	it('focuses a different error that arrives while idle', async () => {
		await show();
		await set({ error: SAME });
		email().focus();
		await set({ error: 'Too many attempts, try again later' });
		expect(alertHasFocus()).toBe(true);
	});

	it('waits for the attempt to finish before focusing an error that changes while busy', async () => {
		await show();
		await set({ error: SAME });
		email().focus();
		await set({ busy: true });
		await set({ error: 'Too many attempts, try again later' });
		expect(alert()?.textContent).toContain('Too many attempts');
		expect(alertHasFocus()).toBe(false);
		await set({ busy: false });
		expect(alertHasFocus()).toBe(true);
	});

	it('does not steal focus when nothing changed', async () => {
		await show();
		await set({ error: SAME });
		email().focus();
		await set({ error: SAME });
		expect(alertHasFocus()).toBe(false);
	});
});

describe('SignInPage email prop', () => {
	it('is an initial value: a later change to it does not overwrite what was typed', async () => {
		await show();
		email().value = 'typed@example.test';
		email().dispatchEvent(new Event('input', { bubbles: true }));
		await set({ email: 'later@example.test' });
		expect(email().value).toBe('typed@example.test');
	});
});
