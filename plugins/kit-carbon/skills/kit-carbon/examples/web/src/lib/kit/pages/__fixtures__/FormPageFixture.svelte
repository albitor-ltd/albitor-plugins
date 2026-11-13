<!-- Test support: FormPage in the shell, for formpage.test.ts alone. Delete it with FormPage.svelte
     and formpage.test.ts. Not routed; nothing links here. -->
<script lang="ts">
	import Field from '../../Field.svelte';
	import FormPage from '../FormPage.svelte';
	import PagesShowcase from './PagesShowcase.svelte';

	let {
		errors = [],
		busy = false
	}: {
		errors?: Array<{ field?: string; message: string }>;
		busy?: boolean;
	} = $props();

	let name = $state('');
	const noop = () => {};
</script>

<PagesShowcase>
	<FormPage title="New report" submitLabel="Create report" cancelHref="/reports" {errors} {busy} onsubmit={noop}>
		<Field
			id="name"
			label="Name"
			bind:value={name}
			hint="Shown in the list."
			error={errors.length ? 'Enter a name' : undefined}
		/>
		<Field label="Notes" multiline />
	</FormPage>
</PagesShowcase>
