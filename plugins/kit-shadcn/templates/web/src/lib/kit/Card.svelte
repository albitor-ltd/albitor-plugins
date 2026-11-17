<!-- A starting point, then yours: a surface that groups related content. With a `title` it is a
     group named by its heading (not a landmark: a page of cards would otherwise be a page of
     regions), and a plain container without one. Elevation, border
     width and radius come from the look through the tokens; do not set them per card.

       <Card title="Recent activity" level={2}>...</Card>
       <Card tone="second">A highlighted note</Card> -->
<script lang="ts">
	import type { Snippet } from 'svelte';

	let {
		title,
		level = 2,
		tone = 'default',
		actions,
		children
	}: {
		title?: string;
		level?: 2 | 3 | 4;
		/** `second` uses the look's second colour as a quiet background. */
		tone?: 'default' | 'second';
		/** Controls at the right of the heading. */
		actions?: Snippet;
		children: Snippet;
	} = $props();

	const id = $props.id();
</script>

<div
	class="card {tone}"
	data-kit="card"
	role={title ? 'group' : undefined}
	aria-labelledby={title ? `${id}-title` : undefined}
>
	{#if title}
		<div class="head">
			<svelte:element this={`h${level}`} id={`${id}-title`}>{title}</svelte:element>
			{#if actions}<div class="actions">{@render actions()}</div>{/if}
		</div>
	{/if}
	{@render children()}
</div>

<style>
	.card {
		padding: var(--space-4);
		background: var(--surface);
		color: var(--foreground);
		border: var(--border-width) solid var(--border);
		border-radius: var(--radius);
		box-shadow: var(--shadow);
	}
	.second {
		background: var(--second-subtle);
	}
	.head {
		display: flex;
		align-items: center;
		justify-content: space-between;
		gap: var(--space-3);
		margin-bottom: var(--space-3);
	}
	.head :global(h2),
	.head :global(h3),
	.head :global(h4) {
		margin: 0;
		font-size: var(--text-heading);
	}
</style>
