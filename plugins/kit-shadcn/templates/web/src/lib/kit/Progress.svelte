<!-- A starting point, then yours: how far along something is, as a bar or a ring. It is a
     progressbar with a name and a value; `showValue` also draws the percentage as text, so the
     figure is never only a length. A `null` value is indeterminate.

       <Progress label="Storage used" value={62} />
       <Progress label="Setup" value={3} max={5} variant="ring" showValue /> -->
<script lang="ts">
	import { Progress } from 'bits-ui';

	let {
		label,
		value,
		max = 100,
		variant = 'bar',
		showValue = false
	}: {
		/** Names the progressbar for assistive technology. */
		label: string;
		value: number | null;
		max?: number;
		variant?: 'bar' | 'ring';
		showValue?: boolean;
	} = $props();

	const pct = $derived(value === null ? 0 : Math.max(0, Math.min(100, (value / max) * 100)));
	const radius = 18;
	const circumference = 2 * Math.PI * radius;
</script>

{#if variant === 'ring'}
	<Progress.Root {value} {max} aria-label={label} class="kit-progress-ring" data-kit="progress">
		<svg viewBox="0 0 44 44" width="44" height="44" aria-hidden="true">
			<circle class="track" cx="22" cy="22" r={radius} />
			<circle
				class="fill"
				cx="22"
				cy="22"
				r={radius}
				stroke-dasharray={circumference}
				stroke-dashoffset={circumference * (1 - pct / 100)}
				transform="rotate(-90 22 22)"
			/>
		</svg>
		{#if showValue}<span class="ring-value">{Math.round(pct)}%</span>{/if}
	</Progress.Root>
{:else}
	<div class="bar-wrap" data-kit="progress">
		<Progress.Root {value} {max} aria-label={label} class="kit-progress-bar">
			<div class="fill" style:inline-size="{pct}%"></div>
		</Progress.Root>
		{#if showValue}<span class="bar-value">{Math.round(pct)}%</span>{/if}
	</div>
{/if}

<style>
	.bar-wrap {
		display: flex;
		align-items: center;
		gap: var(--space-3);
	}
	:global(.kit-progress-bar) {
		flex: 1;
		overflow: hidden;
		block-size: 0.5rem;
		border-radius: calc(var(--radius) * 2);
		background: var(--muted);
	}
	.fill {
		block-size: 100%;
		background: var(--primary);
		border-radius: inherit;
		transition: inline-size var(--motion-duration) var(--motion-ease);
	}
	.bar-value {
		font-size: var(--text-small);
		color: var(--muted-foreground);
		font-variant-numeric: tabular-nums;
	}
	:global(.kit-progress-ring) {
		position: relative;
		display: inline-flex;
		align-items: center;
		justify-content: center;
		inline-size: 2.75rem;
		block-size: 2.75rem;
	}
	circle {
		fill: none;
		stroke-width: 4;
	}
	circle.track {
		stroke: var(--muted);
	}
	circle.fill {
		stroke: var(--primary);
		stroke-linecap: round;
		transition: stroke-dashoffset var(--motion-duration) var(--motion-ease);
	}
	.ring-value {
		position: absolute;
		font-size: 0.6875rem;
		font-weight: 600;
		font-variant-numeric: tabular-nums;
	}
</style>
