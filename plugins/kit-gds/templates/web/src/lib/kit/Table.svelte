<!-- Seeded by Albitor (kit-gds), then yours: a read-only GOV.UK table (govuk-table). Cell values are
     text. For a cell that needs markup, copy this component and give the column a snippet. -->
<script module lang="ts">
	export interface Column {
		key: string;
		label: string;
	}
</script>

<script lang="ts">
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
	<h2 class="govuk-heading-s">{emptyTitle}</h2>
	{#if emptyDescription}<p class="govuk-body">{emptyDescription}</p>{/if}
{:else}
	<!-- A scrollable region must be reachable by keyboard, so it takes focus (axe: scrollable-region-focusable). -->
	<!-- svelte-ignore a11y_no_noninteractive_tabindex -->
	<div class="kit-scroll" role="region" aria-label={caption} tabindex="0">
		<table class="govuk-table">
			<caption class="govuk-table__caption govuk-table__caption--m">{caption}</caption>
			<thead class="govuk-table__head">
				<tr class="govuk-table__row">
					{#each columns as col (col.key)}
						<th scope="col" class="govuk-table__header">{col.label}</th>
					{/each}
				</tr>
			</thead>
			<tbody class="govuk-table__body">
				{#each rows as row, i (i)}
					<tr class="govuk-table__row">
						{#each columns as col, c (col.key)}
							{#if c === 0}
								<th scope="row" class="govuk-table__header">{row[col.key] ?? ''}</th>
							{:else}
								<td class="govuk-table__cell">{row[col.key] ?? ''}</td>
							{/if}
						{/each}
					</tr>
				{/each}
			</tbody>
		</table>
	</div>
{/if}

<style>
	.kit-scroll {
		overflow-x: auto;
	}
	/* GOV.UK's focus state: yellow with a black underline. */
	.kit-scroll:focus {
		outline: 3px solid #fd0;
		box-shadow: 0 0 0 3px #fd0, 0 6px 0 3px #0b0c0c;
	}
</style>
