<!-- Seeded by Albitor (kit-carbon), then yours: what to show instead of a blank list. Say why it is
     empty and what to do next. -->
<script lang="ts">
	import type { Snippet } from 'svelte';

	let {
		title,
		description,
		level = 2,
		action
	}: { title: string; description?: string; level?: 2 | 3 | 4; action?: Snippet } = $props();
</script>

<div class="empty" data-kit="empty-state">
	<svelte:element this={`h${level}`}>{title}</svelte:element>
	{#if description}<p>{description}</p>{/if}
	{#if action}<div class="action">{@render action()}</div>{/if}
</div>

<style>
	.empty {
		padding: 2rem 1rem;
		background: var(--cds-layer);
		color: var(--cds-text-primary);
		border: 1px dashed var(--cds-border-strong);
	}
	.empty :global(h2),
	.empty :global(h3),
	.empty :global(h4) {
		margin: 0 0 0.5rem;
		font-size: 1.25rem;
		font-weight: 400;
		line-height: 1.4;
	}
	p {
		margin: 0 0 1rem;
		color: var(--cds-text-secondary);
	}
</style>
