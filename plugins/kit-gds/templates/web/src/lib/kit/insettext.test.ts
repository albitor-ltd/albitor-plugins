// InsetText, rendered on the server inside the shell (__fixtures__/InsetTextFixture.svelte) and audited with
// axe. Delete this file and the fixture with InsetText.svelte if the app has no inset text.
import type { JSDOM } from 'jsdom';
import { afterEach, describe, expect, it } from 'vitest';
import { audit, serverDom } from './__fixtures__/dom';
import InsetTextFixture from './__fixtures__/InsetTextFixture.svelte';

let dom: JSDOM;

function mount(props: Record<string, unknown> = {}): HTMLElement {
	dom = serverDom(InsetTextFixture, props);
	return dom.window.document.body;
}

afterEach(() => dom?.window.close());

describe('InsetText', () => {
	it('has no axe violations', async () => {
		mount();
		expect(await audit(dom)).toEqual([]);
	});

	it('wraps its content in govuk-inset-text', () => {
		expect(mount().querySelector('.govuk-inset-text')?.textContent).toBe('You have until 31 March to reply.');
	});
});
