// SummaryList and SummaryCard, rendered on the server inside the shell (__fixtures__/SummaryListFixture.svelte) and audited with
// axe. Delete this file and the fixture with SummaryList.svelte if the app has no summary list.
import type { JSDOM } from 'jsdom';
import { afterEach, describe, expect, it } from 'vitest';
import { audit, serverDom } from './__fixtures__/dom';
import SummaryListFixture from './__fixtures__/SummaryListFixture.svelte';

let dom: JSDOM;

function mount(props: Record<string, unknown> = {}): HTMLElement {
	dom = serverDom(SummaryListFixture, props);
	return dom.window.document.body;
}

afterEach(() => dom?.window.close());

describe('SummaryList and SummaryCard', () => {
	it.each([[true], [false]])('has no axe violations (change links: %s)', async (withChange) => {
		mount({ withChange });
		expect(await audit(dom)).toEqual([]);
	});

	it('is a description list of rows, each key beside its value', () => {
		const list = mount().querySelector('dl.govuk-summary-list')!;
		const row = list.querySelector('.govuk-summary-list__row')!;
		expect(row.querySelector('dt')?.textContent).toBe('Name');
		expect(row.querySelector('.govuk-summary-list__value')?.textContent).toBe('Sarah Philips');
	});

	it('names what each change link changes, in hidden text', () => {
		const link = mount().querySelector('.govuk-summary-list__actions a')!;
		expect(link.getAttribute('href')).toBe('/name');
		expect(link.textContent?.replace(/\s+/g, ' ').trim()).toBe('Change name');
		expect(link.querySelector('.govuk-visually-hidden')).not.toBeNull();
	});

	it('marks rows with no change link only when others have one', () => {
		expect(mount({ withChange: false }).querySelector('.govuk-summary-list__row--no-actions')).toBeNull();
	});

	it('titles a card with a heading, and names the card in its actions', () => {
		const card = mount().querySelector('.govuk-summary-card')!;
		expect(card.querySelector('h2.govuk-summary-card__title')?.textContent).toBe('Second applicant');
		const actions = [...card.querySelectorAll('.govuk-summary-card__action a')];
		expect(actions.map((a) => a.textContent?.replace(/\s+/g, ' ').trim())).toEqual([
			'Delete (Second applicant)',
			'Change (Second applicant)'
		]);
		expect(card.querySelector('.govuk-summary-card__content dl.govuk-summary-list')).not.toBeNull();
	});

	it('puts a single card action in a div, not a list', () => {
		const card = mount().querySelectorAll('.govuk-summary-card')[1]!;
		expect(card.querySelector('div.govuk-summary-card__actions')).not.toBeNull();
		expect(card.querySelector('ul')).toBeNull();
	});
});
