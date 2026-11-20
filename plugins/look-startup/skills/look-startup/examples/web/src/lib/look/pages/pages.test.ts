// The look's page examples, rendered on the server the way a build prerenders them, loaded into a
// DOM and audited with axe: each has exactly one h1, no skipped heading level, and no violations,
// in the states it takes. Delete a page's line with the page. Colour contrast is audited from the
// tokens in ../../kit/contrast.test.ts, as jsdom has no layout.
import type { Component } from 'svelte';
import { afterEach, describe, expect, it } from 'vitest';
import { audit, serverDom } from '../../kit/__fixtures__/dom';
import FirstRun from './FirstRun.svelte';
import Home from './Home.svelte';
import NewProject from './NewProject.svelte';
import Overview from './Overview.svelte';
import Preferences from './Preferences.svelte';
import Project from './Project.svelte';
import Projects from './Projects.svelte';
import Release from './Release.svelte';
import Setup from './Setup.svelte';
import Trial from './Trial.svelte';

let dom: ReturnType<typeof serverDom>;
afterEach(() => dom?.window.close());

const cases: Array<[name: string, page: Component<any>, props: Record<string, unknown>]> = [
	['overview', Overview, {}],
	['overview, first run', Overview, { firstRun: true }],
	['overview, loading', Overview, { status: 'loading' }],
	['overview, failed', Overview, { status: 'error' }],
	['projects', Projects, {}],
	['projects, none yet', Projects, { rows: [] }],
	['project', Project, {}],
	['new project', NewProject, {}],
	['new project, with errors', NewProject, { errors: [{ field: 'name', message: 'Enter a project name' }] }],
	['settings', Preferences, {}],
	['first run', FirstRun, {}],
	['setup, step 1', Setup, { step: 1 }],
	['setup, step 3', Setup, { step: 3 }],
	['home (landing)', Home, {}],
	['release note', Release, {}],
	['release note, loading', Release, { status: 'loading' }],
	['trial, question 1', Trial, { step: 1 }],
	['trial, question 2', Trial, { step: 2 }],
	['trial, with errors', Trial, { step: 1, errors: [{ field: 'email', message: 'Enter an email address' }] }],
	['trial, check your answers', Trial, { phase: 'check' }],
	['trial, confirmation', Trial, { phase: 'done' }]
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

describe('the landing-led example app', () => {
	it('leads with one headline and one primary action, and uses no images', () => {
		dom = serverDom(Home, {}, 'Example');
		const doc = dom.window.document;
		expect(doc.querySelector('h1')?.textContent).toBe('Ship the change, not the pipeline');
		expect(doc.querySelector('.hero a')?.textContent?.trim()).toBe('Start a free trial');
		expect(doc.querySelector('img')).toBeNull();
	});
});
