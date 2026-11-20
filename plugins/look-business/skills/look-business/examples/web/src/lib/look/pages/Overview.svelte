<!-- The dashboard first screen, in this look. Use it only when the app's structure names a
     dashboard or an internal tool (a summary of figures that change without the person's own
     input); for any other structure start from the non-dashboard pages beside it (Hub, Renew,
     Letters). The overview: a dense summary strip of exact figures in one bordered row (tabular
     figures, no shadow, the second colour on the one bar that carries data), then a table of what
     needs doing. A starting point, then yours: replace the figures and the table, and pass the
     load `status`. Built from PageHeader and StateNotice (copy the kit's page templates from the
     kit-shadcn skill). Before the app has any data, show FirstRun.svelte instead. -->
<script lang="ts">
	import Plus from '@lucide/svelte/icons/plus';
	import Button from '../../kit/Button.svelte';
	import Card from '../../kit/Card.svelte';
	import Icon from '../../kit/Icon.svelte';
	import Pill from '../../kit/Pill.svelte';
	import Progress from '../../kit/Progress.svelte';
	import Table from '../../kit/Table.svelte';
	import PageHeader from '../../kit/pages/PageHeader.svelte';
	import StateNotice from '../../kit/pages/StateNotice.svelte';

	let { status = 'ready' }: { status?: 'loading' | 'error' | 'ready' } = $props();

	const figures = [
		{ label: 'Open invoices', value: '214', note: '12 more than last week' },
		{ label: 'Value outstanding', value: '£412,860.00', note: 'across 38 suppliers' },
		{ label: 'Overdue', value: '17', note: '£58,420.00' },
		{ label: 'Paid this month', value: '£1,204,310.00', note: '96 invoices' }
	];
	const noop = () => {};
</script>

<PageHeader title="Overview" description="Accounts payable position as at close of business yesterday.">
	{#snippet actions()}
		<Button onclick={noop}><Icon icon={Plus} size={16} /> New invoice</Button>
	{/snippet}
</PageHeader>

{#if status !== 'ready'}
	<StateNotice {status} onretry={noop} loadingLabel="Loading overview" />
{:else}
	<dl class="strip">
		{#each figures as f (f.label)}
			<div>
				<dt>{f.label}</dt>
				<dd>{f.value}</dd>
				<dd class="note">{f.note}</dd>
			</div>
		{/each}
	</dl>

	<div class="grid">
		<Card title="Needs approval">
			<Table
				caption="Invoices waiting for approval"
				columns={[
					{ key: 'ref', label: 'Reference' },
					{ key: 'supplier', label: 'Supplier' },
					{ key: 'due', label: 'Due' },
					{ key: 'amount', label: 'Amount' }
				]}
				rows={[
					{ ref: 'INV-20418', supplier: 'Northwind Freight', due: '2026-11-20', amount: '£12,480.00' },
					{ ref: 'INV-20422', supplier: 'Halden Print', due: '2026-11-24', amount: '£3,150.40' },
					{ ref: 'INV-20431', supplier: 'Marlow Office Supply', due: '2026-11-30', amount: '£942.18' }
				]}
			/>
		</Card>
		<Card title="Ageing">
			<div class="age">
				<Progress label="Not yet due" value={72} showValue />
				<Progress label="1 to 30 days overdue" value={21} showValue />
				<Progress label="Over 30 days overdue" value={7} showValue />
			</div>
			<p class="status"><Pill tone="warning">17 overdue</Pill> Oldest is 41 days past due.</p>
		</Card>
	</div>
{/if}

<style>
	.strip {
		display: grid;
		grid-template-columns: repeat(auto-fit, minmax(11rem, 1fr));
		margin: 0 0 var(--space-6);
		border: var(--border-width) solid var(--border);
		background: var(--surface);
	}
	.strip > div {
		padding: var(--space-3) var(--space-4);
		border-inline-start: var(--border-width) solid var(--border);
	}
	.strip > div:first-child {
		border-inline-start: 0;
	}
	dt {
		color: var(--muted-foreground);
		font-size: var(--text-small);
	}
	dd {
		margin: 0;
		font-size: var(--text-heading);
		font-weight: 600;
		font-variant-numeric: var(--figures);
	}
	dd.note {
		font-size: var(--text-small);
		font-weight: 400;
		color: var(--muted-foreground);
	}
	.grid {
		display: grid;
		grid-template-columns: minmax(0, 2fr) minmax(0, 1fr);
		gap: var(--space-4);
	}
	@media (width < 64rem) {
		.grid {
			grid-template-columns: 1fr;
		}
	}
	.age {
		display: grid;
		gap: var(--space-3);
	}
	.status {
		margin: var(--space-4) 0 0;
		display: flex;
		align-items: center;
		gap: var(--space-2);
	}
</style>
