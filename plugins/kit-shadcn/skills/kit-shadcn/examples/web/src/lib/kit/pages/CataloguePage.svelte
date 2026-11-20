<!-- A catalogue: search and filter a public collection, then open an item. A header, a filter group
     (a search Field and FilterChips, passed as a snippet), a polite result count, then the matches
     as a list of linked items. Filtering is the caller's: pass the items that match, and `total`
     (how many there are before filtering) so the count reads "3 of 12". No match says so and
     offers to clear the filters; a collection with nothing in it yet is a first-run state.
     Loading and failure show where the results will appear.

       <CataloguePage title="Courses" items={matches} total={all.length} {status} onretry={reload}
                      onclear={reset} emptyDescription="Courses are added each term.">
         {#snippet filters()}
           <Field label="Search courses" type="search" bind:value={query} />
           <FilterChip bind:pressed={evenings}>Evenings</FilterChip>
         {/snippet}
       </CataloguePage>

     Each item links to its own page (ItemPage). The whole card is the link target; the item's
     title is the link text, so a screen reader hears one link per item. -->
<script lang="ts">
	import type { Component, Snippet } from 'svelte';
	import Button from '../Button.svelte';
	import EmptyState from '../EmptyState.svelte';
	import Pill from '../Pill.svelte';
	import PageHeader from './PageHeader.svelte';
	import StateNotice from './StateNotice.svelte';

	let {
		title,
		description,
		items,
		total,
		status = 'ready',
		errorMessage,
		onretry,
		onclear,
		emptyTitle = 'Nothing here yet',
		emptyDescription,
		emptyIcon,
		filters
	}: {
		title: string;
		description?: string;
		/** The items that match the current search and filters. */
		items: Array<{
			id: string;
			href: string;
			title: string;
			summary: string;
			/** A short fact line: place, price, date. Text, never markup. */
			meta?: string;
			/** A status or category, drawn as a Pill (a glyph and a word). */
			badge?: { tone: 'neutral' | 'success' | 'warning' | 'danger' | 'info'; label: string };
		}>;
		/** How many items there are before filtering; leave out to show only the matches. */
		total?: number;
		status?: 'loading' | 'error' | 'ready';
		errorMessage?: string;
		onretry?: () => void;
		/** Clears the search and filters; offered when nothing matches. */
		onclear?: () => void;
		/** The first-run state: the collection has nothing in it at all. */
		emptyTitle?: string;
		emptyDescription?: string;
		emptyIcon?: Component<{ size?: number | string; class?: string }>;
		/** A search Field and FilterChips. */
		filters?: Snippet;
	} = $props();

	const count = $derived(
		total === undefined || total === items.length
			? `${items.length} ${items.length === 1 ? 'result' : 'results'}`
			: `${items.length} of ${total} results`
	);
</script>

<PageHeader {title} {description} />

{#if filters}<div class="filters" role="group" aria-label="Search and filters">{@render filters()}</div>{/if}

{#if status !== 'ready'}
	<StateNotice {status} {errorMessage} {onretry} loadingLabel={`Loading ${title.toLowerCase()}`} />
{:else if total === 0}
	<EmptyState title={emptyTitle} description={emptyDescription} icon={emptyIcon} />
{:else}
	<p class="count" role="status">{count}</p>
	{#if items.length === 0}
		<EmptyState title="No matches" description="Try a different search, or remove a filter.">
			{#snippet action()}
				{#if onclear}<Button variant="secondary" onclick={onclear}>Clear search and filters</Button>{/if}
			{/snippet}
		</EmptyState>
	{:else}
		<ul class="items">
			{#each items as item (item.id)}
				<li>
					<h2><a href={item.href}>{item.title}</a></h2>
					<p class="summary">{item.summary}</p>
					{#if item.meta || item.badge}
						<p class="facts">
							{#if item.badge}<Pill tone={item.badge.tone}>{item.badge.label}</Pill>{/if}
							{#if item.meta}<span>{item.meta}</span>{/if}
						</p>
					{/if}
				</li>
			{/each}
		</ul>
	{/if}
{/if}

<style>
	.filters {
		display: flex;
		flex-wrap: wrap;
		align-items: end;
		gap: var(--space-4);
		margin-bottom: var(--space-4);
	}
	.count {
		margin: 0 0 var(--space-3);
		color: var(--muted-foreground);
		font-size: var(--text-small);
	}
	.items {
		display: grid;
		grid-template-columns: repeat(auto-fill, minmax(18rem, 1fr));
		gap: var(--space-4);
		margin: 0;
		padding: 0;
		list-style: none;
	}
	li {
		position: relative;
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
		margin: 0 0 var(--space-2);
		font-size: var(--text-heading);
	}
	h2 a {
		color: var(--link);
		text-decoration: none;
	}
	h2 a:hover {
		text-decoration: underline;
	}
	/* The whole card is the link target; the focus ring is the card's (li:focus-within). */
	h2 a::after {
		content: '';
		position: absolute;
		inset: 0;
	}
	h2 a:focus-visible {
		outline: none;
	}
	.summary {
		margin: 0 0 var(--space-3);
	}
	.facts {
		display: flex;
		flex-wrap: wrap;
		align-items: center;
		gap: var(--space-2) var(--space-3);
		margin: 0;
		color: var(--muted-foreground);
		font-size: var(--text-small);
	}
</style>
