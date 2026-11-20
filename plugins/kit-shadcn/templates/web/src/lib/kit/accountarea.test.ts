// AccountArea, rendered on the server inside the shell's account slot
// (__fixtures__/AccountAreaFixture.svelte) and audited with axe. Delete this file and the fixture
// with AccountArea.svelte if the app has none.
import type { JSDOM } from 'jsdom';
import { afterEach, describe, expect, it } from 'vitest';
import { audit, serverDom } from './__fixtures__/dom';
import AccountAreaFixture from './__fixtures__/AccountAreaFixture.svelte';

let dom: JSDOM;

function mount(props: Record<string, unknown> = {}): HTMLElement {
	dom = serverDom(AccountAreaFixture, props);
	return dom.window.document.body;
}

afterEach(() => dom?.window.close());

describe('AccountArea', () => {
	it('has no axe violations', async () => {
		mount();
		expect(await audit(dom)).toEqual([]);
	});

	it('sits in the header, outside main, as one labelled group', () => {
		const groups = mount().querySelectorAll('[data-kit="account-area"]');
		expect(groups).toHaveLength(1);
		expect(groups[0].getAttribute('aria-label')).toBe('Account');
		expect(groups[0].closest('header')).not.toBeNull();
		expect(groups[0].closest('main')).toBeNull();
	});

	it('names Sign out and the theme switch, which reports its state', () => {
		const buttons = [...mount().querySelectorAll('[data-kit="account-area"] button')];
		expect(buttons.map((b) => b.textContent?.trim())).toEqual(['Dark theme', 'Sign out']);
		expect(buttons[0].getAttribute('aria-pressed')).toBe('false');
	});

	it('reports the dark theme as pressed', () => {
		const theme = mount({ theme: 'dark' }).querySelector('[aria-pressed]');
		expect(theme?.getAttribute('aria-pressed')).toBe('true');
	});

	it('says the save status in words inside a live region, and draws nothing when idle', () => {
		expect(mount({ saveStatus: 'error' }).querySelector('[role="status"]')?.textContent).toContain(
			'Not saved'
		);
		dom.window.close();
		const idle = mount({ saveStatus: 'idle' }).querySelector('[role="status"]');
		expect(idle).not.toBeNull();
		expect(idle?.textContent?.trim()).toBe('');
	});
});
