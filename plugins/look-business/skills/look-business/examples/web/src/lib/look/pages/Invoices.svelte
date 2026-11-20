<!-- A list page, in this look: saved views as filter chips, a search field, a dense table with a
     checkbox on each row, numbers right-aligned in tabular figures, and a bulk bar that appears
     when rows are selected. The kit's Table shows text only, so this one writes its own table
     markup: keep the caption, the `scope` on each header and a label on every checkbox. A
     selected row is a tint plus a 2px left bar, never colour alone (the checkbox says it too).
     A starting point, then yours: replace the columns, the views and the bulk actions. -->
<script lang="ts">
	import Plus from '@lucide/svelte/icons/plus';
	import Button from '../../kit/Button.svelte';
	import Field from '../../kit/Field.svelte';
	import FilterChip from '../../kit/FilterChip.svelte';
	import Icon from '../../kit/Icon.svelte';
	import Pill from '../../kit/Pill.svelte';
	import EmptyState from '../../kit/EmptyState.svelte';
	import EmptyArt from '../EmptyArt.svelte';
	import PageHeader from '../../kit/pages/PageHeader.svelte';
	import StateNotice from '../../kit/pages/StateNotice.svelte';

	type Row = { ref: string; supplier: string; due: string; amount: string; state: 'Approved' | 'Overdue' | 'Draft' };
	const tones = { Approved: 'success', Overdue: 'danger', Draft: 'neutral' } as const;

	let {
		status = 'ready',
		rows = [
			{ ref: 'INV-20418', supplier: 'Northwind Freight', due: '2026-11-20', amount: '£12,480.00', state: 'Approved' },
			{ ref: 'INV-20422', supplier: 'Halden Print', due: '2026-11-04', amount: '£3,150.40', state: 'Overdue' },
			{ ref: 'INV-20431', supplier: 'Marlow Office Supply', due: '2026-11-30', amount: '£942.18', state: 'Draft' }
		]
	}: { status?: 'loading' | 'error' | 'ready'; rows?: Row[] } = $props();

	let search = $state('');
	let overdueOnly = $state(false);
	let draftOnly = $state(false);
	let selected = $state<string[]>([]);

	const shown = $derived(
		rows.filter(
			(r) =>
				(r.ref + r.supplier).toLowerCase().includes(search.toLowerCase()) &&
				(!overdueOnly || r.state === 'Overdue') &&
				(!draftOnly || r.state === 'Draft')
		)
	);
	const allSelected = $derived(shown.length > 0 && shown.every((r) => selected.includes(r.ref)));
	function toggleAll() {
		selected = allSelected ? [] : shown.map((r) => r.ref);
	}
	const noop = () => {};
</script>

<PageHeader title="Invoices" description="Every invoice from every supplier.">
	{#snippet actions()}
		<Button onclick={noop}><Icon icon={Plus} size={16} /> New invoice</Button>
	{/snippet}
</PageHeader>

<div class="tools">
	<div class="views" role="group" aria-label="Saved views">
		<FilterChip bind:pressed={overdueOnly}>Overdue</FilterChip>
		<FilterChip bind:pressed={draftOnly}>Drafts</FilterChip>
	</div>
	<Field label="Search invoices" type="search" bind:value={search} />
</div>

{#if selected.length > 0}
	<div class="bulk" role="region" aria-label="Selected invoices">
		<span>{selected.length} selected</span>
		<Button variant="secondary" onclick={noop}>Approve</Button>
		<Button variant="secondary" onclick={noop}>Export</Button>
		<Button variant="ghost" onclick={() => (selected = [])}>Clear selection</Button>
	</div>
{/if}

{#snippet firstRunArt()}<EmptyArt size={96} />{/snippet}

{#if status !== 'ready'}
	<StateNotice {status} onretry={noop} loadingLabel="Loading invoices" />
{:else if shown.length === 0}
	<EmptyState
		art={rows.length === 0 ? firstRunArt : undefined}
		title={rows.length === 0 ? 'No invoices yet' : 'No invoices match'}
		description={rows.length === 0
			? 'Add the first invoice, or import a file of them.'
			: 'Clear the search or a view to see every invoice.'}
	/>
{:else}
	<!-- A scrollable region must be reachable by keyboard, so it takes focus. -->
	<!-- svelte-ignore a11y_no_noninteractive_tabindex -->
	<div class="scroll" tabindex="0" role="region" aria-label="Invoices table">
		<table>
			<caption class="sr">Invoices</caption>
			<thead>
				<tr>
					<th scope="col" class="check">
						<input type="checkbox" aria-label="Select all invoices" checked={allSelected} onchange={toggleAll} />
					</th>
					<th scope="col">Reference</th>
					<th scope="col">Supplier</th>
					<th scope="col">Status</th>
					<th scope="col">Due</th>
					<th scope="col" class="num">Amount</th>
				</tr>
			</thead>
			<tbody>
				{#each shown as row (row.ref)}
					<tr class:sel={selected.includes(row.ref)}>
						<td class="check">
							<input
								type="checkbox"
								aria-label={`Select ${row.ref}`}
								value={row.ref}
								bind:group={selected}
							/>
						</td>
						<th scope="row">{row.ref}</th>
						<td>{row.supplier}</td>
						<td><Pill tone={tones[row.state]}>{row.state}</Pill></td>
						<td class="num">{row.due}</td>
						<td class="num">{row.amount}</td>
					</tr>
				{/each}
			</tbody>
		</table>
	</div>
{/if}

<style>
	.tools {
		display: flex;
		flex-wrap: wrap;
		align-items: end;
		justify-content: space-between;
		gap: var(--space-3);
		margin-bottom: var(--space-3);
	}
	.views {
		display: flex;
		flex-wrap: wrap;
		gap: var(--space-2);
	}
	.bulk {
		display: flex;
		flex-wrap: wrap;
		align-items: center;
		gap: var(--space-3);
		padding: var(--space-2) var(--space-3);
		margin-bottom: var(--space-2);
		background: var(--second-subtle);
		border: var(--border-width) solid var(--border);
		font-weight: 600;
	}
	.scroll {
		overflow-x: auto;
	}
	table {
		width: 100%;
		border-collapse: collapse;
		font-variant-numeric: var(--figures);
	}
	th,
	td {
		padding: var(--space-2) var(--space-3);
		border-bottom: var(--border-width) solid var(--border);
		text-align: start;
		vertical-align: middle;
	}
	thead th {
		color: var(--muted-foreground);
		font-size: var(--text-small);
		font-weight: 600;
	}
	tbody th {
		font-weight: 600;
	}
	.num {
		text-align: end;
	}
	.check {
		width: 2.75rem;
	}
	tr.sel td,
	tr.sel th {
		background: var(--second-subtle);
	}
	tr.sel :is(td, th):first-child {
		box-shadow: inset 2px 0 0 var(--primary);
	}
	.sr {
		position: absolute;
		width: 1px;
		height: 1px;
		overflow: hidden;
		clip-path: inset(50%);
		white-space: nowrap;
	}
</style>
