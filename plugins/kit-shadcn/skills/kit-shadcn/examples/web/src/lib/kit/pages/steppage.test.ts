// StepPage in each state (StepPageFixture), rendered on the server the way a build prerenders it,
// loaded into a DOM and audited with axe. Delete this file and the fixture with StepPage.svelte and ConfirmationPage.svelte.
import type { JSDOM } from 'jsdom';
import { afterEach, describe, expect, it } from 'vitest';
import { audit, serverDom } from '../__fixtures__/dom';
import Fixture from './__fixtures__/StepPageFixture.svelte';
import { expectOneH1AndNoSkippedLevel } from './__fixtures__/helpers';

let dom: JSDOM;

function mount(props: Record<string, unknown> = {}): HTMLElement {
	dom = serverDom(Fixture, props, 'Pages');
	return dom.window.document.body;
}

afterEach(() => dom?.window.close());

const cases: Array<[name: string, props: Record<string, unknown>]> = [
	['question', {}],
	['first question, no Back', { step: 1 }],
	['question, with errors', { errors: [{ field: 'postcode', message: 'Enter a postcode' }] }],
	['question, saving', { busy: true }],
	['check your answers', { phase: 'check' }],
	['confirmation', { phase: 'done' }]
];

describe('StepPage in each state', () => {
	it.each(cases)('%s has no axe violations', async (_name, props) => {
		mount(props);
		expect(await audit(dom)).toEqual([]);
	});

	it.each(cases)('%s has exactly one h1 and no skipped heading level', (_name, props) => {
		expectOneH1AndNoSkippedLevel(mount(props));
	});
});

describe('StepPage', () => {
	it('says where the person is in words and in a named progress bar', () => {
		const root = mount();
		expect(root.querySelector('.where')?.textContent).toBe('Step 2 of 4');
		const bar = root.querySelector('[role="progressbar"]');
		expect(bar?.getAttribute('aria-valuenow')).toBe('2');
		expect(bar?.getAttribute('aria-valuemax')).toBe('4');
		expect(bar?.getAttribute('aria-label')).toBe('Renew a permit progress');
	});

	it('asks one question as the h1, with Back to the previous step except on the first', () => {
		expect(mount().querySelector('h1')?.textContent).toBe('What is your postcode?');
		expect(dom.window.document.querySelector('.foot a')?.getAttribute('href')).toBe('/renew/1');
		dom.window.close();
		expect(mount({ step: 1 }).querySelector('.foot a')).toBeNull();
	});

	it('links each problem to its control and focuses nothing it cannot reach', () => {
		const root = mount({ errors: [{ field: 'postcode', message: 'Enter a postcode' }] });
		expect(root.querySelector('.summary a')?.getAttribute('href')).toBe('#postcode');
		expect(root.ownerDocument.getElementById('postcode')?.matches('input')).toBe(true);
	});

	it('checks answers with a Change link named for each answer, and a named submit', () => {
		const root = mount({ phase: 'check' });
		const links = [...root.querySelectorAll('.change a')];
		expect(links.map((a) => a.textContent)).toEqual(['Change Postcode', 'Change Vehicle']);
		expect(links[0].getAttribute('href')).toBe('/renew/2');
		expect(root.querySelector('dd')?.textContent).toBe('AB1 2CD');
		expect(root.textContent).toContain('Not answered');
		expect(root.querySelector('button[type="submit"]')?.textContent?.trim()).toBe('Submit renewal');
	});
});

describe('ConfirmationPage', () => {
	it('states the result, the reference and what happens next, with a way onward', () => {
		const root = mount({ phase: 'done' });
		expect(root.querySelector('h1')?.textContent?.trim()).toBe('Renewal sent');
		expect(root.querySelector('.ref strong')?.textContent).toBe('PRM-48213');
		expect(root.querySelectorAll('.done li')).toHaveLength(2);
		expect(root.querySelector('.done > a')?.getAttribute('href')).toBe('/');
	});
});
