// The look's page examples, rendered on the server the way a build prerenders them, loaded into a
// DOM and audited with axe: each has exactly one h1, no skipped heading level, and no violations,
// in the states it takes. Delete a page's line with the page. Colour contrast is audited from the
// tokens in ../../kit/contrast.test.ts, as jsdom has no layout.
import type { Component } from 'svelte';
import { afterEach, describe, expect, it } from 'vitest';
import { audit, serverDom } from '../../kit/__fixtures__/dom';
import FirstRun from './FirstRun.svelte';
import Habit from './Habit.svelte';
import Habits from './Habits.svelte';
import NewHabit from './NewHabit.svelte';
import Overview from './Overview.svelte';
import Preferences from './Preferences.svelte';
import Setup from './Setup.svelte';

let dom: ReturnType<typeof serverDom>;
afterEach(() => dom?.window.close());

const cases: Array<[name: string, page: Component<any>, props: Record<string, unknown>]> = [
	['home', Overview, {}],
	['home, loading', Overview, { status: 'loading' }],
	['home, failed', Overview, { status: 'error' }],
	['habits', Habits, {}],
	['habits, none yet', Habits, { rows: [] }],
	['habits, loading', Habits, { status: 'loading' }],
	['habit', Habit, {}],
	['new habit', NewHabit, {}],
	['new habit, with errors', NewHabit, { errors: [{ field: 'name', message: 'Enter a habit' }] }],
	['settings', Preferences, {}],
	['first run', FirstRun, {}],
	['setup, question 1', Setup, { step: 1 }],
	['setup, question 3', Setup, { step: 3 }]
];

describe.each(cases)('%s', (_name, page, props) => {
	it('has no axe violations', async () => {
		dom = serverDom(page, props, 'Example');
		expect(await audit(dom)).toEqual([]);
	});

	it('has exactly one h1 and no skipped heading level', () => {
		dom = serverDom(page, props, 'Example');
		const doc = dom.window.document;
		expect(doc.querySelectorAll('h1')).toHaveLength(1);
		const levels = [...doc.querySelectorAll('h1, h2, h3, h4')].map((h) => Number(h.tagName[1]));
		levels.forEach((level, i) => {
			if (i > 0) expect(level - levels[i - 1]).toBeLessThanOrEqual(1);
		});
	});
});
