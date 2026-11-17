<!-- A starting point, then yours: a small chart drawn as SVG in the kit's tokens (the primary and
     second colours on the surface). `bar` compares a few values; `sparkline` shows a trend. A chart
     is a picture, so it always has a text equivalent: the `title` names it and a table of the same
     values is in the page for assistive technology (visually hidden), so nothing is conveyed by the
     drawing alone. Keep it to a handful of values; this is not a charting library.

       <Chart title="Orders per day" data={[{ label: 'Mon', value: 4 }, { label: 'Tue', value: 7 }]} />
       <Chart kind="sparkline" title="Signups, last 7 days" data={...} compact /> -->
<script lang="ts">
	let {
		title,
		data,
		kind = 'bar',
		compact = false,
		unit = ''
	}: {
		/** Names the chart and captions the text equivalent. */
		title: string;
		data: Array<{ label: string; value: number }>;
		kind?: 'bar' | 'sparkline';
		/** A short, wide drawing with no axis labels, for a tile. */
		compact?: boolean;
		/** Appended to each value in the text equivalent, for example `%` or ` ms`. */
		unit?: string;
	} = $props();

	const id = $props.id();
	const w = 240;
	const h = $derived(compact ? 40 : 120);
	const pad = 4;
	const max = $derived(Math.max(1, ...data.map((d) => d.value)));
	const min = $derived(Math.min(0, ...data.map((d) => d.value)));
	const span = $derived(max - min || 1);
	const y = (v: number) => pad + (1 - (v - min) / span) * (h - pad * 2);
	const step = $derived(data.length > 1 ? (w - pad * 2) / (data.length - 1) : 0);
	const line = $derived(data.map((d, i) => `${pad + i * step},${y(d.value)}`).join(' '));
	const slot = $derived(data.length ? (w - pad * 2) / data.length : 0);
</script>

<figure class="chart" class:compact data-kit="chart" data-kind={kind}>
	<svg viewBox="0 0 {w} {h}" role="img" aria-labelledby="{id}-t" preserveAspectRatio="none">
		<title id="{id}-t">{title}</title>
		{#if kind === 'sparkline'}
			<polyline class="line" points={line} />
		{:else}
			{#each data as d, i (i)}
				<rect
					class="bar"
					x={pad + i * slot + slot * 0.15}
					y={Math.min(y(d.value), y(0))}
					width={slot * 0.7}
					height={Math.max(1, Math.abs(y(d.value) - y(0)))}
					rx="2"
				/>
			{/each}
		{/if}
	</svg>
	{#if kind === 'bar' && !compact}
		<div class="axis" aria-hidden="true">{#each data as d, i (i)}<span>{d.label}</span>{/each}</div>
	{/if}
	<table class="sr">
		<caption>{title}</caption>
		<thead><tr><th scope="col">Label</th><th scope="col">Value</th></tr></thead>
		<tbody>
			{#each data as d, i (i)}<tr><th scope="row">{d.label}</th><td>{d.value}{unit}</td></tr>{/each}
		</tbody>
	</table>
</figure>

<style>
	.chart {
		margin: 0;
	}
	svg {
		display: block;
		inline-size: 100%;
		block-size: 7.5rem;
		overflow: visible;
	}
	.compact svg {
		block-size: 2.5rem;
	}
	.bar {
		fill: var(--primary);
	}
	.line {
		fill: none;
		stroke: var(--primary);
		stroke-width: 2;
		stroke-linejoin: round;
		stroke-linecap: round;
		vector-effect: non-scaling-stroke;
	}
	.axis {
		display: flex;
		justify-content: space-around;
		margin-top: var(--space-1);
		font-size: var(--text-small);
		color: var(--muted-foreground);
	}
	/* The text equivalent: present for assistive technology, not drawn. */
	.sr {
		position: absolute;
		inline-size: 1px;
		block-size: 1px;
		margin: -1px;
		padding: 0;
		overflow: hidden;
		clip-path: inset(50%);
		white-space: nowrap;
	}
</style>
