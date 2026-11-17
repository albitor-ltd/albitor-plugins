// Chart (bar and sparkline), rendered on the server inside the shell
// (__fixtures__/ChartFixture.svelte) and audited with axe. Delete this file and the fixture with
// Chart.svelte if the app has none.
import type { JSDOM } from 'jsdom';
import { afterEach, describe, expect, it } from 'vitest';
import { audit, serverDom } from './__fixtures__/dom';
import ChartFixture from './__fixtures__/ChartFixture.svelte';

let dom: JSDOM;

function mount(): HTMLElement {
	dom = serverDom(ChartFixture);
	return dom.window.document.body;
}

afterEach(() => dom?.window.close());

describe('Chart', () => {
	it('has no axe violations', async () => {
		mount();
		expect(await audit(dom)).toEqual([]);
	});

	it('is an image named by its title, with the same values as a table', () => {
		const chart = mount().querySelector('[data-kind="bar"]')!;
		const svg = chart.querySelector('svg')!;
		expect(svg.getAttribute('role')).toBe('img');
		expect(chart.querySelector(`#${svg.getAttribute('aria-labelledby')}`)?.textContent).toBe(
			'Orders per day'
		);
		const rows = [...chart.querySelectorAll('tbody tr')].map((r) => r.textContent);
		expect(rows).toEqual(['Mon4 orders', 'Tue7 orders', 'Wed3 orders']);
		expect(chart.querySelector('caption')?.textContent).toBe('Orders per day');
	});

	it('draws one bar per value, the tallest for the largest', () => {
		const bars = [...mount().querySelector('[data-kind="bar"]')!.querySelectorAll('rect.bar')];
		expect(bars).toHaveLength(3);
		const heights = bars.map((b) => Number(b.getAttribute('height')));
		expect(heights[1]).toBeGreaterThan(heights[0]);
		expect(heights[0]).toBeGreaterThan(heights[2]);
	});

	it('draws a sparkline as one line, undistorted, and no axis labels', () => {
		const spark = mount().querySelector('[data-kind="sparkline"]')!;
		expect(spark.querySelectorAll('polyline')).toHaveLength(1);
		expect(spark.querySelectorAll('circle')).toHaveLength(0);
		expect(spark.querySelector('.axis')).toBeNull();
		expect(spark.querySelectorAll('tbody tr')).toHaveLength(3);
	});

	it('accepts repeated labels', () => {
		const root = mount();
		
		expect(root.querySelectorAll('[data-kind="bar"]')[2].querySelectorAll('rect')).toHaveLength(2);
	});

	it('renders no data without an error', () => {
		const empty = [...mount().querySelectorAll('[data-kind="bar"]')][1];
		expect(empty.querySelectorAll('rect')).toHaveLength(0);
	});

	it('uses only tokens for colour', () => {
		const html = mount().innerHTML;
		expect(html).not.toMatch(/(fill|stroke)="#/);
	});
});
