// @vitest-environment jsdom
// Which Carbon theme the page wears, tested through the mounted shell and the attribute Carbon puts
// on <html> (`theme="white"` or `"g100"`), not through theme.ts: that file is the app's to change.
// The contract is behavioural: follow the visitor's colour-scheme preference, let
// <html data-theme="light" | "dark"> force a scheme, and react when either changes.
import { flushSync, mount, tick, unmount } from 'svelte';
import { afterEach, describe, expect, it, vi } from 'vitest';
import Showcase from './Showcase.svelte';

let app: ReturnType<typeof mount> | undefined;
let listeners: Array<() => void> = [];

/** A matchMedia whose preference the test can flip. */
function preference(dark: boolean) {
	const state = { matches: dark };
	listeners = [];
	window.matchMedia = ((query: string) => ({
		get matches() {
			return state.matches;
		},
		media: query,
		addEventListener: (_: string, fn: () => void) => listeners.push(fn),
		removeEventListener: (_: string, fn: () => void) => {
			listeners = listeners.filter((l) => l !== fn);
		}
	})) as unknown as typeof window.matchMedia;
	return {
		set(next: boolean) {
			state.matches = next;
			for (const l of listeners) l();
		}
	};
}

async function show() {
	app = mount(Showcase, { target: document.body });
	flushSync();
	await tick();
}

const theme = () => document.documentElement.getAttribute('theme');

afterEach(() => {
	if (app) unmount(app);
	app = undefined;
	document.body.innerHTML = '';
	document.documentElement.removeAttribute('theme');
	document.documentElement.removeAttribute('data-theme');
});

describe('the Carbon theme', () => {
	it('is white for a visitor who prefers light', async () => {
		preference(false);
		await show();
		await vi.waitFor(() => expect(theme()).toBe('white'));
	});

	it('is Gray 100 for a visitor who prefers dark', async () => {
		preference(true);
		await show();
		await vi.waitFor(() => expect(theme()).toBe('g100'));
	});

	it('follows the preference when it changes', async () => {
		const pref = preference(false);
		await show();
		await vi.waitFor(() => expect(theme()).toBe('white'));
		pref.set(true);
		await vi.waitFor(() => expect(theme()).toBe('g100'));
		pref.set(false);
		await vi.waitFor(() => expect(theme()).toBe('white'));
	});

	it('lets data-theme force a scheme against the preference', async () => {
		preference(false);
		document.documentElement.setAttribute('data-theme', 'dark');
		await show();
		await vi.waitFor(() => expect(theme()).toBe('g100'));
	});

	it('follows data-theme when the page changes it later', async () => {
		preference(true);
		await show();
		await vi.waitFor(() => expect(theme()).toBe('g100'));
		document.documentElement.setAttribute('data-theme', 'light');
		await vi.waitFor(() => expect(theme()).toBe('white'));
		document.documentElement.setAttribute('data-theme', 'dark');
		await vi.waitFor(() => expect(theme()).toBe('g100'));
	});

	it('tells the browser which scheme it is, so form controls and scrollbars match', async () => {
		preference(true);
		await show();
		await vi.waitFor(() => expect(document.documentElement.style.colorScheme).toBe('dark'));
	});
});
