<!-- Seeded by Albitor (kit-carbon), then yours: a list page. A header with a primary action, an
     optional filter area, then the rows in a Table (or an EmptyState saying what to do first).
     Loading and failure show in the place the rows will appear. Fetch in +page.ts or the page, and
     pass `status`, `rows` and `onretry`.

       <ListPage title="Reports" caption="Reports" columns={[{ key: 'name', label: 'Name' }]}
                 {rows} {status} onretry={reload}>
         {#snippet actions()}<Button onclick={create}>New report</Button>{/snippet}
       </ListPage> -->
<script lang="ts">
	import type { Snippet } from 'svelte';
	import EmptyState from '../EmptyState.svelte';
	import Table, { type Column } from '../Table.svelte';
	import PageHeader from './PageHeader.svelte';
	import StateNotice from './StateNotice.svelte';

	let {
		title,
		description,
		caption,
		columns,
		rows,
		status = 'ready',
		errorMessage,
		onretry,
		emptyTitle = 'Nothing here yet',
		emptyDescription,
		actions,
		filters,
		emptyAction
	}: {
		title: string;
		description?: string;
		/** Names the table for assistive technology; usually the same as the title. */
		caption: string;
		columns: Column[];
		rows: Array<Record<string, string | number>>;
		status?: 'loading' | 'error' | 'ready';
		errorMessage?: string;
		onretry?: () => void;
		emptyTitle?: string;
		/** Say why it is empty and what to do next. */
		emptyDescription?: string;
		actions?: Snippet;
		/** Search and filter controls, each a labelled Field. */
		filters?: Snippet;
		/** The button that starts the first item. */
		emptyAction?: Snippet;
	} = $props();
</script>

<PageHeader {title} {description} {actions} />

{#if filters}<div class="filters" role="group" aria-label="Filters">{@render filters()}</div>{/if}

{#if status !== 'ready'}
	<StateNotice {status} {errorMessage} {onretry} loadingLabel={`Loading ${title.toLowerCase()}`} />
{:else if rows.length === 0}
	<EmptyState title={emptyTitle} description={emptyDescription} action={emptyAction} />
{:else}
	<Table {caption} {columns} {rows} />
{/if}

<style>
	.filters {
		display: flex;
		flex-wrap: wrap;
		gap: 1rem;
		margin-bottom: 1rem;
	}
</style>
