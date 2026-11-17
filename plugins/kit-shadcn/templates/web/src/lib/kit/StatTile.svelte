<!-- A starting point, then yours: one headline figure. A `dl` of one term and its values, so a screen
     reader reads "Open items 12", with an optional icon, a delta in words ("up 3 on last week"; the
     arrow is decoration), and a meter (a Progress bar) or a sparkline (a Chart). Put tiles in a
     grid of your own; DashboardPage does.

       <StatTile label="Open items" value="12" icon={Inbox} delta={{ direction: 'up', text: '3 more than last week' }} /> -->
<script lang="ts">
	import type { Component } from 'svelte';
	import ArrowDown from '@lucide/svelte/icons/arrow-down';
	import ArrowUp from '@lucide/svelte/icons/arrow-up';
	import Minus from '@lucide/svelte/icons/minus';
	import Chart from './Chart.svelte';
	import Icon from './Icon.svelte';
	import Progress from './Progress.svelte';

	let {
		label,
		value,
		icon,
		delta,
		meter,
		spark
	}: {
		label: string;
		value: string | number;
		icon?: Component<{ size?: number | string; class?: string }>;
		/** Say what changed in words; `good` says whether the direction is good news. */
		delta?: { direction: 'up' | 'down' | 'flat'; text: string; good?: boolean };
		/** A share of something, 0 to 100, drawn as a bar under the figure. */
		meter?: { value: number; label: string };
		/** Recent values, drawn as a sparkline; `label` names it. */
		spark?: { data: Array<{ label: string; value: number }>; label: string };
	} = $props();

	const arrows = { up: ArrowUp, down: ArrowDown, flat: Minus };
</script>

<dl class="tile" data-kit="stat-tile">
	<dt>
		<span>{label}</span>
		{#if icon}<span class="badge"><Icon {icon} size={18} /></span>{/if}
	</dt>
	<dd class="value">{value}</dd>
	{#if delta}
		<dd class="delta" data-good={delta.good === undefined ? undefined : String(delta.good)}>
			<Icon icon={arrows[delta.direction]} size={14} />
			{delta.text}
		</dd>
	{/if}
	{#if meter}<dd class="extra"><Progress label={meter.label} value={meter.value} /></dd>{/if}
	{#if spark}<dd class="extra"><Chart kind="sparkline" compact title={spark.label} data={spark.data} /></dd>{/if}
</dl>

<style>
	.tile {
		margin: 0;
		padding: var(--space-4);
		background: var(--surface);
		border: var(--border-width) solid var(--border);
		border-radius: var(--radius);
		box-shadow: var(--shadow);
	}
	dt {
		display: flex;
		align-items: center;
		justify-content: space-between;
		gap: var(--space-2);
		color: var(--muted-foreground);
		font-size: var(--text-small);
	}
	.badge {
		display: inline-flex;
		padding: var(--space-2);
		border-radius: var(--radius);
		background: var(--second-subtle);
		color: var(--second-text);
	}
	dd {
		margin: 0;
	}
	.value {
		margin-top: var(--space-1);
		font-family: var(--heading-font);
		font-size: var(--text-title);
		font-weight: var(--heading-weight);
		font-variant-numeric: tabular-nums;
	}
	.delta {
		display: flex;
		align-items: center;
		gap: var(--space-1);
		margin-top: var(--space-1);
		font-size: var(--text-small);
		color: var(--muted-foreground);
	}
	.delta[data-good='true'] :global(.kit-icon) {
		color: var(--success);
	}
	.delta[data-good='false'] :global(.kit-icon) {
		color: var(--destructive);
	}
	.extra {
		margin-top: var(--space-3);
	}
</style>
