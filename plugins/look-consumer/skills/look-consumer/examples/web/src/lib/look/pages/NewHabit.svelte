<!-- A form page, in this look: a few big labelled fields in one roomy Card and one named primary
     action. A starting point, then yours: replace the fields and the validation, and save in
     `onsubmit`. Validate on the server too. Needs the kit's FormPage (copy it from the kit-shadcn
     skill). -->
<script lang="ts">
	import Field from '../../kit/Field.svelte';
	import FormPage from '../../kit/pages/FormPage.svelte';

	let { errors = [] }: { errors?: Array<{ field?: string; message: string }> } = $props();

	let name = $state('');
	let reminder = $state('');
	const error = (field: string) => errors.find((e) => e.field === field)?.message;
	async function save() {
		// Call the API here, then go to the new habit.
	}
</script>

<FormPage
	title="New habit"
	description="Pick something small you can do every time."
	submitLabel="Add habit"
	cancelHref="/habits"
	{errors}
	onsubmit={save}
>
	<Field id="name" label="Habit" hint="For example, evening walk." bind:value={name} error={error('name')} />
	<Field
		id="reminder"
		label="Reminder time"
		hint="Optional. Use 24-hour time, for example 19:30."
		bind:value={reminder}
		error={error('reminder')}
	/>
</FormPage>
