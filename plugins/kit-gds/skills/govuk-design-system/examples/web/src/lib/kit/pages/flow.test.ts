// The task-list flow (__fixtures__/FlowFixture.svelte): task list, one question per page, check
// answers, confirmation. Each page is rendered on the server the way a build prerenders it, loaded
// into a DOM and audited with axe, plus the GOV.UK pattern checks. Delete this file and the
// fixture with the four pages if the app has no such flow.
import type { JSDOM } from 'jsdom';
import { afterEach, describe, expect, it } from 'vitest';
import { audit, serverDom } from '../__fixtures__/dom';
import FlowFixture from './__fixtures__/FlowFixture.svelte';
import { expectOneH1AndNoSkippedLevel, expectSummaryLinksResolve } from './__fixtures__/helpers';

let dom: JSDOM;

function mount(props: Record<string, unknown>): HTMLElement {
	dom = serverDom(FlowFixture, props, 'Pages');
	return dom.window.document.body;
}

afterEach(() => dom?.window.close());

const cases: Array<[name: string, props: Record<string, unknown>]> = [
	['task list', { step: 'tasks' }],
	['task list, all done', { step: 'tasks', allDone: true }],
	['question', { step: 'question' }],
	['question, with an error', { step: 'question', error: 'Enter your full name' }],
	['check answers', { step: 'check' }],
	['confirmation', { step: 'confirmation' }]
];

describe('the task-list flow, page by page', () => {
	it.each(cases)('%s has no axe violations', async (_name, props) => {
		mount(props);
		expect(await audit(dom)).toEqual([]);
	});

	it.each(cases)('%s has exactly one h1 and no skipped heading level', (_name, props) => {
		expectOneH1AndNoSkippedLevel(mount(props));
	});
});

describe('task list page', () => {
	it('counts the completed tasks and offers check answers only when all are done', () => {
		const root = mount({ step: 'tasks' });
		expect(root.textContent).toContain('You have completed 0 of 3 sections.');
		expect(root.querySelector('a[href="/apply/check"]')).toBeNull();
		root.ownerDocument.defaultView?.close();
		const done = mount({ step: 'tasks', allDone: true });
		expect(done.textContent).toContain('You have completed 3 of 3 sections.');
		expect(done.querySelector('a.govuk-button')?.getAttribute('href')).toBe('/apply/check');
	});

	it('uses a tag for each task that is open, and has no back link on the first page', () => {
		const root = mount({ step: 'tasks' });
		expect(root.querySelectorAll('.govuk-task-list .govuk-tag')).toHaveLength(2);
		expect(root.querySelector('.govuk-back-link')).toBeNull();
	});
});

describe('question page', () => {
	it('asks one thing, as the heading, with a back link and one primary button', () => {
		const root = mount({ step: 'question' });
		expect(root.querySelector('h1 label')?.textContent).toBe('What is your full name?');
		expect(root.querySelector('.govuk-back-link')?.getAttribute('href')).toBe('/apply');
		expect(root.querySelectorAll('input, textarea, select')).toHaveLength(1);
		expect(root.querySelectorAll('button.govuk-button')).toHaveLength(1);
		expect(root.querySelector('input')?.getAttribute('aria-describedby')).toBe('name-hint');
	});

	it('shows the error summary above the heading, linked to the control, beside the field error', () => {
		const root = mount({ step: 'question', error: 'Enter your full name' });
		expectSummaryLinksResolve(root);
		expect(root.querySelector('.govuk-error-summary')!.compareDocumentPosition(root.querySelector('h1')!) & 4).toBeTruthy();
		expect(root.querySelector('#name-error')?.textContent).toContain('Enter your full name');
		expect(root.querySelector('input')?.getAttribute('aria-describedby')).toBe('name-hint name-error');
	});
});

describe('check answers page', () => {
	it('lists every answer with a change link naming it, and sends with a named button', () => {
		const root = mount({ step: 'check' });
		const changes = [...root.querySelectorAll('.govuk-summary-list__actions a')].map((a) => a.getAttribute('href'));
		expect(changes).toEqual(['/apply/name', '/apply/premises']);
		expect(root.querySelector('.govuk-back-link')).not.toBeNull();
		expect(root.querySelector('button[type="submit"]')?.textContent).toBe('Accept and send');
	});
});

describe('confirmation page', () => {
	it('gives the reference and what happens next, and has no back link', () => {
		const root = mount({ step: 'confirmation' });
		expect(root.querySelector('h1')?.textContent).toBe('Application sent');
		expect(root.textContent).toContain('HDJ2123F');
		expect(root.querySelector('h2')?.textContent).toBe('What happens next');
		expect(root.querySelector('.govuk-back-link')).toBeNull();
	});
});
