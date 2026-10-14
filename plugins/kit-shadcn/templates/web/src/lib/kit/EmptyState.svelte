<!-- Seeded by Albitor (kit-shadcn), then yours: what to show instead of a blank list. Say why it is
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
		border: 1px dashed var(--input);
		border-radius: var(--radius);
		text-align: start;
	}
	.empty :global(h2),
	.empty :global(h3),
	.empty :global(h4) {
		margin: 0 0 0.5rem;
		font-size: var(--text-heading);
	}
	p {
		margin: 0 0 1rem;
		color: var(--muted-foreground);
	}
</style>
