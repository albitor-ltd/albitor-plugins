// TaskList, rendered on the server inside the shell (__fixtures__/TaskListFixture.svelte) and audited with
// axe. Delete this file and the fixture with TaskList.svelte if the app has no task list.
import type { JSDOM } from 'jsdom';
import { afterEach, describe, expect, it } from 'vitest';
import { audit, serverDom } from './__fixtures__/dom';
import TaskListFixture from './__fixtures__/TaskListFixture.svelte';

let dom: JSDOM;

function mount(props: Record<string, unknown> = {}): HTMLElement {
	dom = serverDom(TaskListFixture, props);
	return dom.window.document.body;
}

afterEach(() => dom?.window.close());

describe('TaskList', () => {
	it('has no axe violations', async () => {
		mount();
		expect(await audit(dom)).toEqual([]);
	});

	it('states each status in words: Completed as text, the in-flight ones as a tag', () => {
		const statuses = [...mount().querySelectorAll('.govuk-task-list__status')];
		expect(statuses.map((s) => s.textContent?.trim())).toEqual([
			'Completed',
			'In progress',
			'Not started',
			'Cannot start yet'
		]);
		expect(statuses.map((s) => s.querySelector('.govuk-tag') !== null)).toEqual([false, true, true, false]);
		expect(statuses[3].classList.contains('govuk-task-list__status--cannot-start-yet')).toBe(true);
	});

	it('links a task that can start, and describes the link by its hint and status', () => {
		const root = mount();
		const link = root.querySelector<HTMLAnchorElement>('a.govuk-task-list__link')!;
		expect(link.getAttribute('href')).toBe('/details');
		const ids = link.getAttribute('aria-describedby')!.split(' ');
		expect(ids).toHaveLength(2);
		expect(ids.map((id) => root.querySelector(`#${id}`)?.textContent?.trim())).toEqual(['Name and email', 'Completed']);
	});

	it('does not link a task that cannot start yet', () => {
		const items = mount().querySelectorAll('.govuk-task-list__item');
		expect(items[3].querySelector('a')).toBeNull();
		expect(items).toHaveLength(4);
	});
});
