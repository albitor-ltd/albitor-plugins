<!-- The person's own collection, grid first: what they have gathered or logged, as a grid of linked
     items with a one-line summary (a count, a streak, "3 finished"), an add action and a first-run
     state. A grid suits things a person recognises by name and picture; for exact values across
     many rows use ListPage. No imagery is supplied: each item is typography and a status Pill, and
     the look's EmptyArt can fill the first-run state through `emptyArt`.

       <CollectionPage title="My plants" summary="12 plants, 2 need water" items={plants} {status}
                       onretry={reload} emptyTitle="No plants yet"
                       emptyDescription="Add the first one to start your log.">
         {#snippet actions()}<Button onclick={add}>Add a plant</Button>{/snippet}
         {#snippet emptyAction()}<Button onclick={add}>Add a plant</Button>{/snippet}
       </CollectionPage> -->
<script lang="ts">
	import type { Component, Snippet } from 'svelte';
	import EmptyState from '../EmptyState.svelte';
	import Pill from '../Pill.svelte';
	import PageHeader from './PageHeader.svelte';
	import StateNotice from './StateNotice.svelte';

	let {
		title,
		description,
		summary,
		items,
		status = 'ready',
		errorMessage,
		onretry,
		emptyTitle = 'Nothing here yet',
		emptyDescription,
		emptyIcon,
		emptyArt,
		actions,
		emptyAction
	}: {
		title: string;
		description?: string;
		/** One line on the whole collection: a count, a total, what needs attention. */
		summary?: string;
		items: Array<{
			id: string;
			href: string;
			title: string;
			subtitle?: string;
			badge?: { tone: 'neutral' | 'success' | 'warning' | 'danger' | 'info'; label: string };
		}>;
		status?: 'loading' | 'error' | 'ready';
		errorMessage?: string;
		onretry?: () => void;
		emptyTitle?: string;
		/** Say why it is empty and what to do next. */
		emptyDescription?: string;
		emptyIcon?: Component<{ size?: number | string; class?: string }>;
		/** The look's decorative art for the first-run state (its EmptyArt); never the only cue. */
		emptyArt?: Snippet;
		actions?: Snippet;
		/** The button that adds the first item. */
		emptyAction?: Snippet;
	} = $props();
</script>

<PageHeader {title} {description} {actions} />

{#if status !== 'ready'}
	<StateNotice {status} {errorMessage} {onretry} loadingLabel={`Loading ${title.toLowerCase()}`} />
{:else if items.length === 0}
	{#if emptyArt}<div class="art">{@render emptyArt()}</div>{/if}
	<EmptyState title={emptyTitle} description={emptyDescription} icon={emptyIcon} action={emptyAction} />
{:else}
	{#if summary}<p class="summary">{summary}</p>{/if}
	<ul class="grid">
		{#each items as item (item.id)}
			<li>
				<h2><a href={item.href}>{item.title}</a></h2>
				{#if item.subtitle}<p class="subtitle">{item.subtitle}</p>{/if}
				{#if item.badge}<p class="badge"><Pill tone={item.badge.tone}>{item.badge.label}</Pill></p>{/if}
			</li>
		{/each}
	</ul>
{/if}

<style>
	.summary {
		margin: 0 0 var(--space-4);
		color: var(--muted-foreground);
	}
	.art {
		margin-bottom: var(--space-4);
	}
	.grid {
		display: grid;
		grid-template-columns: repeat(auto-fill, minmax(13rem, 1fr));
		gap: var(--space-4);
		margin: 0;
		padding: 0;
		list-style: none;
	}
	li {
		position: relative;
		display: flex;
		flex-direction: column;
		gap: var(--space-2);
		min-height: 7rem;
		padding: var(--space-4);
		background: var(--surface);
		border: var(--border-width) solid var(--border);
		border-radius: var(--radius);
		box-shadow: var(--shadow);
	}
	li:hover {
		border-color: var(--input);
	}
	li:focus-within {
		outline: var(--focus-width) solid var(--ring);
		outline-offset: 2px;
	}
	h2 {
		margin: 0;
		font-size: var(--text-heading);
	}
	h2 a {
		color: var(--link);
		text-decoration: none;
	}
	h2 a:hover {
		text-decoration: underline;
	}
	h2 a::after {
		content: '';
		position: absolute;
		inset: 0;
	}
	h2 a:focus-visible {
		outline: none;
	}
	.subtitle {
		margin: 0;
		color: var(--muted-foreground);
		font-size: var(--text-small);
	}
	.badge {
		margin: auto 0 0;
	}
</style>
