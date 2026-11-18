<!-- A detail page, in this look: the status Pill beside the reference, the facts as a dense
     description list, the lines as a table, and a
     Dialog before anything destructive. A starting point, then yours: replace the facts, the
     lines and the actions. Needs the kit's DetailPage (copy it from the kit-shadcn skill). -->
<script lang="ts">
	import Button from '../../kit/Button.svelte';
	import Card from '../../kit/Card.svelte';
	import Dialog from '../../kit/Dialog.svelte';
	import Pill from '../../kit/Pill.svelte';
	import Table from '../../kit/Table.svelte';
	import DetailPage from '../../kit/pages/DetailPage.svelte';

	let { status = 'ready' }: { status?: 'loading' | 'error' | 'ready' } = $props();

	let confirming = $state(false);
	const noop = () => {};
</script>

<DetailPage
	title="INV-20418"
	description="Northwind Freight, due 2026-11-20."
	back={{ href: '/invoices', label: 'All invoices' }}
	fields={[
		{ label: 'Supplier', value: 'Northwind Freight' },
		{ label: 'Purchase order', value: 'PO-7731' },
		{ label: 'Cost centre', value: '4120 Logistics' },
		{ label: 'Approver', value: 'Priya Shah' }
	]}
	{status}
	onretry={noop}
>
	{#snippet badge()}<Pill tone="success">Approved</Pill>{/snippet}
	{#snippet actions()}
		<Button variant="destructive" onclick={() => (confirming = true)}>Void invoice</Button>
		<Button onclick={noop}>Mark as paid</Button>
	{/snippet}
	<Card title="Lines">
		<Table
			caption="Invoice lines"
			columns={[
				{ key: 'item', label: 'Description' },
				{ key: 'qty', label: 'Quantity' },
				{ key: 'amount', label: 'Amount' }
			]}
			rows={[
				{ item: 'Pallet freight, Leeds to Bristol', qty: 8, amount: '£9,600.00' },
				{ item: 'Fuel surcharge', qty: 1, amount: '£1,200.00' },
				{ item: 'VAT at 20%', qty: 1, amount: '£1,680.00' },
				{ item: 'Total', qty: '', amount: '£12,480.00' }
			]}
		/>
	</Card>
</DetailPage>

<Dialog
	bind:open={confirming}
	title="Void this invoice?"
	description="INV-20418 is removed from the payment run. This cannot be undone."
>
	{#snippet actions()}
		<Button variant="secondary" onclick={() => (confirming = false)}>Keep invoice</Button>
		<Button variant="destructive" onclick={() => (confirming = false)}>Void invoice</Button>
	{/snippet}
</Dialog>
