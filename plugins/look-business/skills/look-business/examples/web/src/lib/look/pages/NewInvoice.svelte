<!-- A form page, in this look: compact labelled fields in one Card, an error summary that links to
     the field, one named primary action and a plain Cancel. A starting point, then yours: replace
     the fields and the validation, and save in `onsubmit`. Validate on the server too. Needs the
     kit's FormPage (copy it from the kit-shadcn skill). -->
<script lang="ts">
	import Field from '../../kit/Field.svelte';
	import FormPage from '../../kit/pages/FormPage.svelte';

	let { errors = [] }: { errors?: Array<{ field?: string; message: string }> } = $props();

	let supplier = $state('');
	let reference = $state('');
	let amount = $state('');
	let due = $state('');
	const error = (field: string) => errors.find((e) => e.field === field)?.message;
	async function save() {
		// Call the API here, then go to the new invoice.
	}
</script>

<FormPage
	title="New invoice"
	description="Record an invoice received from a supplier."
	submitLabel="Create invoice"
	cancelHref="/invoices"
	{errors}
	onsubmit={save}
>
	<Field id="supplier" label="Supplier" bind:value={supplier} error={error('supplier')} />
	<Field id="reference" label="Supplier reference" bind:value={reference} error={error('reference')} />
	<Field
		id="amount"
		label="Amount including VAT"
		hint="In pounds, for example 1250.00."
		inputmode="decimal"
		bind:value={amount}
		error={error('amount')}
	/>
	<Field id="due" label="Due date" hint="Year, month, day: 2026-11-30." bind:value={due} error={error('due')} />
</FormPage>
