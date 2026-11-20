<!-- A starting point, then yours: what to show instead of a blank list, and the first-run
     screen of a page with nothing in it yet. Say why it is empty and what to do next. It has no
     box: the shape comes from `art`, a snippet that renders the look's EmptyArt, or from an
     optional Lucide `icon`; both are decorative and either may be left out.

       {#snippet art()}<EmptyArt size={96} />{/snippet}
       <EmptyState title="No reports yet" description="Create the first report." {art} /> -->
<script lang="ts">
	import type { Component, Snippet } from 'svelte';
	import Icon from './Icon.svelte';

	let {
		title,
		description,
		level = 2,
		icon,
		art,
		action
	}: {
		title: string;
		description?: string;
		level?: 2 | 3 | 4;
		icon?: Component<{ size?: number | string; class?: string }>;
		art?: Snippet;
		action?: Snippet;
	} = $props();
</script>

<div class="empty" data-kit="empty-state">
	{#if art}<div class="art-slot">{@render art()}</div>{:else if icon}<span class="art"><Icon {icon} size={32} /></span>{/if}
	<svelte:element this={`h${level}`}>{title}</svelte:element>
	{#if description}<p>{description}</p>{/if}
	{#if action}<div class="action">{@render action()}</div>{/if}
</div>

<style>
	.empty {
		padding: var(--space-8) var(--space-4);
		text-align: center;
	}
	.art-slot {
		margin-bottom: var(--space-3);
	}
	.empty :global(h2),
	.empty :global(h3),
	.empty :global(h4) {
		margin: 0 0 var(--space-2);
		font-size: var(--h2-size);
		line-height: var(--h2-leading);
	}
	.art {
		display: inline-flex;
		margin-bottom: var(--space-3);
		color: var(--second-text);
	}
	p {
		max-width: 44ch;
		margin: 0 auto var(--space-4);
		color: var(--muted-foreground);
	}
</style>
