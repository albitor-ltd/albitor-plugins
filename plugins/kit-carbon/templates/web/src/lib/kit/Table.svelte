<!-- Seeded by Albitor (kit-carbon), then yours: a read-only data table on Carbon's table pieces.
     Cell values are text. For a cell that needs markup, copy this component and give the column a
     snippet. For sorting, selection or search, use Carbon's DataTable instead. -->
<script module lang="ts">
	export interface Column {
		key: string;
		label: string;
	}
</script>

<script lang="ts">
	import {
		Table,
		TableBody,
		TableCell,
		TableHead,
		TableHeader,
		TableRow
	} from 'carbon-components-svelte';
	import EmptyState from './EmptyState.svelte';

	let {
		caption,
		columns,
		rows,
		emptyTitle = 'Nothing here yet',
		emptyDescription
	}: {
		caption: string;
		columns: Column[];
		rows: Array<Record<string, string | number>>;
		emptyTitle?: string;
		emptyDescription?: string;
	} = $props();
</script>

{#if rows.length === 0}
	<EmptyState title={emptyTitle} description={emptyDescription} level={3} />
{:else}
	<!-- A scrollable region must be reachable by keyboard, so it takes focus (axe: scrollable-region-focusable). -->
	<!-- svelte-ignore a11y_no_noninteractive_tabindex -->
	<div class="scroll" role="region" aria-label={caption} tabindex="0">
		<Table useStaticWidth>
			<caption>{caption}</caption>
			<TableHead>
				<TableRow>
					{#each columns as col (col.key)}
						<TableHeader>{col.label}</TableHeader>
					{/each}
				</TableRow>
			</TableHead>
			<TableBody>
				{#each rows as row, i (i)}
					<TableRow>
						{#each columns as col, c (col.key)}
							{#if c === 0}
								<TableHeader scope="row">{row[col.key] ?? ''}</TableHeader>
							{:else}
								<TableCell>{row[col.key] ?? ''}</TableCell>
							{/if}
						{/each}
					</TableRow>
				{/each}
			</TableBody>
		</Table>
	</div>
{/if}

<style>
	.scroll {
		overflow-x: auto;
	}
	.scroll:focus-visible {
		outline: 2px solid var(--cds-focus);
		outline-offset: 2px;
	}
	caption {
		padding: 0.5rem 0;
		text-align: start;
		font-size: 1.25rem;
		line-height: 1.4;
		color: var(--cds-text-primary);
	}
	/* A row header in the body reads like a cell, not like the column headings. */
	.scroll :global(tbody th) {
		background: transparent;
		color: var(--cds-text-primary);
		font-weight: 600;
	}
</style>
