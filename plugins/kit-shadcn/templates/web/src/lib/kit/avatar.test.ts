// Avatar, rendered on the server inside the shell (__fixtures__/AvatarFixture.svelte) and audited
// with axe. Delete this file and the fixture with Avatar.svelte if the app has none.
import type { JSDOM } from 'jsdom';
import { afterEach, describe, expect, it } from 'vitest';
import { audit, serverDom } from './__fixtures__/dom';
import AvatarFixture from './__fixtures__/AvatarFixture.svelte';

let dom: JSDOM;

function mount(): HTMLElement {
	dom = serverDom(AvatarFixture);
	return dom.window.document.body;
}

afterEach(() => dom?.window.close());

describe('Avatar', () => {
	it('has no axe violations', async () => {
		mount();
		expect(await audit(dom)).toEqual([]);
	});

	it('is an image named for the person, with initials hidden from assistive technology', () => {
		const [ada, grace] = [...mount().querySelectorAll('[data-kit="avatar"]')];
		expect(ada.getAttribute('role')).toBe('img');
		expect(ada.getAttribute('aria-label')).toBe('Ada Lovelace');
		expect(ada.textContent?.trim()).toBe('AL');
		expect(ada.querySelector('[aria-hidden="true"]')).not.toBeNull();
		expect(grace.textContent?.trim()).toBe('G');
	});
});
