<!-- A starting point, then yours: what to show instead of a blank list, and the first-run
     screen of a page with nothing in it yet. Say why it is empty and what to do next; an optional
     Lucide `icon` (decorative) gives it a shape. -->
<script lang="ts">
	import type { Component, Snippet } from 'svelte';
	import Icon from './Icon.svelte';

	let {
		title,
		description,
		level = 2,
		icon,
		action
	}: {
		title: string;
		description?: string;
		level?: 2 | 3 | 4;
		icon?: Component<{ size?: number | string; class?: string }>;
		action?: Snippet;
	} = $props();
</script>

<div class="empty" data-kit="empty-state">
	{#if icon}<span class="art"><Icon {icon} size={32} /></span>{/if}
	<svelte:element this={`h${level}`}>{title}</svelte:element>
	{#if description}<p>{description}</p>{/if}
	{#if action}<div class="action">{@render action()}</div>{/if}
</div>

<style>
	.empty {
		padding: var(--space-8) var(--space-4);
		border: var(--border-width) dashed var(--input);
		border-radius: var(--radius);
		text-align: start;
	}
	.empty :global(h2),
	.empty :global(h3),
	.empty :global(h4) {
		margin: 0 0 var(--space-2);
		font-size: var(--text-heading);
	}
	.art {
		display: inline-flex;
		margin-bottom: var(--space-3);
		color: var(--second-text);
	}
	p {
		margin: 0 0 var(--space-4);
		color: var(--muted-foreground);
	}
</style>
