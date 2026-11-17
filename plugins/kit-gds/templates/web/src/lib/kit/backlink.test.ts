// BackLink, rendered on the server inside the shell (__fixtures__/BackLinkFixture.svelte) and audited with
// axe. Delete this file and the fixture with BackLink.svelte if the app has no back link.
import type { JSDOM } from 'jsdom';
import { afterEach, describe, expect, it } from 'vitest';
import { audit, serverDom } from './__fixtures__/dom';
import BackLinkFixture from './__fixtures__/BackLinkFixture.svelte';

let dom: JSDOM;

function mount(props: Record<string, unknown> = {}): HTMLElement {
	dom = serverDom(BackLinkFixture, props);
	return dom.window.document.body;
}

afterEach(() => dom?.window.close());

describe('BackLink', () => {
	it('has no axe violations', async () => {
		mount();
		expect(await audit(dom)).toEqual([]);
	});

	it('is a link to the previous page, labelled Back', () => {
		const a = mount().querySelector<HTMLAnchorElement>('a.govuk-back-link')!;
		expect(a.getAttribute('href')).toBe('/');
		expect(a.textContent).toBe('Back');
	});
});
