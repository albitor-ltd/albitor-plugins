<!-- A starting point, then yours: a read-only data table. Cell values are text. For a
     cell that needs markup, copy this component and give the column a snippet. -->
<script module lang="ts">
	export interface Column {
		key: string;
		label: string;
		/** Priority-columns mode only: `secondary` columns leave the row on a narrow container and
		 * stay reachable in a Details disclosure under the first cell. Default `primary`. */
		priority?: 'primary' | 'secondary';
	}
	/** How a narrow container (under 40rem, measured on the table, not the viewport) lays out:
	 * `scroll` keeps the table and scrolls it sideways (the default); `stacked` turns each row into
	 * a card with every field labelled; `priority` keeps the primary columns in the row. */
	export type Narrow = 'scroll' | 'stacked' | 'priority';
</script>

<script lang="ts">
	import EmptyState from './EmptyState.svelte';

	let {
		caption,
		columns,
		rows,
		emptyTitle = 'Nothing here yet',
		emptyDescription,
		narrow = 'scroll'
	}: {
		caption: string;
		columns: Column[];
		rows: Array<Record<string, string | number>>;
		emptyTitle?: string;
		emptyDescription?: string;
		narrow?: Narrow;
	} = $props();

	const secondary = $derived(
		narrow === 'priority' ? columns.filter((col, i) => i > 0 && col.priority === 'secondary') : []
	);
	const isSecondary = (col: Column, c: number) =>
		narrow === 'priority' && c > 0 && col.priority === 'secondary';
	// Stacked rows are display:block, which drops table semantics in some browsers (Safari with
	// VoiceOver), so restore them with explicit roles. Undefined otherwise: native elements need none.
	const st = $derived(narrow === 'stacked');
</script>

{#if rows.length === 0}
	<EmptyState title={emptyTitle} description={emptyDescription} level={3} />
{:else}
	<!-- A scrollable region must be reachable by keyboard, so it takes focus (axe: scrollable-region-focusable). -->
	<!-- svelte-ignore a11y_no_noninteractive_tabindex -->
	<div class="scroll" data-narrow={narrow} role="region" aria-label={caption} tabindex="0">
		<table role={st ? 'table' : undefined}>
			<caption>{caption}</caption>
			<thead role={st ? 'rowgroup' : undefined}>
				<tr role={st ? 'row' : undefined}>
					{#each columns as col, c (col.key)}
						<th scope="col" role={st ? 'columnheader' : undefined} class:secondary={isSecondary(col, c)}>{col.label}</th>
					{/each}
					{#if secondary.length > 0}
						<th scope="col" class="more"><span class="sr-only">Details</span></th>
					{/if}
				</tr>
			</thead>
			<tbody role={st ? 'rowgroup' : undefined}>
				{#each rows as row, i (i)}
					<tr role={st ? 'row' : undefined}>
						{#each columns as col, c (col.key)}
							{#if c === 0}
								<th scope="row" role={st ? 'rowheader' : undefined}>{row[col.key] ?? ''}</th>
							{:else}
								<td
									data-label={col.label}
									role={st ? 'cell' : undefined}
									class:secondary={isSecondary(col, c)}>{row[col.key] ?? ''}</td
								>
							{/if}
						{/each}
						{#if secondary.length > 0}
							<td class="more">
								<details>
									<summary>Details<span class="sr-only"> for {row[columns[0].key] ?? ''}</span></summary>
									<dl>
										{#each secondary as s (s.key)}
											<dt>{s.label}</dt>
											<dd>{row[s.key] ?? ''}</dd>
										{/each}
									</dl>
								</details>
							</td>
						{/if}
					</tr>
				{/each}
			</tbody>
		</table>
	</div>
{/if}

<style>
	.scroll {
		overflow-x: auto;
		container-type: inline-size;
	}
	.scroll:focus-visible {
		outline: var(--focus-width) solid var(--ring);
		outline-offset: 2px;
	}
	table {
		width: 100%;
		border-collapse: collapse;
		font-variant-numeric: var(--figures);
	}
	caption {
		padding: var(--space-2) 0;
		text-align: start;
		font-family: var(--heading-font);
		font-weight: var(--heading-weight);
		letter-spacing: var(--heading-tracking);
		font-size: var(--text-heading);
	}
	th,
	td {
		height: var(--row-height, auto);
		padding: calc(0.625rem * var(--space)) var(--space-3);
		text-align: start;
		border-bottom: var(--border-width) solid var(--border);
	}
	thead th {
		background: var(--muted);
		border-bottom-color: var(--input);
	}
	tbody th {
		font-weight: 500;
	}
	.more {
		display: none;
	}
	.sr-only {
		position: absolute;
		inline-size: 1px;
		block-size: 1px;
		overflow: hidden;
		clip-path: inset(50%);
		white-space: nowrap;
	}
	@container (max-width: 40rem) {
		/* Stacked rows: each row is a card, each cell shows its column label. The header row stays
		   in the accessibility tree, visually hidden. */
		[data-narrow='stacked'] table,
		[data-narrow='stacked'] tbody,
		[data-narrow='stacked'] tr,
		[data-narrow='stacked'] td,
		[data-narrow='stacked'] tbody th {
			display: block;
		}
		[data-narrow='stacked'] thead {
			position: absolute;
			inline-size: 1px;
			block-size: 1px;
			overflow: hidden;
			clip-path: inset(50%);
		}
		[data-narrow='stacked'] tbody tr {
			border: var(--border-width) solid var(--border);
			border-radius: var(--radius, 0.5rem);
			background: var(--surface);
			box-shadow: var(--shadow);
			padding: var(--space-2) var(--space-3);
			margin-block-end: var(--space-2);
		}
		[data-narrow='stacked'] tbody th,
		[data-narrow='stacked'] td {
			border-bottom: 0;
			height: auto;
			padding: calc(0.125rem * var(--space)) 0;
		}
		[data-narrow='stacked'] td {
			display: flex;
			justify-content: space-between;
			gap: var(--space-3);
		}
		[data-narrow='stacked'] td::before {
			content: attr(data-label);
			flex: none;
			color: var(--muted-foreground);
		}
		/* Priority columns: secondary columns leave the row; the disclosure carries them. */
		[data-narrow='priority'] .secondary {
			display: none;
		}
		[data-narrow='priority'] .more {
			display: table-cell;
			font-weight: 400;
		}
		[data-narrow='priority'] .more summary {
			cursor: pointer;
			font-size: var(--text-small, 0.875rem);
			display: inline-flex;
			align-items: center;
			min-block-size: 24px;
			min-inline-size: 24px;
		}
		[data-narrow='priority'] .more dl {
			display: grid;
			grid-template-columns: auto 1fr;
			gap: calc(0.125rem * var(--space)) var(--space-3);
			margin: calc(0.375rem * var(--space)) 0 0;
		}
		[data-narrow='priority'] .more dd {
			margin: 0;
		}
	}
</style>
