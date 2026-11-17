// Tag, rendered on the server inside the shell (__fixtures__/TagFixture.svelte) and audited with
// axe. Delete this file and the fixture with Tag.svelte if the app has no status labels.
import type { JSDOM } from 'jsdom';
import { afterEach, describe, expect, it } from 'vitest';
import { audit, serverDom } from './__fixtures__/dom';
import TagFixture from './__fixtures__/TagFixture.svelte';

let dom: JSDOM;

function mount(props: Record<string, unknown> = {}): HTMLElement {
	dom = serverDom(TagFixture, props);
	return dom.window.document.body;
}

afterEach(() => dom?.window.close());

describe('Tag', () => {
	it('has no axe violations', async () => {
		mount();
		expect(await audit(dom)).toEqual([]);
	});

	it('is the default tag: words, with no colour variant', () => {
		const tag = mount().querySelector('strong.govuk-tag')!;
		expect(tag.textContent).toBe('In progress');
		expect(tag.className).toBe('govuk-tag');
	});
});
