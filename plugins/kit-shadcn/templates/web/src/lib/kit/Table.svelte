<!-- Seeded by Albitor (kit-shadcn), then yours: a read-only data table. Cell values are text. For a
     cell that needs markup, copy this component and give the column a snippet. -->
<script module lang="ts">
	export interface Column {
		key: string;
		label: string;
	}
</script>

<script lang="ts">
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
		<table>
			<caption>{caption}</caption>
			<thead>
				<tr>
					{#each columns as col (col.key)}
						<th scope="col">{col.label}</th>
					{/each}
				</tr>
			</thead>
			<tbody>
				{#each rows as row, i (i)}
					<tr>
						{#each columns as col, c (col.key)}
							{#if c === 0}
								<th scope="row">{row[col.key] ?? ''}</th>
							{:else}
								<td>{row[col.key] ?? ''}</td>
							{/if}
						{/each}
					</tr>
				{/each}
			</tbody>
		</table>
	</div>
{/if}

<style>
	.scroll {
		overflow-x: auto;
	}
	.scroll:focus-visible {
		outline: 2px solid var(--ring);
		outline-offset: 2px;
	}
	table {
		width: 100%;
		border-collapse: collapse;
	}
	caption {
		padding: 0.5rem 0;
		text-align: start;
		font-weight: 600;
		font-size: var(--text-heading);
	}
	th,
	td {
		padding: 0.625rem 0.75rem;
		text-align: start;
		border-bottom: 1px solid var(--border);
	}
	thead th {
		background: var(--muted);
		border-bottom-color: var(--input);
	}
	tbody th {
		font-weight: 500;
	}
</style>
