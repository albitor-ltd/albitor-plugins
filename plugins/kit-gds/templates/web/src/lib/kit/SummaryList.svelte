<!-- A starting point, then yours: the GOV.UK summary list (Frontend 6.4.0 markup): facts as a
     key and a value, and the heart of a check-answers page. A row's `change` link names what it
     changes in visually hidden text ("Change <hidden>"), so a screen reader hears "Change name",
     not "Change" five times. Keys and values are text, never markup. Give every row a change link
     or none, so the columns line up. -->
<script module lang="ts">
	export interface SummaryRow {
		key: string;
		value: string;
		/** Where the person fixes this answer. `hidden` completes "Change" for a screen reader. */
		change?: { href: string; hidden: string };
	}
</script>

<script lang="ts">
	let { rows, class: className }: { rows: SummaryRow[]; class?: string } = $props();

	const anyAction = $derived(rows.some((r) => r.change));
</script>

<dl class={['govuk-summary-list', className]}>
	{#each rows as row, i (i)}
		<div class={['govuk-summary-list__row', anyAction && !row.change && 'govuk-summary-list__row--no-actions']}>
			<dt class="govuk-summary-list__key">{row.key}</dt>
			<dd class="govuk-summary-list__value">{row.value}</dd>
			{#if row.change}
				<dd class="govuk-summary-list__actions">
					<a class="govuk-link" href={row.change.href}>
						Change<span class="govuk-visually-hidden">{` ${row.change.hidden}`}</span>
					</a>
				</dd>
			{/if}
		</div>
	{/each}
</dl>
