<!-- A starting point, then yours: a filter that is on or off. A toggle button: `aria-pressed`
     carries the state, and the pressed chip also draws a check, so it is not colour alone. Put a
     row of them in a group named "Filters".

       <FilterChip bind:pressed={onlyOpen}>Open only</FilterChip> -->
<script lang="ts">
	import type { Snippet } from 'svelte';
	import Check from '@lucide/svelte/icons/check';
	import Icon from './Icon.svelte';

	let { pressed = $bindable(false), children }: { pressed?: boolean; children: Snippet } = $props();
</script>

<button type="button" class="chip" class:on={pressed} aria-pressed={pressed} data-kit="filter-chip" onclick={() => (pressed = !pressed)}>
	{#if pressed}<Icon icon={Check} size={16} />{/if}
	{@render children()}
</button>

<style>
	/* The hit area reaches 44px where the look draws a smaller chip. */
	.chip::after {
		content: '';
		position: absolute;
		inset-block: var(--hit-inset);
		inset-inline: 0;
	}
	.chip {
		position: relative;
		display: inline-flex;
		align-items: center;
		gap: var(--space-1);
		min-height: var(--control-height);
		padding: 0 var(--space-3);
		font: inherit;
		font-size: var(--text-small);
		line-height: var(--leading-small);
		color: var(--foreground);
		background: var(--background);
		border: var(--border-width) solid var(--input);
		border-radius: calc(var(--radius) * 2);
		cursor: pointer;
		transition: background-color var(--motion-duration) var(--motion-ease);
	}
	.chip:hover {
		background: var(--muted);
	}
	.on {
		background: var(--primary);
		color: var(--primary-foreground);
		border-color: var(--primary);
	}
	.on:hover {
		background: var(--primary);
	}
	.chip:focus-visible {
		outline: var(--focus-width) solid var(--ring);
		outline-offset: 2px;
	}
</style>
